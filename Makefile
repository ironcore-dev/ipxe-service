IMG ?= ipxe-service:latest

ENVTEST_K8S_VERSION ?= 1.25.0
#version v0.13.1
ENVTEST_SHA = 44c5d5029cc3c19bf6e7df3f5c5943977a39637c
ARCHITECTURE = amd64
LOCAL_TESTBIN = $(CURDIR)/testbin

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: build

build:
	go build -o bin/main main.go

run:
	go run main.go

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: setup-envtest
setup-envtest:
	mkdir -p $(LOCAL_TESTBIN)
	GOBIN=$(LOCAL_TESTBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_SHA)
	$(LOCAL_TESTBIN)/setup-envtest use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCAL_TESTBIN)

.PHONY: test
test: setup-envtest fmt vet checklicense
	KUBEBUILDER_ASSETS="$(shell $(LOCAL_TESTBIN)/setup-envtest use $(ENVTEST_K8S_VERSION) -i --bin-dir $(LOCAL_TESTBIN) -p path)" \
	IPXE_DEFAULT_SECRET_PATH="../config/samples/ipxe-default-secret" \
	IPXE_DEFAULT_CONFIGMAP_PATH="../config/samples/ipxe-default-cm" \
	go test ./... -coverprofile cover.out

docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} --build-arpg .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

KUSTOMIZE = $(shell pwd)/bin/kustomize
.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
  $(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.4.1)

install: kustomize ## Install services into the K8s cluster specified in ~/.kube/config. This requires IMG to be available for the cluster.
	#cd config/default/server && $(KUSTOMIZE) edit set image apiserver=${IMG}
	kubectl apply -k config/default

.PHONY: addlicense
addlicense: ## Add license headers to all go files.
	find . -name '*.go' -exec go run github.com/google/addlicense -f hack/license-header.txt {} +

.PHONY: checklicense
checklicense: ## Check that every file has a license header present.
	find . -name '*.go' -exec go run github.com/google/addlicense  -check -c 'IronCore authors' {} +
