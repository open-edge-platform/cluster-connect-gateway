# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

SHELL := bash -eu -o pipefail

# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION            ?= $(shell cat VERSION)
GIT_HASH_SHORT     := $(shell git rev-parse --short=8 HEAD)
VERSION_DEV_SUFFIX := ${GIT_HASH_SHORT}

# Add an identifying suffix for `-dev` builds only.
# Release build versions are verified as unique by the CI build process.
ifeq ($(findstring -dev,$(VERSION)), -dev)
        VERSION := $(VERSION)-$(VERSION_DEV_SUFFIX)
endif

# Add VERSION_SUFFIX as suffix if specified and not empty
ifneq ($(strip $(VERSION_SUFFIX)),)
	VERSION := $(VERSION)$(VERSION_SUFFIX)
endif

HELM_VERSION ?= ${VERSION}

REGISTRY         ?= 080137407410.dkr.ecr.us-west-2.amazonaws.com
REGISTRY_NO_AUTH ?= edge-orch
REPOSITORY       ?= cluster

DOCKER_TAG              ?= ${VERSION}
DOCKER_IMAGE_CONTROLLER ?= ${REGISTRY}/${REGISTRY_NO_AUTH}/${REPOSITORY}/connect-controller:${DOCKER_TAG}
DOCKER_IMAGE_GATEWAY    ?= ${REGISTRY}/${REGISTRY_NO_AUTH}/${REPOSITORY}/connect-gateway:${DOCKER_TAG}
DOCKER_IMAGE_AGENT      ?= ${REGISTRY}/${REGISTRY_NO_AUTH}/${REPOSITORY}/connect-agent:${DOCKER_TAG}
DOCKER_IMAGE_AGENT_LATEST ?= ${REGISTRY}/${REGISTRY_NO_AUTH}/${REPOSITORY}/connect-agent:latest

## Labels to add Docker/Helm/Service CI meta-data.
LABEL_SOURCE       ?= $(shell git remote get-url $(shell git remote) | sed 's|://[^@/]*@|://|g') # Strip credentials from git URLs
LABEL_REVISION     = $(shell git rev-parse HEAD)
LABEL_CREATED      ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")

DOCKER_LABEL_ARGS  ?= \
	--build-arg org_oci_version="${VERSION}" \
	--build-arg org_oci_source="${LABEL_SOURCE}" \
	--build-arg org_oci_revision="${LABEL_REVISION}" \
	--build-arg org_oci_created="${LABEL_CREATED}"

# Docker Build arguments
DOCKER_BUILD_ARGS ?= \
	--build-arg http_proxy="$(http_proxy)" --build-arg https_proxy="$(https_proxy)" \
	--build-arg no_proxy="$(no_proxy)" --build-arg HTTP_PROXY="$(http_proxy)" \
	--build-arg HTTPS_PROXY="$(https_proxy)" --build-arg NO_PROXY="$(no_proxy)"

GOARCH       := $(shell go env GOARCH)
GOEXTRAFLAGS := -trimpath -gcflags="all=-spectre=all -N -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w"
ifeq ($(GOARCH),arm64)
  GOEXTRAFLAGS := -trimpath -gcflags="all=-spectre= -N -l" -asmflags="all=-spectre=" -ldflags="all=-s -w"
endif

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.31.0

## Virtual environment name
VENV_NAME = venv-env

# GoCov versions
GOLANG_GOCOV_VERSION := latest
GOLANG_GOCOV_XML_VERSION := latest
PKG := github.com/open-edge-platform/cluster-connect-gateway
TEST_PATHS := ./internal/...

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

.PHONY: all
all: help

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest gocov helm-test go-test ## Run tests.

.PHONY: go-test
go-test: ## Run go test against code
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ${TEST_PATHS}) -v -race -gcflags -l -coverprofile cover.out -covermode atomic -short
	${GOBIN}/gocov convert cover.out | ${GOBIN}/gocov-xml > coverage.xml
	go tool cover -html=cover.out -o coverage.html

.PHONY: mocks
mocks: ## Generate mock files for unit test using mockery
	mockery --version || go install github.com/vektra/mockery/v2@latest
	go generate $(PKG)/internal/...

.PHONY: coverage
coverage: ## Generate test coverage report.
	echo "TODO: coverage target not implemented"

# TODO(user): To use a different vendor for e2e tests, modify the setup under 'tests/e2e'.
# The default setup assumes Kind is pre-installed and builds/loads the Manager Docker image locally.
# Prometheus and CertManager are installed by default; skip with:
# - PROMETHEUS_INSTALL_SKIP=true
# - CERT_MANAGER_INSTALL_SKIP=true
# To skip Kind cleanup after the test, set the environment variable:
# - SKIP_KIND_CLEANUP=true

# TODO: Fix it
.PHONY: test-e2e
test-e2e: manifests generate fmt vet destroy ## Run the e2e tests. Expected an isolated environment using Kind.
	cd test && SKIP_KIND_CLEANUP=true ./local-e2e.sh

.PHONY: lint
lint: golangci-lint yamllint helmlint ## Run linters
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-config
lint-config: golangci-lint ## Verify golangci-lint linter configuration
	$(GOLANGCI_LINT) config verify

.PHONY: dependency-check-ci
dependency-check: ## Empty for now

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/controller cmd/connect-controller/main.go
	go build -o bin/gateway cmd/connect-gateway/main.go

.PHONY: build-controller
build-controller: ## Build the controller binary.
	go build -o bin/connect-controller ${GOEXTRAFLAGS} cmd/connect-controller/main.go

.PHONY: build-gateway
build-gateway: ## Build the gateway binary.
	go build -o bin/connect-gateway ${GOEXTRAFLAGS} cmd/connect-gateway/main.go

.PHONY: build-agent
build-agent: ## Build the agent binary.
	go build -o bin/connect-agent ${GOEXTRAFLAGS} cmd/connect-agent/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/connect-controller/main.go

.PHONY: vendor
vendor:  ## Build vendor directory of module dependencies.
	GOPRIVATE=* go mod vendor

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
DOCKER_ENV := DOCKER_BUILDKIT=1
docker-build: vendor docker-build-controller docker-build-gateway docker-build-agent ## Build docker images.

.PHONY: docker-build-controller
docker-build-controller: vendor
	$(CONTAINER_TOOL) build -t ${DOCKER_IMAGE_CONTROLLER} . -f build/Dockerfile.connect-controller ${DOCKER_BUILD_ARGS} ${DOCKER_LABEL_ARGS}

.PHONY: docker-build-gateway
docker-build-gateway: vendor
	$(CONTAINER_TOOL) build -t ${DOCKER_IMAGE_GATEWAY} . -f build/Dockerfile.connect-gateway ${DOCKER_BUILD_ARGS} ${DOCKER_LABEL_ARGS}

.PHONY: docker-build-agent
docker-build-agent: vendor
	$(CONTAINER_TOOL) build -t ${DOCKER_IMAGE_AGENT} . -f build/Dockerfile.connect-agent ${DOCKER_BUILD_ARGS} ${DOCKER_LABEL_ARGS}
	$(CONTAINER_TOOL) tag ${DOCKER_IMAGE_AGENT} ${DOCKER_IMAGE_AGENT_LATEST}

.PHONY: docker-push
docker-push: docker-push-controller docker-push-gateway docker-push-agent ## Push docker images.

.PHONY: docker-push-controller
docker-push-controller: ## Push docker image with the controller.
	$(CONTAINER_TOOL) push ${DOCKER_IMAGE_CONTROLLER}

.PHONY: docker-push-gateway
docker-push-gateway: ## Push docker image with the gateway.
	$(CONTAINER_TOOL) push ${DOCKER_IMAGE_GATEWAY}

.PHONY: docker-push-agent
docker-push-agent: ## Push docker image with the agent.
	$(CONTAINER_TOOL) push ${DOCKER_IMAGE_AGENT}
	$(CONTAINER_TOOL) push ${DOCKER_IMAGE_AGENT_LATEST}

.PHONY: docker-list
docker-list: ## Print name of docker container images
	@echo "images:"
	@echo "  connect-controller:"
	@echo "    name: '$(DOCKER_IMAGE_CONTROLLER)'"
	@echo "    version: '$(VERSION)'"
	@echo "    gitTagPrefix: 'v'"
	@echo "    buildTarget: 'docker-build-controller'"
	@echo "  connect-gateway:"
	@echo "    name: '$(DOCKER_IMAGE_GATEWAY)'"
	@echo "    version: '$(VERSION)'"
	@echo "    gitTagPrefix: 'v'"
	@echo "    buildTarget: 'docker-build-gateway'"
	@echo "  connect-agent:"
	@echo "    name: '$(DOCKER_IMAGE_AGENT)'"
	@echo "    version: '$(VERSION)'"
	@echo "    gitTagPrefix: 'v'"
	@echo "    buildTarget: 'docker-build-agent'"

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name cluster-connect-gateway-builder
	$(CONTAINER_TOOL) buildx use cluster-connect-gateway-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm cluster-connect-gateway-builder
	rm Dockerfile.cross

.PHONY: build-installer
build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > dist/install.yaml

HELM_DIRS = $(shell find ./deployment/charts -maxdepth 1 -mindepth 1 -type d -print )
HELM_PKGS = $(shell find . -name "*.tgz" -maxdepth 1 -mindepth 1 -type f -print )

.PHONY: helm-clean
helm-clean: ## Clean helm chart build annotations.
	for d in $(HELM_DIRS); do \
		yq eval -i '.version = "0.0.0"' $$d/Chart.yaml; \
		yq eval -i 'del(.appVersion)' $$d/Chart.yaml; \
		yq eval -i 'del(.annotations.revision)' $$d/Chart.yaml; \
		yq eval -i 'del(.annotations.created)' $$d/Chart.yaml; \
	done
	rm -f $(HELM_PKGS)

.PHONY: helm-test
helm-test: ## Template the charts.
	for d in $(HELM_DIRS); do \
		helm template test $$d; \
	done

.PHONY: helm-build
helm-build: ## Package helm charts.
	for d in $(HELM_DIRS); do \
		yq eval -i '.version = "${HELM_VERSION}"' $$d/Chart.yaml; \
		yq eval -i '.appVersion = "${VERSION}"' $$d/Chart.yaml; \
		yq eval -i '.annotations.revision = "${LABEL_REVISION}"' $$d/Chart.yaml; \
		yq eval -i '.annotations.created = "${LABEL_CREATED}"' $$d/Chart.yaml; \
		helm package --app-version=${VERSION} --version=${HELM_VERSION} --debug -u $$d; \
	done

.PHONY: helm-push
helm-push: ## Push helm charts.
	for c in $(HELM_PKGS); do helm push $$c oci://$(REGISTRY)/$(REGISTRY_NO_AUTH)/$(REPOSITORY)/charts; done

.PHONY: helm-list
helm-list:
	@echo "charts:"
	@for d in $(HELM_DIRS); do \
    cname=$$(grep "^name:" "$$d/Chart.yaml" | cut -d " " -f 2) ;\
    echo "  $$cname:" ;\
    echo -n "    "; grep "^version" "$$d/Chart.yaml"  ;\
    echo "    gitTagPrefix: 'v'" ;\
    echo "    outDir: '.'" ;\
  done

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint

## Tool Versions
KUSTOMIZE_VERSION ?= v5.5.0
CONTROLLER_TOOLS_VERSION ?= v0.17.0
#ENVTEST_VERSION is the version of controller-runtime release branch to fetch the envtest setup script (i.e. release-0.20)
ENVTEST_VERSION ?= $(shell go list -m -f "{{ .Version }}" sigs.k8s.io/controller-runtime | awk -F'[v.]' '{printf "release-%d.%d", $$2, $$3}')
#ENVTEST_K8S_VERSION is the version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31)
ENVTEST_K8S_VERSION ?= $(shell go list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $$3}')
GOLANGCI_LINT_VERSION ?= v1.62.2

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: setup-envtest
setup-envtest: envtest ## Download the binaries required for ENVTEST in the local bin directory.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

##@ Standard targets

.PHONY: cobertura
cobertura:
	go install github.com/boumenot/gocover-cobertura@latest

.PHONY: gocov
gocov:
	go install github.com/axw/gocov/gocov@${GOLANG_GOCOV_VERSION}
	go install github.com/AlekSi/gocov-xml@${GOLANG_GOCOV_XML_VERSION}

$(VENV_NAME): requirements.txt
	echo "Creating virtualenv $@"
	python3 -m venv $@;\
	. ./$@/bin/activate; set -u;\
	python3 -m pip install --upgrade pip;\
	python3 -m pip install -r requirements.txt

.PHONY: license
license: $(VENV_NAME) ## Check licensing with the reuse tool.
	## Check licensing with the reuse tool.
	. ./$</bin/activate; set -u;\
	reuse --version;\
	reuse --root . lint

.PHONY: golint
golint: lint ## Lint Go files.

.PHONY: helmlint
helmlint: ## Lint Helm charts.
	helm lint ./deployment/charts/*

YAML_FILES := $(shell find . -path './venv-env' -prune -o -type f \( -name '*.yaml' -o -name '*.yml' \) -print )

.PHONY: yamllint
yamllint: $(VENV_NAME) ## Lint YAML files.
	. ./$</bin/activate; set -u;\
	yamllint --version;\
	yamllint -c .yamllint -s $(YAML_FILES)

##@ Local KinD test targets

KIND_CLUSTER ?= local-e2e
NAMESPACE ?= e2e-test
HELM_VARS ?=

RS_REGISTRY		 ?= registry-rs.edgeorchestration.intel.com
RS_DOCKER_IMAGE_CONTROLLER ?= ${RS_REGISTRY}/${REGISTRY_NO_AUTH}/${REPOSITORY}/connect-controller:${DOCKER_TAG}
RS_DOCKER_IMAGE_GATEWAY    ?= ${RS_REGISTRY}/${REGISTRY_NO_AUTH}/${REPOSITORY}/connect-gateway:${DOCKER_TAG}
RS_DOCKER_IMAGE_AGENT      ?= ${RS_REGISTRY}/${REGISTRY_NO_AUTH}/${REPOSITORY}/connect-agent:${DOCKER_TAG}

.PHONY: docker-load
docker-load: ## Load docker images into the KinD cluster.
	docker tag ${DOCKER_IMAGE_CONTROLLER} ${RS_DOCKER_IMAGE_CONTROLLER}
	docker tag ${DOCKER_IMAGE_GATEWAY} ${RS_DOCKER_IMAGE_GATEWAY}
	docker tag ${DOCKER_IMAGE_AGENT} ${RS_DOCKER_IMAGE_AGENT}
	kind load docker-image ${RS_DOCKER_IMAGE_CONTROLLER} --name ${KIND_CLUSTER}
	kind load docker-image ${RS_DOCKER_IMAGE_GATEWAY} --name ${KIND_CLUSTER}
	kind load docker-image ${RS_DOCKER_IMAGE_AGENT} --name ${KIND_CLUSTER}

.PHONY: helm-install
helm-install: helm-build ## Install helm charts to the K8s cluster in current context.
	helm upgrade --install cluster-connect-gateway-crd deployment/charts/cluster-connect-gateway-crd -n ${NAMESPACE} --create-namespace
	helm upgrade --install cluster-connect-gateway deployment/charts/cluster-connect-gateway -n ${NAMESPACE} --create-namespace \
		${HELM_VARS}

.PHONY: helm-uninstall
helm-uninstall: ## Uninstall helm charts from the K8s cluster in current context.
	helm uninstall cluster-connect-gateway cluster-connect-gateway-crd -n ${NAMESPACE}

.PHONY: redeploy
redeploy: docker-build docker-load ## Redeploy the pod with the latest codes.
	kubectl delete po -l app.kubernetes.io/instance=cluster-connect-gateway -n ${NAMESPACE}

.PHONY: destroy
destroy: ## Destroy the KinD cluster.
	kind delete cluster --name ${KIND_CLUSTER}

##@ Coder

.PHONY: coder-redeploy
coder-redeploy: KIND_CLUSTER=kind
coder-redeploy: helm-build docker-build docker-load ## Redeploy connect-gateway helm chart in the Coder KinD cluster
	kubectl patch application -n dev root-app --type=merge -p '{"spec":{"syncPolicy":{"automated":{"selfHeal":false}}}}'
	kubectl delete application -n dev cluster-connect-gateway --ignore-not-found=true
	kubectl delete crd clusterconnects.cluster.edge-orchestrator.intel.com --ignore-not-found=true
	helm upgrade --install cluster-connect-gateway-crd deployment/charts/cluster-connect-gateway-crd -n orch-cluster --create-namespace
	helm upgrade --install cluster-connect-gateway deployment/charts/cluster-connect-gateway -n orch-cluster --create-namespace \
		${HELM_VARS}

.PHONY: coder-rebuild-gateway
coder-rebuild-gateway: KIND_CLUSTER=kind
coder-rebuild-gateway: docker-build-gateway docker-load ## Rebuild connect-gateway container from source and redeploy
	kubectl delete po -l app.kubernetes.io/component=gateway -n orch-cluster

.PHONY: coder-rebuild-controller
coder-rebuild-controller: KIND_CLUSTER=kind
coder-rebuild-controller: docker-build-controller docker-load ## Rebuild connect-controller container from source and redeploy
	kubectl delete po -l app.kubernetes.io/component=controller -n orch-cluster
