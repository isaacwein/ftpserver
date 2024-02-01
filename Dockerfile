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

ENV SERVER_ROOT=/static
ENV FTP_SERVER_ROOT=:21

ENTRYPOINT ["./ftpserver"]