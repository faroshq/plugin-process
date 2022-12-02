REPO ?= quay.io/faroshq/
TAG_NAME ?= $(shell git describe --tags --abbrev=0)
JOBDATE		?= $(shell date -u +%Y-%m-%dT%H%M%SZ)
GIT_REVISION ?= $(shell git describe --tags --always --dirty)
PLUGIN_VERSION ?= v$(shell date +'%Y%m%d')

LOCALBIN ?= $(shell pwd)/bin
KUSTOMIZE ?= $(LOCALBIN)/kustomize
TOOLS_DIR=hack/tools
TOOLS_GOBIN_DIR := $(abspath $(TOOLS_DIR))
GO_INSTALL = ./hack/go-install.sh

KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_GEN_VER := v0.10.0
CONTROLLER_GEN_BIN := controller-gen

ARCH := $(shell go env GOARCH)
OS := $(shell go env GOOS)

CONTROLLER_GEN := $(TOOLS_DIR)/$(CONTROLLER_GEN_BIN)-$(CONTROLLER_GEN_VER)
export CONTROLLER_GEN # so hack scripts can use it

LDFLAGS		+= -s -w
LDFLAGS		+= -X github.com/faroshq/plugin-services/pkg/util/version.tag=$(TAG_NAME)
LDFLAGS		+= -X github.com/faroshq/plugin-services/pkg/util/version.commit=$(GIT_REVISION)
LDFLAGS		+= -X github.com/faroshq/plugin-services/pkg/util/version.buildTime=$(JOBDATE)
LDFLAGS		+= -X github.com/faroshq/plugin-services/pkg/util/version.version=$(PLUGIN_VERSION)

tools:$(CONTROLLER_GEN)
.PHONY: tools

PLUGIN_NAME_SYSTEMD ?= faros-systemd

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
$(KUSTOMIZE): ## Download kustomize locally if necessary.
	mkdir -p $(LOCALBIN)
	curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)
	touch $(KUSTOMIZE) # we download an "old" file, so make will re-download to refresh it unless we make it newer than the owning dir

$(CONTROLLER_GEN):
	GOBIN=$(TOOLS_GOBIN_DIR) $(GO_INSTALL) sigs.k8s.io/controller-tools/cmd/controller-gen $(CONTROLLER_GEN_BIN) $(CONTROLLER_GEN_VER)

codegen: $(CONTROLLER_GEN)
	go mod download
	./hack/update-codegen.sh
.PHONY: codegen

.PHONY: apiresourceschemas
apiresourceschemas: $(KUSTOMIZE) ## Convert CRDs from config/crds to APIResourceSchemas. Specify PLUGIN_VERSION as needed.
	rm -rf pkg/plugin/data/*.apiresourceschemas.yaml
	$(KUSTOMIZE) build config/crds | kubectl kcp crd snapshot -f - --prefix $(PLUGIN_VERSION) > pkg/plugin/data/$(PLUGIN_VERSION).apiresourceschemas.yaml

build: codegen apiresourceschemas
	rm -rf ./plugins/*
	go build -ldflags "$(LDFLAGS)" -o ./plugins/${PLUGIN_NAME_SYSTEMD}-${PLUGIN_VERSION}-${ARCH} ./cmd/systemd

