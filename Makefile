K3S_VERSION = v1.25.0-k3s1
RELEASE_TAG ?= dev

### Tests

.PHONY: test
test: unit_test

.PHONY: unit_test
unit_test:
	go test -v -cover -race -timeout=5s ./...

### Dev

.PHONY: run
run: create_cluster install

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
install: install_agent_example install_storage

.PHONY: install_agent_example
install_agent_example: ## install an example in the current cluster
	helm template \
		--set elector.image.devRef=ko://github.com/jlevesy/prometheus-elector/cmd \
		--set prometheus.image.repository=jlevesy/prometheus \
		--set prometheus.image.tag=allow-agent-no-remote-write \
		-f ./example/k8s/agent-values.yaml \
		prometheus-elector-dev ./helm | KO_DOCKER_REPO=prometheus-elector-registry.localhost:5000 ko apply -B -t dev -f -

.PHONY: install_storage
install_storage: ## install storage backend
	kubectl apply -f ./example/k8s/storage

.PHONY: run_agent_local
run_agent_local: dist
	POD_NAME=${POD_NAME} go run ./cmd \
					 -lease-name lease-dev \
					 -lease-namespace default \
					 -kubeconfig /Users/${USER}/.kube/config \
					 -config ./example/config.yaml \
					 -output ./dist/config-${POD_NAME}.yaml \
					 -notify-http-url http://localhost:9091/-/reload

dist:
	mkdir -p dist

### CI

.PHONY: ci_release
ci_release: ci_create_release ci_push_image

.PHONY: ci_create_release
ci_create_release:
	gh release create $(RELEASE_TAG) --generate-notes

.PHONY: ci_push_image
ci_push_image:
	ko publish --bare -t $(RELEASE_TAG) ./cmd
