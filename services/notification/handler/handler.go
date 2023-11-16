package handler

import (
	"context"
	"fmt"
	"github.com/datacommand2/cdm-cloud/common/broker"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/database"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/logger"
	identity "github.com/datacommand2/cdm-cloud/services/identity/proto"
	"github.com/datacommand2/cdm-cloud/services/notification/config"
	"github.com/datacommand2/cdm-cloud/services/notification/event"
	notification "github.com/datacommand2/cdm-cloud/services/notification/proto"
	"github.com/jinzhu/gorm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NotificationHandler Notification 기능을 이용하기 위한 handler 이다.
type NotificationHandler struct{}

// NewNotificationHandler 핸들러를 반환한다.
func NewNotificationHandler() notification.NotificationHandler {
	return &NotificationHandler{}
}

// GetConfig 테넌트의 이벤트 설정을 조회한다.
func (h *NotificationHandler) GetConfig(ctx context.Context, _ *notification.Empty, rsp *notification.GetConfigResponse) error {
	var (
		headerTenant uint64
		err          error
	)

	// 헤더 정보 취득
	headerTenant, _, err = getRequestMetadata(ctx)
	if err != nil {
		logger.Errorf("Could not get config. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_config.failure-get_request_metadata", err)
	}

	err = database.Transaction(func(db *gorm.DB) error {
		rsp.EventConfig, err = config.GetConfig(db, headerTenant)
		return err
	})
	if err != nil {
		logger.Errorf("Could not get config. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_config.failure-get", err)
	}

	rsp.Message = &notification.Message{Code: "cdm-cloud.notification.get_config.success"}
	return errors.StatusOK(ctx, "cdm-cloud.notification.get_config.success", nil)
}

// SetConfig 테넌트의 이벤트 설정을 변경한다.
func (h *NotificationHandler) SetConfig(ctx context.Context, req *notification.SetConfigRequest, rsp *notification.GetConfigResponse) error {
	var (
		headerTenant uint64
		err          error
	)

	// 헤더 정보 취득
	headerTenant, _, err = getRequestMetadata(ctx)
	if err != nil {
		logger.Errorf("Could not set config. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.set_config.failure-get_request_metadata", err)
	}

	err = config.ValidateConfig(req.GetEventConfig())
	if err != nil {
		logger.Errorf("Could not set config. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.set_config.failure-validate", err)
	}

	err = database.Transaction(func(db *gorm.DB) error {
		var err error
		if err = config.ValidateTenant(db, headerTenant); err != nil {
			return err
		}

		if err = config.SetConfig(db, headerTenant, req); err != nil {
			return err
		}

		rsp.EventConfig, err = config.GetConfig(db, headerTenant)

		return err
	})
	if err != nil {
		logger.Errorf("Could not set config. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.set_config.failure-set", err)
	}

	rsp.Message = &notification.Message{Code: "cdm-cloud.notification.set_config.success"}
	return errors.StatusOK(ctx, "cdm-cloud.notification.set_config.success", nil)
}

// GetEvent 테넌트의 이벤트 상세내용을 조회한다.
func (h *NotificationHandler) GetEvent(ctx context.Context, in *notification.GetEventRequest, out *notification.GetEventResponse) error {
	var (
		eventRecord  *event.Record
		headerUser   *identity.User
		headerTenant uint64
		err          error
	)

	headerTenant, headerUser, err = getRequestMetadata(ctx)
	if err != nil {
		logger.Errorf("Could not get event. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_event.failure-get_request_metadata", err)
	}

	if err = checkAuthorization(headerUser); err != nil {
		logger.Errorf("Could not get event. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_event.failure-check_authorization", err)
	}

	err = database.Transaction(func(db *gorm.DB) error {
		eventRecord, err = event.GetEvent(db, in.EventId, headerTenant)
		return err
	})
	if err != nil {
		logger.Errorf("Could not get event. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_event.failure-get", err)
	}

	out.Event = eventRecordToResponse(eventRecord)
	out.Message = &notification.Message{Code: "cdm-cloud.notification.get_event.success"}
	return errors.StatusOK(ctx, "cdm-cloud.notification.get_event.success", nil)
}

// GetEvents 테넌트의 이벤트 목록을 조회한다.
func (h *NotificationHandler) GetEvents(ctx context.Context, in *notification.GetEventsRequest, out *notification.GetEventsResponse) error {
	// 헤더 정보 취득
	headerTenant, headerUser, err := getRequestMetadata(ctx)
	if err != nil {
		logger.Errorf("Could not get events. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_events.failure-get_request_metadata", err)
	}

	if err = checkAuthorization(headerUser); err != nil {
		logger.Errorf("Could not get event. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_event.failure-check_authorization", err)
	}

	err = validateGetEventsParams(in)
	if err != nil {
		logger.Errorf("Could not get events. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_events.failure-validate", err)
	}

	startDate, endDate, err := getValidDate(in.GetStartDate(), in.GetEndDate())
	if err != nil {
		logger.Errorf("Could not get events. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_events.failure-validate", err)
	}
	logger.Infof("Received NotificationHandler.GetEvents request(start : %v, end : %v)", startDate, endDate)
	query := event.NewEventsQuery(headerTenant).
		Limit(in.GetLimit()).
		Offset(in.GetOffset()).
		Solutions(in.GetSolution()).
		From(startDate).
		To(endDate).
		Level(in.GetLevel()).
		Class1(in.GetClass_1()).
		Class2(in.GetClass_2()).
		Class3(in.GetClass_3())

	if query.Error != nil {
		err = errors.InvalidParameterValue("events", 0, query.Error.Error())
		logger.Errorf("Could not generate query. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_events.failure-new_events_query", err)
	}

	// 질의 수행
	events, count, err := event.GetEvents(query)
	if err != nil {
		logger.Errorf("Could not get events. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_events.failure-get", err)
	}

	if count == 0 {
		return createError(ctx, "cdm-cloud.notification.get_events.success-get", errors.ErrNoContent)
	}

	// 사용자 결과
	out.Events = eventRecordsToResponse(events)
	out.Pagination = pagination(count, in.GetLimit(), in.GetOffset())
	out.Message = &notification.Message{Code: "cdm-cloud.notification.get_events.success"}
	return errors.StatusOK(ctx, "cdm-cloud.notification.get_events.success", nil)
}

// GetEventsStream 신규 이벤트 실시간 조회 WebSocket
func (h *NotificationHandler) GetEventsStream(ctx context.Context, in *notification.GetEventsStreamRequest, out notification.Notification_GetEventsStreamStream) error {
	headerTenant, _, err := getRequestMetadata(ctx)
	if err != nil {
		logger.Errorf("Could not get events stream. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_events_stream.failure-get_request_metadata", err)
	}

	err = validateGetEventsStreamParams(in)
	if err != nil {
		logger.Errorf("Could not get events stream. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_events_stream.failure-validate", err)
	}

	var interval uint64
	if in.GetInterval() == nil {
		interval = 5
	} else {
		if err := validateInterval(in.GetInterval().GetValue()); err != nil {
			logger.Errorf("Could not get events stream. cause: %+v", err)
			return createError(ctx, "cdm-cloud.notification.get_events_stream.failure-validate_interval", err)
		}
		interval = in.GetInterval().GetValue()
	}

	lookup := newEventLookup(interval, headerTenant)

	defer lookup.close()

	topic := fmt.Sprintf(constant.TopicNotificationEventCreated, headerTenant)
	s, err := broker.SubscribeTempQueue(topic, lookup.subscribeEvent)
	if err != nil {
		err = errors.UnusableBroker(err)
		logger.Errorf("Could not get events stream. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_events_stream.failure-subscribe_template_queue", err)
	}

	defer func() {
		if err := s.Unsubscribe(); err != nil {
			logger.Warnf("Could not unsubscribe %v. cause: %+v", topic, errors.UnusableBroker(err))
		}
	}()

	var events []event.Record
	for {
		select {
		case ev := <-lookup.C:
			matched := in.GetSolution() == "" || in.GetSolution() == ev.EventCode.Solution
			matched = matched && (in.GetClass_1() == "" || in.GetClass_1() == ev.EventCode.Class1)
			matched = matched && (in.GetClass_2() == "" || in.GetClass_2() == ev.EventCode.Class2)
			matched = matched && (in.GetClass_3() == "" || in.GetClass_3() == ev.EventCode.Class3)
			matched = matched && (in.GetLevel() == "" || in.GetLevel() == ev.EventCode.Level)

			if matched {
				events = append(events, *ev)
			}
		case <-lookup.ticker.C:
			if len(events) == 0 {
				break
			}
			rsp := &notification.GetEventsStreamResponse{Events: eventRecordsToResponse(events)}
			events = nil
			err := out.Send(rsp)
			st, _ := status.FromError(err)
			if st != nil {
				if st.Code() == codes.Unavailable {
					return nil
				}
				err = errors.Unknown(err)
				logger.Errorf("Could not send events. cause: %+v", err)
				return createError(ctx, "cdm-cloud.notification.get_events_stream.failure-send_response", err)
			}
		}
	}
}

// GetEventClassifications 이벤트 코드 분류 목록을 조회한다.
func (h *NotificationHandler) GetEventClassifications(ctx context.Context, _ *notification.Empty, rsp *notification.GetEventClassificationsResponse) error {
	var err error
	rsp.EventClassifications, err = event.GetEventClassifications()
	if err != nil {
		logger.Errorf("Could not get event classifications. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_event_classifications.failure-get", err)
	}

	rsp.Message = &notification.Message{Code: "cdm-cloud.notification.get_event_classifications.success"}
	return errors.StatusOK(ctx, "cdm-cloud.notification.get_event_classifications.success", nil)
}

// GetTenantEventReceives 모든 이벤트에 대한 특정 테넌트의 수신 여부를 조회한다.
func (h *NotificationHandler) GetTenantEventReceives(ctx context.Context, req *notification.GetEventReceivesRequest, rsp *notification.EventReceivesResponse) error {
	err := validateGetEventReceivesRequest(req)
	if err != nil {
		logger.Errorf("Could not get tenant event receives. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_tenant_event_receives.failure-validate", err)
	}

	var (
		headerTenant uint64
	)
	// 헤더 정보 취득
	headerTenant, _, err = getRequestMetadata(ctx)
	if err != nil {
		logger.Errorf("Could not get tenant event receives. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_tenant_event_receives.failure-get_request_metadata", err)
	}

	err = database.Transaction(func(db *gorm.DB) error {
		var err error
		rsp.EventNotifications, err = config.GetTenantEventReceives(db, headerTenant, req)

		return err
	})
	if err != nil {
		logger.Errorf("Could not get tenant event receives. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_tenant_event_receives.failure-get", err)
	}

	rsp.Message = &notification.Message{Code: "cdm-cloud.notification.get_tenant_event_receives.success"}
	return errors.StatusOK(ctx, "cdm-cloud.notification.get_tenant_event_receives.success", nil)
}

// SetTenantEventReceives 특정 테넌트의 이벤트 수신 여부를 변경한다.
func (h *NotificationHandler) SetTenantEventReceives(ctx context.Context, req *notification.EventReceivesRequest, rsp *notification.MessageResponse) error {
	var (
		headerTenant uint64
		err          error
	)

	// 헤더 정보 취득
	headerTenant, _, err = getRequestMetadata(ctx)
	if err != nil {
		logger.Errorf("Could not set tenant event receives. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.set_tenant_event_receives.failure-get_request_metadata", err)
	}

	err = validateEventReceives(req)
	if err != nil {
		logger.Errorf("Could not set tenant event receives. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.set_tenant_event_receives.failure-validate", err)
	}

	err = database.Transaction(func(db *gorm.DB) error {
		if err := config.ValidateTenant(db, headerTenant); err != nil {
			return err
		}

		if err := config.SetTenantEventReceives(db, headerTenant, req); err != nil {
			return err
		}

		return err
	})
	if err != nil {
		logger.Errorf("Could not set tenant event receives. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.set_tenant_event_receives.failure-set", err)
	}

	rsp.Message = &notification.Message{Code: "cdm-cloud.notification.set_tenant_event_receives.success"}
	return errors.StatusOK(ctx, "cdm-cloud.notification.set_tenant_event_receives.success", nil)
}

// GetUserEventReceives 사용자의 모든 이벤트 수신 여부를 조회한다.
func (h *NotificationHandler) GetUserEventReceives(ctx context.Context, req *notification.GetEventReceivesRequest, rsp *notification.EventReceivesResponse) error {
	err := validateGetEventReceivesRequest(req)
	if err != nil {
		logger.Errorf("Could not get user event receives. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_user_event_receives.failure-validate", err)
	}

	var (
		headerUser *identity.User
	)
	// 헤더 정보 취득
	_, headerUser, err = getRequestMetadata(ctx)
	if err != nil {
		logger.Errorf("Could not get user event receives. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_user_event_receives.failure-get_request_metadata", err)
	}

	err = database.Transaction(func(db *gorm.DB) error {
		var err error
		rsp.EventNotifications, err = config.GetUserEventReceives(db, headerUser.Id, req)
		return err
	})
	if err != nil {
		logger.Errorf("Could not get user event receives. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.get_user_event_receives.failure-get", err)
	}

	if len(rsp.EventNotifications) == 0 {
		return createError(ctx, "cdm-cloud.notification.get_user_event_receives.success-get", errors.ErrNoContent)
	}

	rsp.Message = &notification.Message{Code: "cdm-cloud.notification.get_user_event_receives.success"}
	return errors.StatusOK(ctx, "cdm-cloud.notification.get_user_event_receives.success", nil)
}

// SetUserEventReceives 사용자의 이벤트 수신 여부를 변경한다.
func (h *NotificationHandler) SetUserEventReceives(ctx context.Context, req *notification.EventReceivesRequest, rsp *notification.MessageResponse) error {
	var (
		headerUser *identity.User
		err        error
	)
	// 헤더 정보 취득
	_, headerUser, err = getRequestMetadata(ctx)
	if err != nil {
		logger.Errorf("Could not set user event receives. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.set_user_event_receives.failure-get_request_metadata", err)
	}

	err = validateEventReceives(req)
	if err != nil {
		logger.Errorf("Could not set user event receives. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.set_user_event_receives.failure-validate", err)
	}

	err = database.Transaction(func(db *gorm.DB) error {
		if err := config.ValidateUser(db, headerUser.Id); err != nil {
			return err
		}

		if err := config.SetUserEventReceives(db, headerUser.Id, req); err != nil {
			return err
		}

		return err
	})
	if err != nil {
		logger.Errorf("Could not set user event receives. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.set_user_event_receives.failure-set", err)
	}

	rsp.Message = &notification.Message{Code: "cdm-cloud.notification.set_user_event_receives.success"}
	return errors.StatusOK(ctx, "cdm-cloud.notification.set_user_event_receives.success", nil)
}

// ResetUserEventReceives 사용자의 이벤트 수신 여부를 초기화한다.
func (h *NotificationHandler) ResetUserEventReceives(ctx context.Context, _ *notification.Empty, rsp *notification.MessageResponse) error {
	_, headerUser, err := getRequestMetadata(ctx)
	if err != nil {
		logger.Errorf("Could not reset user event receives. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.reset_user_event_receives.failure-get_request_metadata", err)
	}

	err = database.Transaction(func(db *gorm.DB) error {
		if err := config.ValidateUser(db, headerUser.Id); err != nil {
			return err
		}

		if err := config.ResetUserEventReceives(db, headerUser.Id); err != nil {
			return err
		}

		return err
	})
	if err != nil {
		logger.Errorf("Could not reset user event receives. cause: %+v", err)
		return createError(ctx, "cdm-cloud.notification.reset_user_event_receives.failure-reset", err)
	}

	rsp.Message = &notification.Message{Code: "cdm-cloud.notification.reset_user_event_receives.success"}
	return errors.StatusOK(ctx, "cdm-cloud.notification.reset_user_event_receives.success", nil)
}
