# Import environment variables from .env file if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

default: install

REPODIR=/tmp/tf-repo/providers

BINARY=terraform-provider-mcahr
VERSION=1.15.41

BUILD_ENV_VARS=-ldflags " -X 'main.ProviderPath=registry.opentofu.org/NVIDIA/mcahr' -X 'terraform/terraform-provider/provider.RenderedProviderName=\"NVIDIA Mission Control autonomous hardware recovery\"' -X 'terraform/terraform-provider/provider.ProviderShortName=mcahr' -X 'terraform/terraform-provider/provider.EnvVarsNamePrefix=MCAHR' -X 'terraform/terraform-provider/provider.TfLogFile=/tmp/tf_provider.log' -X 'terraform/terraform-provider/provider.DefaultUserName=NVIDIA'"


// NOTE: this only works for 64 bit linux and MacOs ("darwin")
OS=$(shell uname | tr 'A-Z' 'a-z')
SUBPATH=registry.opentofu.org/NVIDIA/mcahr/local/mcahr/$(VERSION)/$(OS)_amd64

generate:
	go generate $(BUILD_ENV_VARS)

build: format generate
	go build $(BUILD_ENV_VARS) -o ./$(BINARY)

test:
	echo unit-tests...

check:
	gofmt -l .

format:
	gofmt -w .

# NOTE: This relies on your ~/.terraformrc pointing to /tmp/tf-repo.
#   See terraformrc in the current dir
install: build
	rm -rf $(REPODIR)/*
	mkdir -p $(REPODIR)/$(SUBPATH)
	cp $(BINARY) $(REPODIR)/$(SUBPATH)/$(BINARY)

# This sets up your ~/.terraformrc (NOTE: need to re-run when the version changes)
use_local: 
	@echo 'Setting up local overrides for terraform provider in ~/.terraformrc'
	@echo 'NOTE: You need to re-run "make use_local" when the version changes."'
	@echo 'provider_installation { dev_overrides { "registry.opentofu.org/NVIDIA/mcahr" = "$(REPODIR)/$(SUBPATH)" } }' > ${HOME}/.terraformrc

use_registry: 
	@echo 'Removing ~/.terraformrc, to use the terraform registry again'
	@rm ${HOME}/.terraformrc

release: 
	GOOS=darwin  GOARCH=amd64 go build $(BUILD_ENV_VARS) -o ./bin/$(BINARY)_$(VERSION)_darwin_amd64
	GOOS=darwin  GOARCH=arm64 go build $(BUILD_ENV_VARS) -o ./bin/$(BINARY)_$(VERSION)_darwin_arm64
	GOOS=linux   GOARCH=amd64 go build $(BUILD_ENV_VARS) -o ./bin/$(BINARY)_$(VERSION)_linux_amd64
	GOOS=linux   GOARCH=arm64 go build $(BUILD_ENV_VARS) -o ./bin/$(BINARY)_$(VERSION)_linux_arm64
	GOOS=linux   GOARCH=arm   go build $(BUILD_ENV_VARS) -o ./bin/$(BINARY)_$(VERSION)_linux_arm
	GOOS=openbsd GOARCH=amd64 go build $(BUILD_ENV_VARS) -o ./bin/$(BINARY)_$(VERSION)_openbsd_amd64
	GOOS=windows GOARCH=amd64 go build $(BUILD_ENV_VARS) -o ./bin/$(BINARY)_$(VERSION)_windows_amd64

version: 
	@echo "version: ${VERSION}\nTo create a release run: \n  git tag v${VERSION}\n  git push origin v${VERSION}"
	@git tag -l | grep '^v${VERSION}$$' >/dev/null && echo "WARNING: Release already exists" || true

# Run acceptance tests
.PHONY: testacc
testacc: 
	@ TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 600s

# no checked in files should contain tokens
scan:
	find . -type f | xargs grep -l -e '[e]yJhb' || echo "scan is clean"


EXAMPLES_ROOT_PATH=./examples/resources/_root

init_ex:
	MCAHR_URL=$(MCAHR_URL) MCAHR_TOKEN=$(MCAHR_TOKEN) MCAHR_DEBUG=$(MCAHR_DEBUG) tofu -chdir=$(EXAMPLES_ROOT_PATH) init

apply_ex:
	MCAHR_URL=$(MCAHR_URL) MCAHR_TOKEN=$(MCAHR_TOKEN) MCAHR_DEBUG=$(MCAHR_DEBUG) tofu -chdir=$(EXAMPLES_ROOT_PATH) apply --auto-approve

apply_ex_na:
	MCAHR_URL=$(MCAHR_URL) MCAHR_TOKEN=$(MCAHR_TOKEN) MCAHR_DEBUG=$(MCAHR_DEBUG) tofu -chdir=$(EXAMPLES_ROOT_PATH) apply

destroy_ex:
	MCAHR_URL=$(MCAHR_URL) MCAHR_TOKEN=$(MCAHR_TOKEN) MCAHR_DEBUG=$(MCAHR_DEBUG) tofu -chdir=$(EXAMPLES_ROOT_PATH) destroy --auto-approve

destroy_ex_na:
	MCAHR_URL=$(MCAHR_URL) MCAHR_TOKEN=$(MCAHR_TOKEN) MCAHR_DEBUG=$(MCAHR_DEBUG) tofu -chdir=$(EXAMPLES_ROOT_PATH) destroy

plan_ex:
	MCAHR_URL=$(MCAHR_URL) MCAHR_TOKEN=$(MCAHR_TOKEN) MCAHR_DEBUG=$(MCAHR_DEBUG) tofu -chdir=$(EXAMPLES_ROOT_PATH) plan

.PHONY: distclean_ex
distclean_ex:
	rm -rf $(EXAMPLES_ROOT_PATH)/terraform.tfstate $(EXAMPLES_ROOT_PATH)/terraform.tfstate.backup $(EXAMPLES_ROOT_PATH)/.terraform $(EXAMPLES_ROOT_PATH)/.terraform.lock.hcl