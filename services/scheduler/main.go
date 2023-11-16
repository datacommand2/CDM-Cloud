package main

import (
	"github.com/datacommand2/cdm-cloud/common"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/database"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/event"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/datacommand2/cdm-cloud/services/scheduler/handler"
	"github.com/datacommand2/cdm-cloud/services/scheduler/internal/scheduler/executor"
	proto "github.com/datacommand2/cdm-cloud/services/scheduler/proto"
	"github.com/jinzhu/gorm"
	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/client/selector"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/server"
)

var (
	// version 은 서비스 버전의 정보이다.
	version string

	// defaultTenant default 테넌트
	defaultTenant model.Tenant
)

func loadDefaultTenantID() error {
	err := database.Transaction(func(db *gorm.DB) error {
		return db.Find(&defaultTenant, model.Tenant{Name: "default"}).Error
	})
	if err == gorm.ErrRecordNotFound {
		return errors.Unknown(err)

	} else if err != nil {
		return errors.UnusableDatabase(err)
	}

	return nil
}

func reportEvent(eventCode, errorCode string, eventContents interface{}) {
	err := event.ReportEvent(defaultTenant.ID, eventCode, errorCode, event.WithContents(eventContents))
	if err != nil {
		logger.Warnf("Could not report event. cause: %+v", errors.Unknown(err))
	}
}

func main() {
	logger.Infof("Initializing service(%s)", constant.ServiceScheduler)
	service := micro.NewService(
		micro.Name(constant.ServiceScheduler),
		micro.Version(version),
		micro.Metadata(map[string]string{
			"CDM_SOLUTION_NAME":       constant.SolutionName,
			"CDM_SERVICE_DESCRIPTION": constant.ServiceSchedulerDescription,
		}),
	)
	service.Init()

	// Registry 의 Kubernetes 기능을 go-micro server 옵션에 추가
	if err := service.Server().Init(server.Registry(registry.DefaultRegistry)); err != nil {
		logger.Fatalf("Cloud not init server options by service(%s). cause: %v", constant.ServiceScheduler, err)
	}

	if err := service.Client().Init(client.Registry(registry.DefaultRegistry)); err != nil {
		logger.Fatalf("Cloud not init client options by service(%s). cause: %v", constant.ServiceScheduler, err)
	}

	if err := selector.DefaultSelector.Init(selector.Registry(registry.DefaultRegistry)); err != nil {
		logger.Fatalf("Cloud not init selector options by service(%s). cause: %v", constant.ServiceScheduler, err)
	}

	defer common.Destroy()

	if err := loadDefaultTenantID(); err != nil {
		logger.Fatalf("Could not load Default TenantID. cause: %+v", err)
	}

	logger.Info("Creating scheduler handler")
	h, err := handler.NewSchedulerHandler(defaultTenant.ID)
	switch {
	case errors.Equal(err, executor.ErrUnsupportedTimezone):
		reportEvent("cdm-cloud.scheduler.main.failure-create_handler", "unsupported_timezone", err)
		logger.Fatalf("Could not create scheduleHandler. cause: %+v", err)

	case errors.Equal(err, executor.ErrUnsupportedScheduleType):
		reportEvent("cdm-cloud.scheduler.main.failure-create_handler", "unsupported_schedule_type", err)
		logger.Fatalf("Could not create scheduleHandler. cause: %+v", err)

	case errors.Equal(err, errors.ErrUnusableDatabase):
		reportEvent("cdm-cloud.scheduler.main.failure-create_handler", "unusable_database", err)
		logger.Fatalf("Could not create scheduleHandler. cause: %+v", err)

	case errors.Equal(err, errors.ErrUnusableBroker):
		reportEvent("cdm-cloud.scheduler.main.failure-create_handler", "unusable_broker", err)
		logger.Fatalf("Could not create scheduleHandler. cause: %+v", err)

	case errors.Equal(err, errors.ErrUnknown):
		reportEvent("cdm-cloud.scheduler.main.failure-create_handler", "unknown", err)
		logger.Fatalf("Could not create scheduleHandler. cause: %+v", err)

	case err != nil:
		err = errors.Unknown(err)
		reportEvent("cdm-cloud.scheduler.main.failure-create_handler", "unknown", err)
		logger.Fatalf("Could not create scheduleHandler. cause: %+v", err)
	}
	defer func() {
		err = h.Close()
		if err != nil {
			reportEvent("cdm-cloud.scheduler.main.failure-close_service", "unusable_broker", err)
			logger.Warnf("Could not close scheduleHandler. cause: %+v", err)
		}
	}()

	logger.Info("Registering scheduler handler")
	err = proto.RegisterSchedulerHandler(service.Server(), h)
	if err != nil {
		err = errors.Unknown(err)
		reportEvent("cdm-cloud.scheduler.main.failure-register_handler", "unknown", err)
		logger.Fatalf("Could not register scheduleHandler. cause: %+v", err)
	}

	logger.Infof("Starting service(%s)", constant.ServiceScheduler)
	if err = service.Run(); err != nil {
		err = errors.Unknown(err)
		reportEvent("cdm-cloud.scheduler.main.failure-run_service", "unknown", err)
		logger.Fatalf("Could not start service(%s). cause: %+v", constant.ServiceAPIGateway, err)
	}
}
