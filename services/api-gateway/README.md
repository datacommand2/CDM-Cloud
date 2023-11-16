API Gateway
==========
[![pipeline status](http://github.com/datacommand2/cdm-cloud/services/api-gateway/badges/master/pipeline.svg)](http://github.com/datacommand2/cdm-cloud/services/api-gateway/-/commits/master)
[![coverage report](http://github.com/datacommand2/cdm-cloud/services/api-gateway/badges/master/coverage.svg)](http://github.com/datacommand2/cdm-cloud/services/api-gateway/-/commits/master)

---

API Gateway 는 내부 서비스의 RPC 를 외부에 Restful API 로 공개하고, 사용자의 Request 를 내부 서비스의 RPC 로 전달해주는 서비스이다.

내부 서비스의 RPC 를 Restful API 로 공개하기 위한 방법은 다음과 같다.

### `proto` 작성 
* proto 파일에서 "google/api/annotations.proto"를 import 해야함  
`import "google/api/annotations.proto";`
* rpc 함수에 option 지정시 API 로 공개됨.  
`option (google.api.http) = { post: "/identity/users/{user_id}"; body:"AddGroupReqeust"; };`

**example**:
```proto
syntax = "proto3";

import "google/api/annotations.proto";
     
service Identity {
		rpc UpdateUserName(Reqeust) returns (Response){
			option (google.api.http) = { post: "/identity/users/{user_id}"; body:"*"; };
		}
}

message Request {
	int64 user_id = 1;
	string name = 2;
}

message Response {
	int64 user_id = 1;
	string name = 2;
	int64 updated_at = 3;
}
```

### `proto` 빌드
`registry.datacommand.co.kr/golang:1.14` 를 사용하여 proto 빌드 시 $GOPATH 를 proto path 에 추가해줘야 함.
`protoc --proto_path=$GOPATH ...`


### 테스트
위 example proto 로 구현된 서비스를 api-gateway 에서 handling 하게 하고,
`curl -X POST -H "Content-Type: application/json" api-gateway:1234/identity/users/123 -d '{"name":"mjj"}'`
와 같이 호출하면, Request 는 `{"user_id": 123, "name": "mjj"}`, Response 는 `{"user_id": 123, "name": "mjj", updated_at: 1601882599}`

Request Header 의 경우, 아래와 같은 방법으로 얻을 수 있습니다.

```go
// import "github.com/micro/go-micro/v2/metadata"

if v, ok := metadata.Get(ctx, "Content-Type"); ok {
    fmt.Printf("Content-Type: %v\n", v)
}
```
### 레퍼런스
* <https://cloud.google.com/endpoints/docs/grpc/transcoding?hl=ko>