FROM golang:1.8
MAINTAINER Mahmoud Azad <mahmoud.azad@acando.de>

ARG GLIDE_VERSION=0.12.3

RUN apt-get update \
 	&& apt-get install -y unzip --no-install-recommends \
	&& rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://github.com/Masterminds/glide/releases/download/v${GLIDE_VERSION}/glide-v${GLIDE_VERSION}-linux-amd64.zip -o glide.zip \
	&& unzip glide.zip  linux-amd64/glide \
	&& mv linux-amd64/glide /usr/local/bin \
	&& rm -rf linux-amd64 \
	&& rm glide.zip