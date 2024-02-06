FROM golang

RUN apt-get update && \
    apt-get upgrade -y && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY ./server ./server
COPY ./users ./users
COPY main.go .
COPY go.mod .

RUN CGO_ENABLED=0 GOOS=linux go build -o ftpserver

ENV FTP_SERVER_ROOT=/static
ENV FTP_SERVER_IPV4=127.0.0.1
ENV FTP_SERVER_PORT=:21
ENV PASV_MIN_PORT=30000
ENV PASV_MAX_PORT=30009
ENTRYPOINT ["./ftpserver"]