module github.com/datacommand2/cdm-cloud/services/scheduler

go 1.14

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

require (
	github.com/datacommand2/cdm-cloud/common v0.0.0-20231116053807-138bc179eb3f
	github.com/golang/protobuf v1.5.2
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	github.com/jinzhu/gorm v1.9.16
	github.com/micro/go-micro/v2 v2.9.1
	github.com/robfig/cron/v3 v3.0.1
	google.golang.org/protobuf v1.27.1

)
