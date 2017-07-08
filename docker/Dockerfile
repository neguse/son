FROM debian:jessie
RUN apt-get update && apt-get install -y --no-install-recommends \
                g++ \
                gcc \
                libc6-dev \
                make \
                pkg-config \
                git \
                ca-certificates \
        && rm -rf /var/lib/apt/lists/*

ENV GOROOT /usr/local/go
ENV GOPATH /volume/go
ENV NODEROOT /usr/local/node-v8.1.3-linux-x64
ENV PATH $PATH:$NODEROOT/bin:$GOROOT/bin:$GOPATH/bin/

ADD https://golang.org/dl/go1.8.3.linux-amd64.tar.gz /usr/local

ADD https://nodejs.org/dist/v8.1.3/node-v8.1.3-linux-x64.tar.xz /usr/local
RUN npm install -g yarn && yarn global add elm@0.18.0

RUN go get github.com/jteeuwen/go-bindata/...
RUN go get github.com/elazarl/go-bindata-assetfs/...

ADD https://github.com/just-containers/s6-overlay/releases/download/v1.19.1.1/s6-overlay-amd64.tar.gz /
ENTRYPOINT ["/init"]
# CMD ["nginx"]
