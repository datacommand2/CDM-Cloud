module github.com/datacommand2/cdm-cloud/services/identity

go 1.14

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

require (
	github.com/asaskevich/govalidator v0.0.0-20200907205600-7a23bdc65eef
	github.com/casbin/casbin/v2 v2.19.8
	github.com/casbin/gorm-adapter/v2 v2.1.0
	github.com/datacommand2/cdm-cloud/common v0.0.0-20231127055611-c20cbc4911cf
	github.com/golang/protobuf v1.5.2
	github.com/jinzhu/gorm v1.9.14
	github.com/micro/go-micro/v2 v2.9.1
	github.com/pkg/errors v0.9.1
	github.com/sethvargo/go-password v0.2.0
	github.com/stretchr/testify v1.7.0 // indirect
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c
	google.golang.org/protobuf v1.27.1
)
