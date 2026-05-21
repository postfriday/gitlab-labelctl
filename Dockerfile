# syntax=docker/dockerfile:1
FROM golang:1.23-alpine3.22 AS builder

RUN apk add --no-cache git ca-certificates
WORKDIR /src

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o /gitlab-labelctl ./cmd/gitlab-labelctl

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /gitlab-labelctl /gitlab-labelctl
USER 65532:65532
ENTRYPOINT ["/gitlab-labelctl"]
