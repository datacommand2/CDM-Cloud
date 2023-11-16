package event

import (
	"encoding/json"
	"fmt"
	"github.com/datacommand2/cdm-cloud/common/broker"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/database"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	e "github.com/datacommand2/cdm-cloud/common/event"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/datacommand2/cdm-cloud/services/notification/notifier"
	notification "github.com/datacommand2/cdm-cloud/services/notification/proto"
	"github.com/jinzhu/gorm"
	"reflect"
)

// Record 이벤트를 저장한다.
type Record struct {
	Event     model.Event     `gorm:"embedded" json:"event"`
	EventCode model.EventCode `gorm:"embedded" json:"event_code"`
	Tenant    model.Tenant    `gorm:"embedded" json:"tenant"`
}

func reportEvent(tid uint64, eventCode, errorCode string, eventContents interface{}) {
	err := e.ReportEvent(tid, eventCode, errorCode, e.WithContents(eventContents))
	if err != nil {
		logger.Warnf("Could not report event. cause: %+v", errors.Unknown(err))
	}
}

// LOG ok
func subscribeEvent(p broker.Event) error {
	var record *Record
	event := model.Event{}
	err := json.Unmarshal(p.Message().Body, &event)
	if err != nil {
		logger.Errorf("Could not subscribe event. cause: %+v", errors.Unknown(err))
		return err
	}

	if event.TenantID == 0 || event.Code == "" {
		return nil
	}

	err = database.Transaction(func(db *gorm.DB) error {
		if err := createEvent(db, &event); err != nil {
			return err
		}
		record, err = GetEvent(db, event.ID, event.TenantID)
		return err
	})
	switch {
	case errors.Equal(err, ErrNotFoundEvent) ||
		errors.Equal(err, errors.ErrUnusableDatabase) ||
		errors.Equal(err, errors.ErrUnknown) ||
		errors.Equal(err, errors.ErrInvalidParameterValue):
		logger.Errorf("Could not subscribe event. cause: %+v", err)
		return err

	case err != nil:
		err = errors.Unknown(err)
		logger.Errorf("Could not subscribe event. cause: %+v", err)
		return err
	}

	go func() {
		b, err := json.Marshal(&record)
		if err != nil {
			logger.Warnf("Could not subscribe event. cause: %+v", errors.Unknown(err))
		}

		topic := fmt.Sprintf(constant.TopicNotificationEventCreated, event.TenantID)
		if err = broker.Publish(topic, &broker.Message{Body: b}); err != nil {
			logger.Warnf("Could not subscribe event. cause: %+v", errors.UnusableBroker(err))
		}
	}()

	go func() {
		err = notifier.ClassifyEvent(&record.Event)
		switch {
		case errors.Equal(err, errors.ErrUnusableDatabase):
			logger.Warnf("Could not subscribe event. cause: %+v", err)
			reportEvent(event.TenantID, "cdm-cloud.notification.subscribe_event.failure-classify_event", "unusable_database", err)

		case errors.Equal(err, errors.ErrUnusableBroker):
			logger.Warnf("Could not subscribe event. cause: %+v", err)
			reportEvent(event.TenantID, "cdm-cloud.notification.subscribe_event.failure-classify_event", "unusable_broker", err)

		case errors.Equal(err, errors.ErrUnknown):
			logger.Warnf("Could not subscribe event. cause: %+v", err)
			reportEvent(event.TenantID, "cdm-cloud.notification.subscribe_event.failure-classify_event", "unknown", err)

		case err != nil:
			err = errors.Unknown(err)
			logger.Warnf("Could not subscribe event. cause: %+v", err)
			reportEvent(event.TenantID, "cdm-cloud.notification.subscribe_event.failure-classify_event", "unknown", err)
		}
	}()

	return nil
}

// Subscribe constant.QueueReportEvent 토픽에 구독하는 subscriber 를 생성한다.
func Subscribe() (broker.Subscriber, error) {
	sub, err := broker.SubscribePersistentQueue(constant.QueueReportEvent, subscribeEvent, false)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func validateContents(contents string) error {
	if contents == "" {
		return nil
	}

	if err := json.Unmarshal([]byte(contents), new(map[string]interface{})); err != nil {
		return errors.Unknown(err)
	}

	return nil
}

func validateEvent(tx *gorm.DB, modelEvent *model.Event) error {
	err := validateContents(modelEvent.Contents)
	if err != nil {
		return err
	}

	_, err = modelEvent.Tenant(tx)
	switch {
	case err != nil && err == gorm.ErrRecordNotFound:
		return errors.InvalidParameterValue("tenant", modelEvent.TenantID, err.Error())

	case err != nil:
		return errors.UnusableDatabase(err)
	}

	_, err = modelEvent.EventCode(tx)
	switch {
	case err != nil && err == gorm.ErrRecordNotFound:
		logger.Warnf("Could not found event code(%+v).", modelEvent.Code)

	case err != nil:
		return errors.UnusableDatabase(err)
	}

	if modelEvent.ErrorCode == nil || *modelEvent.ErrorCode == "" {
		return nil
	}

	_, err = modelEvent.EventError(tx)
	switch {
	case err != nil && err == gorm.ErrRecordNotFound:
		logger.Warnf("Could not found event error code(%+v).", *modelEvent.ErrorCode)

	case err != nil:
		return errors.UnusableDatabase(err)
	}

	return nil
}

// createEvent 이벤트를 database 에 삽입한다.
func createEvent(db *gorm.DB, event *model.Event) error {
	err := validateEvent(db, event)
	if err != nil {
		return err
	}

	// ID 는 무시한다.
	event.ID = 0
	if err = db.Create(&event).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	return nil
}

// GetEvent 이벤트를 database 에서 검색한다.
func GetEvent(db *gorm.DB, eventID, tenantID uint64) (*Record, error) {
	var (
		ret Record
		err error
	)

	err = db.Table("cdm_event").
		Select("*").
		Joins("join cdm_tenant ON  cdm_event.tenant_id = cdm_tenant.id").
		Joins("left join cdm_event_code ON cdm_event.code = cdm_event_code.code").
		Where(&model.Event{ID: eventID, TenantID: tenantID}).
		Scan(&ret).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, NotFoundEvent(eventID)

	case err != nil:
		return nil, errors.UnusableDatabase(err)
	}

	return &ret, nil
}

// GetEvents 이벤트를 database 에서 검색한다.
func GetEvents(query *EventsQuery) ([]Record, uint64, error) {
	var (
		ret   []Record
		count uint64
		err   error
	)

	err = database.Transaction(func(db *gorm.DB) error {
		db = db.Table("cdm_event").
			Select("*").
			Joins("join cdm_tenant ON  cdm_event.tenant_id = cdm_tenant.id").
			Joins("left join cdm_event_code ON cdm_event.code = cdm_event_code.code")

		db = db.Where(&query.event)

		if query.solution != "" {
			db = db.Where("cdm_event_code.solution IN (?)", query.solution)
		}

		if query.eventCode != (model.EventCode{}) {
			db = db.Where(query.eventCode)
		}
		if query.from != 0 {
			db = db.Where("cdm_event.created_at >= ?", query.from)
		}
		if query.to != 0 {
			db = db.Where("cdm_event.created_at <= ?", query.to)
		}

		err := db.Count(&count).Error
		if err != nil {
			return errors.UnusableDatabase(err)
		}

		db = db.Order("cdm_event.created_at desc").Order("cdm_event.id desc")
		if query.limit > 0 {
			db = db.Limit(query.limit)
		}

		if query.offset > 0 {
			db = db.Offset(query.offset)
		}

		err = db.Scan(&ret).Error
		if err != nil {
			return errors.UnusableDatabase(err)
		}

		return nil
	})
	switch {
	case errors.Equal(err, errors.ErrUnusableDatabase):
		return nil, 0, err

	case err != nil:
		return nil, 0, errors.Unknown(err)
	}

	return ret, count, nil
}

// GetEventClassifications 이벤트 분류 목록 조회
func GetEventClassifications() ([]*notification.EventClassification, error) {
	var eventCodes []*model.EventCode
	if err := database.Transaction(func(db *gorm.DB) error {
		// group by 시, PK 가 반드시 포함되어야 한다는 에러 발생함
		// distinct 시, gorm v1 에서는 조회 결과 값을 가져올 수 없음, []map[string]interface{} 조회도 지원 안됨
		// 단순 조회 후 프로그래밍으로 distinct 처리
		if err := db.Select("solution, class_1, class_2, class_3").Find(&eventCodes).Error; err != nil {
			return errors.UnusableDatabase(err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	var ret []*notification.EventClassification
	prev := &model.EventCode{}
	for _, item := range eventCodes {
		if reflect.DeepEqual(prev, item) {
			continue
		}
		ret = append(ret, &notification.EventClassification{
			Solution: item.Solution,
			Class_1:  item.Class1,
			Class_2:  item.Class2,
			Class_3:  item.Class3,
		})
		prev = item
	}
	return ret, nil
}
