package main

import (
	"github.com/datacommand2/cdm-cloud/common"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/database"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/event"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/datacommand2/cdm-cloud/services/identity/handler"
	identity "github.com/datacommand2/cdm-cloud/services/identity/proto"
	"github.com/jinzhu/gorm"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/selector"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/server"
)

var version string

func reportEventWithDefaultTenant(eventCode, errorCode string, eventContents interface{}) {
	tenant := model.Tenant{}

	err := database.Transaction(func(db *gorm.DB) error {
		return db.Find(&tenant, model.Tenant{Name: "default"}).Error
	})
	if err != nil {
		logger.Warnf("Could not report event. cause: %v", err)
		return
	}

	err = event.ReportEvent(tenant.ID, eventCode, errorCode, event.WithContents(eventContents))
	if err != nil {
		logger.Warnf("Could not report event. cause: %v", err)
	}
}

func main() {
	logger.Infof("Creating service(%s)", constant.ServiceIdentity)
	s := micro.NewService(
		micro.Name(constant.ServiceIdentity),
		micro.Version(version),
		micro.Metadata(map[string]string{
			"CDM_SOLUTION_NAME":       constant.SolutionName,
			"CDM_SERVICE_DESCRIPTION": constant.ServiceIdentityDescription,
		}),
	)
	s.Init()

	// Registry 의 Kubernetes 기능을 go-micro server 옵션에 추가
	if err := s.Server().Init(server.Registry(registry.DefaultRegistry)); err != nil {
		logger.Fatalf("Cloud not init server options by service(%s). cause: %v", constant.ServiceIdentity, err)
	}

	if err := s.Client().Init(client.Registry(registry.DefaultRegistry)); err != nil {
		logger.Fatalf("Cloud not init client options by service(%s). cause: %v", constant.ServiceIdentity, err)
	}

	if err := selector.DefaultSelector.Init(selector.Registry(registry.DefaultRegistry)); err != nil {
		logger.Fatalf("Cloud not init selector options by service(%s). cause: %v", constant.ServiceIdentity, err)
	}

	defer common.Destroy()

	logger.Info("Creating identity handler")
	h, err := handler.NewIdentityHandler()
	switch {
	case errors.Equal(err, errors.ErrUnusableDatabase):
		reportEventWithDefaultTenant("cdm-cloud.identity.main.failure-create_handler", "unusable_database", err)
		logger.Fatalf("Could not create identity handler. cause: %+v", err)

	case errors.Equal(err, errors.ErrUnknown):
		reportEventWithDefaultTenant("cdm-cloud.identity.main.failure-create_handler", "unknown", err)
		logger.Fatalf("Could not create identity handler. cause: %+v", err)
	}

	defer func() {
		logger.Info("Closing identity handler")
		if err := h.Close(); err != nil {
			reportEventWithDefaultTenant("cdm-cloud.identity.main.failure-close_handler", "unusable_database", err)
			logger.Warnf("Could not close identity handler. cause: %+v", err)
		}
	}()

	logger.Info("Registering identity handler")
	err = identity.RegisterIdentityHandler(s.Server(), h)
	if err != nil {
		reportEventWithDefaultTenant("cdm-cloud.identity.main.failure-register_handler", "unknown", errors.Unknown(err))
		logger.Fatalf("Could not register identity handler. cause: %v", err)
	}

	logger.Infof("Running service(%s)", constant.ServiceIdentity)
	if err := s.Run(); err != nil {
		reportEventWithDefaultTenant("cdm-cloud.identity.main.failure-run_service", "unknown", errors.Unknown(err))
		logger.Fatalf("Could not run service(%s). cause: %v", constant.ServiceIdentity, err)
	}
}
