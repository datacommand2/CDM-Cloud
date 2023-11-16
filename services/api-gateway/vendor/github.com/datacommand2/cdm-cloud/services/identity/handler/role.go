package handler

import (
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	identity "github.com/datacommand2/cdm-cloud/services/identity/proto"
	"github.com/jinzhu/gorm"
	"unicode/utf8"
)

var (
	roleBoundary = stringBoundary{Enum: []string{"manager", "operator", "viewer"}}
)

func validateRoleList(req *identity.GetRolesRequest) error {
	if utf8.RuneCountInString(req.Solution) > solutionLength {
		return errors.LengthOverflowParameterValue("solution", req.Solution, solutionLength)
	}

	if len(req.Role) != 0 && !roleBoundary.enum(req.Role) {
		return errors.UnavailableParameterValue("role", req.Role, []interface{}{"manager", "operator", "viewer"})
	}

	return nil
}

// getRoles 는 솔루션 역할의 목록을 조회하기위한 함수이다
func getRoles(db *gorm.DB, filters ...roleFilter) ([]*identity.Role, error) {
	var err error
	var conditions = db

	for _, f := range filters {
		conditions, err = f.Apply(conditions)
		if err != nil {
			return nil, errors.UnusableDatabase(err)
		}
	}

	var roles []model.Role
	if err = conditions.Find(&roles).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	var resRoles []*identity.Role
	var role *identity.Role
	for _, r := range roles {
		role, err = roleModelToRsp(&r)
		if err != nil {
			return nil, errors.Unknown(err)
		}
		resRoles = append(resRoles, role)
	}

	return resRoles, nil
}
