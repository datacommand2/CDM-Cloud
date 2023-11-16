package notifier

import "github.com/datacommand2/cdm-cloud/common/errors"

var (
	// ErrNotFoundUser 유저를 찾을 수 없음.
	ErrNotFoundUser = errors.New("not found user")
)

// NotFoundUser 유저를 찾을 수 없음.
func NotFoundUser(id uint64) error {
	return errors.Wrap(
		ErrNotFoundUser,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"id": id,
		}),
	)
}
