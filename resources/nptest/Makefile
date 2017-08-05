
DOCKERREPO       := endianogino/netperf:1.1

all: nptests docker

nptests:
	docker run --rm -v $(shell pwd):/go/src/github.com/mrahbar/kubernetes-inspector/resources/nptest -w /go/src/github.com/mrahbar/kubernetes-inspector/resources/nptest -e CGO_ENABLED=0 golang:1.8.3-stretch go build -a -installsuffix cgo -o nptests main.go

docker: nptests
	mkdir -p Dockerbuild && \
	mv -f nptests Dockerbuild/ &&\
	docker build -t $(DOCKERREPO) Dockerbuild/ 
