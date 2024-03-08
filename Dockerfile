FROM golang

RUN apt-get update && \
    apt-get upgrade -y && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /fileserver
COPY ./ftp ./ftp
COPY ./tls ./tls
COPY ./filesystem ./filesystem
COPY ./http ./http
COPY ./sftp ./sftp
COPY ./users ./users
COPY main.go .
COPY go.mod .
COPY go.sum .
RUN go get -d -v ./... && go mod tidy

RUN CGO_ENABLED=0 GOOS=linux go build -o ./fileserver

ENV FTP_SERVER_ROOT=/static
ENV FTP_SERVER_IPV4=127.0.0.1
ENV FTP_SERVER_ADDR=:21
ENV FTPS_SERVER_ADDR=:990
ENV SFTP_SERVER_ADDR=:22
ENV PASV_MIN_PORT=30000
ENV PASV_MAX_PORT=30009
# LOG_LEVER DEBUG | INFO | WARNING | ERROR
# from "log/slog".Level package
ENV LOG_LEVER=INFO
ENV CRT_FILE=tls/ssl-rsa/localhost.rsa.crt
ENV KEY_FILE=tls/ssl-rsa/localhost.rsa.key
ENTRYPOINT ["./fileserver"]