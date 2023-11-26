module github.com/datacommand2/cdm-cloud/services/notification

go 1.4

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

require (
	github.com/datacommand2/cdm-cloud/common v0.0.0-20231124062432-069e9eb1c852
	github.com/datacommand2/cdm-cloud/services/identity v0.0.0-20231124065504-a141af3cd63a
	github.com/golang/protobuf v1.5.2
	github.com/jinzhu/gorm v1.9.14
	github.com/micro/go-micro/v2 v2.9.1
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c
	google.golang.org/grpc v1.44.0
	google.golang.org/protobuf v1.27.1
)
