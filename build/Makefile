DOCKER_RUN = docker run --rm -v $(shell pwd):/go/src/github.com/mrahbar/kubernetes-inspector -w /go/src/github.com/mrahbar/kubernetes-inspector
GOLANG_CONTAINER = endianogino/golang-glide:1.9-dep
BUILD_IN_CONTAINER = 1
DOCKER_INSPECT_INFO := $(docker inspect $(GOLANG_CONTAINER) > /dev/null 2>&1; echo $$?)
BUILDER_RUN = python build/builder.py --debug --static --skip-dep --outdir ./

all: kubespector

build-container:
ifeq ($(BUILD_IN_CONTAINER),1)
ifeq ($(DOCKER_INSPECT_INFO),1)
	docker build -t $(GOLANG_CONTAINER) -f Dockerfile-builder .
else
	echo "Container $(GOLANG_CONTAINER) already build"
endif
else
	echo "Nothing to do"
endif

kubespector: build-container
ifeq ($(BUILD_IN_CONTAINER),1)
	$(DOCKER_RUN) $(GOLANG_CONTAINER) $(BUILDER_RUN)
else
	$(BUILDER_RUN)
endif


