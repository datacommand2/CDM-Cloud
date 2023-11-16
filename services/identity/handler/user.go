package handler

import (
	"context"
	"fmt"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/golang/protobuf/ptypes/wrappers"
	"time"
	"unicode/utf8"

	vaildator "github.com/asaskevich/govalidator"
	"github.com/datacommand2/cdm-cloud/common/config"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	identity "github.com/datacommand2/cdm-cloud/services/identity/proto"
	"github.com/jinzhu/gorm"
)

func validateUserGet(ctx context.Context, reqUser *identity.User, req *identity.GetUserRequest) error {
	// 일반 사용자의 경우 본인의 계정만 조회할 수 있음
	if !isAdmin(reqUser.Roles) && !isManager(reqUser.Roles) && reqUser.Id != req.UserId {
		return errors.UnauthorizedRequest(ctx)
	}

	return nil
}

// getUser 는 계정 정보를 데이터베이스로부터 얻기 위한 함수이다
// 최고 관리자 역할의 사용자는 모든 사용자 계정을 조회할 수 있으며,
// 관리자 역할의 사용자는 자신이 속한 테넌트의 사용자 계정을 조회할 수 있다.
// 나머지 모든 사용자는 자기 자신의 사용자 계정을 조회할 수 있다.
func getUser(db *gorm.DB, tid, userID uint64) (*identity.User, error) {
	var err error
	var user model.User

	// DB를 tenant id 와 함께 조회 하면 최고관리자가 1번 테넌트에 속하지 않은 유저에 대한 작업을 할 때
	// verifySession 에서 본인(최고관리자)의 정보를 가져오지 못한다.
	err = db.First(&user, userID).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, notFoundUser(userID)
	case err != nil:
		return nil, errors.UnusableDatabase(err)
	}

	rspUser, err := userModelToRsp(&user)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	tenant, err := getTenant(db, user.TenantID)
	if err != nil {
		return nil, err
	}
	rspUser.Tenant = tenant

	roles, err := user.Roles(db)
	if err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	var role *identity.Role
	for _, v := range roles {
		role, err = roleModelToRsp(&v)
		if err != nil {
			return nil, errors.Unknown(err)
		}
		rspUser.Roles = append(rspUser.Roles, role)
	}

	if isAdmin(rspUser.Roles) {
		rspUser.Groups, err = getGroups(db, tid)
		if err != nil {
			return nil, err
		}
	} else {
		groups, err := user.Groups(db)
		if err != nil {
			return nil, errors.UnusableDatabase(err)
		}
		for _, v := range groups {
			g, err := groupModelToRsp(&v)
			if err != nil {
				return nil, errors.Unknown(err)
			}
			rspUser.Groups = append(rspUser.Groups, g)
		}
	}

	return rspUser, nil
}

func getSimpleUser(db *gorm.DB, tid, userID uint64) (*identity.SimpleUser, error) {
	var err error
	var user model.User
	err = db.First(&user, &model.User{ID: userID, TenantID: tid}).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, notFoundUser(userID)
	case err != nil:
		return nil, errors.UnusableDatabase(err)
	}

	rspUser, err := simpleUserModelToRsp(&user)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	tenant, err := getTenant(db, user.TenantID)
	if err != nil {
		return nil, err
	}
	rspUser.Tenant = tenant

	roles, err := user.Roles(db)
	if err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	var role *identity.Role
	for _, v := range roles {
		role, err = roleModelToRsp(&v)
		if err != nil {
			return nil, errors.Unknown(err)
		}
		rspUser.Roles = append(rspUser.Roles, role)
	}

	if isAdmin(rspUser.Roles) {
		rspUser.Groups, err = getGroups(db, tid)
		if err != nil {
			return nil, err
		}
	} else {
		groups, err := user.Groups(db)
		if err != nil {
			return nil, errors.UnusableDatabase(err)
		}
		for _, v := range groups {
			g, err := groupModelToRsp(&v)
			if err != nil {
				return nil, errors.Unknown(err)
			}
			rspUser.Groups = append(rspUser.Groups, g)
		}
	}

	return rspUser, nil
}

// validateUserAdd 는 addUser 함수의 인자를 확인하는 함수이다
// 사용자 계정을 추가한다. 계정 명, 사용자 이름을 필수로 입력받아야하며,
// 추가로 최고 관리자 역할의 사용자가 사용자 계정을 추가할 때는 테넌트를 입력받아야 한다.
// 최고 관리자 역할의 사용자는 추가할 수 없으며, 관리자 역할의 사용자가 생성한 사용자의 테넌트는 관리자의 테넌트여야 한다
func validateUserAdd(ctx context.Context, db *gorm.DB, tenantID uint64, u *identity.User) (*identity.Tenant, error) {
	var err error

	if u.Id != 0 {
		return nil, errors.InvalidParameterValue("user.id", u.Id, "unusable parameter")
	}

	if u.Tenant == nil || u.Tenant.Id == 0 {
		return nil, errors.RequiredParameter("user.tenant.id")
	}

	// 추가하려는 사용자의 테넌트는 요청 대상 테넌트여야 한다.
	if tenantID != u.Tenant.Id {
		return nil, errors.UnauthorizedRequest(ctx)
	}

	// 최고 관리자 역할은 부여할 수 없다
	if isAdmin(u.Roles) {
		return nil, unassignableRole("admin")
	}

	tenant, err := getTenant(db, tenantID)
	switch {
	case errors.Equal(err, errNotFoundTenant):
		return nil, errors.Unknown(err)

	case err != nil:
		return nil, err
	}

	if len(u.Account) == 0 {
		return nil, errors.RequiredParameter("user.account")
	}

	if utf8.RuneCountInString(u.Account) > accountLength {
		return nil, errors.LengthOverflowParameterValue("user.account", u.Account, accountLength)
	}

	if err = db.Where("account = ?", u.Account).First(&model.User{}).Error; err == nil {
		return nil, errors.ConflictParameterValue("user.account", u.Account)
	} else if err != gorm.ErrRecordNotFound {
		return nil, errors.UnusableDatabase(err)
	}

	if len(u.Name) == 0 {
		return nil, errors.RequiredParameter("user.name")
	}

	if utf8.RuneCountInString(u.Name) > nameLength {
		return nil, errors.LengthOverflowParameterValue("user.name", u.Name, nameLength)
	}

	if len(u.Email) != 0 && utf8.RuneCountInString(u.Email) > emailLength {
		return nil, errors.LengthOverflowParameterValue("user.email", u.Email, emailLength)
	}

	if len(u.Email) != 0 && !vaildator.IsExistingEmail(u.Email) {
		return nil, errors.FormatMismatchParameterValue("user.email", u.Email, "email")
	}

	if err = db.Where("email = ?", u.Email).First(&model.User{}).Error; err == nil {
		return nil, errors.ConflictParameterValue("user.email", u.Email)
	} else if err != gorm.ErrRecordNotFound {
		return nil, errors.UnusableDatabase(err)
	}

	for idx, g := range u.Groups {
		var modelGroup = new(model.Group)
		err = db.First(modelGroup, g.Id).Error
		if err == gorm.ErrRecordNotFound {
			return nil, errors.InvalidParameterValue(fmt.Sprintf("user.group[%d].id", idx), g.Id, "not found group")
		} else if err != nil {
			return nil, errors.UnusableDatabase(err)
		}
		if u.Tenant.Id != modelGroup.TenantID {
			return nil, errors.InvalidParameterValue(fmt.Sprintf("user.group[%d].id", idx), g.Id, "unusable group")
		}
	}

	for idx, r := range u.Roles {
		err = db.First(&model.Role{}, r.Id).Error
		if err == gorm.ErrRecordNotFound {
			return nil, errors.InvalidParameterValue(fmt.Sprintf("user.role[%d].id", idx), r.Id, "not found role")
		} else if err != nil {
			return nil, errors.UnusableDatabase(err)
		}
	}

	if utf8.RuneCountInString(u.Department) > departmentLength {
		return nil, errors.LengthOverflowParameterValue("user.department", u.Department, departmentLength)
	}

	if utf8.RuneCountInString(u.Position) > positionLength {
		return nil, errors.LengthOverflowParameterValue("user.position", u.Position, positionLength)
	}

	if utf8.RuneCountInString(u.Contact) > contactLength {
		return nil, errors.LengthOverflowParameterValue("user.contact", u.Contact, contactLength)
	}

	if len(u.LanguageSet) != 0 && !languageBoundary.enum(u.LanguageSet) {
		return nil, errors.UnavailableParameterValue("user.language_set", u.LanguageSet, []interface{}{"eng", "kor"})
	}

	if utf8.RuneCountInString(u.Timezone) > timezoneLength {
		return nil, errors.LengthOverflowParameterValue("user.timezone", u.Timezone, timezoneLength)
	}

	return tenant, nil
}

// validateUserUpdate 는 updateUser 함수의 인자를 확인하는 함수이다
// ID, 계정 명, 암호, 테넌트는 수정할 수 없다.
// 최고 관리자 역할의 사용자 계정은 최고 관리자 역할의 사용자만 수정할 수 있으며, 역할, 그룹은 수정할 수 없다.
// 관리자 역할의 사용자는 자신이 속한 테넌트의 사용자 계정을 수정할 수 있으며, 사용자에게 최고 관리자 역할을 부여할 수 없다.
// 나머지 모든 사용자는 자기 자신의 사용자 계정만 수정할 수 있으며, 역할, 그룹은 수정할 수 없다.
func validateUserUpdate(ctx context.Context, db *gorm.DB, tenantID uint64, reqUser *identity.User, req *identity.UpdateUserRequest) error {
	var err error

	if req.User == nil {
		return errors.RequiredParameter("user")
	}

	if req.User.Tenant == nil || req.User.Tenant.Id == 0 {
		return errors.RequiredParameter("user.tenant.id")
	}

	if req.UserId != req.User.Id {
		return errors.UnchangeableParameter("user.id")
	}

	// 최고 관리자 역할의 사용자 계정은 최고 관리자역할의 사용자만 수정할 수 있다.
	if !isAdmin(reqUser.Roles) && req.User.Id == adminUser.ID {
		return errors.UnauthorizedRequest(ctx)
	}

	// 최고 관리자 역할은 부여할 수 없다
	if req.User.Id != adminUser.ID && isAdmin(req.User.Roles) {
		return unassignableRole("admin")
	}

	// 일반 사용자는 본인의 계정만 수정할 수 있다
	if !isAdmin(reqUser.Roles) && !isManager(reqUser.Roles) && reqUser.Id != req.User.Id {
		return errors.UnauthorizedRequest(ctx)
	}

	t := model.User{}
	err = db.First(&t, &model.User{ID: req.User.Id, TenantID: tenantID}).Error
	if err == gorm.ErrRecordNotFound {
		return notFoundUser(req.User.Id)
	} else if err != nil {
		return errors.UnusableDatabase(err)
	}

	// 테넌트는 수정할 수 없다.
	if req.User.Tenant.Id != t.TenantID {
		return errors.UnchangeableParameter("user.tenant.id")
	}

	// 계정명은 수정할 수 없다.
	if req.User.Account != t.Account {
		return errors.UnchangeableParameter("user.account")
	}

	if len(req.User.Name) == 0 {
		return errors.RequiredParameter("user.name")
	}

	if utf8.RuneCountInString(req.User.Name) > nameLength {
		return errors.LengthOverflowParameterValue("user.name", req.User.Name, nameLength)
	}

	if len(req.User.Email) != 0 && utf8.RuneCountInString(req.User.Email) > emailLength {
		return errors.LengthOverflowParameterValue("user.email", req.User.Email, emailLength)
	}

	if len(req.User.Email) != 0 && !vaildator.IsExistingEmail(req.User.Email) {
		return errors.FormatMismatchParameterValue("user.email", req.User.Email, "email")
	}

	if err = db.Where("NOT id = ? AND email = ?", req.User.Id, req.User.Email).First(&model.User{}).Error; err == nil {
		return errors.ConflictParameterValue("user.email", req.User.Email)
	} else if err != gorm.ErrRecordNotFound {
		return errors.UnusableDatabase(err)
	}

	// 최고 관리자의 그룹과 역할은 수정할 수 없다.
	// 최고 관리자와 매니저의 경우 그룹과 역할을 수정할 수 있으며, 이에따라 미리 그룹과 롤을 확인한다
	// 일반 사용자일 경우 변경 시도를 무시한다.
	if (isAdmin(reqUser.Roles) || isManager(reqUser.Roles)) && req.User.Id != adminUser.ID {
		for idx, g := range req.User.Groups {
			var modelGroup = new(model.Group)
			err := db.First(modelGroup, g.Id).Error
			if err == gorm.ErrRecordNotFound {
				return errors.InvalidParameterValue(fmt.Sprintf("user.group[%d].id", idx), g.Id, "not found group")
			} else if err != nil {
				return errors.UnusableDatabase(err)
			}
			if req.User.Tenant.Id != modelGroup.TenantID {
				return errors.InvalidParameterValue(fmt.Sprintf("user.group[%d].id", idx), req.User.Tenant.Id, "unusable group")
			}
		}

		for idx, r := range req.User.Roles {
			err := db.First(&model.Role{}, r.Id).Error
			if err == gorm.ErrRecordNotFound {
				return errors.InvalidParameterValue(fmt.Sprintf("user.role[%d].id", idx), r.Id, "not found role")
			} else if err != nil {
				return errors.UnusableDatabase(err)
			}
		}
	}

	if utf8.RuneCountInString(req.User.Department) > departmentLength {
		return errors.LengthOverflowParameterValue("user.department", req.User.Department, departmentLength)
	}

	if utf8.RuneCountInString(req.User.Position) > positionLength {
		return errors.LengthOverflowParameterValue("user.position", req.User.Position, positionLength)
	}

	if utf8.RuneCountInString(req.User.Contact) > contactLength {
		return errors.LengthOverflowParameterValue("user.contact", req.User.Contact, contactLength)
	}

	if len(req.User.LanguageSet) != 0 && !languageBoundary.enum(req.User.LanguageSet) {
		return errors.UnavailableParameterValue("user.department", req.User.LanguageSet, []interface{}{"kor", "eng"})
	}

	if utf8.RuneCountInString(req.User.Timezone) > timezoneLength {
		return errors.LengthOverflowParameterValue("user.timezone", req.User.Timezone, timezoneLength)
	}

	return nil
}

// updateUserGroupRelations 은 User와 Group간의 관계를 추가하기위한 함수이다
func updateUserGroupRelations(db *gorm.DB, userID uint64, groups []*identity.Group) error {
	if err := db.Where("user_id = ?", userID).Delete(&model.UserGroup{}).Error; err != nil {
		return err
	}

	for _, g := range groups {
		g := model.UserGroup{
			UserID:  userID,
			GroupID: g.Id,
		}

		if err := db.Save(&g).Error; err != nil {
			return err
		}
	}

	return nil
}

// updateUserRoleRelations 는 User와 Role간의 관계를 추가하기위한 함수이다
func updateUserRoleRelations(db *gorm.DB, userID uint64, roles []*identity.Role) error {
	if err := db.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
		return err
	}

	for _, r := range roles {
		r := model.UserRole{
			UserID: userID,
			RoleID: r.Id,
		}

		if err := db.Save(&r).Error; err != nil {
			return err
		}
	}

	return nil
}

// addUser 는 계정을 추가하기위한 함수이다
func addUser(db *gorm.DB, u *identity.User, password string) (*identity.User, error) {
	var err error

	user, err := userRspToModel(u)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	user.Password = encodePassword(password)
	user.PasswordUpdateFlag = new(bool)
	*user.PasswordUpdateFlag = true

	if err := db.Save(&user).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	if err := updateUserRoleRelations(db, user.ID, u.Roles); err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	if err := updateUserGroupRelations(db, user.ID, u.Groups); err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	roles, err := user.Roles(db)
	if err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	groups, err := user.Groups(db)
	if err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	rspUser, err := userModelToRsp(user)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	var role *identity.Role
	for _, v := range roles {
		role, err = roleModelToRsp(&v)
		if err != nil {
			return nil, errors.Unknown(err)
		}
		rspUser.Roles = append(rspUser.Roles, role)
	}

	var group *identity.Group
	for _, v := range groups {
		group, err = groupModelToRsp(&v)
		if err != nil {
			return nil, errors.Unknown(err)
		}
		rspUser.Groups = append(rspUser.Groups, group)
	}

	return rspUser, nil
}

// updateUser 는 계정 정보를 수정하는 함수이다
// 최고 관리자 : 모든 사용자의 정보를 업데이트 할 수 있으며, 자기 자신에 정보 업데이트 시 role, group 은 업데이트 불 가능
// 관리자: 최고 관리자를 제외한 다른 사용자의 정보 및 자기 자신에 정보를 업데이트 할 수 있음
// 일반 사용자: 자기 자신의 정보만 업데이트를 할 수 있으며, role, group 정보는 업데이트 불 가능
func updateUser(db *gorm.DB, tid uint64, reqUser, u *identity.User) (*identity.User, error) {
	user, err := userRspToModel(u)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	t := model.User{}
	if err := db.First(&t, u.Id).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	user.Password = t.Password
	user.LastLoggedInAt = t.LastLoggedInAt
	user.PasswordUpdatedAt = t.PasswordUpdatedAt
	user.PasswordUpdateFlag = t.PasswordUpdateFlag
	user.OldPassword = t.OldPassword

	if err := db.Save(&user).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	// 최고 관리자의 그룹과 역할은 변경할 수 없다.
	// 최고 관리자 또는 관리자만 그룹과 역할을 변경 할 수 있으며 일반 사용자의 경우 변경을 무시한다.
	if (isAdmin(reqUser.Roles) || isManager(reqUser.Roles)) && u.Id != adminUser.ID {
		if err = updateUserRoleRelations(db, user.ID, u.Roles); err != nil {
			return nil, errors.UnusableDatabase(err)
		}

		if err = updateUserGroupRelations(db, user.ID, u.Groups); err != nil {
			return nil, errors.UnusableDatabase(err)
		}
	}

	rspUser, err := userModelToRsp(user)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	tenant, err := getTenant(db, u.Tenant.Id)
	switch {
	case errors.Equal(err, errNotFoundTenant):
		return nil, errors.Unknown(err)

	case err != nil:
		return nil, err
	}
	rspUser.Tenant = tenant

	roles, err := user.Roles(db)
	if err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	var role *identity.Role
	for _, v := range roles {
		role, err = roleModelToRsp(&v)
		if err != nil {
			return nil, errors.Unknown(err)
		}
		rspUser.Roles = append(rspUser.Roles, role)
	}

	if isAdmin(rspUser.Roles) {
		rspUser.Groups, err = getGroups(db, tid)
		if err != nil {
			return nil, err
		}
	} else {
		groups, err := user.Groups(db)
		if err != nil {
			return nil, errors.UnusableDatabase(err)
		}
		for _, v := range groups {
			g, err := groupModelToRsp(&v)
			if err != nil {
				return nil, errors.Unknown(err)
			}
			rspUser.Groups = append(rspUser.Groups, g)
		}
	}

	return rspUser, nil
}

func validateUserDelete(db *gorm.DB, tenantID, userID uint64) (*model.User, error) {
	if userID == 0 {
		return nil, errors.RequiredParameter("user_id")
	}

	var user model.User
	if err := db.Where(&model.User{ID: userID, TenantID: tenantID}).First(&user).Error; err == gorm.ErrRecordNotFound {
		return nil, notFoundUser(userID)
	} else if err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	userRoles, err := user.Roles(db)
	if err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	// 최고 관리자 역할의 사용자 계정은 삭제할 수 없다.
	for _, r := range userRoles {
		if r.ID == Role[constant.Admin].ID {
			return nil, undeletableUser(userID)
		}
	}

	return &user, nil
}

// deleteUser 는 계정을 삭제하기위한 함수이다
// 최고 관리자 역할의 사용자 계정은 삭제할 수 없다.
// 최고 관리자 역할의 사용자는 모든 사용자 계정을 삭제할 수 있으며,
// 관리자 역할의 사용자는 자신이 속한 테넌트의 사용자 계정을 삭제할 수 있다.
func deleteUser(db *gorm.DB, tenantID, userID uint64) error {
	user, err := validateUserDelete(db, tenantID, userID)
	if err != nil {
		return err
	}

	if err = updateUserGroupRelations(db, userID, nil); err != nil {
		return errors.UnusableDatabase(err)
	}

	if err = updateUserRoleRelations(db, userID, nil); err != nil {
		return errors.UnusableDatabase(err)
	}

	if err = db.Delete(user, userID).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	return nil
}

func validateUserListGet(req *identity.GetUsersRequest) error {
	if utf8.RuneCountInString(req.Solution) > solutionLength {
		return errors.LengthOverflowParameterValue("solution", req.Solution, solutionLength)
	}

	if len(req.Role) != 0 && !roleBoundary.enum(req.Role) {
		return errors.UnavailableParameterValue("role", req.Role, []interface{}{"manager", "operator", "viewer"})
	}

	if utf8.RuneCountInString(req.Name) > nameLength {
		return errors.LengthOverflowParameterValue("user name", req.Name, nameLength)
	}

	if utf8.RuneCountInString(req.Department) > departmentLength {
		return errors.LengthOverflowParameterValue("user department", req.Department, departmentLength)
	}

	if utf8.RuneCountInString(req.Position) > positionLength {
		return errors.LengthOverflowParameterValue("user position", req.Position, positionLength)
	}

	return nil
}

// getUsers 는 유저목록을 반환하는 함수이다.
// 최고 관리자 역할의 사용자는 모든 사용자 계정들을 조회할 수 있으며,
// 관리자 역할의 사용자는 자신이 속한 테넌트의 사용자 계정들을 조회할 수 있다.
func getUsers(db *gorm.DB, tenantID uint64, filters ...userFilter) ([]*identity.User, error) {
	tenant, err := getTenant(db, tenantID)
	if err != nil {
		return nil, err
	}

	var conditions = db.Where("tenant_id = ?", tenantID)
	for _, f := range filters {
		conditions, err = f.Apply(conditions)
		if err != nil {
			return nil, errors.UnusableDatabase(err)
		}
	}

	var users []model.User
	if err = conditions.Find(&users).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	var resUsers []*identity.User
	for _, u := range users {
		rspUser, err := userModelToRsp(&u)
		if err != nil {
			return nil, errors.Unknown(err)
		}
		rspUser.Tenant = tenant

		roles, err := u.Roles(db)
		if err != nil {
			return nil, errors.UnusableDatabase(err)
		}

		var role *identity.Role
		for _, v := range roles {
			role, err = roleModelToRsp(&v)
			if err != nil {
				return nil, errors.Unknown(err)
			}
			rspUser.Roles = append(rspUser.Roles, role)
		}

		if isAdmin(rspUser.Roles) {
			rspUser.Groups, err = getGroups(db, tenantID)
			if err != nil {
				return nil, err
			}
		} else {
			groups, err := u.Groups(db)
			if err != nil {
				return nil, errors.UnusableDatabase(err)
			}
			for _, v := range groups {
				g, err := groupModelToRsp(&v)
				if err != nil {
					return nil, errors.Unknown(err)
				}
				rspUser.Groups = append(rspUser.Groups, g)
			}
		}

		rspUser.Session, err = getSession(u.ID)
		if err != nil {
			return nil, err
		}

		resUsers = append(resUsers, rspUser)
	}
	return resUsers, nil
}

// getUsersPagination 는 유저목록 건수를 반환하는 함수이다.
func getUsersPagination(db *gorm.DB, tenantID uint64, filters ...userFilter) (*identity.Pagination, error) {
	var err error
	var offset, limit uint64

	var conditions = db.Where("tenant_id = ?", tenantID)

	for _, f := range filters {
		if _, ok := f.(*paginationFilter); ok {
			offset = f.(*paginationFilter).Offset
			limit = f.(*paginationFilter).Limit
			continue
		}

		conditions, err = f.Apply(conditions)
		if err != nil {
			return nil, errors.UnusableDatabase(err)
		}
	}

	var total uint64
	if err = conditions.Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	if limit == 0 {
		return &identity.Pagination{
			Page:       &wrappers.UInt64Value{Value: 1},
			TotalPage:  &wrappers.UInt64Value{Value: 1},
			TotalItems: &wrappers.UInt64Value{Value: total},
		}, nil
	}

	return &identity.Pagination{
		Page:       &wrappers.UInt64Value{Value: offset/limit + 1},
		TotalPage:  &wrappers.UInt64Value{Value: (total + limit - 1) / limit},
		TotalItems: &wrappers.UInt64Value{Value: total},
	}, nil
}

func validateUserPasswordUpdate(db *gorm.DB, reqUser *identity.User, req *identity.UpdateUserPasswordRequest) error {
	var err error

	if req.UserId == 0 {
		return errors.RequiredParameter("user_id")
	}

	if reqUser.Id != req.UserId {
		return errors.InvalidParameterValue("user_id", req.UserId, "invalid user id")
	}

	if len(req.NewPassword) == 0 {
		return errors.RequiredParameter("new_password")
	}

	if len(req.CurrentPassword) == 0 {
		return errors.RequiredParameter("current_password")
	}

	if len(req.NewPassword) > passwordLength {
		return errors.LengthOverflowParameterValue("new_password", "******", passwordLength)
	}

	if len(req.CurrentPassword) > passwordLength {
		return errors.LengthOverflowParameterValue("current_password", "******", passwordLength)
	}

	user := model.User{}
	err = db.First(&user, req.UserId).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return notFoundUser(req.UserId)
	case err != nil:
		return errors.UnusableDatabase(err)
	}

	if user.Password != req.CurrentPassword {
		return currentPasswordMismatch()
	}

	var reusable = false
	if cfg := config.TenantConfig(db, reqUser.Tenant.Id, config.UserReuseOldPassword); cfg != nil {
		reusable, err = cfg.Value.Bool()
		if err != nil {
			return errors.Unknown(err)
		}
	}

	if !reusable && user.OldPassword != nil && *user.OldPassword == req.NewPassword {
		return notReusableOldPassword()
	}

	return nil
}

// updateUserPassword 는 유저의 암호를 수정하는 함수이다.
func updateUserPassword(db *gorm.DB, userID uint64, newPassword string) error {
	var err error
	user := model.User{}

	err = db.First(&user, userID).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return notFoundUser(userID)

	case err != nil:
		return errors.UnusableDatabase(err)
	}

	if user.OldPassword == nil {
		user.OldPassword = new(string)
	}

	if user.PasswordUpdatedAt == nil {
		user.PasswordUpdatedAt = new(int64)
	}

	if user.PasswordUpdateFlag == nil {
		user.PasswordUpdateFlag = new(bool)
	}

	*user.OldPassword = user.Password
	*user.PasswordUpdateFlag = false
	*user.PasswordUpdatedAt = time.Now().Unix()
	user.Password = newPassword

	if err = db.Save(&user).Error; err != nil {
		return errors.UnusableDatabase(err)
	}
	return nil
}

func validateUserPasswordReset(ctx context.Context, db *gorm.DB, reqUser *identity.User, tenantID, userID uint64) (*model.User, error) {
	if userID == 0 {
		return nil, errors.RequiredParameter("user_id")
	}

	var err error
	user := model.User{}
	err = db.First(&user, &model.User{ID: userID, TenantID: tenantID}).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, notFoundUser(userID)

	case err != nil:
		return nil, errors.UnusableDatabase(err)
	}

	if isManager(reqUser.Roles) && user.ID == adminUser.ID {
		return nil, errors.UnauthorizedRequest(ctx)
	}

	return &user, err
}

// resetUserPassword 는 유저의 암호를 초기화하는 함수이다.
// 최고 관리자 역할의 사용자는 모든 사용자 계정의 비밀번호를 초기화할 수 있으며,
// 관리자 역할의 사용자는 자신이 속한 테넌트의 사용자 계정 비밀번호를 초기화할 수 있다.
func resetUserPassword(ctx context.Context, db *gorm.DB, reqUser *identity.User, tenantID, userID uint64) (string, error) {
	user, err := validateUserPasswordReset(ctx, db, reqUser, tenantID, userID)
	if err != nil {
		return "", err
	}

	var password string
	password, err = generatePassword()
	if err != nil {
		return "", err
	}

	if user.OldPassword == nil {
		user.OldPassword = new(string)
	}

	if user.PasswordUpdatedAt == nil {
		user.PasswordUpdatedAt = new(int64)
	}

	if user.PasswordUpdateFlag == nil {
		user.PasswordUpdateFlag = new(bool)
	}

	*user.OldPassword = user.Password
	*user.PasswordUpdateFlag = true
	*user.PasswordUpdatedAt = time.Now().Unix()
	user.Password = encodePassword(password)

	if err := db.Save(&user).Error; err != nil {
		return "", errors.UnusableDatabase(err)
	}

	return password, nil
}
