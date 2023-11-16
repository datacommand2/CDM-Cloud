package handler

import "github.com/datacommand2/cdm-cloud/common/errors"

var (
	// ErrNoSolutionRole 특정 솔루션에 대한 사용자의 권한이 없을 때 발생
	ErrNoSolutionRole = errors.New("no role of solution")
)

// NoSolutionRole 특정 솔루션에 대한 사용자의 권한이 없을 때 발생
func NoSolutionRole(solution string, userID uint64) error {
	return errors.Wrap(
		ErrNoSolutionRole,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"solution": solution,
			"user_id":  userID,
		}),
	)
}
