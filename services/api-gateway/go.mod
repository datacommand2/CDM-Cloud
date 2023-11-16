module github.com/datacommand2/cdm-cloud/services/api-gateway

go 1.14

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

require (
	github.com/datacommand2/cdm-cloud/common v0.0.0-20231116045224-15deb31e11c1
	github.com/datacommand2/cdm-cloud/services/identity v0.0.0-20231116045224-15deb31e11c1
	github.com/gobwas/ws v1.0.3
	github.com/golang/protobuf v1.5.2
	github.com/google/uuid v1.3.0
	github.com/gorilla/websocket v1.4.1
	github.com/micro/go-micro/v2 v2.9.1
	github.com/stretchr/testify v1.7.0
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c
	google.golang.org/protobuf v1.27.1

)
