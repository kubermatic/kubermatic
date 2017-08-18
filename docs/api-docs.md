## Swagger/OpenAPI docs

### Quick start guide

#### 1. Install dependency
```
make install
```

#### 2. Run swagger generator
```
$ ./hack/gen-api-docs.sh
2017/08/18 15:08:36 Start parsing
2017/08/18 15:08:39 Finish parsing
2017/08/18 15:08:39 Wrote api/index.json
2017/08/18 15:08:39 Swagger UI files generated
```

#### 3. Build project
```
$ make build
```

#### Run api and navigate to swagger-ui
```
$ ./hack/run-api.sh
$ open http://localhost:8080/swagger-ui/
```
