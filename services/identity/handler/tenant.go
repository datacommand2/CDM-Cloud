package handler

import (
	"fmt"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	identity "github.com/datacommand2/cdm-cloud/services/identity/proto"
	"github.com/jinzhu/gorm"
	"unicode/utf8"
)

func validateTenantListGet(name string) error {
	if utf8.RuneCountInString(name) > nameLength {
		return errors.LengthOverflowParameterValue("name", name, nameLength)
	}
	return nil
}

// getTenants 함수는 테넌트 목록을 조회하는 함수이다.
// 사용 권한: 최고 관리자
func getTenants(db *gorm.DB, filters ...tenantFilter) ([]*identity.Tenant, error) {
	var err error
	var conditions = db

	for _, f := range filters {
		conditions, err = f.Apply(conditions)
		if err != nil {
			return nil, errors.UnusableDatabase(err)
		}
	}

	var tenants []model.Tenant
	if err = conditions.Find(&tenants).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	var resTenants []*identity.Tenant
	var tenant *identity.Tenant
	for _, t := range tenants {
		tenant, err = tenantModelToRsp(&t)
		if err != nil {
			return nil, errors.Unknown(err)
		}
		resTenants = append(resTenants, tenant)
	}

	return resTenants, nil
}

// getTenant 함수는 테넌트를 상세조회하는 함수이다.
// 사용 권한: 최고 관리자
func getTenant(db *gorm.DB, id uint64) (*identity.Tenant, error) {
	tenant := model.Tenant{}
	err := db.First(&tenant, id).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, notFoundTenant(id)
	case err != nil:
		return nil, errors.UnusableDatabase(err)
	}

	rspTenant, err := tenantModelToRsp(&tenant)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	var solutions []model.TenantSolution
	if err := db.Where("tenant_id = ?", tenant.ID).Find(&solutions).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	var solution *identity.Solution
	for _, s := range solutions {
		solution, err = solutionModelToRsp(&s)
		if err != nil {
			return nil, errors.Unknown(err)
		}
		rspTenant.Solutions = append(rspTenant.Solutions, solution)
	}

	return rspTenant, nil
}

func validateTenantAdd(t *identity.AddTenantRequest) error {
	if t.Tenant == nil {
		return errors.RequiredParameter("tenant")
	}

	if t.Tenant.Id != 0 {
		return errors.InvalidParameterValue("tenant.id", t.Tenant.Id, "unusable parameter")
	}

	if len(t.Tenant.Name) == 0 {
		return errors.RequiredParameter("tenant.name")
	}

	if utf8.RuneCountInString(t.Tenant.Name) > nameLength {
		return errors.LengthOverflowParameterValue("tenant.name", t.Tenant.Name, nameLength)
	}

	if t.Tenant.UseFlag == nil {
		return errors.RequiredParameter("tenant.use_flag")
	}

	if len(t.Tenant.Solutions) == 0 {
		return errors.RequiredParameter("tenant.solutions")
	}

	for idx, solutionTenant := range t.Tenant.Solutions {
		if len(solutionTenant.Solution) == 0 {
			return errors.RequiredParameter(fmt.Sprintf("tenant.solutions[%d].solution", idx))
		}

		if utf8.RuneCountInString(solutionTenant.Solution) > solutionLength {
			return errors.LengthOverflowParameterValue(fmt.Sprintf("tenant.solutions[%d].solution", idx), solutionTenant.Solution, solutionLength)
		}
	}

	if utf8.RuneCountInString(t.Tenant.Remarks) > remarksLength {
		return errors.LengthOverflowParameterValue("tenant.remarks", t.Tenant.Remarks, remarksLength)
	}

	return nil
}

// addTenant 함수는 테넌트를 추가하는 함수이다.
// 사용 권한: 최고 관리자
func addTenant(db *gorm.DB, t *identity.AddTenantRequest) (*identity.Tenant, error) {
	err := validateTenantAdd(t)
	if err != nil {
		return nil, err
	}

	tenant, err := tenantRspToModel(t.Tenant)
	if err != nil {
		return nil, errors.Unknown(err)
	}
	if err := db.Save(tenant).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	for _, solutionTenant := range t.Tenant.Solutions {
		if err := db.Save(&model.TenantSolution{TenantID: tenant.ID, Solution: solutionTenant.Solution}).Error; err != nil {
			return nil, errors.UnusableDatabase(err)
		}
	}

	rspTenant, err := tenantModelToRsp(tenant)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	//store TenantSolution
	var solutions []model.TenantSolution
	if err := db.Where("tenant_id = ?", tenant.ID).Find(&solutions).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	var solution *identity.Solution
	for _, s := range solutions {
		solution, err = solutionModelToRsp(&s)
		if err != nil {
			return nil, errors.Unknown(err)
		}
		rspTenant.Solutions = append(rspTenant.Solutions, solution)
	}

	return rspTenant, nil
}

func validateTenantUpdate(t *identity.UpdateTenantRequest) error {
	if t.Tenant == nil {
		return errors.RequiredParameter("tenant")
	}

	if t.TenantId != t.Tenant.Id {
		return errors.UnchangeableParameter("tenant.id")
	}

	if len(t.Tenant.Name) == 0 {
		return errors.RequiredParameter("tenant.name")
	}

	if utf8.RuneCountInString(t.Tenant.Name) > nameLength {
		return errors.LengthOverflowParameterValue("tenant.name", t.Tenant.Name, nameLength)
	}

	if len(t.Tenant.Solutions) == 0 {
		return errors.RequiredParameter("tenant.solutions")
	}

	for idx, solutionTenant := range t.Tenant.Solutions {
		if len(solutionTenant.Solution) == 0 {
			return errors.RequiredParameter(fmt.Sprintf("tenant.solutions[%d].solution", idx))
		}

		if utf8.RuneCountInString(solutionTenant.Solution) > solutionLength {
			return errors.LengthOverflowParameterValue(fmt.Sprintf("tenant.solutions[%d].solution", idx), solutionTenant.Solution, solutionLength)
		}
	}

	if utf8.RuneCountInString(t.Tenant.Remarks) > remarksLength {
		return errors.LengthOverflowParameterValue("tenant.remarks", t.Tenant.Remarks, remarksLength)
	}

	return nil
}

// updateTenant 함수는 테넌트 정보를 수정하는 함수이다.
// 사용 권한: 최고 관리자
func updateTenant(db *gorm.DB, id uint64, t *identity.UpdateTenantRequest) (*identity.Tenant, error) {
	err := validateTenantUpdate(t)
	if err != nil {
		return nil, err
	}

	tenant := model.Tenant{}
	err = db.First(&tenant, id).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, notFoundTenant(id)
	case err != nil:
		return nil, errors.UnusableDatabase(err)
	}

	tenant.Name = t.Tenant.Name
	if tenant.Remarks == nil {
		tenant.Remarks = new(string)
	}
	*tenant.Remarks = t.Tenant.Remarks

	if err := db.Save(&tenant).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	//update TenantSolution
	if err := db.Where("tenant_id = ?", tenant.ID).Delete(&model.TenantSolution{}).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	for _, solutionTenant := range t.Tenant.Solutions {
		if err := db.Save(&model.TenantSolution{TenantID: tenant.ID, Solution: solutionTenant.Solution}).Error; err != nil {
			return nil, errors.UnusableDatabase(err)
		}
	}

	//response tenant information
	rspTenant, err := tenantModelToRsp(&tenant)
	if err != nil {
		return nil, errors.Unknown(err)
	}
	var solutions []model.TenantSolution
	if err := db.Where("tenant_id = ?", tenant.ID).Find(&solutions).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	var solution *identity.Solution
	for _, s := range solutions {
		solution, err = solutionModelToRsp(&s)
		if err != nil {
			return nil, errors.Unknown(err)
		}
		rspTenant.Solutions = append(rspTenant.Solutions, solution)
	}

	return rspTenant, nil
}

// activateTenant 함수는 활성화 여부를 수정하는 함수이다.
// 사용 권한: 최고 관리자
func activateTenant(db *gorm.DB, id uint64) (*identity.Tenant, error) {
	tenant := model.Tenant{}
	err := db.First(&tenant, id).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, notFoundTenant(id)
	case err != nil:
		return nil, errors.UnusableDatabase(err)
	}

	tenant.UseFlag = true

	if err := db.Save(&tenant).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	t, err := tenantModelToRsp(&tenant)
	if err != nil {
		return nil, errors.Unknown(err)
	}
	return t, nil
}

// deactivateTenant 함수는 활성화 여부를 수정하는 함수이다.
// 사용 권한: 최고 관리자
func deactivateTenant(db *gorm.DB, id uint64) (*identity.Tenant, error) {
	tenant := model.Tenant{}
	err := db.First(&tenant, id).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, notFoundTenant(id)
	case err != nil:
		return nil, errors.UnusableDatabase(err)
	}

	tenant.UseFlag = false

	if err := db.Save(&tenant).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	t, err := tenantModelToRsp(&tenant)
	if err != nil {
		return nil, errors.Unknown(err)
	}
	return t, nil
}
