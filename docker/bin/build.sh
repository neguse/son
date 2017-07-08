#!/bin/sh -ex

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
