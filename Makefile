K3S_VERSION=v1.25.0-k3s1

.PHONY: run
run: create_cluster install

### Tests

.PHONY: unit_test
unit_test:
	go test -v -cover -race -timeout=5s ./...

### Dev

.PHONY: create_cluster
create_cluster: ## run a local k3d cluster
	k3d cluster create \
		--image="rancher/k3s:$(K3S_VERSION)" \
		--registry-create=prometheus-elector-registry.localhost:0.0.0.0:5000 \
		prometheus-elector-dev

.PHONY: delete_cluster
delete_cluster:
	k3d cluster delete prometheus-elector-dev

.PHONY: install
install: install_agent install_storage

.PHONY: install_agent
install_agent: ## install an example in the current cluster
	KO_DOCKER_REPO=prometheus-elector-registry.localhost:5000 ko apply -f ./example/k8s/agent

.PHONY: install_storage
install_storage: ## install storage backend
	kubectl apply -f ./example/k8s/storage

### LOCAL
.PHONY: run_agent_local
run_agent_local: dist
	POD_NAME=${POD_NAME} go run ./cmd/main.go \
					 -lease-name lease-dev \
					 -lease-namespace default \
					 -kubeconfig /Users/${USER}/.kube/config \
					 -config ./example/config.yaml \
					 -output ./dist/config-${POD_NAME}.yaml \
					 -reload-url http://localhost:9091/-/reload

dist:
	mkdir -p dist

