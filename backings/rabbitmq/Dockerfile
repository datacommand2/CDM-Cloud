FROM golang:1.14 as builder

WORKDIR /build
COPY common .

RUN GOFLAGS='-mod=vendor' CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o registerd cmd/registerd/registerd.go

FROM rabbitmq:3.8.5

ARG VERSION

COPY entrypoint.sh /sbin/
COPY --from=builder /build/registerd /bin/

RUN chmod +x /sbin/entrypoint.sh

ENV CDM_SERVICE_NAME=cdm-cloud-rabbitmq
ENV CDM_SERVICE_ADVERTISE_PORT=5672
ENV CDM_SERVICE_VERSION=${VERSION:-unknown}
ENV CDM_SOLUTION_NAME="CDM-Cloud"
ENV CDM_SERVICE_DESCRIPTION="Message Broker"

CMD ["/sbin/entrypoint.sh"]
