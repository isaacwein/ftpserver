FROM golang

RUN apt-get update && \
    apt-get upgrade -y && \
    rm -rf /var/lib/apt/lists/*
ENV FTP_SERVER_ROOT=/static

WORKDIR $FTP_SERVER_ROOT

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

RUN CGO_ENABLED=0 GOOS=linux go build -o ./fileserver ./example/


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