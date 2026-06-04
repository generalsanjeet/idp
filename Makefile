run:
	go run ./cmd/server/...

build:
	go build -o bin/idp ./cmd/server/...

test:
	go test ./... -v

port-forwards:
	kubectl port-forward -n monitoring svc/loki 3100:3100 &
	kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9091:9090 &
	kubectl port-forward -n argocd svc/argocd-server 8888:443 &
	echo "all port-forwards started"
