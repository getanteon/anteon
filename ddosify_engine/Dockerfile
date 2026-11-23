# Builds ddosify from source: docker build -t ddosify ddosify_engine
FROM golang:1.27.4-alpine@sha256:70bd7ddea8b6cbe5d6943cde279d6b67feb10cc924fb9100d1d8757c1575b280 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
ARG VERSION=development
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.GitVersion=${VERSION}" -o /out/ddosify .

FROM gcr.io/distroless/static-debian13:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329
COPY --from=builder /out/ddosify /usr/bin/ddosify
USER 65532:65532
ENTRYPOINT ["/usr/bin/ddosify"]
CMD ["--help"]
