package config

import "github.com/datacommand2/cdm-cloud/common/errors"

var (
	// ErrNotFoundUser user 가 존재하지 않음
	ErrNotFoundUser = errors.New("not found user")
	// ErrNotFoundTenant tenant 가 존재하지 않음
	ErrNotFoundTenant = errors.New("not found tenant")
)

// NotFoundUser 요청한 id 의 user 가 존재하지 않음
func NotFoundUser(id uint64) error {
	return errors.Wrap(
		ErrNotFoundUser,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"id": id,
		}),
	)
}

// NotFoundTenant 요청한 id 의 tenant 가 존재하지 않음
func NotFoundTenant(id uint64) error {
	return errors.Wrap(
		ErrNotFoundTenant,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"id": id,
		}),
	)
}
