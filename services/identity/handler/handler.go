package handler

import (
	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	gormAdapter "github.com/casbin/gorm-adapter/v2"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/database"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/datacommand2/cdm-cloud/common/metadata"
	identity "github.com/datacommand2/cdm-cloud/services/identity/proto"
	"github.com/jinzhu/gorm"

	"context"
	"crypto/rsa"
)

// IdentityHandler 는 Identity 기능을 이용하기위한 handler 이다
type IdentityHandler struct {
	privateKey *rsa.PrivateKey
	enforcer   *casbin.Enforcer
	db         *database.ConnectionWrapper
}

var (
	//Role 은 admin 과 manager 정보를 캐싱하기위한 전역 변수이다.
	Role      map[string]model.Role
	adminUser model.User
)

// casbinModelConf 은 casbin authorization 을 위한 모델 conf 이다.
const casbinModelConf = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[policy_effect]
e = some(where (p.eft == allow))

[role_definition]
g = _, _, _

[matchers]
m = g(r.sub , p.sub, r.obj) && r.obj == p.obj && r.act == p.act || r.sub == 'admin'
`

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

// GetUser 는 사용자 정보를 반환하는 함수
func (h *IdentityHandler) GetUser(ctx context.Context, req *identity.GetUserRequest, rsp *identity.UserResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not find user. cause: %+v", err)
			}
		}()

		reqTenantID, reqUser, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_user.failure-get_request_metadata", err)
		}

		err = validateUserGet(ctx, reqUser, req)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_user.failure-validate", err)
		}

		rsp.User, err = getUser(tx, reqTenantID, req.UserId)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_user.failure-get", err)
		}

		if reqTenantID != rsp.User.Tenant.Id {
			err = notFoundUser(req.UserId)
			logger.Errorf("Could not find user. cause: %+v", err)
			return createError(ctx, "cdm-cloud.identity.get_user.failure-get", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.get_user.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.get_user.success", nil)
	})
}

// GetSimpleUser 는 간소화 된 사용자 정보를 반환하는 함수
func (h *IdentityHandler) GetSimpleUser(ctx context.Context, req *identity.GetSimpleUserRequest, rsp *identity.SimpleUserResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not find user. cause: %+v", err)
			}
		}()

		reqTenantID, _, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_simple_user.failure-get_request_metadata", err)
		}

		rsp.SimpleUser, err = getSimpleUser(tx, reqTenantID, req.UserId)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_simple_user.failure-get", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.get_simple_user.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.get_simple_user.success", nil)
	})
}

// AddUser 함수는 사용자 계정을 추가하는 함수이다.
func (h *IdentityHandler) AddUser(ctx context.Context, req *identity.AddUserRequest, rsp *identity.AddUserResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not add user. cause: %+v", err)
			}
		}()

		tenantID, _, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.add_user.failure-get_request_metadata", err)
		}

		tenant, err := validateUserAdd(ctx, tx, tenantID, req.User)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.add_user.failure-validate", err)
		}

		var password string
		password, err = generatePassword()
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.add_user.failure-generate_password", err)
		}

		rsp.User, err = addUser(tx, req.User, password)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.add_user.failure-add", err)
		}

		rsp.Password = password
		rsp.User.Tenant = tenant
		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.add_user.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.add_user.success", nil)
	})
}

// UpdateUser 함수는 사용자 정보를 갱신하는 함수이다.
// Todo : 정보 수정을 요청하는 계정이 admin 인지 본인인지 확인하고, 수정할 수 있는 정보를 제한해야한다.
func (h *IdentityHandler) UpdateUser(ctx context.Context, req *identity.UpdateUserRequest, rsp *identity.UserResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not update user. cause: %+v", err)
			}
		}()

		tenantID, reqUser, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.update_user.failure-get_request_metadata", err)
		}

		err = validateUserUpdate(ctx, tx, tenantID, reqUser, req)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.update_user.failure-validate", err)
		}

		rsp.User, err = updateUser(tx, tenantID, reqUser, req.User)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.update_user.failure-update", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.update_user.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.update_user.success", nil)
	})
}

// DeleteUser 함수는 데이터 베이스로 사용자 계정 정보 삭제 요청을 보내는 함수이다.
func (h *IdentityHandler) DeleteUser(ctx context.Context, req *identity.DeleteUserRequest, rsp *identity.MessageResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not delete user. cause: %+v", err)
			}
		}()

		tenantID, _, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.delete_user.failure-get_request_metadata", err)
		}

		err = deleteUser(tx, tenantID, req.UserId)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.delete_user.failure-delete", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.delete_user.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.delete_user.success", nil)
	})
}

func (h *IdentityHandler) makeUserFilter(tx *gorm.DB, req *identity.GetUsersRequest) []userFilter {
	var filters []userFilter

	if len(req.Solution) != 0 {
		filters = append(filters, &userSolutionFilter{DB: tx, Solution: req.Solution, Role: req.Role})
	}

	if req.GroupId != 0 {
		filters = append(filters, &userGroupFilter{DB: tx, GroupID: req.GroupId})
	}

	if req.ExcludeGroupId != 0 {
		filters = append(filters, &userExcludeGroupFilter{DB: tx, ExcludeGroupID: req.ExcludeGroupId})
	}

	if len(req.Name) != 0 {
		filters = append(filters, &userNameFilter{Name: req.Name})
	}

	if len(req.Department) != 0 {
		filters = append(filters, &userDepartmentFilter{Department: req.Department})
	}

	if len(req.Position) != 0 {
		filters = append(filters, &userPositionFilter{Position: req.Position})
	}

	if req.GetOffset() != nil && (req.GetLimit() != nil && req.GetLimit().GetValue() != 0) {
		filters = append(filters, &paginationFilter{Offset: req.GetOffset().GetValue(), Limit: req.GetLimit().GetValue()})
	}

	if req.LoginOnly {
		filters = append(filters, &loginFilter{})
	}

	return filters
}

// GetUsers 는 유저목록을 반환하는 함수이다.
func (h *IdentityHandler) GetUsers(ctx context.Context, req *identity.GetUsersRequest, rsp *identity.UsersResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not get user list. cause: %+v", err)
			}
		}()

		tenantID, _, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_users.failure-get_request_metadata", err)
		}

		err = validateUserListGet(req)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_users.failure-validate", err)
		}

		filters := h.makeUserFilter(tx, req)

		rsp.Users, err = getUsers(tx, tenantID, filters...)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_users.failure-get", err)
		}

		if len(rsp.Users) == 0 {
			return createError(ctx, "cdm-cloud.identity.get_users.success-get", errors.ErrNoContent)
		}

		rsp.Pagination, err = getUsersPagination(tx, tenantID, filters...)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_users.failure-get_user_count", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.get_users.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.get_users.success", nil)
	})
}

// UpdateUserPassword 는 사용자 비밀번호를 수정하는 함수이다.
func (h *IdentityHandler) UpdateUserPassword(ctx context.Context, req *identity.UpdateUserPasswordRequest, rsp *identity.MessageResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not update user password. cause: %+v", err)
			}
		}()

		_, reqUser, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.update_user_password.failure-get_request_metadata", err)
		}

		err = validateUserPasswordUpdate(tx, reqUser, req)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.update_user_password.failure-validate", err)
		}

		err = updateUserPassword(tx, req.UserId, req.NewPassword)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.update_user_password.failure-update", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.update_user_password.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.update_user_password.success", nil)
	})
}

// ResetUserPassword 는 사용자 비밀번호를 초기화하는 함수이다.
func (h *IdentityHandler) ResetUserPassword(ctx context.Context, req *identity.ResetUserPasswordRequest, res *identity.UserPasswordResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Error("Could not reset user password. cause: %+v", err)
			}
		}()

		tenantID, reqUser, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.reset_user_password.failure-get_request_metadata", err)
		}

		res.Password, err = resetUserPassword(ctx, tx, reqUser, tenantID, req.UserId)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.reset_user_password.failure-reset", err)
		}

		res.Message = &identity.Message{Code: "cdm-cloud.identity.reset_user_password.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.reset_user_password.success", nil)
	})
}

// GetGroups 는 그룹 목록을 조회하는 함수이다
func (h *IdentityHandler) GetGroups(ctx context.Context, req *identity.GetGroupsRequest, res *identity.GroupsResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not get group list. cause: %+v", err)
			}
		}()

		tenantID, _, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_groups.failure-get_request_metadata", err)
		}

		err = validateGroupListGet(req.Name)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_groups.failure-validate", err)
		}

		var filters []groupFilter
		if len(req.Name) != 0 {
			filters = append(filters, &groupNameFilter{Name: req.Name})
		}

		res.Groups, err = getGroups(tx, tenantID, filters...)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_groups.failure-get", err)
		}

		if len(res.Groups) == 0 {
			return createError(ctx, "cdm-cloud.identity.get_groups.success-get", errors.ErrNoContent)
		}

		res.Message = &identity.Message{Code: "cdm-cloud.identity.get_groups.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.get_groups.success", nil)
	})
}

// GetGroup 은 그룹 정보를 상세조회하기위한 함수이다
func (h *IdentityHandler) GetGroup(ctx context.Context, req *identity.GetGroupRequest, res *identity.GroupResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not find group. cause: %+v", err)
			}
		}()

		var tenantID uint64
		tenantID, _, err = getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_group.failure-get_request_metadata", err)
		}

		res.Group, err = getGroup(ctx, tx, tenantID, req.GroupId)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_group.failure-get", err)
		}

		res.Message = &identity.Message{Code: "cdm-cloud.identity.get_group.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.get_group.success", nil)
	})
}

// AddGroup 는 그룹을 추가하기위한 함수이다
func (h *IdentityHandler) AddGroup(ctx context.Context, req *identity.AddGroupRequest, res *identity.GroupResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not add group. cause: %+v", err)
			}
		}()

		tenantID, _, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.add_group.failure-get_request_metadata", err)
		}

		res.Group, err = addGroup(ctx, tx, tenantID, req.Group)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.add_group.failure-add", err)
		}

		res.Message = &identity.Message{Code: "cdm-cloud.identity.add_group.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.add_group.success", nil)
	})
}

// UpdateGroup 은 그룹정보를 수정하기위한 함수이다
func (h *IdentityHandler) UpdateGroup(ctx context.Context, req *identity.UpdateGroupRequest, res *identity.GroupResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not update group. cause: %+v", err)
			}
		}()

		tenantID, _, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.update_group.failure-get_request_metadata", err)
		}

		err = validateUpdateGroup(tx, tenantID, req)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.update_group.failure-validate", err)
		}

		res.Group, err = updateGroup(tx, tenantID, req.Group)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.update_group.failure-update", err)
		}

		res.Message = &identity.Message{Code: "cdm-cloud.identity.update_group.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.update_group.success", nil)
	})
}

// DeleteGroup 은 그룹을 삭제하기위한 함수이다
func (h *IdentityHandler) DeleteGroup(ctx context.Context, req *identity.DeleteGroupRequest, rsp *identity.MessageResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not delete group. cause: %+v", err)
			}
		}()

		tenantID, _, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.delete_group.failure-get_request_metadata", err)
		}

		err = deleteGroup(ctx, tx, tenantID, req.GroupId)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.delete_group.failure-delete", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.delete_group.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.delete_group.success", nil)
	})
}

// SetGroupUsers 는 사용자 그룹 사용자 목록 수정하는 함수이다.
// 해당 사용자 그룹의 사용자 목록을 덮어쓴다.
func (h *IdentityHandler) SetGroupUsers(ctx context.Context, req *identity.SetGroupUsersRequest, rsp *identity.UsersResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not set users in group. cause: %+v", err)
			}
		}()

		tenantID, _, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.set_group_users.failure-get_request_metadata", err)
		}

		rsp.Users, err = setGroupUsers(ctx, tx, tenantID, req)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.set_group_users.failure-set", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.set_group_users.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.set_group_users.success", nil)
	})
}

// GetRoles 는 솔루션 역할 목록을 조회하기위한 함수이다
func (h *IdentityHandler) GetRoles(ctx context.Context, req *identity.GetRolesRequest, rsp *identity.RolesResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not get role list. cause: %+v", err)
			}
		}()

		tenantID, _, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_roles.failure-get_request_metadata", err)
		}

		err = validateRoleList(req)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_roles.failure-validate", err)
		}

		var filters = []roleFilter{&tenantSolutionFilter{DB: tx, TenantID: tenantID}}
		if len(req.Solution) != 0 {
			filters = append(filters, &solutionNameFilter{Solution: req.Solution})
		}

		if len(req.Role) != 0 {
			filters = append(filters, &roleNameFilter{Role: req.Role})
		}

		rsp.Roles, err = getRoles(tx, filters...)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_roles.failure-get", err)
		}

		if len(rsp.Roles) == 0 {
			return createError(ctx, "cdm-cloud.identity.get_roles.success-get", errors.ErrNoContent)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.get_roles.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.get_roles.success", nil)
	})
}

// Login 은 로그인을 수행하기위한 함수이다
func (h *IdentityHandler) Login(ctx context.Context, req *identity.LoginRequest, rsp *identity.UserResponse) error {
	var err error

	defer func() {
		// 패스워드 불일치로 로그인 실패시 로그인 실패 내역 업데이트
		if errors.Equal(err, errIncorrectPassword) {
			_ = database.Transaction(func(tx *gorm.DB) error {
				err := updateUserLoginFailedInfo(tx, req)
				switch {
				case errors.Equal(err, errors.ErrUnusableDatabase):
					reportEvent(ctx, "cdm-cloud.identity.login.warning-update_user_login_failed_info", "unusable_database", err)

				case errors.Equal(err, errors.ErrUnknown):
					reportEvent(ctx, "cdm-cloud.identity.login.warning-update_user_login_failed_info", "unknown", err)
				}

				logger.Warnf("Could not update user login fail information. cause : %+v", err)

				return nil
			})
		}
	}()

	return database.Transaction(func(tx *gorm.DB) error {
		defer func() {
			if err != nil {
				logger.Errorf("Could not login. cause: %+v", err)
			}
		}()
		// 데이터베이스 조회
		rsp.User, err = loginUser(ctx, tx, req)
		if err != nil {
			if errors.Equal(err, errNotFoundUser) {
				err = errors.UnauthenticatedRequest(ctx)
			}
			return createError(ctx, "cdm-cloud.identity.login.failure-login", err)
		}

		// 새 세션 발급
		rsp.User.Session = new(identity.Session)
		rsp.User.Session.Key, err = newSession(ctx, tx, rsp.User.Tenant.Id, rsp.User.Id, h.privateKey, req.Force)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.login.failure-new_session", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.login.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.login.success", nil)
	})
}

// Logout 은 로그아웃을 수행하기위한 함수이다
func (h *IdentityHandler) Logout(ctx context.Context, _ *identity.Empty, rsp *identity.MessageResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		session, err := metadata.GetAuthenticatedSession(ctx)
		if err != nil {
			err = errors.InvalidRequest(ctx)
			logger.Errorf("Could not logout. Getting session key failed. cause: %+v", err)
			return createError(ctx, "cdm-cloud.identity.logout.failure-get_authenticated_session", err)
		}

		reqUser, err := metadata.GetAuthenticatedUser(ctx)
		if err != nil {
			err = errors.InvalidRequest(ctx)
			logger.Errorf("Could not logout. Getting user failed. cause: %+v", err)
			return createError(ctx, "cdm-cloud.identity.logout.failure-get_authenticated_user", err)
		}

		defer func() {
			if err != nil {
				logger.Errorf("Could not logout. cause : %+v", err)
			}
		}()

		id, err := validateDeleteSession(reqUser, session)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.logout.failure-validate_delete_session", err)
		}

		// 세션 삭제
		err = deleteSession(id)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.logout.failure-delete_session", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.logout.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.logout.success", nil)
	})
}

// VerifySession 함수는 세션의 유효성을 확인하기위한 함수이다
func (h *IdentityHandler) VerifySession(ctx context.Context, _ *identity.Empty, rsp *identity.UserResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		session, err := metadata.GetAuthenticatedSession(ctx)
		if err != nil {
			err = errors.UnauthenticatedRequest(ctx)
			logger.Errorf("Could not verify session key. Getting session key failed, cause: %+v", err)
			return createError(ctx, "cdm-cloud.identity.verify_session.failure-get_authenticated_session", err)
		}

		tenantID, err := metadata.GetTenantID(ctx)
		if err != nil {
			logger.Errorf("Could not verify session key. cause: %+v", err)
			return createError(ctx, "cdm-cloud.identity.verify_session.failure-get_tenant_id", err)
		}

		defer func() {
			if err != nil {
				logger.Errorf("Could not verify session. cause : %+v", err)
			}
		}()

		// 세션의 유효성 확인
		var payload *SessionPayload
		payload, err = verifySession(ctx, session, h.privateKey)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.verify_session.failure-verify", err)
		}

		// 유저 정보 반환을 위한 데이터베이스 조회
		rsp.User, err = getUser(tx, tenantID, payload.ID)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.verify_session.failure-get_user", err)
		}

		// 세션 키 업데이트
		rsp.User.Session = new(identity.Session)
		rsp.User.Session.Key, err = updateSession(ctx, tx, rsp.User.Id, rsp.User.Tenant.Id, payload.MagicNumber, h.privateKey)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.verify_session.failure-update_session", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.verify_session.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.verify_session.success", nil)
	})
}

// RevokeSession 은 접속한 사용자를 강제로 로그아웃하기위한 함수이다.
func (h *IdentityHandler) RevokeSession(ctx context.Context, req *identity.RevokeSessionRequest, rsp *identity.MessageResponse) error {
	return database.Transaction(func(tx *gorm.DB) error {
		var err error
		defer func() {
			if err != nil {
				logger.Errorf("Could not revoke session. cause : %+v", err)
			}
		}()

		tenantID, reqUser, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.revoke_session.failure-get_request_metadata", err)
		}

		id, err := validateRevokeSession(tx, reqUser, tenantID, req.SessionKey)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.logout.failure-validate_revoke_session", err)
		}

		err = deleteSession(id)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.revoke_session.failure-delete_session", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.revoke_session.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.revoke_session.success", nil)
	})
}

// GetConfig 는 사용자 인증에 관련된 설정을 조회하는 함수이다.
func (h *IdentityHandler) GetConfig(ctx context.Context, _ *identity.Empty, rsp *identity.ConfigResponse) error {
	return database.Transaction(func(tx *gorm.DB) (err error) {
		defer func() {
			if err != nil {
				logger.Errorf("Could not get config. cause : %+v", err)
			}
		}()

		tenantID, _, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_config.failure-get_request_metadata", err)
		}

		err = getConfig(tx, tenantID, rsp)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_config.failure-get", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.get_config.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.get_config.success", nil)
	})
}

// SetConfig 는 사용자 인증에 관련된 설정을 수정하는 함수이다.
func (h *IdentityHandler) SetConfig(ctx context.Context, req *identity.ConfigRequest, rsp *identity.ConfigResponse) error {
	return database.Transaction(func(tx *gorm.DB) (err error) {
		defer func() {
			if err != nil {
				logger.Errorf("Could not set config. cause : %+v", err)
			}
		}()

		tenantID, _, err := getRequestMetadata(ctx)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.set_config.failure-get_request_metadata", err)
		}

		err = setConfig(tx, tenantID, req)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.set_config.failure-set", err)
		}

		err = getConfig(tx, tenantID, rsp)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.set_config.failure-get", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.set_config.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.set_config.success", nil)
	})
}

// CheckAuthorization 는 사용자 인가 여부를 확인하는 함수이다.
func (h *IdentityHandler) CheckAuthorization(ctx context.Context, req *identity.CheckAuthorizationRequest, rsp *identity.MessageResponse) error {
	var err error
	defer func() {
		if err != nil {
			logger.Errorf("Could not check authorization. cause: %+v", err)
		}
	}()

	tenantID, user, err := getRequestMetadata(ctx)
	if err != nil {
		return createError(ctx, "cdm-cloud.identity.check_authorization.failure-get_request_metadata", err)
	}

	err = database.Transaction(func(db *gorm.DB) error {
		err := db.First(&model.Tenant{}, tenantID).Error
		if err == gorm.ErrRecordNotFound {
			return notFoundTenant(tenantID)
		} else if err != nil {
			return errors.UnusableDatabase(err)
		}
		return nil
	})
	if err != nil {
		return createError(ctx, "cdm-cloud.identity.check_authorization.failure-transaction", err)
	}

	if !isAdmin(user.Roles) && tenantID != user.Tenant.Id {
		err = errors.UnauthorizedRequest(ctx)
		return createError(ctx, "cdm-cloud.identity.check_authorization.failure-mismatch_user", err)
	}

	var roles []*identity.Role
	roles = append(user.Roles, &identity.Role{Role: constant.User, Solution: constant.SolutionName})
	for _, r := range roles {
		if ok, err := h.enforcer.Enforce(r.Role, r.Solution, req.Endpoint); ok {
			// authorized
			rsp.Message = &identity.Message{Code: "cdm-cloud.identity.check_authorization.success"}
			return errors.StatusOK(ctx, "cdm-cloud.identity.check_authorization.success", nil)

		} else if err != nil {
			return createError(ctx, "cdm-cloud.identity.check_authorization.failure-enforce", err)
		}
	}

	err = errors.UnauthorizedRequest(ctx)
	return createError(ctx, "cdm-cloud.identity.check_authorization.failure", err)
}

// GetTenants 는 테넌트 목록을 조회하는 함수이다.
func (h *IdentityHandler) GetTenants(ctx context.Context, req *identity.TenantsRequest, rsp *identity.TenantsResponse) error {
	return database.Transaction(func(tx *gorm.DB) (err error) {
		defer func() {
			if err != nil {
				logger.Errorf("Could not get tenant list. cause: %+v", err)
			}
		}()

		err = validateTenantListGet(req.Name)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_tenants.failure-validate", err)
		}

		var filters []tenantFilter
		if len(req.Name) != 0 {
			filters = append(filters, &tenantNameFilter{Name: req.Name})
		}

		rsp.Tenants, err = getTenants(tx, filters...)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_tenants.failure-get", err)
		}

		if len(rsp.Tenants) == 0 {
			return createError(ctx, "cdm-cloud.identity.get_tenants.success-get", errors.ErrNoContent)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.get_tenants.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.get_tenants.success", nil)
	})
}

// GetTenant 는 테넌트 정보를 상세조회하는 함수이다.
func (h *IdentityHandler) GetTenant(ctx context.Context, req *identity.TenantRequest, rsp *identity.TenantResponse) error {
	return database.Transaction(func(tx *gorm.DB) (err error) {
		defer func() {
			if err != nil {
				logger.Errorf("Could not get tenant. cause: %+v", err)
			}
		}()

		rsp.Tenant, err = getTenant(tx, req.TenantId)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.get_tenant.failure-get", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.get_tenant.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.get_tenant.success", nil)
	})
}

// AddTenant 는 테넌트를 추가하는 함수이다.
func (h *IdentityHandler) AddTenant(ctx context.Context, req *identity.AddTenantRequest, rsp *identity.TenantResponse) error {
	return database.Transaction(func(tx *gorm.DB) (err error) {
		defer func() {
			if err != nil {
				logger.Errorf("Could not add tenant. cause: %+v", err)
			}
		}()

		rsp.Tenant, err = addTenant(tx, req)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.add_tenant.failure-add", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.add_tenant.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.add_tenant.success", nil)
	})
}

// UpdateTenant 는 테넌트 정보를 수정하기위한 함수이다.
func (h *IdentityHandler) UpdateTenant(ctx context.Context, req *identity.UpdateTenantRequest, rsp *identity.TenantResponse) error {
	return database.Transaction(func(tx *gorm.DB) (err error) {
		defer func() {
			if err != nil {
				logger.Errorf("Could not update tenant. cause: %+v", err)
			}
		}()

		rsp.Tenant, err = updateTenant(tx, req.TenantId, req)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.update_tenant.failure-update", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.update_tenant.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.update_tenant.success", nil)
	})
}

// ActivateTenant 는 테넌트를 활성화하는 함수이다.
func (h *IdentityHandler) ActivateTenant(ctx context.Context, req *identity.TenantRequest, rsp *identity.TenantResponse) error {
	return database.Transaction(func(tx *gorm.DB) (err error) {
		defer func() {
			if err != nil {
				logger.Errorf("Could not activate tenant. cause: %+v", err)
			}
		}()

		rsp.Tenant, err = activateTenant(tx, req.TenantId)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.activate_tenant.failure-activate", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.activate_tenant.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.activate_tenant.success", nil)
	})
}

// DeactivateTenant 는 테넌트를 비활성화하는 함수이다.
func (h *IdentityHandler) DeactivateTenant(ctx context.Context, req *identity.TenantRequest, rsp *identity.TenantResponse) error {
	return database.Transaction(func(tx *gorm.DB) (err error) {
		defer func() {
			if err != nil {
				logger.Errorf("Could not deactivate tenant. cause: %+v", err)
			}
		}()

		rsp.Tenant, err = deactivateTenant(tx, req.TenantId)
		if err != nil {
			return createError(ctx, "cdm-cloud.identity.deactivate_tenant.failure-deactivate", err)
		}

		rsp.Message = &identity.Message{Code: "cdm-cloud.identity.deactivate_tenant.success"}
		return errors.StatusOK(ctx, "cdm-cloud.identity.deactivate_tenant.success", nil)
	})
}

// Close 는 Identity Handler 가 사용하는 리소스를 닫아주는 함수이다.
func (h *IdentityHandler) Close() error {
	if err := h.db.Close(); err != nil {
		return errors.UnusableDatabase(err)
	}
	return nil
}

func (h *IdentityHandler) init() error {
	var err error
	if h.privateKey, err = getPrivateKeyFromFile(defaultPrivateKeyPath + "identity.pem"); err != nil {
		return errors.Unknown(err)
	}

	if h.db, err = database.OpenDefault(); err != nil {
		return errors.UnusableDatabase(err)
	}

	var adapter *gormAdapter.Adapter
	if adapter, err = gormAdapter.NewAdapterByDBUsePrefix(h.db.DB, "cdm_"); err != nil {
		return errors.UnusableDatabase(err)
	}

	var casbinModel casbinmodel.Model
	if casbinModel, err = casbinmodel.NewModelFromString(casbinModelConf); err != nil {
		return errors.Unknown(err)
	}

	if h.enforcer, err = casbin.NewEnforcer(casbinModel, adapter); err != nil {
		return errors.Unknown(err)
	}

	if err = h.enforcer.LoadPolicy(); err != nil {
		return errors.Unknown(err)
	}
	return nil
}

// NewIdentityHandler 는 Identity 서비스의 DB를 초기화하고
// 핸들러를 반환하는 함수이다.
func NewIdentityHandler() (*IdentityHandler, error) {
	var (
		err     error
		admin   model.Role
		manager model.Role
	)
	Role = make(map[string]model.Role)
	err = database.Transaction(func(db *gorm.DB) error {
		if err := db.Select("id").Where(&model.User{Account: constant.Admin}).First(&adminUser).Error; err != nil {
			return err
		}

		if err := db.Where(&model.Role{Solution: constant.SolutionName, Role: constant.Admin}).First(&admin).Error; err != nil {
			return err
		}

		return db.Where(&model.Role{Solution: constant.SolutionName, Role: constant.Manager}).First(&manager).Error
	})
	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, errors.Unknown(err)

	case err != nil:
		return nil, errors.UnusableDatabase(err)
	}

	Role[constant.Admin] = admin
	Role[constant.Manager] = manager

	i := &IdentityHandler{}
	if err = i.init(); err != nil {
		return nil, err
	}
	return i, nil
}
