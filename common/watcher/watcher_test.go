package watcher

import (
	"github.com/datacommand2/cdm-cloud/common/broker"
	"github.com/datacommand2/cdm-cloud/common/config"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/database"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/datacommand2/cdm-cloud/common/test/helper"
	"github.com/jinzhu/gorm"
	mlogger "github.com/micro/go-micro/v2/logger"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	var err error
	if err = helper.Init(); err != nil {
		panic(err)
	} else {
		defer helper.Close()
	}

	if code := m.Run(); code != 0 {
		os.Exit(code)
	}
}

func TestLoggingLevelUpdate(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var err error
		var globalConfig model.GlobalConfig
		var serviceConfig model.ServiceConfig
		var serviceName = "test-service"

		err = db.Where("key = ? AND name = ?", config.ServiceLogLevel, serviceName).First(&serviceConfig).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			panic(err)
		} else {
			serviceConfig.Key = config.ServiceLogLevel
			serviceConfig.Name = serviceName
		}

		err = db.Where("key = ?", config.GlobalLogLevel).First(&globalConfig).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			panic(err)
		} else {
			globalConfig.Key = config.GlobalLogLevel
		}

		// set service logging level to error
		serviceConfig.Value = mlogger.ErrorLevel.String()
		if err := db.Save(&serviceConfig).Error; err != nil {
			panic(err)
		}

		// set global logging level to warn
		globalConfig.Value = mlogger.WarnLevel.String()
		if err := db.Save(&globalConfig).Error; err != nil {
			panic(err)
		}

		// start watching
		assert.NoError(t, Watch(serviceName))

		// global: warn, service: error
		assert.Equal(t, mlogger.ErrorLevel, logger.DefaultLogger.Options().Level)

		// wait for subscribe
		time.Sleep(2 * time.Second)

		// set service logging level to null
		if err := db.Delete(&serviceConfig).Error; err != nil {
			panic(err)
		}

		// publish service logging level updated notice
		if err := broker.Publish(
			constant.TopicNoticeServiceLoggingLevelUpdated,
			&broker.Message{Body: []byte(serviceName)},
		); err != nil {
			panic(err)
		}

		// wait for subscribe message and update
		time.Sleep(2 * time.Second)

		// global: warn, service: nil
		assert.Equal(t, mlogger.WarnLevel, logger.DefaultLogger.Options().Level)

		// set global logging level to info
		globalConfig.Value = mlogger.InfoLevel.String()
		if err := db.Save(&globalConfig).Error; err != nil {
			panic(err)
		}

		// publish global logging level updated notice
		if err := broker.Publish(
			constant.TopicNoticeGlobalLoggingLevelUpdated,
			&broker.Message{Body: []byte("")},
		); err != nil {
			panic(err)
		}

		// wait for subscribe message and update
		time.Sleep(2 * time.Second)

		// global: info, service: nil
		assert.Equal(t, mlogger.InfoLevel, logger.DefaultLogger.Options().Level)

		// stop watching
		assert.NoError(t, Stop())

		// set global logging level to error
		globalConfig.Value = mlogger.ErrorLevel.String()
		if err := db.Save(&globalConfig).Error; err != nil {
			panic(err)
		}

		// publish global logging level updated notice
		if err := broker.Publish(
			constant.TopicNoticeGlobalLoggingLevelUpdated,
			&broker.Message{Body: []byte("")},
		); err != nil {
			panic(err)
		}

		// after does not update logging level
		assert.Equal(t, mlogger.InfoLevel, logger.DefaultLogger.Options().Level)
	})
}
