FROM golang:1.18.2-alpine

RUN apk add build-base
WORKDIR /app

RUN apk update && apk add bash

COPY go.mod ./
COPY go.sum ./
COPY database/ ./database/
COPY internal/ ./internal/
COPY services/ ./services/
COPY model/ ./model/
COPY swaggerui/ ./swaggerui/
COPY utils/ ./utils/
COPY .openapi-generator-ignore ./
COPY wait-for-it.sh ./
RUN go mod download

COPY *.go ./

RUN apk add --update nodejs npm
RUN apk add openjdk11
RUN npm install @openapitools/openapi-generator-cli -g
RUN npx @openapitools/openapi-generator-cli generate -i ./swaggerui/openapi.yaml -g go-server -o ./ --additional-properties=sourceFolder=openApi,packageName=openApi
RUN npx @openapitools/openapi-generator-cli generate -i https://raw.githubusercontent.com/CHainGate/backend/main/swaggerui/internal/openapi.yaml -g go -o ./backendClientApi --ignore-file-override=.openapi-generator-ignore --additional-properties=sourceFolder=backendClientApi,packageName=backendClientApi
RUN npx @openapitools/openapi-generator-cli generate -i https://raw.githubusercontent.com/CHainGate/proxy-service/main/swaggerui/openapi.yaml -g go -o ./proxyClientApi --ignore-file-override=.openapi-generator-ignore --additional-properties=sourceFolder=proxyClientApi,packageName=proxyClientApi
RUN go install golang.org/x/tools/cmd/goimports@latest
RUN goimports -w .

RUN ["chmod", "+x", "wait-for-it.sh"]

RUN go build -o /ethereum-service

EXPOSE 9000

CMD [ "/ethereum-service" ]