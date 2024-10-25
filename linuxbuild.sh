#!/bin/bash -x
set GOOS=linux
set GOARCH=amd64
go build .
(cd mediamtxlaunch; go build .; )
cp mediamtxlaunch/camerahls camerahls
rm linuxmediamtx.zip
7z a -tzip -r linuxmediamtx mediamtx  camerahls mediamtx.yml web -x!camerahls