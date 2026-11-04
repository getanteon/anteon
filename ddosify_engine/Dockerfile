# Builds ddosify from source: docker build -t ddosify ddosify_engine
FROM golang:1.26.10-alpine@sha256:a2012b4a1d1f9f7e9f44e0d39b805a901e856b1dd80b4e6ee4338db547050308 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
ARG VERSION=development
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.GitVersion=${VERSION}" -o /out/ddosify .

FROM gcr.io/distroless/static-debian13:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329
COPY --from=builder /out/ddosify /usr/bin/ddosify
USER nonroot:nonroot
ENTRYPOINT ["/usr/bin/ddosify"]
CMD ["--help"]
