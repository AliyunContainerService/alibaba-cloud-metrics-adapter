all: build

PREFIX?=registry.aliyuncs.com/acs
FLAGS=
ARCH?=amd64
ALL_ARCHITECTURES=amd64 arm arm64 ppc64le s390x
ML_PLATFORMS=linux/amd64,linux/arm,linux/arm64,linux/ppc64le,linux/s390x


VERSION?=v0.2.0-alpha
GIT_COMMIT:=$(shell git rev-parse --short HEAD)


fmt:
	find . -type f -name "*.go" | grep -v "./vendor*" | xargs gofmt -s -w

build: clean
	GOARCH=$(ARCH) CGO_ENABLED=0 go build -o alibaba-cloud-metrics-adapter github.com/AliyunContainerService/alibaba-cloud-metrics-adapter

sanitize:
	hack/check_gofmt.sh
	hack/run_vet.sh

test-unit: clean sanitize build

ifeq ($(ARCH),amd64)
	GOARCH=$(ARCH) go test --test.short -race ./... $(FLAGS)
else
	GOARCH=$(ARCH) go test --test.short ./... $(FLAGS)
endif

test-unit-cov: clean sanitize build
	hack/coverage.sh

docker-container:
	docker build --pull -t $(PREFIX)/alibaba-cloud-metrics-adapter-$(ARCH):$(VERSION)-$(GIT_COMMIT) -f deploy/Dockerfile .

clean:
	rm -f alibaba-cloud-metrics-adapter

.PHONY: all build sanitize test-unit test-unit-cov docker-container clean fmt
