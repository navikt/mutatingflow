DATE        := $(shell date "+%Y-%m-%d-%H%M%S")
LAST_COMMIT := $(shell git --no-pager log -1 --pretty=%h)
VERSION     := $(DATE)-$(LAST_COMMIT)
LDFLAGS     := -X github.com/navikt/mutatingflow/pkg/version.Revision=$(shell git rev-parse --short HEAD) -X github.com/navikt/mutatingflow/pkg/version.Version=$(VERSION)
ROOT_DIR    := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

build:
	go build

test:
	go test ./... -count=1

release:
	CGO_ENABLED=0 \
	GOOS=linux \
	go build -a -installsuffix cgo -o mutatingflow -ldflags "-s $(LDFLAGS)"

setup-local:
	./gen-cert.sh
	./ca-bundle.sh
	kubectl apply -f ./webhook.yaml

local:
	./mutatingflow --cert ./cert.pem --key ./key.pem

codegen-crd:
	${ROOT_DIR}/codegen/update-codegen.sh

docker:
	docker build -t navikt/mutatingflow:latest -t navikt/mutatingflow:${VERSION} .
	docker push navikt/mutatingflow:latest
	docker push navikt/mutatingflow:${VERSION}
