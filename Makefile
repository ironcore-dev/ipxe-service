GOPRIVATE ?= "github.com/onmetal/*"
IMG ?= ipxe-service:latest

ENVTEST_K8S_VERSION ?= 1.25.0
#version v0.13.1
ENVTEST_SHA = 44c5d5029cc3c19bf6e7df3f5c5943977a39637c
ARCHITECTURE = amd64
LOCAL_TESTBIN = $(CURDIR)/testbin

GITHUB_PAT_PATH ?=
ifeq (,$(GITHUB_PAT_PATH))
GITHUB_PAT_MOUNT ?=
else
GITHUB_PAT_MOUNT ?= --secret id=github_pat,src=$(GITHUB_PAT_PATH)
endif

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build

build:
	/usr/local/go/bin/go build -o bin/main main.go

run:
	/usr/local/go/bin/go run main.go

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: test
test:
	mkdir -p $(LOCAL_TESTBIN)
	GOBIN=$(GOBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_SHA)
	KUBEBUILDER_ASSETS="$(shell $(GOBIN)/setup-envtest use $(ENVTEST_K8S_VERSION) -i --bin-dir $(LOCAL_TESTBIN) -p path)" \
	ASSUME_NO_MOVING_GC_UNSAFE_RISK_IT_WITH=go1.19 \
	IPXE_DEFAULT_SECRET_PATH="../config/samples/ipxe-default-secret" \
	IPXE_DEFAULT_CONFIGMAP_PATH="../config/samples/ipxe-default-cm" \
	go test ./... -coverprofile cover.out

image: test
	podman build . -t ${IMG} --build-arg GOPRIVATE=${GOPRIVATE} --build-arg GIT_USER=${GIT_USER} --build-arg GIT_PASSWORD=${GIT_PASSWORD}

docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} --build-arg GOPRIVATE=${GOPRIVATE} $(GITHUB_PAT_MOUNT) .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

KUSTOMIZE = $(shell pwd)/bin/kustomize
.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
  $(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.4.1)

install: kustomize ## Install services into the K8s cluster specified in ~/.kube/config. This requires IMG to be available for the cluster.
	#cd config/default/server && $(KUSTOMIZE) edit set image apiserver=${IMG}
	kubectl apply -k config/default
