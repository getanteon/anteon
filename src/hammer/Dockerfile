FROM golang:1.16.6-alpine

RUN apk update && apk add git && apk add gcc && apk add musl-dev

WORKDIR /workspace

RUN go get -v golang.org/x/tools/gopls@v0.7.1 
RUN go get -v github.com/rogpeppe/godef@v1.1.2
RUN go get -v github.com/golangci/golangci-lint/cmd/golangci-lint@v1.41.1
RUN go get -v github.com/rakyll/gotest@v0.0.6
RUN go get -v github.com/ramya-rao-a/go-outline@1.0.0


COPY go.mod ./
COPY go.sum ./
RUN go mod download