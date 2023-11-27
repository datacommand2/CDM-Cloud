module github.com/datacommand2/cdm-cloud/common

go 1.14

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

require (
	github.com/cockroachdb/cockroach-go v2.0.1+incompatible
	github.com/coreos/etcd v3.3.27+incompatible
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/google/uuid v1.3.0
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/jinzhu/gorm v1.9.14
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lestrrat-go/strftime v1.0.6 // indirect
	github.com/lib/pq v1.10.7
	github.com/micro/cli/v2 v2.1.2
	github.com/micro/go-micro/v2 v2.9.1
	github.com/pkg/errors v0.9.1
	github.com/streadway/amqp v1.0.0
	google.golang.org/grpc v1.44.0
)
