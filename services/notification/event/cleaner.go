package event

import (
	"github.com/datacommand2/cdm-cloud/common/config"
	"github.com/datacommand2/cdm-cloud/common/database"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/jinzhu/gorm"
	"time"
)

const monthInSeconds = 30 * 24 * 3600

var defaultEventStorePeriod = int64(12)

// Cleaner 자정마다 보유기간이 지난 이벤트를 지우는 기능을 수행한다.
type Cleaner struct {
	done     chan bool
	tenantID uint64
}

// NewCleaner Cleaner 생성
func NewCleaner(tenantID uint64) *Cleaner {
	return &Cleaner{
		tenantID: tenantID,
	}
}

func cleanExpiredTenantEvent(db *gorm.DB, tenant model.Tenant) error {
	now := time.Now().UTC()
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	period := defaultEventStorePeriod
	if cfg := config.TenantConfig(db, tenant.ID, config.EventStorePeriod); cfg != nil {
		if p, err := cfg.Value.Int64(); err == nil {
			period = p
		}
	}

	if period == 0 {
		return nil
	}

	err := db.Where("created_at < ?", midnight.Unix()-period*monthInSeconds).
		Delete(&model.Event{}, &model.Event{TenantID: tenant.ID}).Error
	if err != nil {
		return errors.UnusableDatabase(err)
	}

	return nil
}

func (c *Cleaner) cleanExpiredEvent() {
	err := database.Transaction(func(db *gorm.DB) error {
		var tenants []model.Tenant

		if err := db.Find(&tenants).Error; err != nil {
			return err
		}

		for _, tenant := range tenants {
			if err := cleanExpiredTenantEvent(db, tenant); err != nil {
				logger.Warnf("Could not clean Expired Tenant(%s) Event. cause: %+v", tenant.Name, err)
				reportEvent(tenant.ID, "cdm-cloud.notification.main.failure-clean_expired_event", "unusable_database", err)
			}
		}

		return nil
	})
	switch {
	case err != nil:
		err = errors.UnusableDatabase(err)
		logger.Warnf("Could not clean Expired Event. cause: %+v", err)
		reportEvent(c.tenantID, "cdm-cloud.notification.main.failure-clean_expired_event", "unusable_database", err)
	}
}

func nextMidnightTicker() *time.Ticker {
	now := time.Now().UTC()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)

	return time.NewTicker(nextMidnight.Sub(now))
}

func (c *Cleaner) cleaner() {
	for {
		c.cleanExpiredEvent()

		ticker := nextMidnightTicker()
		select {
		case <-ticker.C:
			ticker.Stop()
			continue

		case <-c.done:
			ticker.Stop()
			return
		}
	}
}

// Start 시작
func (c *Cleaner) Start() {
	c.done = make(chan bool)
	go c.cleaner()
}

// Stop 종료
func (c *Cleaner) Stop() {
	c.done <- true
	close(c.done)
}
