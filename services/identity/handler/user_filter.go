package handler

import (
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/store"

	"github.com/jinzhu/gorm"
	"strconv"
	"strings"
)

// userFilter 는 유저 목록 검색에 필터를 적용하기위한 인터페이스이다.
type userFilter interface {
	Apply(*gorm.DB) (*gorm.DB, error)
}

// userSolutionFilter 는 솔루션 필터를 위해 솔루션 정보를 전달하는 자료구조이다.
type userSolutionFilter struct {
	DB       *gorm.DB
	Solution string
	Role     string
}

// Apply 유저목록 검색에 솔루션 필터를 적용하기위한 함수이다.
func (f *userSolutionFilter) Apply(db *gorm.DB) (*gorm.DB, error) {
	c := f.DB.Where("solution = ?", f.Solution)
	if len(f.Role) != 0 {
		c = c.Where("role = ?", f.Role)
	}

	var roles []model.Role
	err := c.Select("id").Find(&roles).Error
	switch {
	case err != nil:
		return nil, err

	case len(roles) == 0:
		return db.Where("1<>1"), nil
	}

	var rids []uint64
	for _, r := range roles {
		rids = append(rids, r.ID)
	}

	var userRoles []model.UserRole
	if err := f.DB.Select("DISTINCT user_id").Where("role_id in (?)", rids).Find(&userRoles).Error; err != nil {
		return nil, err
	} else if len(userRoles) == 0 {
		return db.Where("1<>1"), nil
	}

	var uids []uint64
	for _, ur := range userRoles {
		uids = append(uids, ur.UserID)
	}

	return db.Where("id in (?)", uids), nil
}

// userExcludeGroupFilter 는 그룹 필터를 위해 Exclude그룹 정보를 전달하는 자료구조이다.
type userExcludeGroupFilter struct {
	DB             *gorm.DB
	ExcludeGroupID uint64
}

// Apply 유저목록 검색에 Exclude그룹 필터를 적용하기 위한 함수이다.
func (f *userExcludeGroupFilter) Apply(db *gorm.DB) (*gorm.DB, error) {
	var userGroups []model.UserGroup

	err := f.DB.Where("group_id = ?", f.ExcludeGroupID).Select("user_id").Find(&userGroups).Error
	switch {
	case err != nil:
		return nil, err

	case len(userGroups) == 0:
		return db.Where("1=1"), nil
	}

	var uids []uint64
	for _, ug := range userGroups {
		uids = append(uids, ug.UserID)
	}

	return db.Where("id not in (?)", uids), nil
}

// userGroupFilter 는 그룹 필터를 위해 그룹 정보를 전달하는 자료구조이다.
type userGroupFilter struct {
	DB      *gorm.DB
	GroupID uint64
}

// Apply 유저목록 검색에 그룹 필터를 적용하기위한 함수이다.
func (f *userGroupFilter) Apply(db *gorm.DB) (*gorm.DB, error) {
	var userGroups []model.UserGroup

	err := f.DB.Where("group_id = ?", f.GroupID).Select("user_id").Find(&userGroups).Error
	switch {
	case err != nil:
		return nil, err

	case len(userGroups) == 0:
		return db.Where("1<>1"), nil
	}

	var uids []uint64
	for _, ug := range userGroups {
		uids = append(uids, ug.UserID)
	}

	return db.Where("id in (?)", uids), nil
}

// userNameFilter 는 유저이름 필터를 위해 유저이름 정보를 전달하는 자료구조이다.
type userNameFilter struct {
	Name string
}

// Apply 는 유저목록 검색에 이름 필터를 적용하기위한 함수이다.
func (f *userNameFilter) Apply(db *gorm.DB) (*gorm.DB, error) {
	return db.Where("name LIKE ?", "%"+f.Name+"%"), nil
}

// userDepartmentFilter 는 부서 필터를 위해 부서 정보를 전달하는 자료구조이다.
type userDepartmentFilter struct {
	Department string
}

// Apply 는 유저목록 검색에 솔루션 필터를 적용하기위한 함수이다.
func (f *userDepartmentFilter) Apply(db *gorm.DB) (*gorm.DB, error) {
	return db.Where("department LIKE ?", "%"+f.Department+"%"), nil
}

// userPositionFilter 는 직책 필터를 위해 직책 정보를 전달하는 자료구조이다.
type userPositionFilter struct {
	Position string
}

// Apply 는 유저목록 검색에 직책 필터를 적용하기위한 함수이다.
func (f *userPositionFilter) Apply(db *gorm.DB) (*gorm.DB, error) {
	return db.Where("position LIKE ?", "%"+f.Position+"%"), nil
}

// paginationFilter 는 페지이 정보를 전달하는 자료구조이다.
type paginationFilter struct {
	Offset uint64
	Limit  uint64
}

// Apply 는 유저목록 검색에 페이지 정보를 적용하기위한 함수이다.
func (f *paginationFilter) Apply(db *gorm.DB) (*gorm.DB, error) {
	return db.Offset(f.Offset).Limit(f.Limit), nil
}

// logFilter 는 로그인 필터를 위한 자료 구조이다.
type loginFilter struct {
}

// Apply 는 유저목록 검색에 로그인 정보를 적용하기위한 함수이다.
func (f *loginFilter) Apply(db *gorm.DB) (*gorm.DB, error) {
	storeKey, err := store.List(storeKeyPrefix)
	switch {
	case err == store.ErrNotFoundKey:
		return db.Where("1<>1"), nil

	case err != nil:
		return nil, errors.UnusableStore(err)
	}

	var uids []uint64
	for _, k := range storeKey {
		id, err := strconv.Atoi(strings.TrimPrefix(k, storeKeyPrefix+"."))
		if err != nil {
			continue
		}
		uids = append(uids, uint64(id))
	}

	return db.Where("id in (?)", uids), nil
}
