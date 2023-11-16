package handler

import (
	"encoding/json"
	"github.com/datacommand2/cdm-cloud/common/broker"
	"github.com/datacommand2/cdm-cloud/common/config"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/database"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/logger"
	types "github.com/datacommand2/cdm-cloud/services/scheduler/constants"
	"github.com/jinzhu/gorm"
	"math"
	"time"
)

func getDeleteLogFilesSchedule(topic string) (*model.Schedule, error) {
	var (
		err      error
		schedule model.Schedule
	)

	if err = database.Execute(func(db *gorm.DB) error {
		return db.Table(model.Schedule{}.TableName()).
			Where(&model.Schedule{Topic: topic}).
			First(&schedule).Error
	}); err != nil {
		return nil, err
	}

	return &schedule, nil
}

func createDeleteLogFilesSchedule(topic string) error {
	interval := uint(1)
	zero := uint(0)
	// 매일 자정에 동작하도록 하는 schedule
	schedule := &model.Schedule{
		ActivationFlag: true,
		Topic:          topic,
		StartAt:        time.Now().Unix(),
		EndAt:          math.MaxInt64,
		Type:           types.ScheduleTypeDaily,
		IntervalDay:    &interval,
		Hour:           &zero,
		Minute:         &zero,
		Timezone:       "Asia/Seoul", // 임시(변경필요시 수정)
	}

	if err := database.GormTransaction(func(db *gorm.DB) error {
		return createSchedule(db, schedule)
	}); err != nil {
		logger.Errorf("[createDeleteLogFilesSchedule] Could not create schedule. Cause: %v", err)
		return err
	}

	return nil
}

func updateDeleteLogFilesSchedule(nodeType string) error {
	var (
		err                     error
		expectedTopic, oldTopic string
	)

	if nodeType == config.ServiceNodeTypeSingle {
		expectedTopic = constant.TopicNoticeSingleDeleteExpiredLogFiles
		oldTopic = constant.TopicNoticeMultipleDeleteExpiredLogFiles
	} else { // config.ServiceNodeTypeMultiple
		expectedTopic = constant.TopicNoticeMultipleDeleteExpiredLogFiles
		oldTopic = constant.TopicNoticeSingleDeleteExpiredLogFiles
	}

	schedule, err := getDeleteLogFilesSchedule(expectedTopic)
	switch err {
	// 이미 존재
	case nil:
		//logger.Infof("[updateDeleteLogFilesSchedule] Same topic(%s:%s) schedule is already existed.", nodeType, expectedTopic)
		return nil

	// 존재하지 않음
	case gorm.ErrRecordNotFound:
		// 설정과는 다른 type 의 schedule 이 존재하는지 확인
		schedule, err = getDeleteLogFilesSchedule(oldTopic)
		// 존재하면 수정
		if err == nil {
			schedule.Topic = expectedTopic
			if err = database.GormTransaction(func(db *gorm.DB) error {
				return updateSchedule(db, schedule)
			}); err != nil {
				logger.Errorf("[updateDeleteLogFilesSchedule] Could not update schedule. Cause: %v", err)
				return err
			}

		} else if err == gorm.ErrRecordNotFound {
			// 존재하지 않으면 생성
			if err = createDeleteLogFilesSchedule(expectedTopic); err != nil {
				return err
			}

		} else {
			logger.Errorf("[updateDeleteLogFilesSchedule] Could not get schedule for init schedule.")
			return err
		}

	default:
		logger.Errorf("[updateDeleteLogFilesSchedule] Could not get schedule for init schedule.")
		return err
	}

	return nil
}

// InitDeleteLogFilesSchedule delete log files schedule 초기화
func (h *SchedulerHandler) InitDeleteLogFilesSchedule() error {
	logger.Infof("Initializing schedule for deleting expired log files .")
	var (
		err      error
		nodeType string
	)

	database.Execute(func(db *gorm.DB) error {
		cfg := config.ServiceConfig(db, config.ServiceNode, config.ServiceNodeType)
		// node type 이 multiple 일때
		if cfg != nil && cfg.Value.String() == config.ServiceNodeTypeMultiple {
			nodeType = config.ServiceNodeTypeMultiple
		} else {
			// node type 이 설정되지 않았거나(default), single 일 때
			nodeType = config.ServiceNodeTypeSingle
		}
		return nil
	})

	if err = updateDeleteLogFilesSchedule(nodeType); err != nil {
		return err
	}

	// 파일 유지기한이 지난 파일들 삭제
	if hour := time.Now().Hour(); hour != 0 && hour != 23 { // 00시이거나 23시 일때는 진행하지 않음
		if err = config.DeleteExpiredServiceLogFiles(); err != nil {
			logger.Errorf("Could not complete the deleting expired log files. Cause: %v", err)
		}
	}

	return nil
}

func (h *SchedulerHandler) updateLogFilesScheduleHandler(e broker.Event) error {
	logger.Infof("[updateLogFilesScheduleHandler] Updating delete expired log files schedule.")
	var (
		err error
		msg string
	)

	if err = json.Unmarshal(e.Message().Body, &msg); err != nil {
		err = errors.Unknown(err)
		logger.Errorf("[updateLogFilesScheduleHandler] Could not delete schedule job. cause: %+v", err)
		return err
	}

	if err = updateDeleteLogFilesSchedule(msg); err != nil {
		return err
	}
	return nil
}

func (h *SchedulerHandler) deleteLogFilesScheduleHandler(e broker.Event) error {
	if err := config.DeleteExpiredServiceLogFiles(); err != nil {
		logger.Errorf("[deleteLogFilesScheduleHandler] Could not complete the deleting expired log files. Cause: %v", err)
	}
	return nil
}

// RunDeleteLogFilesSchedule delete log files schedule 관련된 publish 한 메세지를 처리 하기 위한 subscriber 를 등록
func (h *SchedulerHandler) RunDeleteLogFilesSchedule() error {
	sub, err := broker.SubscribeTempQueue(constant.TopicNoticeServiceLogNodeTypeUpdated, h.updateLogFilesScheduleHandler, broker.RequeueOnError())
	if err != nil {
		err = errors.UnusableBroker(err)
		logger.Errorf("[RunDeleteLogFilesSchedule] Could not subscribe queue(%s). cause: %+v", constant.TopicNoticeServiceLogNodeTypeUpdated, err)
		return err
	}
	h.subs = append(h.subs, sub)

	sub, err = broker.SubscribeTempQueue(constant.TopicNoticeSingleDeleteExpiredLogFiles, h.deleteLogFilesScheduleHandler)
	if err != nil {
		err = errors.UnusableBroker(err)
		logger.Errorf("[RunDeleteLogFilesSchedule] Could not subscribe queue(%s). cause: %+v", constant.TopicNoticeSingleDeleteExpiredLogFiles, err)
		return err
	}
	h.subs = append(h.subs, sub)

	return nil
}
