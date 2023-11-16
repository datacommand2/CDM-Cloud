FROM golang:1.14 as builder

WORKDIR /build
COPY common .

RUN GOFLAGS='-mod=vendor' CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o registerd cmd/registerd/registerd.go

FROM cockroach:v20.1.4

ARG VERSION

COPY pkg /cockroach/pkg
COPY entrypoint.sh /cockroach
COPY --from=builder /build/registerd /bin/

RUN dpkg -i /cockroach/pkg/*.deb
RUN chmod +x /cockroach/entrypoint.sh

ENV CDM_SERVICE_NAME=cdm-cloud-cockroach
ENV CDM_SERVICE_ADVERTISE_PORT=26257
ENV CDM_SERVICE_METADATA="dialect=postgres"
ENV CDM_SERVICE_VERSION=${VERSION:-unknown}
ENV CDM_SOLUTION_NAME="CDM-Cloud"
ENV CDM_SERVICE_DESCRIPTION="Database"

ENTRYPOINT ["/cockroach/entrypoint.sh"]
