FROM golang:alpine

RUN apk add build-base
WORKDIR /app

COPY go.mod ./
COPY go.sum ./
COPY services/ ./services/
COPY swaggerui/ ./swaggerui/
COPY .openapi-generator-ignore ./
COPY blockchain-service.yaml ./
RUN go mod download

COPY *.go ./

RUN apk add --update nodejs npm
RUN apk add openjdk11
RUN npm install @openapitools/openapi-generator-cli -g
RUN npx @openapitools/openapi-generator-cli generate -i ./blockchain-service.yaml -g go-server -o ./ --additional-properties=sourceFolder=openApi,packageName=openApi
RUN go install golang.org/x/tools/cmd/goimports@latest
RUN goimports -w .

RUN ["chmod", "+x", "wait-for-it.sh"]

RUN go build -o /ethereum-service

EXPOSE 9000

CMD [ "/ethereum-service" ]