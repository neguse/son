#!/bin/sh -ex

GOROOT=/usr/local/go
GOPATH=/volume/go
NODEROOT=/usr/local/node-v8.1.3-linux-x64
PATH=$PATH:$NODEROOT/bin:$GOROOT/bin:$GOPATH/bin/

REPO=github.com/neguse/son

go get -u https://$REPO
cd $GOPATH/src/$REPO
cd client
elm make Main.elm --yes --output ../assets/main.js
cd ..

go-bindata-assetfs assets/...
mv bindata_assetfs.go server/

cd server
go install
