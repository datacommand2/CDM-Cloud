package handler

import (
	"github.com/datacommand2/cdm-cloud/common/errors"
)

var (
	// ErrNotFoundSchedule 스케줄을 데이터베이스에서 찾지 못할 경우
	ErrNotFoundSchedule = errors.New("not found schedule")
	// ErrInvalidID 스케줄 추가 및 삭제 시 schedule ID 값이 설정되지 않을 경우
	ErrInvalidID = errors.New("invalid id")
	// ErrInvalidTimezone 유효 하지 않는 타임 존 인 경우
	ErrInvalidTimezone = errors.New("invalid timezone")
	// ErrInvalidTopic Topic 이 설정 되지 않는 경우
	ErrInvalidTopic = errors.New("invalid topic")
	// ErrInvalidMessage Message 가 설정되지 않는 경우
	ErrInvalidMessage = errors.New("invalid message")
	// ErrInvalidStartAt 유효 하지 않는 스케줄 시작(유효) 시간
	ErrInvalidStartAt = errors.New("invalid start_at")
	// ErrInvalidEndAt 유효 하지 않는 종료 시간
	ErrInvalidEndAt = errors.New("invalid end_at")
)

// NotFoundSchedule 스케줄을 찾을 수 없음
func NotFoundSchedule(id uint64) error {
	return errors.Wrap(
		ErrNotFoundSchedule,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"id": id,
		}),
	)
}

// InvalidTimezone 유효 하지 않는 타임 존 인 경우
func InvalidTimezone(tz string) error {
	return errors.Wrap(
		ErrInvalidTimezone,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"timezone": tz,
		}),
	)
}

// InvalidTopic Topic 이 설정 되지 않는 경우
func InvalidTopic() error {
	return errors.Wrap(
		ErrInvalidTopic,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"length": 0,
		}),
	)
}

// InvalidMessage Message 가 설정되지 않는 경우
func InvalidMessage() error {
	return errors.Wrap(
		ErrInvalidMessage,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"length": 0,
		}),
	)
}

// InvalidStartAt 유효 하지 않는 스케줄 시작(유효) 시간
func InvalidStartAt(startAt int64) error {
	return errors.Wrap(
		ErrInvalidStartAt,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"start_at": startAt,
		}),
	)
}

// InvalidEndAt 유효 하지 않는 종료 시간
func InvalidEndAt(endAt int64) error {
	return errors.Wrap(
		ErrInvalidEndAt,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"end_at": endAt,
		}),
	)
}
