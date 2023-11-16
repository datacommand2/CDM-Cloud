package notifier

import (
	"encoding/json"
	"github.com/datacommand2/cdm-cloud/common/broker"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/database"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	e "github.com/datacommand2/cdm-cloud/common/event"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/datacommand2/cdm-cloud/services/notification/config"
	"github.com/datacommand2/cdm-cloud/services/notification/notifier/email"
	"github.com/jinzhu/gorm"
)

type delivery struct {
	Event *model.Event `json:"event"`
	User  *model.User  `json:"user"`
}

type notifier interface {
	notify(dlv *delivery) error // 실패시, 다시 큐에 삽입
}

// TODO: 향후 QueueNotify* 들에 대한 notifier 들을 만들고, 추가해야 함
var (
	//defaultTenantID default 테넌트 ID
	defaultTenantID uint64
	notifierMap     = map[string]notifier{
		constant.QueueNotifyEmail: &emailNotifier{},
	}
)

func reportEvent(tid uint64, eventCode, errorCode string, eventContents interface{}) {
	err := e.ReportEvent(tid, eventCode, errorCode, e.WithContents(eventContents))
	if err != nil {
		logger.Warnf("Could not report event. cause: %+v", errors.Unknown(err))
	}
}

// ClassifyEvent 이벤트의 notify 여부를 판단하고, 이를 각 Queue 에 삽입한다.
func ClassifyEvent(ev *model.Event) error {
	// Log 일부 허용, 전부 구현되면 로그 제거
	return database.Transaction(func(db *gorm.DB) error {
		var kinds []string
		conf, err := config.GetConfig(db, ev.TenantID)
		if err != nil {
			return err
		}

		if conf.GetEventNotificationEnable().GetValue() == false {
			return nil
		}

		if conf.GetEventEmailNotificationEnable().GetValue() == true {
			kinds = append(kinds, constant.QueueNotifyEmail)
		}
		if conf.GetEventSmsNotificationEnable().GetValue() == true {
			//kinds = append(kinds, QueueNotifySMS)
			logger.Warnf("%v notification is not implemented", constant.QueueNotifySMS)
		}
		if conf.GetEventDesktopNotificationEnable().GetValue() == true {
			//kinds = append(kinds, QueueNotifyDesktop)
			logger.Warnf("%v notification is not implemented", constant.QueueNotifyDesktop)
		}
		if conf.GetEventPopupNotificationEnable().GetValue() == true {
			//kinds = append(kinds, QueueNotifyBrowser)
			logger.Warnf("%v notification is not implemented", constant.QueueNotifyBrowser)
		}
		if conf.GetEventCustomNotificationEnable().GetValue() == true {
			//kinds = append(kinds, QueueNotifyCustom)
			logger.Warnf("%v notification is not implemented", constant.QueueNotifyCustom)
		}

		if len(kinds) == 0 {
			return nil
		}

		var users []model.User
		// caution: IFNULL is builtin function for cockroach database, if you use another database, this may be not work.
		err = db.Joins("JOIN cdm_user_receive_event ON cdm_user_receive_event.user_id = cdm_user.id").
			Joins("JOIN cdm_tenant_receive_event ON cdm_tenant_receive_event.tenant_id = cdm_user.tenant_id").
			Where(model.User{TenantID: ev.TenantID}).
			Where(model.TenantReceiveEvent{Code: ev.Code}).
			Where("IFNULL(cdm_user_receive_event.receive_flag, cdm_tenant_receive_event.receive_flag) = true").
			Find(&users).Error
		if err != nil {
			return errors.UnusableDatabase(err)
		}

		for _, user := range users {
			for _, kind := range kinds {
				var dlv delivery
				dlv.Event = ev
				dlv.User = &user
				b, err := json.Marshal(&dlv)
				if err != nil {
					return errors.Unknown(err)
				}
				err = broker.Publish(kind, &broker.Message{Body: b})
				if err != nil {
					return errors.UnusableBroker(err)
				}
			}
		}

		return nil
	})
}

// Log OK
func notifyEvent(p broker.Event) error {
	dlv := delivery{}

	err := json.Unmarshal(p.Message().Body, &dlv)
	if err != nil {
		err = errors.Unknown(err)
		logger.Errorf("Could not notify event. cause: %+v", err)
		reportEvent(defaultTenantID, "cdm-cloud.notification.notify_event.failure-unmarshal", "unknown", err)
		return err
	}

	n, ok := notifierMap[p.Topic()]
	if !ok {
		logger.Warnf("Could not notify event. cause: unsupported notifier %v", p.Topic())
		return nil
	}

	err = n.notify(&dlv)
	switch {
	case errors.Equal(err, errors.ErrUnusableDatabase):
		logger.Errorf("Could not notify event. cause: %+v", err)
		reportEvent(dlv.Event.TenantID, "cdm-cloud.notification.notify_event.failure-notify", "unusable_database", err)
		return err

	case errors.Equal(err, email.ErrUnsupportedEncryption):
		logger.Errorf("Could not notify event. cause: %+v", err)
		reportEvent(dlv.Event.TenantID, "cdm-cloud.notification.notify_event.failure-notify", "unsupported_encryption", err)
		return err

	case errors.Equal(err, email.ErrUnsupportedAuth):
		logger.Errorf("Could not notify event. cause: %+v", err)
		reportEvent(dlv.Event.TenantID, "cdm-cloud.notification.notify_event.failure-notify", "unsupported_auth", err)
		return err

	case errors.Equal(err, ErrNotFoundUser):
		logger.Errorf("Could not notify event. cause: %+v", err)
		reportEvent(dlv.Event.TenantID, "cdm-cloud.notification.notify_event.failure-notify", "not_found_user", err)
		return err

	case errors.Equal(err, errors.ErrUnknown):
		logger.Errorf("Could not notify event. cause: %+v", err)
		reportEvent(dlv.Event.TenantID, "cdm-cloud.notification.notify_event.failure-notify", "unknown", err)
		return err

	case err != nil:
		err = errors.Unknown(err)
		logger.Errorf("Could not notify event. cause: %+v", err)
		reportEvent(dlv.Event.TenantID, "cdm-cloud.notification.notify_event.failure-notify", "unknown", err)
		return err
	}

	return nil
}

// Subscribe QueueNotify* 토픽들을 구독하는 subscriber 를 생성한다.
func Subscribe(tenantID uint64) (broker.Subscriber, error) {
	defaultTenantID = tenantID

	sub, err := broker.SubscribePersistentQueue(constant.QueueNotifyEmail, notifyEvent, true)
	if err != nil {
		return nil, err
	}

	// TODO: 향후 QueueNotify* 들을 추가해야함

	return sub, nil
}
