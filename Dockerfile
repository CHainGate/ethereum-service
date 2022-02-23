FROM golang:alpine

RUN apk add build-base
WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /ethereum-service

EXPOSE 8080

CMD [ "/ethereum-service" ]