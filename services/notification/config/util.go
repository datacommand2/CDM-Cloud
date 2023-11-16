package config

import (
	"encoding/json"
	"github.com/datacommand2/cdm-cloud/common/config"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	notification "github.com/datacommand2/cdm-cloud/services/notification/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/jinzhu/gorm"
	"strconv"
)

func validateEventStorePeriod(period uint32) error {
	if period > 120 {
		return errors.OutOfRangeParameterValue("event_store_period", period, 0, 120)
	}

	return nil
}

func validateServerAddress(address string) error {
	if address == "" {
		return errors.RequiredParameter("event_smtp_notifier.server_address")
	}

	if len(address) > 1024 {
		return errors.LengthOverflowParameterValue("event_smtp_notifier.server_address", address, 1024)
	}

	return nil
}

func validateServerPort(port uint32) error {
	if port == 0 || port > 65535 {
		return errors.OutOfRangeParameterValue("event_smtp_notifier.server_port", port, 1, 65535)
	}

	return nil
}

func validateEncryption(enc string) error {
	list := []interface{}{"none", "ssl/tls", "starttls"}

	if enc == "" {
		return errors.RequiredParameter("event_smtp_notifier.encryption")
	}

	for _, l := range list {
		if l == enc {
			return nil
		}
	}

	return errors.UnavailableParameterValue("event_smtp_notifier.encryption", enc, list)
}

func validateAuthMechanism(mec string) error {
	list := []interface{}{"PLAIN", "LOGIN", "CRAM-MD5"}

	if mec == "" {
		return errors.RequiredParameter("event_smtp_notifier.auth_mechanism")
	}

	for _, l := range list {
		if l == mec {
			return nil
		}
	}

	return errors.UnavailableParameterValue("event_smtp_notifier.auth_mechanism", mec, list)
}

func validateAuthUsername(name string) error {
	if name == "" {
		return errors.RequiredParameter("event_smtp_notifier.auth_username")
	}

	if len(name) > 1024 {
		return errors.LengthOverflowParameterValue("event_smtp_notifier.auth_username", name, 1024)
	}

	return nil
}

func validateAuthPassword(password string) error {
	if password == "" {
		return errors.RequiredParameter("auth_username")
	}

	if len(password) > 1024 {
		return errors.LengthOverflowParameterValue("event_smtp_notifier.auth_password", password, 1024)
	}

	return nil
}

func validateSender(sender string) error {
	if len(sender) > 1024 {
		return errors.LengthOverflowParameterValue("event_smtp_notifier.sender", sender, 1024)
	}

	return nil
}

func validateProvider(provider string) error {
	if provider == "" {
		return errors.RequiredParameter("event_sms_notifier.provider")
	}

	if len(provider) > 1024 {
		return errors.LengthOverflowParameterValue("event_sms_notifier.provider", provider, 1024)
	}

	return nil
}

func validateVersion(version string) error {
	if version == "" {
		return errors.RequiredParameter("event_sms_notifier.version")
	}

	if len(version) > 1024 {
		return errors.LengthOverflowParameterValue("event_sms_notifier.version", version, 1024)
	}

	return nil
}

var defaultConfig = map[string]string{
	config.EventNotificationEnable:        "false",
	config.EventEmailNotificationEnable:   "false",
	config.EventDesktopNotificationEnable: "false",
	config.EventPopupNotificationEnable:   "false",
	config.EventSmsNotificationEnable:     "false",
	config.EventCustomNotificationEnable:  "false",
	config.EventStorePeriod:               "12",
	config.EventEmailNotifier:             "{}",
	config.EventSMSNotifier:               "{}",
}

func getConfigOne(db *gorm.DB, tenantID uint64, key string) (string, error) {
	var ret model.TenantConfig

	err := db.Find(&ret, &model.TenantConfig{
		TenantID: tenantID,
		Key:      key,
	}).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		ret.Value = defaultConfig[key]
	case err != nil:
		return "", errors.UnusableDatabase(err)
	}
	return ret.Value, nil
}

func getConfigBool(db *gorm.DB, tenantID uint64, key string) (*wrappers.BoolValue, error) {
	s, err := getConfigOne(db, tenantID, key)
	if err != nil {
		return nil, err
	}

	v, err := strconv.ParseBool(s)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	return &wrappers.BoolValue{
		Value: v,
	}, nil
}

func getConfigUint32(db *gorm.DB, tenantID uint64, key string) (*wrappers.UInt32Value, error) {
	s, err := getConfigOne(db, tenantID, key)
	if err != nil {
		return nil, err
	}

	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	return &wrappers.UInt32Value{
		Value: uint32(v),
	}, nil
}

func getConfigEventSMTPNotifier(db *gorm.DB, tenantID uint64, key string) (*notification.EventSMTPNotifier, error) {
	s, err := getConfigOne(db, tenantID, key)
	if err != nil {
		return nil, err
	}

	var ret notification.EventSMTPNotifier
	err = json.Unmarshal([]byte(s), &ret)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	return &ret, nil
}

func getConfigEventSMSNotifier(db *gorm.DB, tenantID uint64, key string) (*notification.EventSMSNotifier, error) {
	s, err := getConfigOne(db, tenantID, key)
	if err != nil {
		return nil, err
	}

	var ret notification.EventSMSNotifier
	err = json.Unmarshal([]byte(s), &ret)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	return &ret, nil
}

func setConfigOne(db *gorm.DB, tenantID uint64, key, value string) error {
	if err := db.Save(&model.TenantConfig{
		TenantID: tenantID,
		Key:      key,
		Value:    value,
	}).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	return nil
}

func setConfigBool(db *gorm.DB, tenantID uint64, key string, value *wrappers.BoolValue) error {
	if value == nil {
		return errors.RequiredParameter(key)
	}

	return setConfigOne(db, tenantID, key, strconv.FormatBool(value.GetValue()))
}

func setConfigUint32(db *gorm.DB, tenantID uint64, key string, value *wrappers.UInt32Value) error {
	if value == nil {
		return errors.RequiredParameter(key)
	}

	return setConfigOne(db, tenantID, key, strconv.FormatUint(uint64(value.GetValue()), 10))
}

func setConfigEventSMTPNotifier(db *gorm.DB, tenantID uint64, key string, value *notification.EventSMTPNotifier) error {
	if value == nil {
		return errors.RequiredParameter(key)
	}

	b, err := json.Marshal(value)
	if err != nil {
		return errors.Unknown(err)
	}

	return setConfigOne(db, tenantID, key, string(b))
}

func setConfigEventSMSNotifier(db *gorm.DB, tenantID uint64, key string, value *notification.EventSMSNotifier) error {
	if value == nil {
		return errors.RequiredParameter(key)
	}

	b, err := json.Marshal(value)
	if err != nil {
		return errors.Unknown(err)
	}

	return setConfigOne(db, tenantID, key, string(b))
}
