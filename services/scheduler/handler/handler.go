package handler

import (
	"context"
	"github.com/datacommand2/cdm-cloud/common/broker"
	"github.com/datacommand2/cdm-cloud/common/database"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/datacommand2/cdm-cloud/services/scheduler/internal/scheduler"
	proto "github.com/datacommand2/cdm-cloud/services/scheduler/proto"
	"github.com/jinzhu/gorm"
	"time"
)

// SchedulerHandler 스케줄러 서비스의 rpc handler
type SchedulerHandler struct {
	scheduler *scheduler.Scheduler
	subs      []broker.Subscriber
}

// Close handler, db, 스케줄러 익스큐터 종료 함수
func (h *SchedulerHandler) Close() error {
	for _, s := range h.subs {
		if err := s.Unsubscribe(); err != nil {
			logger.Warnf("[Close] Could not unsubscribe queue (%s). Cause: %+v", s.Topic(), err)
		}
	}

	return h.scheduler.Close()
}

// GetSchedule 스케줄 조회 요청을 처리하기 위한 핸들러 함수
func (h *SchedulerHandler) GetSchedule(ctx context.Context, req *proto.ScheduleRequest, rsp *proto.ScheduleResponse) error {
	return database.Transaction(func(db *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("[handler] Could not get schedule(%d). cause: %v", req.Schedule.Id, err)
			}
		}()

		s, err := getSchedule(db, req.Schedule.Id)
		if err != nil {
			return createError(ctx, "cdm-cloud.scheduler.get_schedule.failure-get_schedule", err)
		}

		rsp.Schedule = new(proto.Schedule)
		if err = rsp.Schedule.SetFromModel(s); err != nil {
			return createError(ctx, "cdm-cloud.scheduler.get_schedule.failure-set_from_model", err)
		}

		t, err := h.scheduler.CalculateScheduleJobNextRunTime(s)
		if err != nil {
			return createError(ctx, "cdm-cloud.scheduler.get_schedule.failure-calculate_schedule_job_next_runtime", err)
		}

		rsp.NextRuntime = *t
		return nil
	})
}

// CreateSchedule handler 에서 스케줄 저장 요청을 처리하기 위한 함수
// 유효성 확인 후 database 에 해당 스케줄을 저장 하며, 해당 스케줄을 스케줄러 익스큐터에 등록을 위해
// 메세지를 broker 로 publish 를 한다.
func (h *SchedulerHandler) CreateSchedule(ctx context.Context, req *proto.ScheduleRequest, rsp *proto.ScheduleResponse) error {
	return database.Transaction(func(db *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("[handler] Could not create schedule(%+v). cause: %v", req.Schedule, err)
			}
		}()

		s, err := req.Schedule.Model()
		if err != nil {
			return createError(ctx, "cdm-cloud.scheduler.create_schedule.failure-model", err)
		}

		if err = validateScheduleBrokerMessage(s); err != nil {
			return createError(ctx, "cdm-cloud.scheduler.create_schedule.failure-validate_schedule_broker_message", err)
		}

		if err = validateSchedule(s); err != nil {
			return createError(ctx, "cdm-cloud.scheduler.create_schedule.failure-validate_schedule", err)
		}

		if err = createSchedule(db, s); err != nil {
			return createError(ctx, "cdm-cloud.scheduler.create_schedule.failure-create_schedule", err)
		}

		rsp.Schedule = new(proto.Schedule)
		if err = rsp.Schedule.SetFromModel(s); err != nil {
			return createError(ctx, "cdm-cloud.scheduler.create_schedule.failure-set_from_model", err)
		}

		t, err := h.scheduler.CalculateScheduleJobNextRunTime(s)
		if err != nil {
			return createError(ctx, "cdm-cloud.scheduler.create_schedule.failure-calculate_schedule_job_next_runtime", err)
		}

		rsp.NextRuntime = *t
		logger.Infof("[handler] Create schedule success. id: %d, topic: %s, message: %s", s.ID, s.Topic, s.Message)
		return nil
	})
}

// UpdateSchedule handler 에서 스케줄 수정 요청을 처리하기 위한 함수
// 유효성 확인 후 database 에 해당 스케줄을 저장 하며, 해당 스케줄을 스케줄러 익스큐터에 등록 및 삭제를 위해
// 메세지를 broker 로 publish 를 한다.
func (h *SchedulerHandler) UpdateSchedule(ctx context.Context, req *proto.ScheduleRequest, rsp *proto.ScheduleResponse) error {
	return database.Transaction(func(db *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("[handler] Could not update schedule(%+v). cause: %v", req.Schedule, err)
			}
		}()

		s, err := req.Schedule.Model()
		if err != nil {
			return createError(ctx, "cdm-cloud.scheduler.update_schedule.failure-model", err)
		}

		if err = validateScheduleBrokerMessage(s); err != nil {
			return createError(ctx, "cdm-cloud.scheduler.update_schedule.failure-validate_schedule_broker_message", err)
		}

		if err = validateSchedule(s); err != nil {
			return createError(ctx, "cdm-cloud.scheduler.update_schedule.failure-validate_schedule", err)
		}

		if err = updateSchedule(db, s); err != nil {
			return createError(ctx, "cdm-cloud.scheduler.update_schedule.failure-update_schedule", err)
		}

		rsp.Schedule = new(proto.Schedule)
		if err = rsp.Schedule.SetFromModel(s); err != nil {
			return createError(ctx, "cdm-cloud.scheduler.update_schedule.failure-set_from_model", err)
		}

		t, err := h.scheduler.CalculateScheduleJobNextRunTime(s)
		if err != nil {
			return createError(ctx, "cdm-cloud.scheduler.update_schedule.failure-calculate_schedule_job_next_runtime", err)
		}

		rsp.NextRuntime = *t
		logger.Infof("[handler] Update schedule success. id: %d, topic: %s, message: %s", s.ID, s.Topic, s.Message)
		return nil
	})
}

// DeleteSchedule handler 에서 스케줄 삭제 요청을 처리하기 위한 함수
// 해당 스케줄을 database 에서 삭제 하며, 해당 스케줄을 스케줄러 익스큐터에서 삭제를 위해
// 메세지를 broker 로 publish 를 한다.
func (h *SchedulerHandler) DeleteSchedule(ctx context.Context, req *proto.ScheduleRequest, _ *proto.Empty) error {
	return database.Transaction(func(db *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("[handler] Could not delete schedule(%+v). cause: %v", req.Schedule, err)
			}
		}()

		s, err := req.Schedule.Model()
		if err != nil {
			return createError(ctx, "cdm-cloud.scheduler.delete_schedule.failure-model", err)
		}

		if err = deleteSchedule(db, s); err != nil {
			return createError(ctx, "cdm-cloud.scheduler.delete_schedule.failure-delete_schedule", err)
		}

		logger.Infof("[handler] Delete schedule success. id: %d", s.ID)
		return nil
	})
}

// CalculateNextRuntime handler 에서 특정 스케줄에 다음 실행 시간을 처리 하기 위한 함수
func (h *SchedulerHandler) CalculateNextRuntime(ctx context.Context, req *proto.ScheduleRequest, rsp *proto.ScheduleNextRuntimeResponse) error {
	var err error
	defer func() {
		if err != nil {
			logger.Errorf("[handler] Could not calculate schedule(%+v) next runtime. cause: %v", req.Schedule, err)
		}
	}()

	s, err := req.Schedule.Model()
	if err != nil {
		return createError(ctx, "cdm-cloud.scheduler.calculate_next_runtime.failure-model", err)
	}

	if err = validateSchedule(s); err != nil {
		return createError(ctx, "cdm-cloud.scheduler.calculate_next_runtime.failure-validate_schedule", err)
	}

	t, err := h.scheduler.CalculateScheduleJobNextRunTime(s)
	if err != nil {
		return createError(ctx, "cdm-cloud.scheduler.calculate_next_runtime.failure-calculate_schedule_job_next_runtime", err)
	}

	rsp.NextRuntime = *t
	return nil
}

// NewSchedulerHandler SchedulerHandler 생성 함수
func NewSchedulerHandler(tid uint64) (*SchedulerHandler, error) {
	s := scheduler.NewScheduler(tid)
	if err := s.Start(); err != nil {
		return nil, err
	}
	//스케줄러 subscribe 를 위한 대기 시간
	time.Sleep(2 * time.Second)

	h := &SchedulerHandler{scheduler: s}

	h.RunDeleteLogFilesSchedule()

	if err := h.InitDeleteLogFilesSchedule(); err != nil {
		logger.Errorf("[handler] Could not initialize delete log files schedule. Cause: %v", err)
	}

	return h, nil
}
