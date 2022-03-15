# ethereum-service

swagger url: http://localhost:9000/api/swaggerui/


openapi gen:
 ```
docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli generate -i /local/blockchain-service.yaml -g go-server -o /local/ --additional-properties=sourceFolder=openApi,packageName=openApi
goimports -w .
 ```