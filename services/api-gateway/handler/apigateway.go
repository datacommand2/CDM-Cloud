package handler

import (
	api "github.com/micro/go-micro/v2/api/handler"
	"github.com/micro/go-micro/v2/api/handler/rpc"
	"github.com/micro/go-micro/v2/api/router/registry"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/grpc"
	"net/http"
	"time"
)

// NewAPIGatewayHandler APIGateway에 http handler를 생성
func NewAPIGatewayHandler(opts ...Option) http.Handler {
	var (
		option  Options
		handler http.Handler = rpc.NewHandler(api.WithRouter(registry.NewRouter()), api.WithClient(grpc.NewClient(client.RequestTimeout(30*time.Second))))
	)
	for _, o := range opts {
		o(&option)
	}

	for _, h := range option.wrappers {
		handler = h.Wrap(handler)
	}
	return handler
}
