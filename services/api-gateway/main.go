package main

import (
	"fmt"
	"github.com/datacommand2/cdm-cloud/common"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/datacommand2/cdm-cloud/services/api-gateway/handler"
	"github.com/datacommand2/cdm-cloud/services/api-gateway/handler/wrapper"
  "github.com/micro/cli/v2"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/selector"
	"github.com/micro/go-micro/v2/config/cmd"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/server"
	"github.com/micro/go-micro/v2/web"
	"time"
)

// Version 은 서비스 버전의 정보이다.
var version string

func main() {
	var port int
	var ttl int
	common.Init(common.WithFlags(
		&cli.IntFlag{
			Name:        "cdm_cloud_api_gateway_port",
			Usage:       "Port to used by api-gateway",
			Required:    true,
			Destination: &port,
		},
		&cli.IntFlag{
			Name:        "cdm_cloud_api_gateway_ttl",
			Usage:       "TTL to used by api-gateway",
			Value:       30,
			Required:    false,
			Destination: &ttl,
		},
	))

	if err := cmd.DefaultCmd.Init(); err != nil {
		logger.Fatalf("Could not init service(%s). cause: %v", constant.ServiceAPIGateway, err)
		panic(err)
	}
	defer common.Destroy()

	// 먼저 집어넣은 순서대로 post process 가 진행되고,
	// 나중에 넣은 것부터 pre process 가 진행됨
	h := handler.NewAPIGatewayHandler(
		handler.WithWrapper(wrapper.NewAuthHandler()),
		handler.WithWrapper(wrapper.NewErrorFilterHandler()),
		handler.WithWrapper(wrapper.NewErrorConversionHandler()),
		handler.WithWrapper(wrapper.NewWebSocketHandler()),
		handler.WithWrapper(wrapper.NewCORSHandler()),
	)

	s := web.NewService(
		web.Name(constant.ServiceAPIGateway),
		web.Address(fmt.Sprintf(":%d", port)),
		web.Handler(h),
		web.Version(version),
		web.Metadata(map[string]string{
			"CDM_SOLUTION_NAME":       constant.SolutionName,
			"CDM_SERVICE_DESCRIPTION": constant.ServiceAPIGatewayDescription,
		}),
	)

	logger.Infof("Initializing service(%s)", constant.ServiceAPIGateway)
	if err := s.Init(); err != nil {
		logger.Fatalf("Could not initialize service(%s). cause: %v", constant.ServiceAPIGateway, err)
	}

	// Registry 의 Kubernetes 기능을 go-micro client 옵션에 추가
	if err := s.Options().Service.Client().Init(client.Registry(registry.DefaultRegistry), client.PoolTTL(time.Duration(ttl)*time.Minute)); err != nil {
		logger.Fatalf("Cloud not init client options by service(%s). cause: %v", constant.ServiceAPIGateway, err)
	}

	if err := s.Options().Service.Server().Init(server.Registry(registry.DefaultRegistry)); err != nil {
		logger.Fatalf("Cloud not init server options by service(%s). cause: %v", constant.ServiceAPIGateway, err)
	}

	if err := selector.DefaultSelector.Init(selector.Registry(registry.DefaultRegistry)); err != nil {
		logger.Fatalf("Cloud not init selector options by service(%s). cause: %v", constant.ServiceAPIGateway, err)
	}

	logger.Infof("Starting service(%s)", constant.ServiceAPIGateway)
	if err := s.Run(); err != nil {
		logger.Fatalf("Could not start service(%s). cause: %v", constant.ServiceAPIGateway, err)
	}
}
