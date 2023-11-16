package scheduler

import (
	"encoding/json"
	"github.com/datacommand2/cdm-cloud/common/broker"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/database"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/event"
	"github.com/datacommand2/cdm-cloud/common/logger"
	types "github.com/datacommand2/cdm-cloud/services/scheduler/constants"
	"github.com/datacommand2/cdm-cloud/services/scheduler/internal/scheduler/executor"
	"github.com/jinzhu/gorm"
	"sync"
	"time"
)

// Scheduler 스케줄러 서비스에서 스케줄링을 위한 구조체
// Simple 형식과 cron 형식으로 스케줄링을 진행 한다.
type Scheduler struct {
	lock            *sync.Mutex
	defaultTenantID uint64

	simpleExecutorMap map[string]executor.Executor
	cronExecutorMap   map[string]executor.Executor

	subCreate broker.Subscriber
	subUpdate broker.Subscriber
	subDelete broker.Subscriber
}

func parseScheduleID(e broker.Event) (uint64, error) {
	var id uint64

	if err := json.Unmarshal(e.Message().Body, &id); err != nil {
		return 0, err
	}
	return id, nil
}

func findSchedule(id uint64) (*model.Schedule, error) {
	var (
		schedule model.Schedule
		err      error
	)

	timeout := time.After(time.Second * 3)
	for {
		select {
		case <-timeout:
			logger.Errorf("[findSchedule] Could not find schedule(%d). cause: %v", id, err)
			return nil, err

		default:
			err = database.Transaction(func(db *gorm.DB) error {
				return db.First(&schedule, id).Error
			})
			if err == nil {
				return &schedule, nil
			}
			logger.Warnf("[findSchedule] Could not find schedule(%d). cause: %v", id, err)
			time.Sleep(1 * time.Second)
		}
	}
}

func (s *Scheduler) reportEvent(eventCode, errorCode string, eventContents interface{}) {
	err := event.ReportEvent(s.defaultTenantID, eventCode, errorCode, event.WithContents(eventContents))
	if err != nil {
		logger.Warnf("[reportEvent] Could not report event. cause: %+v", errors.Unknown(err))
	}
}

func (s *Scheduler) createScheduleHandler(e broker.Event) error {
	var id uint64
	var err error

	defer func() {
		if err != nil {
			logger.Errorf("[createScheduleHandler] Could not create schedule job. cause: %+v", err)
		}
	}()

	id, err = parseScheduleID(e)
	if err != nil {
		err = errors.Unknown(err)
		s.reportEvent("cdm-cloud.scheduler.create_schedule_handler.failure-parse_schedule_id", "unknown", err)
		return err
	}

	schedule, err := findSchedule(id)
	if err != nil {
		err = errors.UnusableDatabase(err)
		s.reportEvent("cdm-cloud.scheduler.create_schedule_handler.failure-find_schedule", "unusable_database", err)
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	err = s.createScheduleJob(schedule)
	switch {
	case errors.Equal(err, executor.ErrUnsupportedScheduleType):
		s.reportEvent("cdm-cloud.scheduler.create_schedule_handler.failure-create_schedule_job", "unsupported_schedule_type", err)
		return err

	case errors.Equal(err, executor.ErrUnsupportedTimezone):
		s.reportEvent("cdm-cloud.scheduler.create_schedule_handler.failure-create_schedule_job", "unsupported_timezone", err)
		return err

	case errors.Equal(err, errors.ErrUnknown):
		s.reportEvent("cdm-cloud.scheduler.create_schedule_handler.failure-create_schedule_job", "unknown", err)
		return err
	}

	logger.Infof("[createScheduleHandler] ScheduleJob(%d:%s-%s) is created.", schedule.ID, schedule.Topic, schedule.Type)
	return nil
}

func (s *Scheduler) updateScheduleHandler(e broker.Event) error {
	var id uint64
	var err error

	defer func() {
		if err != nil {
			logger.Errorf("[updateScheduleHandler] Could not update schedule job. cause: %+v", err)
		}
	}()

	id, err = parseScheduleID(e)
	if err != nil {
		err = errors.Unknown(err)
		s.reportEvent("cdm-cloud.scheduler.update_schedule_handler.failure-parse_schedule_id", "unknown", err)
		return err
	}

	schedule, err := findSchedule(id)
	if err != nil {
		err = errors.UnusableDatabase(err)
		s.reportEvent("cdm-cloud.scheduler.update_schedule_handler.failure-find_schedule", "unusable_database", err)
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	// delete schedule job
	s.deleteScheduleJob(schedule)

	// create schedule job
	err = s.createScheduleJob(schedule)
	switch {
	case errors.Equal(err, executor.ErrUnsupportedScheduleType):
		s.reportEvent("cdm-cloud.scheduler.update_schedule_handler.failure-create_schedule_job", "unsupported_schedule_type", err)
		return err

	case errors.Equal(err, executor.ErrUnsupportedTimezone):
		s.reportEvent("cdm-cloud.scheduler.update_schedule_handler.failure-create_schedule_job", "unsupported_timezone", err)
		return err

	case errors.Equal(err, errors.ErrUnknown):
		s.reportEvent("cdm-cloud.scheduler.update_schedule_handler.failure-create_schedule_job", "unknown", err)
		return err
	}

	logger.Infof("[updateScheduleHandler] ScheduleJob(%d:%s-%s) is updated.", schedule.ID, schedule.Topic, schedule.Type)
	return nil
}

func (s *Scheduler) deleteScheduleHandler(e broker.Event) error {
	id, err := parseScheduleID(e)
	if err != nil {
		err = errors.Unknown(err)
		logger.Errorf("[deleteScheduleHandler] Could not delete schedule job. cause: %+v", err)
		s.reportEvent("cdm-cloud.scheduler.delete_schedule_handler.failure-parse_schedule_id", "unknown", err)
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.deleteScheduleJob(&model.Schedule{ID: id})

	logger.Infof("[deleteScheduleHandler] ScheduleJob(%d) is deleted", id)
	return nil
}

func (s *Scheduler) getExecutorMap(schedule *model.Schedule) (executor.Executor, error) {
	var (
		ok bool
		e  executor.Executor
	)

	if _, err := time.LoadLocation(schedule.Timezone); err != nil || schedule.Timezone == "" {
		return nil, executor.UnsupportedTimezone(schedule.Timezone)
	}

	switch schedule.Type {
	case types.ScheduleTypeDayOfMonthly, types.ScheduleTypeWeekOfMonthly:
		if e, ok = s.cronExecutorMap[schedule.Timezone]; !ok {
			e = executor.NewCronExecutor(schedule.Timezone)
			s.cronExecutorMap[schedule.Timezone] = e
		}

	case types.ScheduleTypeSpecified, types.ScheduleTypeMinutely, types.ScheduleTypeHourly, types.ScheduleTypeDaily, types.ScheduleTypeWeekly:
		if e, ok = s.simpleExecutorMap[schedule.Timezone]; !ok {
			e = executor.NewSimpleExecutor(schedule.Timezone)
			s.simpleExecutorMap[schedule.Timezone] = e
		}

	default:
		return nil, executor.UnsupportedScheduleType(schedule.Type)
	}
	return e, nil
}

// CalculateScheduleJobNextRunTime 스케줄 실행 시간을 반환 하는 함수
func (s *Scheduler) CalculateScheduleJobNextRunTime(schedule *model.Schedule) (*int64, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	e, err := s.getExecutorMap(schedule)
	if err != nil {
		return nil, err
	}
	return e.CalculateJobNextRunTime(schedule)
}

func (s *Scheduler) createScheduleJob(schedule *model.Schedule) error {
	e, err := s.getExecutorMap(schedule)
	if err != nil {
		return err
	}
	return e.CreateJob(schedule)
}

func (s *Scheduler) deleteScheduleJob(schedule *model.Schedule) {
	for _, e := range s.cronExecutorMap {
		e.DeleteJob(schedule)
	}

	for _, e := range s.simpleExecutorMap {
		e.DeleteJob(schedule)
	}
}

// Start 스케줄러 시작 함수
// 시작 시 database 에 저장된 스케줄 정보로 스케줄을 등록 하며
// SchdulerHandler 에서 publish 한 메세지를 처리 하기 위한 subscriber를 등록 한다.
func (s *Scheduler) Start() error {
	var (
		err       error
		schedules []*model.Schedule
	)

	err = database.Transaction(func(db *gorm.DB) error {
		return db.Find(&schedules).Error
	})
	if err != nil {
		err = errors.UnusableDatabase(err)
		logger.Errorf("[Start] Could not load schedule. cause: %+v", err)
		return err
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	for _, schedule := range schedules {
		if err := s.createScheduleJob(schedule); err != nil {
			logger.Errorf("[Start] Error occurred in schedule create (%v)", err)
			return err
		}
	}

	s.subCreate, err = broker.SubscribeTempQueue(constant.TopicNoticeCreateSchedule, s.createScheduleHandler)
	if err != nil {
		err = errors.UnusableBroker(err)
		logger.Errorf("[Start] Could not subscribe topic(%s). cause: %+v", constant.TopicNoticeCreateSchedule, err)
		return err
	}

	s.subUpdate, err = broker.SubscribeTempQueue(constant.TopicNoticeUpdateSchedule, s.updateScheduleHandler)
	if err != nil {
		err = errors.UnusableBroker(err)
		logger.Errorf("[Start] Could not subscribe topic(%s). cause: %+v", constant.TopicNoticeUpdateSchedule, err)
		return err
	}

	s.subDelete, err = broker.SubscribeTempQueue(constant.TopicNoticeDeleteSchedule, s.deleteScheduleHandler)
	if err != nil {
		err = errors.UnusableBroker(err)
		logger.Errorf("[Start] Could not subscribe topic(%s). cause: %+v", constant.TopicNoticeDeleteSchedule, err)
		return err
	}
	return nil
}

// Close 스케줄러 종료 함수
// Suscriber 취소 및 현재 등록된 스케줄링을 종료 한다.
func (s *Scheduler) Close() error {
	if s.subCreate != nil {
		if err := s.subCreate.Unsubscribe(); err != nil {
			err = errors.UnusableBroker(err)
			logger.Errorf("[Start] Could not unsubscribe topic(%s). cause: %+v", s.subCreate.Topic(), err)
			return err
		}
	}
	s.subCreate = nil

	if s.subUpdate != nil {
		if err := s.subUpdate.Unsubscribe(); err != nil {
			err = errors.UnusableBroker(err)
			logger.Errorf("[Start] Could not unsubscribe topic(%s). cause: %+v", s.subUpdate.Topic(), err)
			return err
		}
	}
	s.subUpdate = nil

	if s.subDelete != nil {
		if err := s.subDelete.Unsubscribe(); err != nil {
			err = errors.UnusableBroker(err)
			logger.Errorf("[Start] Could not unsubscribe topic(%s). cause: %+v", s.subDelete.Topic(), err)
			return err
		}
	}
	s.subDelete = nil

	for _, e := range s.cronExecutorMap {
		e.Close()
	}

	for _, e := range s.simpleExecutorMap {
		e.Close()
	}
	return nil
}

// NewScheduler Scheduler 구조체 생성
func NewScheduler(tid uint64) *Scheduler {
	return &Scheduler{
		lock:              &sync.Mutex{},
		defaultTenantID:   tid,
		simpleExecutorMap: make(map[string]executor.Executor),
		cronExecutorMap:   make(map[string]executor.Executor),
	}
}
