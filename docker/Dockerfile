FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive \
    GOPROXY=direct

RUN apt-get update && \
        apt-get install -y golang-go git && \
        rm -rf /var/lib/apt/lists/*

WORKDIR /root/src
CMD ["go", "run", "main.go"]
