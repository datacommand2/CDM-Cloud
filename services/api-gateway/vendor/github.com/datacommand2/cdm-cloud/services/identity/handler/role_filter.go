package handler

import (
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/jinzhu/gorm"
)

// roleFilter 는 그룹 목록 검색에 필터를 적용하기위한 인터페이스이다
type roleFilter interface {
	Apply(*gorm.DB) (*gorm.DB, error)
}

// tenantSolutionFilter 는 역할 목록의 테넌트 솔루션 필터 자료구조이다.
type tenantSolutionFilter struct {
	DB       *gorm.DB
	TenantID uint64
}

// Apply 역할 목록 검색에 테넌트 솔루션 필터를 적용하는 함수이다.
func (f *tenantSolutionFilter) Apply(db *gorm.DB) (*gorm.DB, error) {
	var tenantSolution []model.TenantSolution
	if err := f.DB.Where("tenant_id = ?", f.TenantID).Find(&tenantSolution).Error; err != nil {
		return nil, err
	}

	var solutions []string
	for _, ts := range tenantSolution {
		solutions = append(solutions, ts.Solution)
	}

	return db.Where("solution in (?)", solutions), nil
}

// roleNameFilter 는 그룹 이름 필터를 위해 문자열을 전달하는 자료구조이다
type roleNameFilter struct {
	Role string
}

// Apply 그룹 목록 검색에 이름 문자열을 적용하기위한 함수이다
func (f *roleNameFilter) Apply(db *gorm.DB) (*gorm.DB, error) {
	return db.Where("role LIKE ?", "%"+f.Role+"%"), nil
}

// solutionNameFilter 는 그룹 이름 필터를 위해 문자열을 전달하는 자료구조이다
type solutionNameFilter struct {
	Solution string
}

// Apply 그룹 목록 검색에 이름 문자열을 적용하기위한 함수이다
func (f *solutionNameFilter) Apply(db *gorm.DB) (*gorm.DB, error) {
	return db.Where("solution LIKE ?", "%"+f.Solution+"%"), nil
}
