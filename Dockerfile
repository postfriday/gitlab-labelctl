# syntax=docker/dockerfile:1
FROM golang:1.23-alpine3.22 AS builder

RUN apk add --no-cache git ca-certificates
WORKDIR /src

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w -X github.com/postfriday/gitlab-labelctl/pkg/version.Version=${VERSION}" -o /gitlab-labelctl ./cmd/gitlab-labelctl

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /gitlab-labelctl /gitlab-labelctl
USER 65532:65532
ENTRYPOINT ["/gitlab-labelctl"]
