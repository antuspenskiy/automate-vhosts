#!/bin/bash

version="1.0.1-alpha"
githash=`git rev-parse HEAD`
buildtime=`date -u '+%Y-%m-%d_%I:%M:%S%p'`

gobuild="/usr/local/Cellar/go/1.9.2/libexec/bin/go build"
gobin="/Users/auspenskii/Documents/go/bin"
gosrc="/Users/auspenskii/Documents/go/src/github.com/antuspenskiy/automate-vhosts/cmd/automate-vhosts"
env="env GOOS=linux GOARCH=amd64"

function build {
    ${env} ${gobuild} -i -ldflags "-X main.Version=${version} -X main.BuildTime=${buildtime} -X main.GitHash=${githash} -s -w" -o ${gobin}/dbimport ${gosrc}/dbimport/dbimport.go
    ${env} ${gobuild} -i -ldflags "-X main.Version=${version} -X main.BuildTime=${buildtime} -X main.GitHash=${githash} -s -w" -o ${gobin}/dbdump ${gosrc}/dbdump/dbdump.go
    ${env} ${gobuild} -i -ldflags "-X main.Version=${version} -X main.BuildTime=${buildtime} -X main.GitHash=${githash} -s -w" -o ${gobin}/prepare ${gosrc}/prepare/prepare.go
    ${env} ${gobuild} -i -ldflags "-X main.Version=${version} -X main.BuildTime=${buildtime} -X main.GitHash=${githash} -s -w" -o ${gobin}/createconfigs ${gosrc}/createconfigs/createconfigs.go
    ${env} ${gobuild} -i -ldflags "-X main.Version=${version} -X main.BuildTime=${buildtime} -X main.GitHash=${githash} -s -w" -o ${gobin}/deletestuff ${gosrc}/deletestuff/deletestuff.go
}

build

exit $?
