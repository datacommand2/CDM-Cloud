package handler

import (
	"context"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/util"
	"github.com/datacommand2/cdm-cloud/services/scheduler/internal/scheduler/executor"
)

// createError 각 서비스의 internal error 를 처리
func createError(ctx context.Context, eventCode string, err error) error {
	if err == nil {
		return nil
	}

	var errorCode string
	switch {
	// handler
	case errors.Equal(err, ErrNotFoundSchedule):
		errorCode = "not_found_schedule"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	case errors.Equal(err, ErrInvalidID):
		errorCode = "invalid_schedule_id"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	case errors.Equal(err, ErrInvalidTimezone):
		errorCode = "invalid_timezone"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	case errors.Equal(err, ErrInvalidTopic):
		errorCode = "invalid_topic"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	case errors.Equal(err, ErrInvalidMessage):
		errorCode = "invalid_message"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	case errors.Equal(err, ErrInvalidStartAt):
		errorCode = "invalid_start_at"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	case errors.Equal(err, ErrInvalidEndAt):
		errorCode = "invalid_end_at"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	// executor
	case errors.Equal(err, executor.ErrUnsupportedTimezone):
		errorCode = "unsupported_timezone"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	case errors.Equal(err, executor.ErrUnsupportedScheduleType):
		errorCode = "unsupported_schedule_type"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	default:
		if err := util.CreateError(ctx, eventCode, err); err != nil {
			return err
		}
	}

	return nil
}

// boundary 유효값 정리를 위한 구조체
type numericBoundary struct {
	Min  uint64
	Max  uint64
	Enum []uint64
}

func (b *numericBoundary) minMax(i uint64) bool {
	return !(i < b.Min || i > b.Max)
}

func (b *numericBoundary) enum(i uint64) bool {
	for _, v := range b.Enum {
		if v == i {
			return true
		}
	}
	return false
}

type stringBoundary struct {
	Min  string
	Max  string
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

var (
	hourBoundary           = numericBoundary{Min: 0, Max: 23}
	minuteBoundary         = numericBoundary{Min: 0, Max: 59}
	intervalMinuteBoundary = numericBoundary{Min: 1, Max: 59}
	intervalHourBoundary   = numericBoundary{Min: 1, Max: 23}
	intervalDayBoundary    = numericBoundary{Min: 1, Max: 30}
	intervalWeekBoundary   = numericBoundary{Min: 1, Max: 4}
	intervalMonthBoundary  = numericBoundary{Enum: []uint64{1, 2, 3, 4, 6, 12}}

	dayOfWeekBoundary = stringBoundary{Enum: []string{
		"mon", "tue", "wed", "thu", "fri", "sat", "sun",
	}}

	dayOfMonthBoundary = stringBoundary{Enum: []string{
		"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11",
		"12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22",
		"23", "24", "25", "26", "27", "28", "29", "30", "31", "L",
	}}

	weekOfMonthBoundary = stringBoundary{Enum: []string{
		"#1", "#2", "#3", "#4", "#5", "L",
	}}
)
