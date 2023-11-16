package handler

import (
	"context"
	"database/sql"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/datacommand2/cdm-cloud/common/metadata"
	"github.com/datacommand2/cdm-cloud/common/util"
	identity "github.com/datacommand2/cdm-cloud/services/identity/proto"
	"github.com/datacommand2/cdm-cloud/services/notification/config"
	"github.com/datacommand2/cdm-cloud/services/notification/event"
	"github.com/datacommand2/cdm-cloud/services/notification/notifier"
	notification "github.com/datacommand2/cdm-cloud/services/notification/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
	"strconv"
	"time"
)

// createError 각 서비스의 internal error 를 처리
func createError(ctx context.Context, eventCode string, err error) error {
	if err == nil {
		return nil
	}

	var errorCode string
	switch {
	// handler
	case errors.Equal(err, ErrNoSolutionRole):
		errorCode = "unknown_user"
		return errors.StatusUnauthorized(ctx, eventCode, errorCode, err)

	// config
	case errors.Equal(err, config.ErrNotFoundTenant):
		errorCode = "not_found_tenant"
		return errors.StatusInternalServerError(ctx, eventCode, errorCode, err)

	case errors.Equal(err, config.ErrNotFoundUser):
		errorCode = "not_found_user"
		return errors.StatusInternalServerError(ctx, eventCode, errorCode, err)

	// event
	case errors.Equal(err, event.ErrNotFoundEvent):
		errorCode = "not_found_event"
		return errors.StatusNotFound(ctx, eventCode, errorCode, err)

	// notifier
	case errors.Equal(err, notifier.ErrNotFoundUser):
		errorCode = "not_found_user"
		return errors.StatusInternalServerError(ctx, eventCode, errorCode, err)

	default:
		if err := util.CreateError(ctx, eventCode, err); err != nil {
			return err
		}
	}

	return nil
}

func eventRecordToResponse(in *event.Record) *notification.Event {
	var eventError string
	if in.Event.ErrorCode != nil {
		eventError = *in.Event.ErrorCode
	}

	return &notification.Event{
		Id: in.Event.ID,
		Tenant: &notification.Tenant{
			Id:   in.Tenant.ID,
			Name: in.Tenant.Name,
		},
		Code:       in.Event.Code,
		EventError: eventError,
		Contents:   in.Event.Contents,
		CreatedAt:  in.Event.CreatedAt,
	}
}

func eventRecordsToResponse(in []event.Record) []*notification.Event {
	var ret []*notification.Event

	for _, item := range in {
		ret = append(ret, eventRecordToResponse(&item))
	}

	return ret
}

func checkAuthorization(user *identity.User) error {
	for _, role := range user.Roles {
		if role.Role == constant.Manager || role.Role == constant.Admin {
			return nil
		}

		if role.Solution == constant.SolutionName {
			return nil
		}
	}

	return NoSolutionRole(constant.SolutionName, user.Id)
}

func getValidDate(startDate, endDate int64) (int64, int64, error) {
	var (
		from int64
		to   int64
		err  error
		s    time.Time
		e    time.Time
	)

	// TODO: config 에서 지역(location)을 읽어 처리하도록 수정 필요
	loc, _ := time.LoadLocation("Asia/Seoul")
	if startDate != 0 {
		s, err = time.ParseInLocation("200601021504", strconv.FormatInt(startDate, 10), loc)
		if err != nil {
			return 0, 0, errors.InvalidParameterValue("start_date", startDate, "Wrong format. must be YYYYMMDDHHmm format")
		}

		from = s.Unix()
	}

	if endDate != 0 {
		e, err = time.ParseInLocation("200601021504", strconv.FormatInt(endDate, 10), loc)
		if err != nil {
			return 0, 0, errors.InvalidParameterValue("end_date", endDate, "wrong format. must be YYYYMMDDHHmm format")
		}

		if startDate > endDate {
			return 0, 0, errors.InvalidParameterValue("start_date", startDate, "Start date is later than end date.")
		}
		to = e.Unix()
	}

	return from, to, nil
}

func pagination(itemNum, limit, offset uint64) *notification.Pagination {
	if limit == 0 {
		return &notification.Pagination{
			Page:       1,
			TotalPage:  1,
			TotalItems: itemNum,
		}
	}

	return &notification.Pagination{
		Page:       offset/limit + 1,
		TotalPage:  (itemNum + limit - 1) / limit,
		TotalItems: itemNum,
	}
}

func newBool(b sql.NullBool) *wrappers.BoolValue {
	if b.Valid {
		return &wrappers.BoolValue{Value: b.Bool}
	}

	return nil
}

func getRequestMetadata(ctx context.Context) (uint64, *identity.User, error) {
	id, err := metadata.GetTenantID(ctx)
	if err != nil {
		return 0, nil, errors.InvalidRequest(ctx)
	}

	user, err := metadata.GetAuthenticatedUser(ctx)
	if err != nil {
		return 0, nil, errors.InvalidRequest(ctx)
	}

	return id, user, nil
}

func validateLevel(level string) error {
	list := []interface{}{"trace", "info", "warn", "error", "fatal"}

	for _, l := range list {
		if l == level {
			return nil
		}
	}

	return errors.UnavailableParameterValue("level", level, list)
}

func validateInterval(interval uint64) error {
	if interval < 2 || interval > 30 {
		return errors.OutOfRangeParameterValue("interval", interval, 2, 30)
	}

	return nil
}

func validateEventCode(code string) error {
	if code == "" {
		return errors.RequiredParameter("code")
	}

	return nil
}

func validateGetEventsParams(in *notification.GetEventsRequest) error {
	if in.GetLimit() > 0 && in.GetLimit() < 10 {
		logger.Warnf("The limit must be more than 10.")
	}

	if in.GetLimit() > 100 {
		logger.Warnf("The limit must be less than 100.")
	}

	if in.GetSolution() != "" && len(in.GetSolution()) > 100 {
		return errors.LengthOverflowParameterValue("solution", in.GetSolution(), 100)
	}

	if in.GetClass_1() != "" && len(in.GetClass_1()) > 100 {
		return errors.LengthOverflowParameterValue("class_1", in.GetClass_1(), 100)
	}

	if in.GetClass_2() != "" && len(in.GetClass_2()) > 100 {
		return errors.LengthOverflowParameterValue("class_2", in.GetClass_2(), 100)
	}

	if in.GetClass_3() != "" && len(in.GetClass_3()) > 100 {
		return errors.LengthOverflowParameterValue("class_3", in.GetClass_3(), 100)
	}

	if in.GetLevel() != "" {
		if err := validateLevel(in.GetLevel()); err != nil {
			return err
		}
	}

	return nil
}

func validateGetEventsStreamParams(in *notification.GetEventsStreamRequest) error {
	if in.GetSolution() != "" && len(in.GetSolution()) > 100 {
		return errors.LengthOverflowParameterValue("solution", in.GetSolution(), 100)
	}

	if in.GetClass_1() != "" && len(in.GetClass_1()) > 100 {
		return errors.LengthOverflowParameterValue("class_1", in.GetClass_1(), 100)
	}

	if in.GetClass_2() != "" && len(in.GetClass_2()) > 100 {
		return errors.LengthOverflowParameterValue("class_2", in.GetClass_2(), 100)
	}

	if in.GetClass_3() != "" && len(in.GetClass_3()) > 100 {
		return errors.LengthOverflowParameterValue("class_3", in.GetClass_3(), 100)
	}

	if in.GetLevel() != "" {
		if err := validateLevel(in.GetLevel()); err != nil {
			return err
		}
	}

	return nil
}

func validateEventReceives(in *notification.EventReceivesRequest) error {
	if in.GetEventNotifications() == nil {
		return errors.RequiredParameter("event_notifications")
	}

	for _, receive := range in.GetEventNotifications() {
		if err := validateEventCode(receive.Code); err != nil {
			return err
		}
	}

	return nil
}

func validateGetEventReceivesRequest(in *notification.GetEventReceivesRequest) error {
	if in.GetSolution() != "" && len(in.GetSolution()) > 100 {
		return errors.LengthOverflowParameterValue("solution", in.GetSolution(), 100)
	}

	if in.GetClass_1() != "" && len(in.GetClass_1()) > 100 {
		return errors.LengthOverflowParameterValue("class_1", in.GetClass_1(), 100)
	}

	if in.GetClass_2() != "" && len(in.GetClass_2()) > 100 {
		return errors.LengthOverflowParameterValue("class_2", in.GetClass_2(), 100)
	}

	if in.GetClass_3() != "" && len(in.GetClass_3()) > 100 {
		return errors.LengthOverflowParameterValue("class_3", in.GetClass_3(), 100)
	}

	if in.GetLevel() != "" {
		if err := validateLevel(in.GetLevel()); err != nil {
			return err
		}
	}

	return nil
}
