# syntax=docker/dockerfile:1
ARG GO_VERSION=1.22
# minor version is 1.22 because of the router

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS builder

RUN apt-get update && \
    apt-get upgrade -y && \
    rm -rf /var/lib/apt/lists/*

ARG TARGETOS
ARG TARGETARCH


WORKDIR /fileserver
COPY ./ftp ./ftp
COPY ./filesystem ./filesystem
COPY ./httphandler ./httphandler
COPY ./sftp ./sftp
COPY ./users ./users
COPY ./tools ./tools
COPY ./example ./example
COPY ./keys ./keys
COPY go.mod .
COPY go.sum .
RUN go get -d -v ./... && go mod tidy

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o ./fileserver ./example/

FROM --platform=$BUILDPLATFORM scratch

ENV FTP_SERVER_ROOT=/static
WORKDIR $FTP_SERVER_ROOT

WORKDIR /fileserver
COPY --from=builder /fileserver/fileserver .
COPY --from=builder /fileserver/example/tls/ssl-rsa/localhost.rsa.crt /fileserver/example/tls/ssl-rsa/
COPY --from=builder /fileserver/example/tls/ssl-rsa/localhost.rsa.key /fileserver/example/tls/ssl-rsa/


ENV FTP_SERVER_IPV4=127.0.0.1
ENV FTP_SERVER_ADDR=:21
ENV FTPS_SERVER_ADDR=:990
ENV SFTP_SERVER_ADDR=:22
ENV HTTP_SERVER_ADDR=:80
ENV HTTPS_SERVER_ADDR=:443
ENV PASV_MIN_PORT=30000
ENV PASV_MAX_PORT=30009
# LOG_LEVER DEBUG | INFO | WARNING | ERROR
# from "log/slog".Level package
ENV LOG_LEVEL=INFO
ENV CRT_FILE=example/tls/ssl-rsa/localhost.rsa.crt
ENV KEY_FILE=example/tls/ssl-rsa/localhost.rsa.key
ENTRYPOINT ["./fileserver"]