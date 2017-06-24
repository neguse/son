#!/bin/sh -ex

pushd client
elm make Main.elm --output ../assets/main.js
popd

go-bindata-assetfs assets/...
mv bindata_assetfs.go server/

pushd server
go build -o main *.go
popd
