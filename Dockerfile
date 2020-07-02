FROM golang:alpine
LABEL maintainer="Talha Altinel <talhaaltinel@hotmail.com>"

ENV GO111MODULE=on

RUN apk update && apk add bash git gcc g++ libc-dev

RUN mkdir /chat_service_app
RUN mkdir -p /chat_service_app/proto


WORKDIR /chat_service_app

COPY ./proto/chat.pb.go /chat_service_app/proto
COPY ./server/server.go /chat_service_app

COPY go.mod go.sum ./
RUN go mod download

RUN go build -o chat_service_app .
EXPOSE 8080
CMD ["./chat_service_app"]
