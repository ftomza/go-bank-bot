FROM golang:1.15 AS build_base

WORKDIR /tmp/service

LABEL VER=2020.1.4

RUN git clone https://github.com/ftomza/go-bank-bot

RUN go get "github.com/DATA-DOG/go-sqlmock"
RUN go get "github.com/stretchr/testify/assert"

RUN go get -d -v ./go-bank-bot/...

RUN go test -v ./go-bank-bot/...

RUN go build -ldflags "-linkmode external -extldflags -static" -o service ./go-bank-bot/cmd/app.go

FROM alpine:3.9
RUN apk add ca-certificates

RUN mkdir /app

COPY --from=build_base /tmp/service/service /app/service

WORKDIR /app

ENTRYPOINT ["/app/service"]