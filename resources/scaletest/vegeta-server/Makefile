DOCKERREPO       := endianogino/vegeta-server:1.0
GOLANG_CONTAINER = endianogino/golang-glide:1.9-dep
DOCKER_RUN = docker run --rm -e CGO_ENABLED=0 -v $(shell pwd):/go/src/github.com/mrahbar/kubernetes-inspector/resources/scaletest/vegeta-server -w /go/src/github.com/mrahbar/kubernetes-inspector/resources/scaletest/vegeta-server
all: push
DOCKER_INSPECT_INFO := $(docker inspect $(GOLANG_CONTAINER) > /dev/null 2>&1; echo $$?)

build-container:
ifeq ($(DOCKER_INSPECT_INFO),1)
	docker build -t $(GOLANG_CONTAINER) -f Dockerfile-builder .
else
	echo "Container $(GOLANG_CONTAINER) already build"
endif

deps: build-container
	$(DOCKER_RUN) $(GOLANG_CONTAINER) dep ensure -v

vegeta-server: deps
	$(DOCKER_RUN) $(GOLANG_CONTAINER) go build -a -installsuffix cgo -o vegeta-server vegeta-server.go

container: vegeta-server
	mkdir -p Dockerbuild && \
	cp -f Dockerfile Dockerbuild/ &&\
	cp -f vegeta-server Dockerbuild/ &&\
	docker build -t $(DOCKERREPO) Dockerbuild/

push: container
	docker push $(DOCKERREPO)