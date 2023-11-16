package event

import "github.com/datacommand2/cdm-cloud/common/errors"

var (
	// ErrNotFoundEvent event 가 존재하지 않음
	ErrNotFoundEvent = errors.New("not found event")
)

// NotFoundEvent 요청한 id 의 event 가 존재하지 않음
func NotFoundEvent(id uint64) error {
	return errors.Wrap(
		ErrNotFoundEvent,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"id": id,
		}),
	)
}
