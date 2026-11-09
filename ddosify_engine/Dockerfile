# Builds ddosify from source: docker build -t ddosify ddosify_engine
FROM golang:1.27.3-alpine@sha256:fa3d596898d9f126af7c01d14ee7597bab3f39cccf392ef36b69fe77cd681f18 AS builder
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
