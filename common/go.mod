module github.com/datacommand2/cdm-cloud/common

go 1.14

require (
	github.com/cockroachdb/cockroach-go v2.0.1+incompatible
	github.com/coreos/etcd v3.3.22+incompatible
	github.com/datacommand2/cdm-cloud/services/identity v0.0.0-20231116043826-dcb4d64b8828
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.4.0
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/jinzhu/gorm v1.9.16
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.6 // indirect
	github.com/lib/pq v1.3.0
	github.com/micro/cli/v2 v2.1.2
	github.com/micro/go-micro/v2 v2.9.1
	github.com/pkg/errors v0.9.1
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/testify v1.8.4
	google.golang.org/grpc v1.59.0

)
