FROM golang:1.14 as builder

WORKDIR /build
COPY common .

RUN GOFLAGS='-mod=vendor' CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o registerd cmd/registerd/registerd.go

RUN mkdir .etcd_auth \
	&& openssl genrsa -out .etcd_auth/.jwt_RS256 4096 \
 	&& openssl rsa -in .etcd_auth/.jwt_RS256 -pubout > .etcd_auth/.jwt_RS256.pub

FROM etcd:v3.3.22

ARG VERSION

COPY etcd_run.sh /bin/
COPY etcd_auth.sh /bin/
COPY etcd_updatemember.sh /bin/
COPY --from=builder /build/registerd /bin/
COPY --from=builder /build/.etcd_auth /bin/.etcd_auth
COPY pkg /bin/pkg

RUN apk add  --repositories-file=/bin/pkg --allow-untrusted --no-network /bin/pkg/*.apk

VOLUME /data
RUN chmod 700 /data
EXPOSE 2379 2380
ENV ETCDCTL_API=3
ENV CDM_SERVICE_NAME=cdm-cloud-etcd
ENV CDM_SERVICE_ADVERTISE_PORT=2379
ENV CDM_SERVICE_VERSION=${VERSION:-unknown}
ENV CDM_SOLUTION_NAME="CDM-Cloud"
ENV CDM_SERVICE_DESCRIPTION="Key-Value Store"

ENTRYPOINT ["/bin/sh","/bin/etcd_run.sh"]
