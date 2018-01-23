BINARY = /Users/auspenskii/Documents/go/bin
GOARCH = amd64

VERSION=1.0.6-beta
COMMIT=$(shell git rev-parse HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
BUILDTIME=`date -u '+%Y-%m-%d_%I:%M:%S%p'`

# Symlink into GOPATH
GITHUB_USERNAME=antuspenskiy
BUILD_DIR=${GOPATH}/src/github.com/${GITHUB_USERNAME}/automate-vhosts/cmd

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS = -ldflags "-X main.VERSION=${VERSION} -X main.BUILDTIME=${BUILDTIME} -X main.COMMIT=${COMMIT} -X main.BRANCH=${BRANCH} -s -w"

# Build the project
all: clean vet linux

linux: 
	GOOS=linux GOARCH=${GOARCH} go build -i ${LDFLAGS} -o ${BINARY}/dbimport-linux-${GOARCH} ${BUILD_DIR}/dbimport/main.go; \
	GOOS=linux GOARCH=${GOARCH} go build -i ${LDFLAGS} -o ${BINARY}/dbdump-linux-${GOARCH} ${BUILD_DIR}/dbdump/main.go; \
	GOOS=linux GOARCH=${GOARCH} go build -i ${LDFLAGS} -o ${BINARY}/prepare-linux-${GOARCH} ${BUILD_DIR}/prepare/main.go; \
	GOOS=linux GOARCH=${GOARCH} go build -i ${LDFLAGS} -o ${BINARY}/createconfigs-linux-${GOARCH} ${BUILD_DIR}/createconfigs/main.go; \
	GOOS=linux GOARCH=${GOARCH} go build -i ${LDFLAGS} -o ${BINARY}/deletestuff-linux-${GOARCH} ${BUILD_DIR}/deletestuff/main.go;

vet:
	cd ${BUILD_DIR}; \
	go tool vet .

fmt:
	cd ${BUILD_DIR}; \
	go fmt $$(go list ./... | grep -v /vendor/)

clean:
	rm -f ${BINARY}/dbimport-linux-*
	rm -f ${BINARY}/dbdump-linux-*
	rm -f ${BINARY}/prepare-linux-*
	rm -f ${BINARY}/createconfigs-linux-*
	rm -f ${BINARY}/deletestuff-linux-*

.PHONY: linux vet fmt clean