#!/bin/bash

#export GOPATH=`pwd`
export GOBIN=`pwd`/../bin

go build -o ../bin/recorder recorder.go
go build -o ../bin/replayer replayer.go
go build -o ../bin/spec spec.go
go build -o ../bin/envhash envhash.go
