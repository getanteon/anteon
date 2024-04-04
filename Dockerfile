FROM golang:1.22.2-alpine as builder
WORKDIR /app
COPY . ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/ddosify main.go


FROM alpine:3.15.4
ENV ENV="/root/.ashrc"
WORKDIR /root
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/ddosify /bin/

COPY assets/ddosify.profile /tmp/profile
RUN cat /tmp/profile >> "$ENV"
