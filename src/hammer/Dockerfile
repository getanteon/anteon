FROM golang:1.16.6-alpine

RUN apk update && apk add git

RUN go get -v \
    golang.org/x/tools/gopls \
    github.com/ramya-rao-a/go-outline \
    golang.org/x/tools/cmd/goimports