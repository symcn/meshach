.PHONY: build deploy

VERSION ?= v0.0.1
# Image URL to use all building/pushing image targets
IMG_ADDR ?= symcn.tencentcloudcr.com/symcn/mesh-operator

# This repo's root import path (under GOPATH).
ROOT := github.com/mesh-operator

GO_VERSION := 1.14.2
ARCH     ?= $(shell go env GOARCH)
BUILD_DATE = $(shell date +'%Y-%m-%dT%H:%M:%SZ')
COMMIT    = $(shell git rev-parse --short HEAD)
GOENV    := CGO_ENABLED=0 GOOS=$(shell uname -s | tr A-Z a-z) GOARCH=$(ARCH) GOPROXY=https://goproxy.io,direct
GO       := $(GOENV) go build

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif


docker-build:
	docker run --rm -v "$$PWD":/go/src/${ROOT} -v ${GOPATH}/pkg/mod:/go/pkg/mod -w /go/src/${ROOT} golang:${GO_VERSION} make build

docker-push:
	docker build -t ${IMG_ADDR}:${VERSION} -f ./build/Dockerfile .
	docker push ${IMG_ADDR}:${VERSION}

build:
	$(GO) -v -o build/_output/bin/mesh-operator -ldflags "-s -w -X  $(ROOT)/pkg/version.Release=$(VERSION) -X  $(ROOT)/pkg/version.Commit=$(COMMIT)   \
	-X  $(ROOT)/pkg/version.BuildDate=$(BUILD_DATE)" cmd/mesh-operator/main.go

deploy:
	kubectl apply -f deploy/service_account.yaml
	kubectl apply -f deploy/role.yaml
	kubectl apply -f deploy/role_binding.yaml
	kubectl apply -f deploy/operator.yaml

clear:
	kubectl delete -f deploy/service_account.yaml
	kubectl delete -f deploy/role.yaml
	kubectl delete -f deploy/role_binding.yaml
	kubectl delete -f deploy/operator.yaml

