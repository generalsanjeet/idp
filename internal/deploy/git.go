package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"context"
	"log/slog"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kyaml "sigs.k8s.io/yaml"
)

// Store handles all GitOps deploy operations.
type Store struct {
	repoURL   string // remote GitHub repo URL
	localPath string // where to clone the repo locally
	token     string // GitHub personal access token
	k8sClient  dynamic.Interface // dynamic client for applying ArgoCD CRDs
}

// NewStore creates a new GitOps store.
func NewStore(repoURL, localPath, token, kubeconfigPath string) (*Store, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Store{
		repoURL:   repoURL,
		localPath: localPath,
		token:     token,
		k8sClient: dynamicClient,
	}, nil
}

// Deploy updates the Helm values.yaml for a service and pushes to Git.
// ArgoCD detects the push and deploys to Kubernetes automatically.
func (s *Store) Deploy(serviceName, image string) error {
	repo, err := s.syncRepo()
	if err != nil {
		return fmt.Errorf("failed to sync repo: %w", err)
	}

	if err := s.updateValues(serviceName, image); err != nil {
		return fmt.Errorf("failed to update values: %w", err)
	}

	if err := s.commitAndPush(repo, serviceName, image); err != nil {
		return fmt.Errorf("failed to commit and push: %w", err)
	}

	return nil
}

// Bootstrap creates the initial Helm chart structure and ArgoCD
// Application manifest for a new service in the gitops repo.
// Called once when a service is first registered.
func (s *Store) Bootstrap(serviceName string) error {
	repo, err := s.syncRepo()
	if err != nil {
		return fmt.Errorf("failed to sync repo: %w", err)
	}

	if err := s.createHelmChart(serviceName); err != nil {
		return fmt.Errorf("failed to create helm chart: %w", err)
	}

	if err := s.createArgoCDApp(serviceName, s.repoURL); err != nil {
		return fmt.Errorf("failed to create argocd app: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	if err := w.AddGlob("."); err != nil {
		return fmt.Errorf("failed to stage files: %w", err)
	}

	commitMsg := fmt.Sprintf("bootstrap %s service", serviceName)
	_, err = w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "IDP Bot",
			Email: "idp@platform.internal",
			When:  time.Now(),
		},
	})
	if err != nil {
		if err.Error() == "cannot create empty commit: clean working tree" {
			// Chart already exists — still apply the ArgoCD manifest
			// in case it was deleted from the cluster.
			return s.applyArgoCDApp(context.Background(), serviceName)
		}
		return fmt.Errorf("failed to commit: %w", err)
	}

	if err := repo.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: "idp",
			Password: s.token,
		},
	}); err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to push: %w", err)
	}

	// Apply the ArgoCD Application to the cluster.
	if err := s.applyArgoCDApp(context.Background(), serviceName); err != nil {
		return fmt.Errorf("failed to apply argocd app: %w", err)
	}

	return nil
}

// createHelmChart creates the Helm chart folder structure for a service.
func (s *Store) createHelmChart(serviceName string) error {
	templatesDir := filepath.Join(s.localPath, "apps", serviceName, "helm", "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Chart.yaml — note: backtick content starts at column 0, no leading whitespace.
	chartYaml := fmt.Sprintf(`apiVersion: v2
name: %s
description: Helm chart for %s service managed by IDP
type: application
version: 0.1.0
appVersion: "1.0.0"
`, serviceName, serviceName)

	if err := os.WriteFile(
		filepath.Join(s.localPath, "apps", serviceName, "helm", "Chart.yaml"),
		[]byte(chartYaml), 0644,
	); err != nil {
		return err
	}

	// values.yaml
	valuesYaml := `image:
  repository: nginx
  tag: latest

replicaCount: 1

service:
  port: 80
`

	if err := os.WriteFile(
		filepath.Join(s.localPath, "apps", serviceName, "helm", "values.yaml"),
		[]byte(valuesYaml), 0644,
	); err != nil {
		return err
	}

	// templates/deployment.yaml
	deploymentYaml := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}
  labels:
    app: {{ .Release.Name }}
    managed-by: idp
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app: {{ .Release.Name }}
  template:
    metadata:
      labels:
        app: {{ .Release.Name }}
    spec:
      containers:
        - name: {{ .Release.Name }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          ports:
            - containerPort: {{ .Values.service.port }}
`

	if err := os.WriteFile(
		filepath.Join(templatesDir, "deployment.yaml"),
		[]byte(deploymentYaml), 0644,
	); err != nil {
		return err
	}

	// templates/service.yaml
	serviceYaml := `apiVersion: v1
kind: Service
metadata:
  name: {{ .Release.Name }}
  labels:
    app: {{ .Release.Name }}
    managed-by: idp
spec:
  selector:
    app: {{ .Release.Name }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: {{ .Values.service.port }}
`

	if err := os.WriteFile(
		filepath.Join(templatesDir, "service.yaml"),
		[]byte(serviceYaml), 0644,
	); err != nil {
		return err
	}

	return nil
}

// createArgoCDApp creates the ArgoCD Application manifest for a service.
func (s *Store) createArgoCDApp(serviceName, repoURL string) error {
	argocdDir := filepath.Join(s.localPath, "argocd")
	if err := os.MkdirAll(argocdDir, 0755); err != nil {
		return fmt.Errorf("failed to create argocd directory: %w", err)
	}

	appYaml := fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: argocd
spec:
  project: default
  source:
    repoURL: %s
    targetRevision: HEAD
    path: apps/%s/helm
  destination:
    server: https://kubernetes.default.svc
    namespace: default
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
`, serviceName, repoURL, serviceName)

	return os.WriteFile(
		filepath.Join(argocdDir, serviceName+".yaml"),
		[]byte(appYaml), 0644,
	)
}

// syncRepo clones the repo if it doesn't exist locally,
// or pulls the latest changes if it does.
func (s *Store) syncRepo() (*git.Repository, error) {
	if _, err := os.Stat(s.localPath); os.IsNotExist(err) {
		repo, err := git.PlainClone(s.localPath, false, &git.CloneOptions{
			URL: s.repoURL,
			Auth: &http.BasicAuth{
				Username: "idp",
				Password: s.token,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to clone repo: %w", err)
		}
		return repo, nil
	}

	repo, err := git.PlainOpen(s.localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	err = w.Pull(&git.PullOptions{
		Auth: &http.BasicAuth{
			Username: "idp",
			Password: s.token,
		},
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return nil, fmt.Errorf("failed to pull repo: %w", err)
	}

	return repo, nil
}

// updateValues writes a new values.yaml for the service with the updated image.
func (s *Store) updateValues(serviceName, image string) error {
	valuesPath := filepath.Join(s.localPath, "apps", serviceName, "helm", "values.yaml")

	repository, tag := parseImage(image)

	content := fmt.Sprintf(`image:
  repository: %s
  tag: %s

replicaCount: 1

service:
  port: 80
`, repository, tag)

	if err := os.WriteFile(valuesPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write values.yaml: %w", err)
	}

	return nil
}

// commitAndPush stages values.yaml, creates a commit, and pushes to GitHub.
func (s *Store) commitAndPush(repo *git.Repository, serviceName, image string) error {
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	stagePath := filepath.Join("apps", serviceName, "helm", "values.yaml")
	if _, err := w.Add(stagePath); err != nil {
		return fmt.Errorf("failed to stage file: %w", err)
	}

	commitMsg := fmt.Sprintf("deploy %s with image %s", serviceName, image)
	_, err = w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "IDP Bot",
			Email: "idp@platform.internal",
			When:  time.Now(),
		},
	})
	if err != nil {
		if err.Error() == "cannot create empty commit: clean working tree" {
			return nil
		}
		return fmt.Errorf("failed to commit: %w", err)
	}

	if err := repo.Push(&git.PushOptions{
		Auth: &http.BasicAuth{
			Username: "idp",
			Password: s.token,
		},
	}); err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// parseImage splits "nginx:1.25" into ("nginx", "1.25").
// If no tag is present, defaults to "latest".
func parseImage(image string) (repository, tag string) {
	for i := len(image) - 1; i >= 0; i-- {
		if image[i] == ':' {
			return image[:i], image[i+1:]
		}
	}
	return image, "latest"
}

// applyArgoCDApp applies the ArgoCD Application manifest to the cluster.
// This is equivalent to: kubectl apply -f argocd/service.yaml
// We use the dynamic client because ArgoCD Application is a CRD —
// not a built-in Kubernetes type.
func (s *Store) applyArgoCDApp(ctx context.Context, serviceName string) error {
	// ArgoCD Application GVR — Group, Version, Resource.
	// This identifies the CRD we're working with.
	gvr := schema.GroupVersionResource{
		Group:    "argoproj.io",
		Version:  "v1alpha1",
		Resource: "applications",
	}

	// Read the manifest file we just wrote to the gitops repo.
	manifestPath := filepath.Join(s.localPath, "argocd", serviceName+".yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read argocd manifest: %w", err)
	}

	// Parse YAML into an unstructured object.
	// Unstructured is Go's way of representing arbitrary k8s resources
	// without needing the concrete type.
	obj := &unstructured.Unstructured{}
	if err := obj.UnmarshalJSON(
		mustConvertYAMLToJSON(data),
	); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	slog.Info("applying argocd application", "service", serviceName, "namespace", "argocd")

	// Try to create the Application.
	_, err = s.k8sClient.Resource(gvr).
		Namespace("argocd").
		Create(ctx, obj, metav1.CreateOptions{})
	if err == nil {
		slog.Info("argocd application created successfully", "service", serviceName)
		return nil // created successfully
	}

	slog.Error("create failed", "error", err, "service", serviceName)

	// If it already exists, update it.
	if errors.IsAlreadyExists(err) {
		slog.Info("application already exists, updating", "service", serviceName)
		// Get existing to retrieve resourceVersion (required for updates).
		existing, err := s.k8sClient.Resource(gvr).
			Namespace("argocd").
			Get(ctx, serviceName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get existing application: %w", err)
		}
		obj.SetResourceVersion(existing.GetResourceVersion())

		_, err = s.k8sClient.Resource(gvr).
			Namespace("argocd").
			Update(ctx, obj, metav1.UpdateOptions{})
		if err != nil {
			return fmt.Errorf("failed to update application: %w", err)
		}
		slog.Info("argocd application updated successfully", "service", serviceName)
		return nil
	}

	return fmt.Errorf("failed to apply argocd application: %w", err)
}

// mustConvertYAMLToJSON converts YAML bytes to JSON bytes.
// Used because the dynamic client expects JSON.
func mustConvertYAMLToJSON(data []byte) []byte {
	// Use the k8s YAML library which handles multi-doc YAML correctly.
	json, err := kyaml.YAMLToJSON(data)
	if err != nil {
		panic(fmt.Sprintf("failed to convert yaml to json: %v", err))
	}
	return json
}
