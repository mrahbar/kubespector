all: push

TAG = 1.0.1-SNAPSHOT
PREFIX = kubernetes-ingress
BUILD_DATE := $(shell date -u)

DOCKER_RUN = docker run --rm -v $(shell pwd):/go/src/github.com/mrahbar/kubernetes-inspector -w /go/src/github.com/mrahbar/kubernetes-inspector
GOLANG_CONTAINER = golang-glide:1.8
BUILD_IN_CONTAINER = 1


kubernetes-inspector:
ifeq ($(BUILD_IN_CONTAINER),1)
	docker build -t $(GOLANG_CONTAINER) .
	$(DOCKER_RUN) -e CGO_ENABLED=0 $(GOLANG_CONTAINER) glide install
	$(DOCKER_RUN) -e CGO_ENABLED=0 $(GOLANG_CONTAINER) go build -a -installsuffix cgo -ldflags "-w -X main.version=$(TAG) -X 'main.buildDate=$(BUILD_DATE)'" -o kubernetes-inspector-$(TAG) *.go
else
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w' -o kubernetes-inspector *.go
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

osx:
ifeq ($(BUILD_IN_CONTAINER),1)
	$(DOCKER_RUN) -e CGO_ENABLED=0 -e GOOS=darwin $(GOLANG_CONTAINER) go build -a -installsuffix cgo -ldflags '-w' -o kubernetes-inspector *.go
else
	CGO_ENABLED=0 GOOS=darwin go build -a -installsuffix cgo -ldflags '-w' -o osx-kubernetes-inspector *.go
endif

clean:
	rm -f kubernetes-inspector
