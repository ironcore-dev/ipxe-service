GOPRIVATE ?= "github.com/onmetal/*"
IMG ?= ipxe-service:latest

GITHUB_PAT_PATH ?=
ifeq (,$(GITHUB_PAT_PATH))
GITHUB_PAT_MOUNT ?=
else
GITHUB_PAT_MOUNT ?= --secret id=github_pat,src=$(GITHUB_PAT_PATH)
endif

all: build

build:
	/usr/local/go/bin/go build -o bin/main main.go

run:
	/usr/local/go/bin/go run main.go

test:
	kubectl apply -f config/samples/ipam.onmetal.de_ips.yaml --force
	kubectl apply -f config/samples/ipam.onmetal.de_subnets.yaml --force
	kubectl apply -f config/samples/machine.onmetal.de_inventories.yaml --force
	kubectl apply -f config/samples/for_tests.yaml --force
	go test pkg -v

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
