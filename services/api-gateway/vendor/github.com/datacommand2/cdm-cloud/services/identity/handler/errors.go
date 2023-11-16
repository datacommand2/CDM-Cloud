package handler

import (
	"fmt"
	"github.com/datacommand2/cdm-cloud/common/errors"
	withStackErrors "github.com/pkg/errors"
)

var (
	// not found
	errNotFoundUser   = errors.New("not found user")
	errNotFoundTenant = errors.New("not found tenant")
	errNotFoundGroup  = errors.New("not found group")

	// user
	errNotReusableOldPassword  = errors.New("not reusable old password")
	errCurrentPasswordMismatch = errors.New("current password mismatch")
	errUndeletableUser         = errors.New("undeletable user")

	// group
	errAlreadyDeletedGroup = errors.New("already deleted group")
	errUndeletableGroup    = errors.New("undeletable group")

	// session
	errAlreadyLogin      = errors.New("already login account")
	errLoginRestricted   = errors.New("account restrict")
	errIncorrectPassword = errors.New("password mismatch")
	errUnknownSession    = errors.New("not found session")
	errExpiredSession    = errors.New("expired session")
	errUnverifiedSession = errors.New("unverified session")
	errInvalidSession    = errors.New("invalid session")

	// config
	errNotfoundTenantConfig = errors.New("not found tenant config")
	errInvalidTenantConfig  = errors.New("invalid tenant config")

	// role
	errUnassignableRole = errors.New("unassignable role")
)

func notFoundUser(value interface{}) error {
	return errors.Wrap(
		errNotFoundUser,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"id": value,
		}),
	)
}

func notFoundTenant(id uint64) error {
	return errors.Wrap(
		errNotFoundTenant,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"id": id,
		}),
	)
}

func notFoundGroup(id uint64) error {
	return errors.Wrap(
		errNotFoundGroup,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"id": id,
		}),
	)
}

func currentPasswordMismatch() error {
	return errors.Wrap(
		errCurrentPasswordMismatch,
		errors.CallerSkipCount(1),
	)
}

func notReusableOldPassword() error {
	return errors.Wrap(
		errNotReusableOldPassword,
		errors.CallerSkipCount(1),
	)
}

func alreadyDeletedGroup(groupID uint64) error {
	return errors.Wrap(
		errAlreadyDeletedGroup,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"group_id": groupID,
		}),
	)
}

func incorrectPassword() error {
	return errors.Wrap(
		errIncorrectPassword,
		errors.CallerSkipCount(1),
	)
}

func alreadyLogin(userID uint64) error {
	return errors.Wrap(
		errAlreadyLogin,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"user_id": userID,
		}),
	)
}

func unknownSession(id uint64) error {
	return errors.Wrap(
		errUnknownSession,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"id": id,
		}),
	)
}

func invalidSession(session string) error {
	return errors.Wrap(
		errInvalidSession,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"session": session,
		}),
	)
}

func expiredSession(session string) error {
	return errors.Wrap(
		errExpiredSession,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"session": session,
		}),
	)
}

func unverifiedSession(session string, cause error) error {
	return errors.Wrap(
		errUnverifiedSession,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"session": session,
			"cause":   fmt.Sprintf("%+v", withStackErrors.WithStack(cause)),
		}),
	)
}

func loginRestricted(account string, failedCount uint, lastFailedAt, until int64) error {
	return errors.Wrap(
		errLoginRestricted,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"account":        account,
			"failed_count":   failedCount,
			"last_failed_at": lastFailedAt,
			"until":          until,
		}),
	)
}

func notfoundTenantConfig(key string) error {
	return errors.Wrap(
		errNotfoundTenantConfig,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"key": key,
		}),
	)
}

func invalidTenantConfig(key, value string) error {
	return errors.Wrap(
		errInvalidTenantConfig,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"key":   key,
			"value": value,
		}),
	)
}

func unassignableRole(role string) error {
	return errors.Wrap(
		errUnassignableRole,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"role": role,
		}),
	)
}

func undeletableUser(userID uint64) error {
	return errors.Wrap(
		errUndeletableUser,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"user_id": userID,
		}),
	)
}

func undeletableGroup(groupID uint64) error {
	return errors.Wrap(
		errUndeletableGroup,
		errors.CallerSkipCount(1),
		errors.WithValue(map[string]interface{}{
			"group_id": groupID,
		}),
	)
}
