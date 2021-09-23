FROM golang:1.17.1

RUN apt update && apt install -y git gcc musl-dev curl iputils-ping telnet && rm -rf /var/lib/apt/lists/*

ENV GOPATH /go
ENV GOBIN /go/bin

WORKDIR /workspace

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.42.0
RUN curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh -s -- -b /usr/bin

RUN go get -v golang.org/x/tools/gopls@v0.7.2
RUN go get -v github.com/rogpeppe/godef@v1.1.2
RUN go get -v github.com/rakyll/gotest@v0.0.6
RUN go get -v github.com/cweill/gotests@v1.6.0
RUN go get -v github.com/ramya-rao-a/go-outline@1.0.0
RUN go get -v github.com/go-delve/delve/cmd/dlv@v1.7.1
RUN GOBIN=/tmp/ go install github.com/go-delve/delve/cmd/dlv@master && mv /tmp/dlv $GOPATH/bin/dlv-dap

COPY go.mod ./
COPY go.sum ./
RUN go mod download


# Aliases
RUN echo "alias ll='ls -alF'" >> /etc/bash.bashrc
RUN echo "alias hammer-clean='go clean -testcache'" >> /etc/bash.bashrc
RUN echo "alias hammer-race-check='go run -race main.go -t https://proxytest.ddosifytech.com/ -p https -d 1 -n 1500 -a testuser:rp6RHlNMaOneEN1Inr5ZcQd8F'" >> /etc/bash.bashrc
RUN echo "alias hammer-test-n-cover='gotest -coverpkg=./... -coverprofile=coverage.out ./... && go tool cover -func coverage.out'" >> /etc/bash.bashrc
