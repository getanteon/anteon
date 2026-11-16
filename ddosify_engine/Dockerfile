# Builds ddosify from source: docker build -t ddosify ddosify_engine
FROM golang:1.26.10-alpine@sha256:a2012b4a1d1f9f7e9f44e0d39b805a901e856b1dd80b4e6ee4338db547050308 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
ARG VERSION=development
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.GitVersion=${VERSION}" -o /out/ddosify .

FROM gcr.io/distroless/static-debian13:nonroot@sha256:aafe993665bf6200db38a02c11a9351e782a84b6a45e7f33b6695638c88e3931
COPY --from=builder /out/ddosify /usr/bin/ddosify
USER 65532:65532
ENTRYPOINT ["/usr/bin/ddosify"]
CMD ["--help"]
