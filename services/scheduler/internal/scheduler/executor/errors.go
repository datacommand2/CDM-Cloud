package executor

import "github.com/datacommand2/cdm-cloud/common/errors"

var (
	// ErrUnsupportedScheduleType 지원하지 않는 스케쥴 주기
	ErrUnsupportedScheduleType = errors.New("unsupported schedule type")

	// ErrUnsupportedTimezone 유효하지 않은 타임 존
	ErrUnsupportedTimezone = errors.New("unsupported timezone")
)

// UnsupportedScheduleType 지원하지 않는 스케쥴 주기
func UnsupportedScheduleType(t string) error {
	return errors.Wrap(
		ErrUnsupportedScheduleType,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"type": t,
		}),
	)
}

// UnsupportedTimezone 유효하지 않은 타임 존
func UnsupportedTimezone(tz string) error {
	return errors.Wrap(
		ErrUnsupportedTimezone,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"timezone": tz,
		}),
	)
}
