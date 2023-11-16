package handler

import (
	"github.com/datacommand2/cdm-cloud/common/errors"
	"strconv"
	"time"

	"github.com/datacommand2/cdm-cloud/common/config"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	identity "github.com/datacommand2/cdm-cloud/services/identity/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/jinzhu/gorm"
)

func validateConfig(req *identity.ConfigRequest) error {
	if req.IdentityConfig == nil {
		return errors.RequiredParameter("identity_config")
	}

	var (
		timezone                     = req.IdentityConfig.GetGlobalTimezone()
		globalLanguageSet            = req.IdentityConfig.GetGlobalLanguageSet()
		userLoginRestrictionEnable   = req.IdentityConfig.GetUserLoginRestrictionEnable()
		userLoginRestrictionTryCount = req.IdentityConfig.GetUserLoginRestrictionTryCount()
		userLoginRestrictionTime     = req.IdentityConfig.GetUserLoginRestrictionTime()
		userReuseOldPassword         = req.IdentityConfig.GetUserReuseOldPassword()
		userPasswordChangeCycle      = req.IdentityConfig.GetUserPasswordChangeCycle()
		userSessionTimeout           = req.IdentityConfig.GetUserSessionTimeout()
		err                          error
	)

	// completeness check
	if timezone == nil {
		return errors.RequiredParameter("global_timezone")
	}
	if globalLanguageSet == nil {
		return errors.RequiredParameter("global_language_set")
	}
	if userReuseOldPassword == nil {
		return errors.RequiredParameter("user_reuse_old_password")
	}
	if userPasswordChangeCycle == nil {
		return errors.RequiredParameter("user_password_change_cycle")
	}
	if userSessionTimeout == nil {
		return errors.RequiredParameter("user_session_timeout")
	}
	if userLoginRestrictionEnable == nil {
		return errors.RequiredParameter("user_login_restriction_enable")
	}

	// validity check
	if _, err = time.LoadLocation(timezone.GetValue()); err != nil {
		return errors.UnavailableParameterValue("global_timezone", timezone.GetValue(), []interface{}{"https://en.wikipedia.org/wiki/List_of_tz_database_time_zones"})
	}
	if !languageBoundary.enum(globalLanguageSet.GetValue()) {
		return errors.UnavailableParameterValue("global_language_set", globalLanguageSet.GetValue(), []interface{}{"kor", "eng"})
	}
	if !userPasswordChangeCycleBoundary.minMax(userPasswordChangeCycle.GetValue()) {
		return errors.OutOfRangeParameterValue("user_password_change_cycle", userPasswordChangeCycle.GetValue(), 0, 180)
	}
	if !userSessionTimeoutBoundary.minMax(userSessionTimeout.GetValue()) {
		return errors.OutOfRangeParameterValue("user_session_timeout", userSessionTimeout.GetValue(), 1, 1440)
	}

	// userLoginRestriction
	if userLoginRestrictionEnable.GetValue() {
		// completeness check
		if userLoginRestrictionTryCount == nil {
			return errors.RequiredParameter("user_login_restriction_try_count")
		}
		if userLoginRestrictionTime == nil {
			return errors.RequiredParameter("user_login_restriction_time")
		}

		// validity check
		if !userLoginRestrictionTryCountBoundary.minMax(userLoginRestrictionTryCount.GetValue()) {
			return errors.OutOfRangeParameterValue("user_login_restriction_try_count", userLoginRestrictionTryCount.GetValue(), 5, 30)
		}
		if !userLoginRestrictionTimeBoundary.minMax(uint64(userLoginRestrictionTime.GetValue())) {
			return errors.OutOfRangeParameterValue("user_login_restriction_time", userLoginRestrictionTime.GetValue(), 10, 7200)
		}
	}

	return nil
}

func getConfig(db *gorm.DB, tenantID uint64, rsp *identity.ConfigResponse) error {
	var cfg *config.Config
	if cfg = config.TenantConfig(db, tenantID, config.GlobalLanguageSet); cfg == nil {
		return notfoundTenantConfig(config.GlobalLanguageSet)
	}

	var rspCfg = &identity.Config{}
	rspCfg.GlobalLanguageSet = &wrappers.StringValue{Value: cfg.Value.String()}

	if cfg = config.TenantConfig(db, tenantID, config.GlobalTimeZone); cfg == nil {
		return notfoundTenantConfig(config.GlobalTimeZone)
	}
	rspCfg.GlobalTimezone = &wrappers.StringValue{Value: cfg.Value.String()}

	if cfg = config.TenantConfig(db, tenantID, config.UserPasswordChangeCycle); cfg == nil {
		return notfoundTenantConfig(config.UserPasswordChangeCycle)
	}
	uint64Value, err := cfg.Value.Uint64()
	if err != nil {
		return invalidTenantConfig(config.UserPasswordChangeCycle, cfg.Value.String())
	}
	rspCfg.UserPasswordChangeCycle = &wrappers.UInt64Value{Value: uint64Value}

	if cfg = config.TenantConfig(db, tenantID, config.UserReuseOldPassword); cfg == nil {
		return notfoundTenantConfig(config.UserReuseOldPassword)
	}
	boolValue, err := cfg.Value.Bool()
	if err != nil {
		return invalidTenantConfig(config.UserReuseOldPassword, cfg.Value.String())
	}
	rspCfg.UserReuseOldPassword = &wrappers.BoolValue{Value: boolValue}

	if cfg = config.TenantConfig(db, tenantID, config.UserSessionTimeout); cfg == nil {
		return notfoundTenantConfig(config.UserSessionTimeout)
	}
	uint64Value, err = cfg.Value.Uint64()
	if err != nil {
		return invalidTenantConfig(config.UserSessionTimeout, cfg.Value.String())
	}
	rspCfg.UserSessionTimeout = &wrappers.UInt64Value{Value: uint64Value}

	if cfg = config.TenantConfig(db, tenantID, config.UserLoginRestrictionEnable); cfg == nil {
		return notfoundTenantConfig(config.UserLoginRestrictionEnable)
	}
	boolValue, err = cfg.Value.Bool()
	if err != nil {
		return invalidTenantConfig(config.UserLoginRestrictionEnable, cfg.Value.String())
	}
	rspCfg.UserLoginRestrictionEnable = &wrappers.BoolValue{Value: boolValue}

	if rspCfg.GetUserLoginRestrictionEnable().GetValue() {
		if cfg = config.TenantConfig(db, tenantID, config.UserLoginRestrictionTryCount); cfg == nil {
			return notfoundTenantConfig(config.UserLoginRestrictionTryCount)
		}
		uint64Value, err = cfg.Value.Uint64()
		if err != nil {
			return invalidTenantConfig(config.UserLoginRestrictionTryCount, cfg.Value.String())
		}
		rspCfg.UserLoginRestrictionTryCount = &wrappers.UInt64Value{Value: uint64Value}

		if cfg = config.TenantConfig(db, tenantID, config.UserLoginRestrictionTime); cfg == nil {
			return notfoundTenantConfig(config.UserLoginRestrictionTime)
		}
		int64Value, err := cfg.Value.Int64()
		if err != nil {
			return invalidTenantConfig(config.UserLoginRestrictionTime, cfg.Value.String())
		}
		rspCfg.UserLoginRestrictionTime = &wrappers.Int64Value{Value: int64Value}
	}

	rsp.IdentityConfig = rspCfg
	return nil
}

func setConfig(db *gorm.DB, tenantID uint64, req *identity.ConfigRequest) error {
	var err error
	if err = validateConfig(req); err != nil {
		return err
	}

	var cfg = model.TenantConfig{TenantID: tenantID}

	cfg.Key = config.GlobalLanguageSet
	cfg.Value = req.IdentityConfig.GetGlobalLanguageSet().GetValue()
	if err = db.Save(&cfg).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	cfg.Key = config.GlobalTimeZone
	cfg.Value = req.IdentityConfig.GetGlobalTimezone().GetValue()
	if err = db.Save(&cfg).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	cfg.Key = config.UserPasswordChangeCycle
	cfg.Value = strconv.Itoa(int(req.IdentityConfig.GetUserPasswordChangeCycle().GetValue()))
	if err = db.Save(&cfg).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	cfg.Key = config.UserReuseOldPassword
	cfg.Value = strconv.FormatBool(req.IdentityConfig.UserReuseOldPassword.GetValue())
	if err = db.Save(&cfg).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	cfg.Key = config.UserSessionTimeout
	cfg.Value = strconv.Itoa(int(req.IdentityConfig.GetUserSessionTimeout().GetValue()))
	if err = db.Save(&cfg).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	cfg.Key = config.UserLoginRestrictionEnable
	cfg.Value = strconv.FormatBool(req.IdentityConfig.GetUserLoginRestrictionEnable().GetValue())
	if err = db.Save(&cfg).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	if req.IdentityConfig.GetUserLoginRestrictionEnable().GetValue() {
		cfg.Key = config.UserLoginRestrictionTryCount
		cfg.Value = strconv.Itoa(int(req.IdentityConfig.GetUserLoginRestrictionTryCount().GetValue()))
		if err = db.Save(&cfg).Error; err != nil {
			return errors.UnusableDatabase(err)
		}

		cfg.Key = config.UserLoginRestrictionTime
		cfg.Value = strconv.Itoa(int(req.IdentityConfig.GetUserLoginRestrictionTime().GetValue()))
		if err = db.Save(&cfg).Error; err != nil {
			return errors.UnusableDatabase(err)
		}
	}

	return nil
}

type stringBoundary struct {
	Enum []string
}

func (b *stringBoundary) enum(s string) bool {
	for _, v := range b.Enum {
		if v == s {
			return true
		}
	}
	return false
}

type numericBoundary struct {
	min uint64
	max uint64
}

func (b *numericBoundary) minMax(i uint64) bool {
	return !(i < b.min || i > b.max)
}

var (
	languageBoundary                     = stringBoundary{Enum: []string{"eng", "kor"}}
	userPasswordChangeCycleBoundary      = numericBoundary{max: 180}
	userSessionTimeoutBoundary           = numericBoundary{min: 1, max: 1440}
	userLoginRestrictionTryCountBoundary = numericBoundary{min: 5, max: 30}
	userLoginRestrictionTimeBoundary     = numericBoundary{min: 10, max: 7200}
)
