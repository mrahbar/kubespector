TAG = 1.1.3-SNAPSHOT
PREFIX = kubespector
BUILD_DATE := $(shell date -u)

DOCKER_RUN = docker run --rm -u $(shell id -u):$(shell id -g) -v $(shell pwd):/go/src/github.com/mrahbar/kubernetes-inspector -w /go/src/github.com/mrahbar/kubernetes-inspector
GOLANG_CONTAINER = endianogino/golang-glide:1.9-dep
BUILD_IN_CONTAINER = 1

build-container:
ifeq ($(BUILD_IN_CONTAINER),1)
DOCKER_INSPECT_INFO := $(docker inspect $(GOLANG_CONTAINER) > /dev/null 2>&1; echo $$?)
ifeq ($(DOCKER_INSPECT_INFO),1)
docker build -t $(GOLANG_CONTAINER) -f Dockerfile-builder .
else
echo "Container $(GOLANG_CONTAINER) already build"
endif
else
echo "Nothing to do"
endif

deps: build-container
ifeq ($(BUILD_IN_CONTAINER),1)
	$(DOCKER_RUN) -e CGO_ENABLED=0 $(GOLANG_CONTAINER) dep ensure
else
	CGO_ENABLED=0 dep ensure
endif

kubernetes-inspector: deps
ifeq ($(BUILD_IN_CONTAINER),1)
	$(DOCKER_RUN) -e CGO_ENABLED=0 $(GOLANG_CONTAINER) go build -a -installsuffix cgo -ldflags "-w -X main.version=$(TAG) -X 'main.buildDate=$(BUILD_DATE)'" -o $(PREFIX)-$(TAG) *.go
else
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w' -o $(PREFIX)-$(TAG) *.go
endif

test:
ifeq ($(BUILD_IN_CONTAINER),1)
	$(DOCKER_RUN) $(GOLANG_CONTAINER) go test ./...
else
	go test ./...
endif

container: test kubernetes-inspector
	docker build -t $(PREFIX):$(TAG) .

push: container
	docker push $(PREFIX):$(TAG)

clean:
	rm -f $(PREFIX)-$(TAG)
