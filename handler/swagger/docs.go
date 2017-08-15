
package main
//This file is generated automatically. Do not try to edit it manually.

var resourceListingJson = `{
    "apiVersion": "1.4.0",
    "swaggerVersion": "1.2",
    "basePath": "http://loodse.com/",
    "apis": [
        {
            "path": "/testapi",
            "description": "get string by ID"
        },
        {
            "path": "/api",
            "description": "Kubermatic API"
        }
    ],
    "info": {
        "title": "Kubermatic REST API",
        "description": "Kubermatic REST API",
        "contact": "andrii.soldatenko@gmail.com",
        "termsOfServiceUrl": "https://github.com/kubermatic/api",
        "license": "TBD",
        "licenseUrl": "https://github.com/kubermatic/api"
    }
}`
var apiDescriptionsJson = map[string]string{"testapi":`{
    "apiVersion": "1.4.0",
    "swaggerVersion": "1.2",
    "basePath": "http://loodse.com/",
    "resourcePath": "/testapi",
    "produces": [
        "application/json"
    ],
    "apis": [
        {
            "path": "/testapi/get-string-by-int/{some_id}",
            "description": "get string by ID",
            "operations": [
                {
                    "httpMethod": "GET",
                    "nickname": "GetStringByInt",
                    "type": "string",
                    "items": {},
                    "summary": "get string by ID",
                    "parameters": [
                        {
                            "paramType": "path",
                            "name": "some_id",
                            "description": "Some ID",
                            "dataType": "int",
                            "type": "int",
                            "format": "",
                            "allowMultiple": false,
                            "required": true,
                            "minimum": 0,
                            "maximum": 0
                        }
                    ],
                    "responseMessages": [
                        {
                            "code": 200,
                            "message": "",
                            "responseType": "object",
                            "responseModel": "string"
                        },
                        {
                            "code": 400,
                            "message": "We need ID!!",
                            "responseType": "object",
                            "responseModel": "github.com.kubermatic.api.handler.APIError"
                        },
                        {
                            "code": 404,
                            "message": "Can not find ID",
                            "responseType": "object",
                            "responseModel": "github.com.kubermatic.api.handler.APIError"
                        }
                    ],
                    "produces": [
                        "application/json"
                    ]
                }
            ]
        }
    ],
    "models": {
        "github.com.kubermatic.api.handler.APIError": {
            "id": "github.com.kubermatic.api.handler.APIError",
            "properties": {
                "ErrorCode": {
                    "type": "int",
                    "description": "",
                    "items": {},
                    "format": ""
                },
                "ErrorMessage": {
                    "type": "string",
                    "description": "",
                    "items": {},
                    "format": ""
                }
            }
        }
    }
}`,}
