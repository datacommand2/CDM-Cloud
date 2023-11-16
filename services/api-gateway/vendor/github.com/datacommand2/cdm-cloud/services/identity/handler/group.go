package handler

import (
	"context"
	"fmt"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	identity "github.com/datacommand2/cdm-cloud/services/identity/proto"
	"github.com/jinzhu/gorm"
	"unicode/utf8"
)

func validateGroupListGet(name string) error {
	if utf8.RuneCountInString(name) > nameLength {
		return errors.LengthOverflowParameterValue("group.name", name, nameLength)
	}
	return nil
}

// getGroups 는 그룹의 목록을 조회하기위한 함수이다
func getGroups(db *gorm.DB, tenantID uint64, filters ...groupFilter) ([]*identity.Group, error) {
	tenant, err := getTenant(db, tenantID)
	if err != nil {
		return nil, err
	}

	var conditions = db.Where("tenant_id = ? AND deleted_flag = false", tenantID)
	for _, f := range filters {
		conditions, err = f.Apply(conditions)
		if err != nil {
			return nil, errors.UnusableDatabase(err)
		}
	}

	var groups []model.Group
	if err = conditions.Find(&groups).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	var resGroups []*identity.Group
	for _, g := range groups {
		rspGroup, err := groupModelToRsp(&g)
		if err != nil {
			return nil, errors.Unknown(err)
		}
		rspGroup.Tenant = tenant
		resGroups = append(resGroups, rspGroup)
	}

	return resGroups, nil
}

// getGroup 은 그룹정보를 상세조회하기위한 함수이다
func getGroup(ctx context.Context, db *gorm.DB, tenantID, groupID uint64) (*identity.Group, error) {
	if groupID == 0 {
		return nil, errors.RequiredParameter("group_id")
	}

	group := model.Group{}
	err := db.Where("id = ?", groupID).
		Where("tenant_id = ?", tenantID).
		Where("deleted_flag = false").
		First(&group).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, notFoundGroup(groupID)
	case err != nil:
		return nil, errors.UnusableDatabase(err)
	}

	tenant, err := getTenant(db, tenantID)
	if err != nil {
		return nil, err
	}

	rspGroup, err := groupModelToRsp(&group)
	if err != nil {
		return nil, errors.Unknown(err)
	}
	rspGroup.Tenant = tenant
	return rspGroup, nil
}

func validateGroupAdd(ctx context.Context, db *gorm.DB, tenantID uint64, g *identity.Group) (*identity.Tenant, error) {
	var err error
	if g == nil {
		return nil, errors.RequiredParameter("group")
	}

	if g.Id != 0 {
		return nil, errors.InvalidParameterValue("group.id", g.Id, "unusable parameter")
	}

	if g.Tenant == nil || g.Tenant.Id == 0 {
		return nil, errors.RequiredParameter("group.tenant.id")
	}

	if tenantID != g.Tenant.Id {
		return nil, errors.UnauthorizedRequest(ctx)
	}

	if len(g.Name) == 0 {
		return nil, errors.RequiredParameter("group.name")
	}

	if utf8.RuneCountInString(g.Name) > nameLength {
		return nil, errors.LengthOverflowParameterValue("group.name", g.Name, nameLength)
	}

	err = db.Where("name = ?", g.Name).
		Where("tenant_id = ?", g.Tenant.Id).
		Where("deleted_flag = false").
		First(&model.Group{}).Error
	if err == nil {
		return nil, errors.ConflictParameterValue("group.name", g.Name)
	} else if err != gorm.ErrRecordNotFound {
		return nil, errors.UnusableDatabase(err)
	}

	if utf8.RuneCountInString(g.Remarks) > remarksLength {
		return nil, errors.LengthOverflowParameterValue("group.remarks", g.Remarks, remarksLength)
	}

	tenant, err := getTenant(db, tenantID)
	if err != nil {
		return nil, err
	}

	return tenant, nil
}

// addGroup 는 그룹을 추가하기위한 함수이다
func addGroup(ctx context.Context, db *gorm.DB, tenantID uint64, g *identity.Group) (*identity.Group, error) {
	var tenant *identity.Tenant
	var err error
	if tenant, err = validateGroupAdd(ctx, db, tenantID, g); err != nil {
		return nil, err
	}

	group, err := groupRspToModel(g)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	if err = db.Save(&group).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	rspGroup, err := groupModelToRsp(group)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	rspGroup.Tenant = tenant

	return rspGroup, nil
}

func validateUpdateGroup(db *gorm.DB, tenantID uint64, req *identity.UpdateGroupRequest) error {
	var err error
	if req.Group == nil {
		return errors.RequiredParameter("group")
	}

	if req.Group.Tenant == nil || req.Group.Tenant.Id == 0 {
		return errors.RequiredParameter("group.tenant.id")
	}

	if req.GroupId != req.Group.Id {
		return errors.UnchangeableParameter("group.id")
	}

	var t model.Group
	err = db.Where("id = ?", req.Group.Id).
		Where("tenant_id = ?", tenantID).
		Where("deleted_flag = false").
		First(&t).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return notFoundGroup(req.Group.Id)
	case err != nil:
		return errors.UnusableDatabase(err)
	}

	if req.Group.Tenant.Id != t.TenantID {
		return errors.UnchangeableParameter("group.tenant.id")
	}

	if len(req.Group.Name) == 0 {
		return errors.RequiredParameter("group.name")
	}

	if utf8.RuneCountInString(req.Group.Name) > nameLength {
		return errors.LengthOverflowParameterValue("group.name", req.Group.Name, nameLength)
	}

	err = db.Where("NOT id = ?", req.Group.Id).
		Where("tenant_id = ?", req.Group.Tenant.Id).
		Where("name = ?", req.Group.Name).
		Where("deleted_flag = false").
		First(&model.Group{}).Error
	if err == nil {
		return errors.ConflictParameterValue("group.name", req.Group.Name)
	} else if err != gorm.ErrRecordNotFound {
		return errors.UnusableDatabase(err)
	}

	if utf8.RuneCountInString(req.Group.Remarks) > remarksLength {
		return errors.LengthOverflowParameterValue("group.remarks", req.Group.Remarks, remarksLength)
	}

	return nil
}

// updateGroup 는 그룹정보를 수정하기위한 함수이다
func updateGroup(db *gorm.DB, tenantID uint64, g *identity.Group) (*identity.Group, error) {
	tenant, err := getTenant(db, tenantID)
	if err != nil {
		return nil, err
	}

	group, err := groupRspToModel(g)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	if err := db.Save(group).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	rspGroup, err := groupModelToRsp(group)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	rspGroup.Tenant = tenant
	return rspGroup, nil
}

func validDeleteGroup(ctx context.Context, db *gorm.DB, tid, gid uint64) (*model.Group, error) {
	if gid == 0 {
		return nil, errors.RequiredParameter("group_id")
	}

	group := model.Group{}
	err := db.First(&group, "id = ?", gid).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, notFoundGroup(gid)
	case err != nil:
		return nil, errors.UnusableDatabase(err)
	}

	if group.Name == "default" {
		return nil, undeletableGroup(gid)
	}

	if group.DeletedFlag {
		return nil, alreadyDeletedGroup(gid)
	}

	if tid != group.TenantID {
		return nil, errors.UnauthorizedRequest(ctx)
	}

	return &group, nil
}

// deleteGroup 는 그룹을 삭제하기위한 함수이다
func deleteGroup(ctx context.Context, db *gorm.DB, tenantID, groupID uint64) error {
	var group *model.Group
	var err error
	if group, err = validDeleteGroup(ctx, db, tenantID, groupID); err != nil {
		return err
	}

	if err = db.Where("group_id = ?", groupID).Delete(&model.UserGroup{}).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	group.DeletedFlag = true
	if err = db.Save(&group).Error; err != nil {
		return errors.UnusableDatabase(err)
	}
	return nil
}

func validateGroupUserSet(ctx context.Context, db *gorm.DB, tenantID uint64, req *identity.SetGroupUsersRequest) (*identity.Tenant, error) {
	if req.GroupId == 0 {
		return nil, errors.RequiredParameter("group_id")
	}

	var group model.Group
	err := db.First(&group, "id = ? AND deleted_flag = false", req.GroupId).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, notFoundGroup(req.GroupId)

	case err != nil:
		return nil, errors.UnusableDatabase(err)

	case tenantID != group.TenantID:
		return nil, errors.UnauthorizedRequest(ctx)
	}

	if len(req.Users) != 0 {
		for idx, user := range req.Users {
			if user.Id == 0 {
				return nil, errors.RequiredParameter(fmt.Sprintf("user[%d].id", idx))
			}

			err = db.Where("id = ? AND tenant_id = ?", user.Id, group.TenantID).First(&model.User{}).Error
			switch {
			case err == gorm.ErrRecordNotFound:
				return nil, errors.InvalidParameterValue(fmt.Sprintf("users[%d].id", idx), user.Id, "user not found")

			case err != nil:
				return nil, errors.UnusableDatabase(err)
			}
		}
	}

	tenant, err := getTenant(db, tenantID)
	if err != nil {
		return nil, err
	}

	return tenant, nil
}

func setGroupUsers(ctx context.Context, db *gorm.DB, tenantID uint64, req *identity.SetGroupUsersRequest) ([]*identity.User, error) {
	var err error
	var tenant *identity.Tenant
	if tenant, err = validateGroupUserSet(ctx, db, tenantID, req); err != nil {
		return nil, err
	}

	err = db.Where("group_id = ?", req.GroupId).Delete(&model.UserGroup{}).Error
	if err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	var resUsers []*identity.User
	for _, u := range req.Users {
		g := model.UserGroup{
			UserID:  u.Id,
			GroupID: req.GroupId,
		}
		if err = db.Save(&g).Error; err != nil {
			return nil, errors.UnusableDatabase(err)
		}

		var user = &model.User{ID: u.Id}
		err = db.First(user).Error
		switch {
		case err != nil:
			return nil, errors.UnusableDatabase(err)
		}

		resUser, err := userModelToRsp(user)
		if err != nil {
			return nil, errors.Unknown(err)
		}
		resUser.Tenant = tenant
		resUsers = append(resUsers, resUser)
	}
	return resUsers, nil
}
