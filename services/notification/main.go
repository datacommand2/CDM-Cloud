package main

import (
	"github.com/datacommand2/cdm-cloud/common"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/database"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/event"
	"github.com/datacommand2/cdm-cloud/common/logger"
	notificationEvent "github.com/datacommand2/cdm-cloud/services/notification/event"
	"github.com/datacommand2/cdm-cloud/services/notification/handler"
	"github.com/datacommand2/cdm-cloud/services/notification/notifier"
	notification "github.com/datacommand2/cdm-cloud/services/notification/proto"
	"github.com/jinzhu/gorm"
	"github.com/micro/go-micro/v2"
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
	var (
		notificationHandler notification.NotificationHandler
		cleaner             *notificationEvent.Cleaner
		err                 error
	)

	logger.Infof("Creating service(%s)", constant.ServiceNotification)
	s := micro.NewService(
		micro.Name(constant.ServiceNotification),
		micro.Version(version),
		micro.Metadata(map[string]string{
			"CDM_SOLUTION_NAME":       constant.SolutionName,
			"CDM_SERVICE_DESCRIPTION": constant.ServiceNotificationDescription,
		}),
	)
	s.Init()

	// Registry 의 Kubernetes 기능을 go-micro server 옵션에 추가
	if err = s.Server().Init(server.Registry(registry.DefaultRegistry)); err != nil {
		logger.Fatalf("Cloud not init server options by service(%s). cause: %v", constant.ServiceNotification, err)
	}

	if err = s.Client().Init(client.Registry(registry.DefaultRegistry)); err != nil {
		logger.Fatalf("Cloud not init client options by service(%s). cause: %v", constant.ServiceNotification, err)
	}

	if err = selector.DefaultSelector.Init(selector.Registry(registry.DefaultRegistry)); err != nil {
		logger.Fatalf("Cloud not init selector options by service(%s). cause: %v", constant.ServiceNotification, err)
	}

	defer common.Destroy()

	if err := loadDefaultTenantID(); err != nil {
		logger.Fatalf("Could not load Default TenantID. cause: %+v", err)
	}

	logger.Info("Creating subscriber")
	sub, err := notificationEvent.Subscribe()
	if err != nil {
		err = errors.UnusableBroker(err)
		reportEvent("cdm-cloud.notification.main.failure-notification_event_subscribe", "unusable_broker", err)
		logger.Fatalf("Could not create notificationEvent subscriber. cause: %+v", err)
	}

	defer func() {
		logger.Info("release notificationEvent subscriber")

		if err := sub.Unsubscribe(); err != nil {
			err = errors.UnusableBroker(err)
			reportEvent("cdm-cloud.notification.main.failure-notification_event_unsubscribe", "unusable_broker", err)
			logger.Warnf("Could not release notificationEvent subscriber. cause: %+v", err)
		}
	}()

	n, err := notifier.Subscribe(defaultTenant.ID)
	if err != nil {
		err = errors.UnusableBroker(err)
		reportEvent("cdm-cloud.notification.main.failure-notifier_subscribe", "unusable_broker", err)
		logger.Fatalf("Could not create notifier subscriber. cause: %+v", err)
	}

	defer func() {
		logger.Info("release notifier subscriber")

		if err := n.Unsubscribe(); err != nil {
			err = errors.UnusableBroker(err)
			reportEvent("cdm-cloud.notification.main.failure-notifier_unsubscribe", "unusable_broker", err)
			logger.Warnf("Could not release notifier subscriber. cause: %+v", err)
		}
	}()

	logger.Info("Creating cleaner")
	cleaner = notificationEvent.NewCleaner(defaultTenant.ID)
	cleaner.Start()
	defer func() {
		logger.Info("release cleaner")
		cleaner.Stop()
	}()

	logger.Info("Creating notification handler")
	notificationHandler = handler.NewNotificationHandler()

	logger.Info("Registering notification handler")
	err = notification.RegisterNotificationHandler(s.Server(), notificationHandler)
	if err != nil {
		err = errors.Unknown(err)
		reportEvent("cdm-cloud.notification.main.failure-register_handler", "unknown", err)
		logger.Fatalf("Could not register notification handler. cause: %+v", err)
	}

	logger.Infof("Running service(%s)", constant.ServiceNotification)
	if err := s.Run(); err != nil {
		err = errors.Unknown(err)
		reportEvent("cdm-cloud.notification.main.failure-run_service", "unknown", err)
		logger.Fatalf("Could not run service(%s). cause: %+v", constant.ServiceNotification, err)
	}
}
