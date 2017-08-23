
DOCKERREPO       := endianogino/simple-webserver:1.0

all: simple-webserver container push

simple-webserver:
	docker run --rm -v $(shell pwd):/go/src/github.com/mrahbar/kubernetes-inspector/resources/scaletest/simple-webserver -w /go/src/github.com/mrahbar/kubernetes-inspector/resources/scaletest/simple-webserver -e CGO_ENABLED=0 golang:1.8.3-stretch go build -a -installsuffix cgo -o simple-webserver simple-webserver.go

container: simple-webserver
	mkdir -p Dockerbuild && \
	mv -f Dockerfile Dockerbuild/ &&\
	mv -f simple-webserver Dockerbuild/ &&\
	docker build -t $(DOCKERREPO) Dockerbuild/

push: container
	docker push $(DOCKERREPO)