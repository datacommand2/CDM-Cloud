module github.com/datacommand2/cdm-cloud/common

go 1.14

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

require (
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/cockroachdb/cockroach-go v2.0.1+incompatible
	github.com/coreos/etcd v3.3.27+incompatible
	github.com/datacommand2/cdm-cloud/services/identity v0.0.0-20231116052203-cfca20d50a93
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/go-git/go-git/v5 v5.4.2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/jinzhu/copier v0.0.0-20190924061706-b57f9002281a
	github.com/jinzhu/gorm v1.9.14
	github.com/lestrrat-go/file-rotatelogs v2.4.0+incompatible
	github.com/lib/pq v1.10.7
	github.com/micro/cli/v2 v2.1.2
	github.com/micro/go-micro/v2 v2.9.1
	github.com/miekg/dns v1.1.43 // indirect
	github.com/mitchellh/hashstructure v1.1.0 // indirect
	github.com/nats-io/nats.go v1.13.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0 // indirect
	github.com/streadway/amqp v1.0.0
	go.uber.org/zap v1.17.0 // indirect
	golang.org/x/crypto v0.0.0-20210513164829-c07d793c2f9a // indirect
	golang.org/x/net v0.0.0-20210525063256-abc453219eb5 // indirect
	google.golang.org/grpc v1.44.0
	gopkg.in/yaml.v2 v2.4.0 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)
