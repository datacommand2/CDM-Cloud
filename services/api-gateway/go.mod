module github.com/datacommand2/cdm-cloud/services/api-gateway

go 1.14

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

require (
	github.com/datacommand2/cdm-cloud/common v0.0.0-20231128060710-080c7906e48b
	github.com/datacommand2/cdm-cloud/services/identity v0.0.0-20231129020632-c054325b27f7
	github.com/gobwas/ws v1.0.3
	github.com/golang/protobuf v1.5.2
	github.com/micro/cli/v2 v2.1.2
	github.com/micro/go-micro/v2 v2.9.1
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c
	google.golang.org/protobuf v1.27.1

)
