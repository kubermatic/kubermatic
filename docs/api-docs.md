## Swagger/OpenAPI docs

### Quick start giude

#### 1. Install & compile swagger
```
go get github.com/yvasiyarov/swagger
go install
```

#### 2. Run swagger generator
```
swagger -apiPackage="github.com/kubermatic/api/cmd/kubermatic-api" \
-mainApiFile="/Users/andrii/workspace/src/github.com/kubermatic/api/cmd/kubermatic-api/main.go" \
-format="swagger" \
-output="handler/api-docs"
```

