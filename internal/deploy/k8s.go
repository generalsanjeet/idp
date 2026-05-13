package deploy

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Store handles all Kubernetes operations.
type Store struct {
	client kubernetes.Interface
}

// NewStore creates a k8s client from your local kubeconfig.
func NewStore(kubeconfigPath string) (*Store, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	return &Store{client: client}, nil
}

// Deploy creates a Kubernetes Deployment for the given service.
// If the deployment already exists, it updates it.
func (s *Store) Deploy(ctx context.Context, serviceName, image string) error {
	namespace := "default"
	replicas := int32(1)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":        serviceName,
				"managed-by": "idp", // marks it as IDP-managed
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": serviceName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": serviceName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  serviceName,
							Image: image,
							Ports: []corev1.ContainerPort{
								{ContainerPort: 8080},
							},
						},
					},
				},
			},
		},
	}

	// Try to create first.
	_, err := s.client.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err == nil {
		return nil // created successfully
	}

	// If it already exists, update it instead.
	_, err = s.client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to deploy service %s: %w", serviceName, err)
	}

	return nil
}
