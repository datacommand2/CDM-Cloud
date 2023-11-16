package config

import (
	"database/sql"
	"encoding/json"
	"github.com/datacommand2/cdm-cloud/common/config"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	notification "github.com/datacommand2/cdm-cloud/services/notification/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/jinzhu/gorm"
)

// Receive 코드의 수신 여부를 저장한다
type Receive struct {
	Code        string
	ReceiveFlag sql.NullBool
}

// Config 알림 설정
type Config struct {
	EventNotificationEnable        string
	EventEmailNotificationEnable   string
	EventDesktopNotificationEnable string
	EventPopupNotificationEnable   string
	EventSmsNotificationEnable     string
	EventCustomNotificationEnable  string
	EventStorePeriod               string
	EventSMTPNotifier              string
	EventSMSNotifier               string
}

// TODO: 향후 global config 가 아닌, notification 을 위한 별도의 table 을 추가해 처리해야 할 것 같음

// ValidateUser 테넌트의 유효성 여부를 판단한다.
func ValidateUser(db *gorm.DB, userID uint64) error {
	err := db.Find(&model.User{}, &model.User{ID: userID}).Error
	if err != nil && err == gorm.ErrRecordNotFound {
		return NotFoundUser(userID)
	}

	if err != nil {
		return errors.UnusableDatabase(err)
	}

	return nil
}

// ValidateTenant 테넌트의 유효성 여부를 판단한다.
func ValidateTenant(db *gorm.DB, tenantID uint64) error {
	err := db.Find(&model.Tenant{}, &model.Tenant{ID: tenantID}).Error
	switch {
	case err != nil && err == gorm.ErrRecordNotFound:
		return NotFoundTenant(tenantID)

	case err != nil:
		return errors.UnusableDatabase(err)
	}

	return nil
}

// GetConfigEmailNotifier 테넌트의 이메일 정보를 조회한다.
func GetConfigEmailNotifier(db *gorm.DB, tenantID uint64) (*notification.EventSMTPNotifier, error) {
	emailStr, err := getConfigOne(db, tenantID, config.EventEmailNotifier)
	if err != nil {
		return nil, err
	}

	var email notification.EventSMTPNotifier
	err = json.Unmarshal([]byte(emailStr), &email)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	return &email, nil
}

// GetConfig 설정을 조회한다.
// caller: must call ValidateTenant before GetConfig
func GetConfig(db *gorm.DB, tenantID uint64) (*notification.Config, error) {
	var (
		ret notification.Config
		err error
	)

	ret.EventNotificationEnable, err = getConfigBool(db, tenantID, config.EventNotificationEnable)
	if err != nil {
		return nil, err
	}

	ret.EventEmailNotificationEnable, err = getConfigBool(db, tenantID, config.EventEmailNotificationEnable)
	if err != nil {
		return nil, err
	}

	ret.EventDesktopNotificationEnable, err = getConfigBool(db, tenantID, config.EventDesktopNotificationEnable)
	if err != nil {
		return nil, err
	}

	ret.EventPopupNotificationEnable, err = getConfigBool(db, tenantID, config.EventPopupNotificationEnable)
	if err != nil {
		return nil, err
	}

	ret.EventSmsNotificationEnable, err = getConfigBool(db, tenantID, config.EventSmsNotificationEnable)
	if err != nil {
		return nil, err
	}

	ret.EventCustomNotificationEnable, err = getConfigBool(db, tenantID, config.EventCustomNotificationEnable)
	if err != nil {
		return nil, err
	}

	ret.EventStorePeriod, err = getConfigUint32(db, tenantID, config.EventStorePeriod)
	if err != nil {
		return nil, err
	}

	ret.EventSmtpNotifier, err = getConfigEventSMTPNotifier(db, tenantID, config.EventEmailNotifier)
	if err != nil {
		return nil, err
	}

	ret.EventSmsNotifier, err = getConfigEventSMSNotifier(db, tenantID, config.EventSMSNotifier)
	if err != nil {
		return nil, err
	}

	return &ret, nil
}

// SetConfig 설정을 삭제 혹은 업데이트한다.
// caller: must call ValidateTenant before SetConfig
func SetConfig(db *gorm.DB, tenantID uint64, req *notification.SetConfigRequest) error {
	if err := setConfigBool(db, tenantID, config.EventNotificationEnable, req.GetEventConfig().GetEventNotificationEnable()); err != nil {
		return err
	}

	if err := setConfigBool(db, tenantID, config.EventEmailNotificationEnable, req.GetEventConfig().GetEventEmailNotificationEnable()); err != nil {
		return err
	}

	if err := setConfigBool(db, tenantID, config.EventDesktopNotificationEnable, req.GetEventConfig().GetEventDesktopNotificationEnable()); err != nil {
		return err
	}

	if err := setConfigBool(db, tenantID, config.EventPopupNotificationEnable, req.GetEventConfig().GetEventPopupNotificationEnable()); err != nil {
		return err
	}

	if err := setConfigBool(db, tenantID, config.EventSmsNotificationEnable, req.GetEventConfig().GetEventSmsNotificationEnable()); err != nil {
		return err
	}

	if err := setConfigBool(db, tenantID, config.EventCustomNotificationEnable, req.GetEventConfig().GetEventCustomNotificationEnable()); err != nil {
		return err
	}

	if err := setConfigUint32(db, tenantID, config.EventStorePeriod, req.GetEventConfig().GetEventStorePeriod()); err != nil {
		return err
	}

	if err := setConfigEventSMTPNotifier(db, tenantID, config.EventEmailNotifier, req.GetEventConfig().GetEventSmtpNotifier()); err != nil {
		return err
	}

	if err := setConfigEventSMSNotifier(db, tenantID, config.EventSMSNotifier, req.GetEventConfig().GetEventSmsNotifier()); err != nil {
		return err
	}

	return nil
}

type tenantReceive struct {
	TenantReceiveEvent model.TenantReceiveEvent `gorm:"embedded"`
	EventCode          model.EventCode          `gorm:"embedded"`
}

// GetTenantEventReceives 특정 테넌트의 이벤트 수신 여부를 조회한다
func GetTenantEventReceives(db *gorm.DB, tenantID uint64, req *notification.GetEventReceivesRequest) ([]*notification.EventReceive, error) {
	if req.GetSolution() != "" ||
		req.GetClass_1() != "" || req.GetClass_2() != "" || req.GetClass_3() != "" ||
		req.GetLevel() != "" {
		err := db.Find(&model.EventCode{}, &model.EventCode{
			Solution: req.GetSolution(),
			Class1:   req.GetClass_1(),
			Class2:   req.GetClass_2(),
			Class3:   req.GetClass_3(),
			Level:    req.GetLevel(),
		}).Error
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, errors.InvalidParameterValue("request", req, err.Error())
		case nil:
		default:
			return nil, errors.UnusableDatabase(err)
		}
	}

	var receives []*tenantReceive
	err := db.Model(&model.TenantReceiveEvent{}).
		Select("*").
		Joins("JOIN cdm_event_code ON cdm_event_code.code = cdm_tenant_receive_event.code").
		Where(&model.TenantReceiveEvent{TenantID: tenantID}).
		Where(&model.EventCode{
			Solution: req.GetSolution(),
			Class1:   req.GetClass_1(),
			Class2:   req.GetClass_2(),
			Class3:   req.GetClass_3(),
			Level:    req.GetLevel(),
		}).Scan(&receives).Error
	if err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	var ret []*notification.EventReceive
	for _, item := range receives {
		ret = append(ret, &notification.EventReceive{
			Code: item.TenantReceiveEvent.Code,
			Enable: &wrappers.BoolValue{
				Value: item.TenantReceiveEvent.ReceiveFlag,
			},
		})
	}
	return ret, nil
}

// SetTenantEventReceives 특정 테넌트의 수신 여부를 저장한다.
func SetTenantEventReceives(db *gorm.DB, tenantID uint64, req *notification.EventReceivesRequest) error {
	// TODO: 해당하는 cdm event code 가 없는 경우 에러를 떨구는 처리
	// 기억용: 벌크 업데이트 불가, gorm2 부터 가능
	for _, item := range req.GetEventNotifications() {
		// 기억용: update 는 0을 무시하지만, save 에서의 update 는 0을 무시하지 않음
		// gorm2부터는 update 에서도 0을 무시하지 않게 하는 방법 제공
		if err := db.Save(&model.TenantReceiveEvent{
			Code:        item.Code,
			TenantID:    tenantID,
			ReceiveFlag: item.GetEnable().GetValue(),
		}).Error; err != nil {
			return errors.UnusableDatabase(err)
		}
	}

	return nil
}

type userReceive struct {
	UserReceiveEvent model.UserReceiveEvent `gorm:"embedded"`
	EventCode        model.EventCode        `gorm:"embedded"`
}

// GetUserEventReceives 특정 사용자의 수신 여부를 조회한다.
func GetUserEventReceives(db *gorm.DB, userID uint64, req *notification.GetEventReceivesRequest) ([]*notification.EventReceive, error) {
	if req.GetSolution() != "" ||
		req.GetClass_1() != "" || req.GetClass_2() != "" || req.GetClass_3() != "" ||
		req.GetLevel() != "" {
		err := db.Find(&model.EventCode{}, &model.EventCode{
			Solution: req.GetSolution(),
			Class1:   req.GetClass_1(),
			Class2:   req.GetClass_2(),
			Class3:   req.GetClass_3(),
			Level:    req.GetLevel(),
		}).Error
		switch err {
		case gorm.ErrRecordNotFound:
			return nil, errors.InvalidParameterValue("request", req, err.Error())
		case nil:
		default:
			return nil, errors.UnusableDatabase(err)
		}
	}

	var receives []*userReceive

	err := db.Model(&model.UserReceiveEvent{}).
		Select("*").
		Joins("JOIN cdm_event_code ON cdm_event_code.code = cdm_user_receive_event.code").
		Where(&model.UserReceiveEvent{UserID: userID}).
		Where(&model.EventCode{
			Solution: req.GetSolution(),
			Class1:   req.GetClass_1(),
			Class2:   req.GetClass_2(),
			Class3:   req.GetClass_3(),
			Level:    req.GetLevel(),
		}).Scan(&receives).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.UnusableDatabase(err)
	}

	var ret []*notification.EventReceive
	for _, item := range receives {
		if !item.UserReceiveEvent.ReceiveFlag.Valid {
			continue
		}
		ret = append(ret, &notification.EventReceive{
			Code: item.UserReceiveEvent.Code,
			Enable: &wrappers.BoolValue{
				Value: item.UserReceiveEvent.ReceiveFlag.Bool,
			},
		})
	}

	return ret, nil
}

// SetUserEventReceives 특정 사용자의 수신 여부를 저장한다.
func SetUserEventReceives(db *gorm.DB, userID uint64, req *notification.EventReceivesRequest) error {
	// TODO: 해당하는 cdm event code 가 없는 경우 에러를 떨구는 처리

	for _, item := range req.GetEventNotifications() {
		if item.Enable == nil {
			if err := db.Delete(&model.UserReceiveEvent{
				Code:   item.GetCode(),
				UserID: userID,
			}).Error; err != nil {
				return errors.UnusableDatabase(err)
			}
		} else {
			if err := db.Save(&model.UserReceiveEvent{
				Code:   item.GetCode(),
				UserID: userID,
				ReceiveFlag: sql.NullBool{
					Bool:  item.GetEnable().GetValue(),
					Valid: true,
				},
			}).Error; err != nil {
				return errors.UnusableDatabase(err)
			}
		}
	}

	return nil
}

// ResetUserEventReceives 특정 사용자의 이벤트 수신 여부 전부 제거
func ResetUserEventReceives(db *gorm.DB, userID uint64) error {
	if err := db.Delete(&model.UserReceiveEvent{
		UserID: userID,
	}).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	return nil
}

// ValidateConfig config 값의 제약 조건을 확인한다
func ValidateConfig(conf *notification.Config) error {
	if conf.GetEventNotificationEnable() == nil {
		return errors.RequiredParameter("event_notification_enable")
	}

	if conf.GetEventEmailNotificationEnable() == nil {
		return errors.RequiredParameter("event_email_notification_enable")
	}

	if conf.GetEventDesktopNotificationEnable() == nil {
		return errors.RequiredParameter("event_desktop_notification_enable")
	}

	if conf.GetEventPopupNotificationEnable() == nil {
		return errors.RequiredParameter("event_popup_notification_enable")
	}

	if conf.GetEventSmsNotificationEnable() == nil {
		return errors.RequiredParameter("event_sms_notification_enable")
	}

	if conf.GetEventCustomNotificationEnable() == nil {
		return errors.RequiredParameter("event_custom_notification_enable")
	}

	if conf.GetEventStorePeriod() == nil {
		return errors.RequiredParameter("event_store_period")
	}

	if conf.GetEventSmtpNotifier() == nil {
		return errors.RequiredParameter("event_notification_enable")
	}

	if conf.GetEventSmsNotifier() == nil {
		return errors.RequiredParameter("event_notification_enable")
	}

	if err := validateEventStorePeriod(conf.GetEventStorePeriod().GetValue()); err != nil {
		return err
	}

	if conf.GetEventEmailNotificationEnable().GetValue() == true {
		if err := validateServerAddress(conf.GetEventSmtpNotifier().GetServerAddress()); err != nil {
			return err
		}

		if err := validateServerPort(conf.GetEventSmtpNotifier().GetServerPort()); err != nil {
			return err
		}

		if err := validateEncryption(conf.GetEventSmtpNotifier().GetEncryption()); err != nil {
			return err
		}

		if err := validateAuthMechanism(conf.GetEventSmtpNotifier().GetAuthMechanism()); err != nil {
			return err
		}

		if err := validateAuthUsername(conf.GetEventSmtpNotifier().GetAuthUsername()); err != nil {
			return err
		}

		if err := validateAuthPassword(conf.GetEventSmtpNotifier().GetAuthPassword()); err != nil {
			return err
		}

		if err := validateSender(conf.GetEventSmtpNotifier().GetSender()); err != nil {
			return err
		}
	}

	if conf.GetEventSmsNotificationEnable().GetValue() == true {
		if err := validateProvider(conf.GetEventSmsNotifier().GetProvider()); err != nil {
			return err
		}

		if err := validateVersion(conf.GetEventSmsNotifier().GetVersion()); err != nil {
			return err
		}
	}

	return nil
}
