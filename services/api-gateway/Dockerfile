FROM golang:1.14 as builder

WORKDIR /build
COPY . .

ARG VERSION

RUN make clean build

FROM scratch

COPY --from=builder /build/api-gateway /

ENTRYPOINT ["/api-gateway"]