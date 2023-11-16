package handler

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	random "math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/datacommand2/cdm-cloud/common/config"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/database"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	commonmeta "github.com/datacommand2/cdm-cloud/common/metadata"
	"github.com/datacommand2/cdm-cloud/common/store"
	"github.com/datacommand2/cdm-cloud/common/test/helper"
	identity "github.com/datacommand2/cdm-cloud/services/identity/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/metadata"
	"github.com/stretchr/testify/assert"
)

var (
	handler *IdentityHandler

	cloudAdminEndpoint           = "cloud.admin"
	cloudAdminAndManagerEndPoint = "cloud.admin_manager"
	cloudAllEndpoint             = "cloud.all"
)

func fatalIfError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func setCtx(reqUser *identity.User, tenantID uint64) context.Context {
	ctx := context.Background()
	ctx = metadata.Set(ctx, commonmeta.HeaderClientIP, "192.168.1.1")
	b, _ := json.Marshal(reqUser)
	ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))
	ctx = metadata.Set(ctx, commonmeta.HeaderTenantID, strconv.Itoa(int(tenantID)))
	return ctx
}

func setup() {
	var err error

	_, _ = exec.Command("mkdir", defaultPrivateKeyPath).Output()
	_, _ = exec.Command("openssl", "genrsa", "-out", defaultPrivateKeyPath+"identity.pem", "2048").Output()

	if err = helper.Init(); err != nil {
		panic(err)
	}

	database.Test(func(db *gorm.DB) {
		if handler, err = NewIdentityHandler(); err != nil {
			panic(err)
		}
	})

}

func teardown() {
	helper.Close()
}

func TestMain(m *testing.M) {
	setup()
	defer teardown()

	if code := m.Run(); code != 0 {
		os.Exit(code)
	}
}

func TestNewIdentityHandler(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		h, err := NewIdentityHandler()
		assert.NoError(t, err)
		assert.NotNil(t, h)
		assert.NoError(t, h.Close())
	})
}

func TestGetUser(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var defaultTenant model.Tenant
		var admin model.User
		fatalIfError(t, db.First(&defaultTenant, model.Tenant{Name: "default"}).Error)
		fatalIfError(t, db.First(&admin, model.User{Account: "admin"}).Error)

		user := model.User{
			Account:           uuid.New().String()[:30],
			Name:              uuid.New().String()[:30],
			Password:          uuid.New().String(),
			Department:        new(string),
			Position:          new(string),
			Email:             new(string),
			Contact:           new(string),
			Timezone:          new(string),
			LanguageSet:       new(string),
			LastLoggedInAt:    new(int64),
			PasswordUpdatedAt: new(int64),
			TenantID:          defaultTenant.ID,
		}
		*user.Department = uuid.New().String()[:30]
		*user.Position = uuid.New().String()[:30]
		*user.Email = uuid.New().String()[:30]
		*user.Contact = uuid.New().String()[:20]
		*user.Timezone = uuid.New().String()[:30]
		*user.LanguageSet = uuid.New().String()[:30]
		*user.LastLoggedInAt = time.Now().Unix()
		*user.PasswordUpdatedAt = time.Now().Unix()

		group := model.Group{
			Name:     uuid.New().String()[:30],
			Remarks:  new(string),
			TenantID: defaultTenant.ID,
		}
		*group.Remarks = uuid.New().String()[:30]

		role := model.Role{
			Solution: uuid.New().String()[:30],
			Role:     uuid.New().String()[:30],
		}
		if err := db.Save(&user).Error; err != nil {
			panic(err)
		}
		if err := db.Save(&group).Error; err != nil {
			panic(err)
		}
		if err := db.Save(&role).Error; err != nil {
			panic(err)
		}
		if err := db.Save(&model.UserGroup{UserID: user.ID, GroupID: group.ID}).Error; err != nil {
			panic(err)
		}
		if err := db.Save(&model.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
			panic(err)
		}

		for _, tc := range []struct {
			Desc  string
			User  model.User
			Error error
		}{
			{
				Desc:  "normal case: user",
				User:  user,
				Error: nil,
			},
			{
				Desc:  "normal case: admin",
				User:  admin,
				Error: nil,
			},
			{
				Desc:  "abnormal case: unknown user id",
				User:  model.User{ID: uint64(999999999)},
				Error: errors.NotFound(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			reqUser := identity.User{
				Account: constant.Admin,
				Name:    constant.Admin,
				Tenant:  &identity.Tenant{Id: defaultTenant.ID},
			}
			tmp := Role[constant.Admin]
			r, _ := roleModelToRsp(&tmp)
			reqUser.Roles = append(reqUser.Roles, r)

			ctx := setCtx(&reqUser, defaultTenant.ID)

			var rsp = new(identity.UserResponse)

			err := handler.GetUser(ctx, &identity.GetUserRequest{UserId: tc.User.ID}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.User.ID, rsp.User.Id, tc.Desc)
				assert.Equal(t, defaultTenant.ID, rsp.User.Tenant.Id)
				assert.Equal(t, defaultTenant.Name, rsp.User.Tenant.Name)
				assert.Equal(t, tc.User.Account, rsp.User.Account, tc.Desc)
				assert.Equal(t, tc.User.Name, rsp.User.Name, tc.Desc)
				if tc.User.Account != "admin" {
					assert.Equal(t, *tc.User.Department, rsp.User.Department, tc.Desc)
					assert.Equal(t, *tc.User.Position, rsp.User.Position, tc.Desc)
					assert.Equal(t, *tc.User.Email, rsp.User.Email, tc.Desc)
					assert.Equal(t, *tc.User.Contact, rsp.User.Contact, tc.Desc)
					assert.Equal(t, *tc.User.Timezone, rsp.User.Timezone, tc.Desc)
					assert.Equal(t, *tc.User.LanguageSet, rsp.User.LanguageSet, tc.Desc)
					assert.Equal(t, *tc.User.LastLoggedInAt, rsp.User.LastLoggedInAt, tc.Desc)
					assert.Equal(t, *tc.User.PasswordUpdatedAt, rsp.User.PasswordUpdatedAt, tc.Desc)
				}
				assert.Equal(t, tc.User.CreatedAt, rsp.User.CreatedAt, tc.Desc)
				assert.Equal(t, tc.User.UpdatedAt, rsp.User.UpdatedAt, tc.Desc)

				if tc.User.Account != "admin" {
					assert.Len(t, rsp.User.Groups, 1, tc.Desc)
					assert.Equal(t, group.ID, rsp.User.Groups[0].Id, tc.Desc)
					assert.Equal(t, group.Name, rsp.User.Groups[0].Name, tc.Desc)
					assert.Equal(t, *group.Remarks, rsp.User.Groups[0].Remarks, tc.Desc)

					assert.Len(t, rsp.User.Roles, 1, tc.Desc)
					assert.Equal(t, role.ID, rsp.User.Roles[0].Id, tc.Desc)
					assert.Equal(t, role.Solution, rsp.User.Roles[0].Solution, tc.Desc)
					assert.Equal(t, role.Role, rsp.User.Roles[0].Role, tc.Desc)
				} else {
					assert.Len(t, rsp.User.Groups, 2, tc.Desc)
					assert.Len(t, rsp.User.Roles, 1, tc.Desc)
				}
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetSimpleUser(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var defaultTenant model.Tenant
		var admin model.User
		fatalIfError(t, db.First(&defaultTenant, model.Tenant{Name: "default"}).Error)
		fatalIfError(t, db.First(&admin, model.User{Account: "admin"}).Error)

		user := model.User{
			Account:           uuid.New().String()[:30],
			Name:              uuid.New().String()[:30],
			Password:          uuid.New().String(),
			Department:        new(string),
			Position:          new(string),
			Email:             new(string),
			Contact:           new(string),
			Timezone:          new(string),
			LanguageSet:       new(string),
			LastLoggedInAt:    new(int64),
			PasswordUpdatedAt: new(int64),
			TenantID:          defaultTenant.ID,
		}
		*user.Department = uuid.New().String()[:30]
		*user.Position = uuid.New().String()[:30]
		*user.Email = uuid.New().String()[:30]
		*user.Contact = uuid.New().String()[:20]
		*user.Timezone = uuid.New().String()[:30]
		*user.LanguageSet = uuid.New().String()[:30]
		*user.LastLoggedInAt = time.Now().Unix()
		*user.PasswordUpdatedAt = time.Now().Unix()

		group := model.Group{
			Name:     uuid.New().String()[:30],
			Remarks:  new(string),
			TenantID: defaultTenant.ID,
		}
		*group.Remarks = uuid.New().String()[:30]

		role := model.Role{
			Solution: uuid.New().String()[:30],
			Role:     constant.Admin,
		}
		if err := db.Save(&user).Error; err != nil {
			panic(err)
		}
		if err := db.Save(&group).Error; err != nil {
			panic(err)
		}
		if err := db.Save(&role).Error; err != nil {
			panic(err)
		}
		if err := db.Save(&model.UserGroup{UserID: user.ID, GroupID: group.ID}).Error; err != nil {
			panic(err)
		}
		if err := db.Save(&model.UserRole{UserID: user.ID, RoleID: role.ID}).Error; err != nil {
			panic(err)
		}

		for _, tc := range []struct {
			Desc  string
			User  model.User
			Error error
		}{
			{
				Desc:  "normal case: user",
				User:  user,
				Error: nil,
			},
			{
				Desc:  "normal case: admin",
				User:  admin,
				Error: nil,
			},
			{
				Desc:  "abnormal case: unknown user id",
				User:  model.User{ID: uint64(999999999)},
				Error: errors.NotFound(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			reqUser := identity.User{
				Account: constant.User,
				Name:    constant.User,
				Tenant:  &identity.Tenant{Id: defaultTenant.ID},
			}
			tmp := Role[constant.User]
			r, _ := roleModelToRsp(&tmp)
			reqUser.Roles = append(reqUser.Roles, r)

			ctx := setCtx(&reqUser, defaultTenant.ID)

			var rsp = new(identity.SimpleUserResponse)

			err := handler.GetSimpleUser(ctx, &identity.GetSimpleUserRequest{UserId: tc.User.ID}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.User.ID, rsp.SimpleUser.Id, tc.Desc)
				assert.Equal(t, defaultTenant.ID, rsp.SimpleUser.Tenant.Id)
				assert.Equal(t, defaultTenant.Name, rsp.SimpleUser.Tenant.Name)
				assert.Equal(t, tc.User.Account, rsp.SimpleUser.Account, tc.Desc)
				assert.Equal(t, tc.User.Name, rsp.SimpleUser.Name, tc.Desc)

				if tc.User.Account != "admin" {
					assert.Equal(t, *tc.User.Department, rsp.SimpleUser.Department, tc.Desc)
					assert.Equal(t, *tc.User.Position, rsp.SimpleUser.Position, tc.Desc)
					assert.Equal(t, *tc.User.Email, rsp.SimpleUser.Email, tc.Desc)
					assert.Equal(t, *tc.User.Contact, rsp.SimpleUser.Contact, tc.Desc)
				}

				if tc.User.Account != "admin" {
					assert.Len(t, rsp.SimpleUser.Groups, 1, tc.Desc)
					assert.Equal(t, group.ID, rsp.SimpleUser.Groups[0].Id, tc.Desc)
					assert.Equal(t, group.Name, rsp.SimpleUser.Groups[0].Name, tc.Desc)
					assert.Equal(t, *group.Remarks, rsp.SimpleUser.Groups[0].Remarks, tc.Desc)

					assert.Len(t, rsp.SimpleUser.Roles, 1, tc.Desc)
					assert.Equal(t, role.ID, rsp.SimpleUser.Roles[0].Id, tc.Desc)
					assert.Equal(t, role.Solution, rsp.SimpleUser.Roles[0].Solution, tc.Desc)
					assert.Equal(t, role.Role, rsp.SimpleUser.Roles[0].Role, tc.Desc)
				} else {
					assert.Len(t, rsp.SimpleUser.Groups, 2, tc.Desc)
					assert.Len(t, rsp.SimpleUser.Roles, 1, tc.Desc)
				}
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetUserAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}
		user := model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			Password: uuid.New().String(),
			TenantID: tenant.ID,
		}
		if err := db.Save(&user).Error; err != nil {
			panic(err)
		}

		var tmp model.Role
		var r *identity.Role
		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ = roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		authManager := identity.User{
			Tenant: &identity.Tenant{Id: tenant.ID},
		}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		authManager.Roles = append(authManager.Roles, r)

		unauthManager := identity.User{}
		unauthManager.Roles = append(unauthManager.Roles, &identity.Role{Solution: "unknwon", Role: constant.Manager})

		authUser := identity.User{Id: user.ID}
		authUser.Roles = append(authUser.Roles, &identity.Role{Solution: "unknwon", Role: ""})

		unauthUser := identity.User{Id: 999999999}
		unauthUser.Roles = append(authUser.Roles, &identity.Role{Solution: "unknwon", Role: ""})

		for _, tc := range []struct {
			Desc    string
			ID      uint64
			reqUser *identity.User
			Error   error
		}{
			{
				Desc:    "normal case: get request by admin",
				ID:      user.ID,
				reqUser: &admin,
				Error:   nil,
			},
			{
				Desc:    "normal case: get request by auth manager",
				ID:      user.ID,
				reqUser: &authManager,
				Error:   nil,
			},
			{
				Desc:    "normal case: get request by auth user",
				ID:      user.ID,
				reqUser: &authUser,
				Error:   nil,
			},
			{
				Desc:    "abnormal case: get request by unauth manager",
				ID:      user.ID,
				reqUser: &unauthManager,
				Error:   errors.New(constant.ServiceIdentity, "unauthorized user", 403),
			},
			{
				Desc:    "normal case: get request by unauth user",
				ID:      user.ID,
				reqUser: &unauthUser,
				Error:   errors.New(constant.ServiceIdentity, "unauthorized user", 403),
			},
		} {
			ctx := setCtx(tc.reqUser, tenant.ID)

			var rsp = new(identity.UserResponse)

			err := handler.GetUser(ctx, &identity.GetUserRequest{UserId: tc.ID}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, user.ID, rsp.User.Id, tc.Desc)
				assert.Equal(t, tenant.ID, rsp.User.Tenant.Id)
				assert.Equal(t, tenant.Name, rsp.User.Tenant.Name)
				assert.Equal(t, user.Account, rsp.User.Account, tc.Desc)
				assert.Equal(t, user.Name, rsp.User.Name, tc.Desc)

			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestAddUser(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var users []*model.User
		var groups []*model.Group
		var roles []*model.Role
		var tenants []*model.Tenant

		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})
		for _, tenant := range tenants {
			if err := db.Save(&tenant).Error; err != nil {
				panic(err)
			}
		}

		groups = append(groups, &model.Group{
			Name:     uuid.New().String()[:30],
			TenantID: tenants[0].ID,
		})
		groups = append(groups, &model.Group{
			Name:     uuid.New().String()[:30],
			TenantID: tenants[0].ID,
		})

		roles = append(roles, &model.Role{
			Solution: uuid.New().String()[:30],
			Role:     uuid.New().String()[:30],
		})
		roles = append(roles, &model.Role{
			Solution: uuid.New().String()[:30],
			Role:     uuid.New().String()[:30],
		})

		for _, g := range groups {
			if err := db.Save(g).Error; err != nil {
				panic(err)
			}
		}
		for _, r := range roles {
			if err := db.Save(r).Error; err != nil {
				panic(err)
			}
		}

		authManager := identity.User{
			Tenant: &identity.Tenant{Id: tenants[0].ID},
		}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		authManager.Roles = append(authManager.Roles, r)

		for _, tc := range []struct {
			Desc        string
			Tenant      *identity.Tenant
			TenantName  string
			Account     string
			Name        string
			Department  string
			Position    string
			Email       string
			Contact     string
			Timezone    string
			LanguageSet string
			Roles       []uint64
			Groups      []uint64
			ReqUser     *identity.User
			Error       error
		}{
			{
				Desc:       "normal case 1",
				Account:    "account1",
				Email:      "test1@localhost",
				Name:       "tester1",
				Tenant:     &identity.Tenant{Id: tenants[0].ID},
				TenantName: tenants[0].Name,
				ReqUser:    &authManager,
				Error:      nil,
			},
			{
				Desc:        "normal case 2",
				Account:     "account2-2",
				Name:        "tester2-2",
				Department:  "department2-2",
				Position:    "Position2-2",
				Contact:     "222-2222",
				Timezone:    "Asia/Chita",
				LanguageSet: "kor",
				Roles:       []uint64{roles[0].ID, roles[1].ID},
				Groups:      []uint64{groups[0].ID, groups[1].ID},
				Tenant:      &identity.Tenant{Id: tenants[0].ID},
				TenantName:  tenants[0].Name,
				ReqUser:     &authManager,
				Error:       nil,
			},
			{
				Desc:    "abnormal case: conflict account",
				Account: "account1", //ID conflict
				Name:    "tester3",
				Tenant:  &identity.Tenant{Id: tenants[0].ID},
				ReqUser: &authManager,
				Error:   errors.Conflict(constant.ServiceIdentity, "conflict parameter"),
			},
			{
				Desc:    "abnormal case: empty account",
				Account: "", // ID empty
				Name:    "tester4",
				Tenant:  &identity.Tenant{Id: tenants[0].ID},
				ReqUser: &authManager,
				Error:   errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: length of account over max size",
				Account:    uuid.New().String()[:32],
				Name:       uuid.New().String()[:30],
				Tenant:     &identity.Tenant{Id: tenants[0].ID},
				TenantName: tenants[0].Name,
				ReqUser:    &authManager,
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:    "abnormal case: invalid email parameter",
				Account: "account1222222",
				Name:    "tester32",
				Email:   "test3@daq21",
				Tenant:  &identity.Tenant{Id: tenants[0].ID},
				ReqUser: &authManager,
				Error:   errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:    "abnormal case: conflict email",
				Account: "account5",
				Name:    "tester5",
				Email:   "test1@localhost",
				Tenant:  &identity.Tenant{Id: tenants[0].ID},
				ReqUser: &authManager,
				Error:   errors.Conflict(constant.ServiceIdentity, "conflict parameter"),
			},
			{
				Desc:    "abnormal case: empty name",
				Account: "account6",
				Name:    "", //name tmpty
				Tenant:  &identity.Tenant{Id: tenants[0].ID},
				ReqUser: &authManager,
				Error:   errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: length of name over max size",
				Account:    uuid.New().String()[:30],
				Name:       generateString(nameLength + 1),
				Tenant:     &identity.Tenant{Id: tenants[0].ID},
				TenantName: tenants[0].Name,
				ReqUser:    &authManager,
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:    "abnormal case: unknown role id",
				Account: "account7",
				Name:    "tester7",
				Tenant:  &identity.Tenant{Id: tenants[0].ID},
				Roles:   []uint64{77777777}, //Unknown role id
				ReqUser: &authManager,
				Error:   errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:    "abnormal case: unknown role id",
				Account: "account7-1",
				Name:    "tester7-1",
				Tenant:  &identity.Tenant{Id: tenants[0].ID},
				Roles:   []uint64{1}, //admin role id
				ReqUser: &authManager,
				Error:   errors.BadRequest(constant.ServiceIdentity, "unassignable role(admin)"),
			},
			{
				Desc:    "abnormal case: unknown group id",
				Account: "account8",
				Name:    "tester8",
				Tenant:  &identity.Tenant{Id: tenants[0].ID},
				Groups:  []uint64{888888888}, //Unknown group id
				ReqUser: &authManager,
				Error:   errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:    "abnormal case: unknown tenant id",
				Account: "account9",
				Name:    "tester9",
				Tenant:  &identity.Tenant{Id: 999999999}, //Unknown tenant id
				ReqUser: &authManager,
				Error:   errors.InternalServerError(constant.ServiceIdentity, "unknown error"),
			},
			{
				Desc:    "abnormal case: user tenant id and group tenant id mismatch",
				Account: "account9",
				Name:    "tester9",
				Groups:  []uint64{groups[0].ID},
				Tenant:  &identity.Tenant{Id: tenants[1].ID},
				ReqUser: &authManager,
				Error:   errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: length of department over max size",
				Account:    uuid.New().String()[:30],
				Name:       uuid.New().String()[:30],
				Tenant:     &identity.Tenant{Id: tenants[0].ID},
				TenantName: tenants[0].Name,
				Department: generateString(departmentLength + 1),
				ReqUser:    &authManager,
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: length of position over max size",
				Account:    uuid.New().String()[:30],
				Name:       uuid.New().String()[:30],
				Tenant:     &identity.Tenant{Id: tenants[0].ID},
				TenantName: tenants[0].Name,
				Position:   generateString(positionLength + 1),
				ReqUser:    &authManager,
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: length of contact over max size",
				Account:    uuid.New().String()[:30],
				Name:       uuid.New().String()[:30],
				Tenant:     &identity.Tenant{Id: tenants[0].ID},
				TenantName: tenants[0].Name,
				Contact:    uuid.New().String()[:30],
				ReqUser:    &authManager,
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:        "abnormal case: length of language set over max size",
				Account:     uuid.New().String()[:30],
				Name:        uuid.New().String()[:30],
				Tenant:      &identity.Tenant{Id: tenants[0].ID},
				TenantName:  tenants[0].Name,
				LanguageSet: uuid.New().String()[:30],
				ReqUser:     &authManager,
				Error:       errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:        "abnormal case: unknown language set",
				Account:     uuid.New().String()[:30],
				Name:        uuid.New().String()[:30],
				Tenant:      &identity.Tenant{Id: tenants[0].ID},
				TenantName:  tenants[0].Name,
				LanguageSet: "aaa",
				ReqUser:     &authManager,
				Error:       errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: length of timezone over max size",
				Account:    uuid.New().String()[:30],
				Name:       uuid.New().String()[:30],
				Tenant:     &identity.Tenant{Id: tenants[0].ID},
				TenantName: tenants[0].Name,
				Timezone:   generateString(timezoneLength + 1),
				ReqUser:    &authManager,
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			ctx := setCtx(tc.ReqUser, tc.Tenant.Id)

			var req = new(identity.AddUserRequest)
			req.User = new(identity.User)
			req.User.Tenant = tc.Tenant
			req.User.Account = tc.Account
			req.User.Name = tc.Name
			req.User.Department = tc.Department
			req.User.Position = tc.Position
			req.User.Email = tc.Email
			req.User.Contact = tc.Contact
			req.User.Timezone = tc.Timezone
			req.User.LanguageSet = tc.LanguageSet

			for _, RoleID := range tc.Roles {
				req.User.Roles = append(req.User.Roles, &identity.Role{Id: RoleID})
			}

			for _, GroupID := range tc.Groups {
				req.User.Groups = append(req.User.Groups, &identity.Group{Id: GroupID})
			}

			var rsp = new(identity.AddUserResponse)

			err := handler.AddUser(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.NotEqual(t, 0, rsp.User.Id, tc.Desc)
				assert.Equal(t, tc.Tenant.Id, rsp.User.Tenant.Id)
				assert.Equal(t, tc.TenantName, rsp.User.Tenant.Name)
				assert.Equal(t, tc.Account, rsp.User.Account, tc.Desc)
				assert.Equal(t, tc.Name, rsp.User.Name, tc.Desc)
				assert.Equal(t, tc.Department, rsp.User.Department, tc.Desc)
				assert.Equal(t, tc.Position, rsp.User.Position, tc.Desc)
				assert.Equal(t, tc.Email, rsp.User.Email, tc.Desc)
				assert.Equal(t, tc.Contact, rsp.User.Contact, tc.Desc)
				assert.Equal(t, tc.Timezone, rsp.User.Timezone, tc.Desc)
				assert.Equal(t, tc.LanguageSet, rsp.User.LanguageSet, tc.Desc)
				assert.NotEmpty(t, rsp.Password, tc.Desc)

				var roles []uint64
				for _, r := range rsp.User.Roles {
					roles = append(roles, r.Id)
				}

				assert.ElementsMatch(t, tc.Roles, roles, tc.Desc)

				var groups []uint64
				for _, g := range rsp.User.Groups {
					groups = append(groups, g.Id)
				}

				assert.ElementsMatch(t, tc.Groups, groups, tc.Desc)

			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}

			if err == nil && rsp.User != nil {
				users = append(users, &model.User{ID: rsp.User.Id})
			}
		}
	})
}

func TestAddUserNilPointer(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30]}

		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		authManager := identity.User{
			Tenant: &identity.Tenant{Id: tenant.ID},
		}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		authManager.Roles = append(authManager.Roles, r)

		ctx := setCtx(&authManager, tenant.ID)

		var req = new(identity.AddUserRequest)
		req.User = new(identity.User)
		req.User.Tenant = nil
		req.User.Account = "account"
		req.User.Name = "name"

		var rsp = new(identity.AddUserResponse)
		err := handler.AddUser(ctx, req, rsp)

		assert.Equal(t, int32(400), err.(*errors.Error).Code)
	})
}

func TestAddUserWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var users []*model.User
		var tenants []*model.Tenant

		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})
		for _, tenant := range tenants {
			if err := db.Save(&tenant).Error; err != nil {
				panic(err)
			}
		}
		var tmp model.Role

		admin := identity.User{
			Account: constant.Admin,
		}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		authManager := identity.User{Account: "authManager",
			Tenant: &identity.Tenant{Id: tenants[0].ID},
		}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		authManager.Roles = append(authManager.Roles, r)

		//unauthManager := identity.User{Account: "unauthManager", TenantId: tenants[1].ID}
		//unauthManager.Roles = append(unauthManager.Roles, &identity.Role{Solution: "unknwon", Role: Manager})

		//user := identity.User{Account: "user"}

		for _, tc := range []struct {
			Desc       string
			Tenant     *identity.Tenant
			TenantName string
			Account    string
			Name       string
			Email      string
			ReqUser    *identity.User
			Error      error
		}{
			{
				Desc:       "normal case 1: admin add user",
				Account:    "account1",
				Name:       "tester1",
				Email:      "test822@localhost",
				Tenant:     &identity.Tenant{Id: tenants[0].ID},
				TenantName: tenants[0].Name,
				ReqUser:    &admin,
				Error:      nil,
			},
			{
				Desc:       "normal case 2: auth manager add user",
				Account:    "account2",
				Name:       "tester2",
				Tenant:     &identity.Tenant{Id: tenants[0].ID},
				TenantName: tenants[0].Name,
				ReqUser:    &authManager,
				Error:      nil,
			},
		} {
			ctx := setCtx(tc.ReqUser, tc.Tenant.Id)

			var req = new(identity.AddUserRequest)
			req.User = new(identity.User)
			req.User.Tenant = tc.Tenant
			req.User.Account = tc.Account
			req.User.Name = tc.Name
			req.User.Email = tc.Email

			var rsp = new(identity.AddUserResponse)

			err := handler.AddUser(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.NotEqual(t, 0, rsp.User.Id, tc.Desc)
				assert.Equal(t, tc.Tenant.Id, rsp.User.Tenant.Id)
				assert.Equal(t, tc.TenantName, rsp.User.Tenant.Name)
				assert.Equal(t, tc.Account, rsp.User.Account, tc.Desc)
				assert.Equal(t, tc.Name, rsp.User.Name, tc.Desc)
				assert.Equal(t, tc.Email, rsp.User.Email, tc.Desc)
				assert.NotEmpty(t, rsp.Password, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}

			if err == nil && rsp.User != nil {
				users = append(users, &model.User{ID: rsp.User.Id})
			}
		}
	})
}

func TestAddUserWithId(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var users []*model.User

		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		user := model.User{Account: uuid.New().String()[:30], Name: uuid.New().String()[:30], TenantID: tenant.ID}
		if err := db.Save(&user).Error; err != nil {
			panic(err)
		}

		authManager := identity.User{Tenant: &identity.Tenant{Id: tenant.ID}}

		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		authManager.Roles = append(authManager.Roles, r)

		for _, tc := range []struct {
			Desc       string
			ID         uint64
			Tenant     *identity.Tenant
			TenantName string
			Account    string
			Name       string
			Email      string
			ReqUser    *identity.User
			Error      error
		}{
			{
				Desc:       "normal case 1",
				Account:    "account1",
				Email:      "test1@localhost",
				Name:       "tester1",
				Tenant:     &identity.Tenant{Id: tenant.ID},
				TenantName: tenant.Name,
				ReqUser:    &authManager,
				Error:      nil,
			},
			{
				Desc:    "abnormal case: group id (already exist)",
				Account: "account2",
				ID:      user.ID,
				Name:    "tester3",
				Tenant:  &identity.Tenant{Id: tenant.ID},
				ReqUser: &authManager,
				Error:   errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			ctx := setCtx(tc.ReqUser, tc.Tenant.Id)

			var req = new(identity.AddUserRequest)
			req.User = new(identity.User)
			req.User.Id = tc.ID
			req.User.Tenant = tc.Tenant
			req.User.Account = tc.Account
			req.User.Name = tc.Name
			req.User.Email = tc.Email

			var rsp = new(identity.AddUserResponse)

			err := handler.AddUser(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.NotEqual(t, 0, rsp.User.Id, tc.Desc)
				assert.Equal(t, tc.Tenant.Id, rsp.User.Tenant.Id)
				assert.Equal(t, tc.TenantName, rsp.User.Tenant.Name)
				assert.Equal(t, tc.Account, rsp.User.Account, tc.Desc)
				assert.Equal(t, tc.Name, rsp.User.Name, tc.Desc)
				assert.Equal(t, tc.Email, rsp.User.Email, tc.Desc)
				assert.NotEmpty(t, rsp.Password, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}

			if err == nil && rsp.User != nil {
				users = append(users, &model.User{ID: rsp.User.Id})
			}
		}
	})
}

func TestUpdateUser(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var users []*model.User
		var groups []*model.Group
		var roles []*model.Role
		var tenants []*model.Tenant

		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})

		for _, tenant := range tenants {
			if err := db.Save(&tenant).Error; err != nil {
				panic(err)
			}
		}

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenants[0].ID,
		})
		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenants[0].ID,
		})

		groups = append(groups, &model.Group{
			Name:     uuid.New().String()[:30],
			TenantID: tenants[0].ID,
		})
		groups = append(groups, &model.Group{
			Name:     uuid.New().String()[:30],
			TenantID: tenants[0].ID,
		})

		roles = append(roles, &model.Role{
			Solution: uuid.New().String()[:30],
			Role:     uuid.New().String()[:30],
		})
		roles = append(roles, &model.Role{
			Solution: uuid.New().String()[:30],
			Role:     uuid.New().String()[:30],
		})

		for _, u := range users {
			if err := db.Save(u).Error; err != nil {
				panic(err)
			}
		}
		for _, g := range groups {
			if err := db.Save(g).Error; err != nil {
				panic(err)
			}
		}
		for _, r := range roles {
			if err := db.Save(r).Error; err != nil {
				panic(err)
			}
		}

		if err := db.Save(&model.UserGroup{UserID: users[0].ID, GroupID: groups[0].ID}).Error; err != nil {
			panic(err)
		}

		if err := db.Save(&model.UserRole{UserID: users[0].ID, RoleID: roles[0].ID}).Error; err != nil {
			panic(err)
		}

		var admin identity.User
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		for _, tc := range []struct {
			Desc        string
			UserID      uint64
			ID          uint64
			Tenant      *identity.Tenant
			TenantName  string
			Account     string
			Name        string
			Department  string
			Position    string
			Email       string
			Contact     string
			Timezone    string
			LanguageSet string
			Roles       []uint64
			Groups      []uint64
			Error       error
		}{
			{
				Desc:        "normal case",
				UserID:      users[0].ID,
				ID:          users[0].ID,
				Tenant:      &identity.Tenant{Id: tenants[0].ID},
				TenantName:  tenants[0].Name,
				Account:     users[0].Account,
				Name:        "user1-1",
				Department:  "department1",
				Position:    "position1",
				Email:       "test1-2221@localhost",
				Contact:     "111-1111",
				Timezone:    "Asia/Chita",
				LanguageSet: "kor",
				Roles:       []uint64{roles[0].ID, roles[1].ID},
				Groups:      []uint64{groups[0].ID, groups[1].ID},
				Error:       nil,
			},
			{
				Desc:    "abnormal case: id mismatch",
				UserID:  0,
				ID:      users[1].ID,
				Tenant:  &identity.Tenant{Id: tenants[0].ID},
				Account: users[1].Account,
				Name:    "user1",
				Error:   errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:    "abnormal case: invalid email address",
				UserID:  0,
				ID:      users[1].ID,
				Tenant:  &identity.Tenant{Id: tenants[0].ID},
				Account: users[1].Account,
				Name:    "user1",
				Email:   "tester1@test.com",
				Error:   errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:    "abnormal case: unknown user id",
				UserID:  99999999,
				ID:      99999999,
				Tenant:  &identity.Tenant{Id: tenants[1].ID},
				Account: "account1",
				Name:    "user1",
				Error:   errors.NotFound(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			ctx := setCtx(&admin, tc.Tenant.Id)

			var req = new(identity.UpdateUserRequest)
			req.UserId = tc.UserID
			req.User = new(identity.User)
			req.User.Id = tc.ID
			req.User.Tenant = tc.Tenant
			req.User.Account = tc.Account
			req.User.Name = tc.Name
			req.User.Department = tc.Department
			req.User.Position = tc.Position
			req.User.Email = tc.Email
			req.User.Contact = tc.Contact
			req.User.Timezone = tc.Timezone
			req.User.LanguageSet = tc.LanguageSet

			for _, RoleID := range tc.Roles {
				req.User.Roles = append(req.User.Roles, &identity.Role{Id: RoleID})
			}

			for _, GroupID := range tc.Groups {
				req.User.Groups = append(req.User.Groups, &identity.Group{Id: GroupID})
			}

			var rsp = new(identity.UserResponse)

			err := handler.UpdateUser(ctx, req, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.ID, rsp.User.Id, tc.Desc)
				assert.Equal(t, tc.Tenant.Id, rsp.User.Tenant.Id, tc.Desc)
				assert.Equal(t, tc.TenantName, rsp.User.Tenant.Name, tc.Desc)
				assert.Equal(t, tc.Account, rsp.User.Account, tc.Desc)
				assert.Equal(t, tc.Name, rsp.User.Name, tc.Desc)
				assert.Equal(t, tc.Department, rsp.User.Department, tc.Desc)
				assert.Equal(t, tc.Position, rsp.User.Position, tc.Desc)
				assert.Equal(t, tc.Contact, rsp.User.Contact, tc.Desc)
				assert.Equal(t, tc.Timezone, rsp.User.Timezone, tc.Desc)
				assert.Equal(t, tc.LanguageSet, rsp.User.LanguageSet, tc.Desc)
				assert.Equal(t, tc.Email, rsp.User.Email, tc.Desc)

				var roles []uint64
				for _, r := range rsp.User.Roles {
					roles = append(roles, r.Id)
				}

				assert.ElementsMatch(t, tc.Roles, roles, tc.Desc)

				var groups []uint64
				for _, g := range rsp.User.Groups {
					groups = append(groups, g.Id)
				}

				assert.ElementsMatch(t, tc.Groups, groups, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestUpdateUserNilPointer(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30]}

		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		authManager := identity.User{
			Tenant: &identity.Tenant{Id: tenant.ID},
		}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		authManager.Roles = append(authManager.Roles, r)

		ctx := setCtx(&authManager, tenant.ID)

		var req = new(identity.UpdateUserRequest)
		req.User = new(identity.User)
		req.User.Tenant = nil
		req.User.Account = "account"
		req.User.Name = "name"

		var rsp = new(identity.UserResponse)
		err := handler.UpdateUser(ctx, req, rsp)

		assert.Equal(t, int32(400), err.(*errors.Error).Code)
	})
}

func TestUpdateUserWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var users []*model.User
		var tenants []*model.Tenant

		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})

		for _, tenant := range tenants {
			if err := db.Save(&tenant).Error; err != nil {
				panic(err)
			}
		}

		// 0 번 인덱스 유저: 일반 유저
		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenants[0].ID,
		})
		// 1 번 인덱스 유저: 최고 관리자
		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenants[0].ID,
		})

		for _, u := range users {
			if err := db.Save(u).Error; err != nil {
				panic(err)
			}
		}

		var tmp model.Role
		admin := identity.User{
			Account: constant.Admin,
		}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		authManager := identity.User{
			Account: "authManager",
			Tenant:  &identity.Tenant{Id: tenants[0].ID},
		}
		tmp = Role[constant.Manager]
		authManager.Roles = append(authManager.Roles, r)

		unauthManager := identity.User{
			Tenant: &identity.Tenant{Id: 9999999},
		}
		unauthManager.Roles = append(unauthManager.Roles, r)

		authUser := identity.User{
			Id:      users[0].ID,
			Name:    users[0].Name,
			Account: users[0].Account,
			Tenant:  &identity.Tenant{Id: users[0].TenantID},
		}
		authUser.Roles = append(authUser.Roles, &identity.Role{Solution: constant.SolutionName, Role: ""})

		unauthUser := identity.User{
			Id:      users[1].ID,
			Name:    users[1].Name,
			Account: users[1].Account,
			Tenant:  &identity.Tenant{Id: users[1].TenantID},
		}
		unauthUser.Roles = append(unauthUser.Roles, &identity.Role{Solution: "unknown", Role: ""})
		if err := db.Save(&model.UserRole{UserID: users[1].ID, RoleID: Role[constant.Admin].ID}).Error; err != nil {
			panic(err)
		}

		for _, tc := range []struct {
			Desc       string
			UserID     uint64
			ID         uint64
			Tenant     *identity.Tenant
			TenantName string
			Account    string
			Name       string
			Roles      []uint64
			ReqUser    *identity.User
			Error      error
		}{
			{
				Desc:       "normal case: Request update user from admin",
				UserID:     users[0].ID,
				ID:         users[0].ID,
				Tenant:     &identity.Tenant{Id: tenants[0].ID},
				TenantName: tenants[0].Name,
				Account:    users[0].Account,
				Name:       "user1-1",
				ReqUser:    &admin,
				Error:      nil,
			},
			{
				Desc:       "normal case: Request update user from authorized manager",
				UserID:     users[0].ID,
				ID:         users[0].ID,
				Tenant:     &identity.Tenant{Id: tenants[0].ID},
				TenantName: tenants[0].Name,
				Account:    users[0].Account,
				Name:       "user1-1",
				ReqUser:    &authManager,
				Error:      nil,
			},
			{
				Desc:       "normal case",
				UserID:     users[0].ID,
				ID:         users[0].ID,
				Tenant:     &identity.Tenant{Id: tenants[0].ID},
				TenantName: tenants[0].Name,
				Account:    users[0].Account,
				Name:       "user1-1",
				ReqUser:    &authUser,
				Error:      nil,
			},
			{
				Desc:    "abnormal case: Request update user from authorized user",
				UserID:  users[0].ID,
				ID:      users[0].ID,
				Tenant:  &identity.Tenant{Id: tenants[0].ID},
				Account: users[0].Account,
				Name:    "user1-1",
				ReqUser: &unauthUser,
				Error:   errors.New(constant.ServiceIdentity, "unauthorized user", 403),
			},
			{
				Desc:    "abnormal case: Request update user from authorized user",
				UserID:  users[0].ID,
				ID:      users[0].ID,
				Tenant:  &identity.Tenant{Id: tenants[0].ID},
				Account: users[0].Account,
				Name:    "user1-1",
				ReqUser: &unauthUser,
				Error:   errors.New(constant.ServiceIdentity, "unauthorized user", 403),
			},
			{
				Desc:       "normal case: Request update admin from manager",
				UserID:     users[1].ID,
				ID:         users[1].ID,
				Tenant:     &identity.Tenant{Id: tenants[1].ID},
				TenantName: tenants[1].Name,
				Account:    users[1].Account,
				Name:       "user1-1",
				ReqUser:    &authManager,
				Error:      errors.New(constant.ServiceIdentity, "not found user", 404),
			},
			{
				Desc:    "normal case: Request update admin from user",
				UserID:  users[1].ID,
				ID:      users[1].ID,
				Tenant:  &identity.Tenant{Id: tenants[1].ID},
				Account: users[1].Account,
				Name:    "user1-1",
				ReqUser: &authUser,
				Error:   errors.New(constant.ServiceIdentity, "unauthorized user", 403),
			},
			{
				Desc:       "normal case: Request update user with ",
				UserID:     users[1].ID,
				ID:         users[1].ID,
				Tenant:     &identity.Tenant{Id: tenants[1].ID},
				TenantName: tenants[1].Name,
				Account:    users[1].Account,
				Name:       "user1-1",
				ReqUser:    &authUser,
				Error:      errors.New(constant.ServiceIdentity, "unauthorized user", 403),
			},
		} {
			ctx := setCtx(tc.ReqUser, tc.Tenant.Id)

			var req = new(identity.UpdateUserRequest)
			req.UserId = tc.UserID
			req.User = new(identity.User)
			req.User.Id = tc.ID
			req.User.Tenant = tc.Tenant
			req.User.Account = tc.Account
			req.User.Name = tc.Name

			for _, RoleID := range tc.Roles {
				req.User.Roles = append(req.User.Roles, &identity.Role{Id: RoleID})
			}

			var rsp = new(identity.UserResponse)

			err := handler.UpdateUser(ctx, req, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.ID, rsp.User.Id, tc.Desc)
				assert.Equal(t, tc.Tenant.Id, rsp.User.Tenant.Id, tc.Desc)
				assert.Equal(t, tc.TenantName, rsp.User.Tenant.Name, tc.Desc)
				assert.Equal(t, tc.Account, rsp.User.Account, tc.Desc)
				assert.Equal(t, tc.Name, rsp.User.Name, tc.Desc)

				var roles []uint64
				for _, r := range rsp.User.Roles {
					roles = append(roles, r.Id)
				}

				assert.ElementsMatch(t, tc.Roles, roles, tc.Desc)

			} else {
				assert.Error(t, err, tc.Desc)
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestUpdateUserRole(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var users []*model.User
		var roles []*model.Role
		var tenants []*model.Tenant

		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})
		for _, tenant := range tenants {
			if err := db.Save(&tenant).Error; err != nil {
				panic(err)
			}
		}

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenants[0].ID,
		})

		roles = append(roles, &model.Role{
			Solution: uuid.New().String()[:30],
			Role:     uuid.New().String()[:30],
		})

		roles = append(roles, &model.Role{
			Solution: uuid.New().String()[:30],
			Role:     uuid.New().String()[:30],
		})

		for _, u := range users {
			if err := db.Save(u).Error; err != nil {
				panic(err)
			}
		}
		for _, r := range roles {
			if err := db.Save(r).Error; err != nil {
				panic(err)
			}
		}

		if err := db.Save(&model.UserRole{UserID: users[0].ID, RoleID: roles[0].ID}).Error; err != nil {
			panic(err)
		}

		authUser := identity.User{
			Id:      users[0].ID,
			Name:    users[0].Name,
			Account: users[0].Account,
			Tenant:  &identity.Tenant{Id: users[0].TenantID},
		}
		authUser.Roles = append(authUser.Roles, &identity.Role{Solution: constant.SolutionName, Role: ""})

		if err := db.Select("id").Where(&model.User{Account: constant.Admin}).First(&adminUser).Error; err != nil {
			panic(err)
		}

		var defaultTenant model.Tenant
		if err := db.First(&defaultTenant, model.Tenant{Name: "default"}).Error; err != nil {
			panic(err)
		}

		admin := identity.User{}
		adminRole := Role[constant.Admin]
		r, _ := roleModelToRsp(&adminRole)
		admin.Roles = append(admin.Roles, r)

		for _, tc := range []struct {
			Desc       string
			UserID     uint64
			ID         uint64
			Tenant     *identity.Tenant
			TenantName string
			Account    string
			Name       string
			Roles      []uint64
			Error      error
		}{
			{
				Desc:       "normal case 1",
				UserID:     users[0].ID,
				ID:         users[0].ID,
				Tenant:     &identity.Tenant{Id: tenants[0].ID},
				TenantName: tenants[0].Name,
				Account:    users[0].Account,
				Name:       "user1-1",
				Roles:      []uint64{roles[0].ID},
				Error:      nil,
			},
			{
				Desc:       "normal case 2 admin user",
				UserID:     adminUser.ID,
				ID:         adminUser.ID,
				Tenant:     &identity.Tenant{Id: defaultTenant.ID},
				TenantName: defaultTenant.Name,
				Account:    constant.Admin,
				Name:       "user1-1",
				Roles:      []uint64{roles[0].ID},
				Error:      nil,
			},
			{
				Desc:       "normal case 3",
				UserID:     users[0].ID,
				ID:         users[0].ID,
				Tenant:     &identity.Tenant{Id: tenants[0].ID},
				TenantName: tenants[0].Name,
				Account:    users[0].Account,
				Name:       "user1-1",
				Roles:      []uint64{roles[1].ID},
				Error:      nil,
			},
		} {
			ctx := setCtx(&admin, tc.Tenant.Id)

			var req = new(identity.UpdateUserRequest)
			req.UserId = tc.UserID
			req.User = new(identity.User)
			req.User.Id = tc.ID
			req.User.Tenant = tc.Tenant
			req.User.Account = tc.Account
			req.User.Name = tc.Name

			for _, RoleID := range tc.Roles {
				req.User.Roles = append(req.User.Roles, &identity.Role{Id: RoleID})
			}

			var rsp = new(identity.UserResponse)

			err := handler.UpdateUser(ctx, req, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.ID, rsp.User.Id, tc.Desc)
				assert.Equal(t, tc.Tenant.Id, rsp.User.Tenant.Id, tc.Desc)
				assert.Equal(t, tc.TenantName, rsp.User.Tenant.Name, tc.Desc)
				assert.Equal(t, tc.Account, rsp.User.Account, tc.Desc)
				assert.Equal(t, tc.Name, rsp.User.Name, tc.Desc)

				var rspRoles []uint64
				for _, r := range rsp.User.Roles {
					rspRoles = append(rspRoles, r.Id)
				}
				if tc.UserID == adminUser.ID {
					assert.Equal(t, len(admin.Roles), len(rspRoles))
					for k, id := range rspRoles {
						assert.Equal(t, id, admin.Roles[k].Id)
					}
					assert.Equal(t, rsp.User.Groups[0].Name, "default")
				} else if tc.Roles[0] == adminRole.ID {
					assert.ElementsMatch(t, roles, rspRoles, tc.Desc)
				} else {
					assert.ElementsMatch(t, tc.Roles, rspRoles, tc.Desc)
				}
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestDeleteUser(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		unknownTenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&unknownTenant).Error; err != nil {
			panic(err)
		}

		var users []*model.User
		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		})
		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		})
		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: unknownTenant.ID,
		})

		group := model.Group{
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		}

		role := model.Role{
			Solution: uuid.New().String()[:30],
			Role:     uuid.New().String()[:30],
		}
		for _, u := range users {
			if err := db.Save(&u).Error; err != nil {
				panic(err)
			}
		}

		if err := db.Save(&group).Error; err != nil {
			panic(err)
		}

		if err := db.Save(&role).Error; err != nil {
			panic(err)
		}

		userGroup := model.UserGroup{
			UserID:  users[0].ID,
			GroupID: group.ID,
		}
		if err := db.Save(&userGroup).Error; err != nil {
			panic(err)
		}

		if err := db.Save(&model.UserRole{UserID: users[0].ID, RoleID: role.ID}).Error; err != nil {
			panic(err)
		}
		if err := db.Save(&model.UserRole{UserID: users[1].ID, RoleID: Role[constant.Admin].ID}).Error; err != nil {
			panic(err)
		}
		if err := db.Save(&model.UserRole{UserID: users[2].ID, RoleID: role.ID}).Error; err != nil {
			panic(err)
		}

		admin := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		for _, tc := range []struct {
			Desc  string
			ID    uint64
			Error error
		}{
			{
				Desc:  "normal case",
				ID:    users[0].ID,
				Error: nil,
			},
			{
				Desc:  "abnormal case: unknown user id",
				ID:    uint64(0),
				Error: errors.BadRequest(constant.ServiceIdentity, "required parameter"),
			},
			{
				Desc:  "abnormal case: unknown user id",
				ID:    uint64(999999999),
				Error: errors.NotFound(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:  "abnormal case: admin account",
				ID:    users[1].ID,
				Error: errors.BadRequest(constant.ServiceIdentity, "undeletable user"),
			},
			{
				Desc:  "abnormal case: unknown tenant user",
				ID:    users[2].ID,
				Error: errors.NotFound(constant.ServiceIdentity, "unknown user"),
			},
		} {
			ctx := setCtx(&admin, tenant.ID)

			var rsp = new(identity.MessageResponse)

			err := handler.DeleteUser(ctx, &identity.DeleteUserRequest{UserId: tc.ID}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestDeleteUserWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		var users []*model.User
		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		})
		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		})
		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		})

		for _, u := range users {
			if err := db.Save(&u).Error; err != nil {
				panic(err)
			}
		}

		var tmp model.Role

		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		authManager := identity.User{
			Tenant: &identity.Tenant{Id: tenant.ID},
		}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		authManager.Roles = append(authManager.Roles, r)

		unauthManager := identity.User{
			Tenant: &identity.Tenant{Id: 99999999999},
		}

		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		unauthManager.Roles = append(unauthManager.Roles, r)

		user := identity.User{}
		user.Roles = append(user.Roles, &identity.Role{})

		for _, tc := range []struct {
			Desc    string
			ID      uint64
			ReqUser *identity.User
			Tenant  *identity.Tenant
			Error   error
		}{
			{
				Desc:    "normal case: admin has sent a request to delete a user",
				ID:      users[0].ID,
				ReqUser: &admin,
				Tenant:  &identity.Tenant{Id: tenant.ID},
				Error:   nil,
			},
			{
				Desc:    "normal case: authorized manager has sent a request to delete a user",
				ID:      users[1].ID,
				ReqUser: &authManager,
				Tenant:  &identity.Tenant{Id: tenant.ID},
				Error:   nil,
			},
			{
				Desc:    "abnormal case: unauthorized manager has sent a request to delete a user",
				ID:      users[2].ID,
				ReqUser: &unauthManager,
				Tenant:  &identity.Tenant{Id: 99999999999},
				Error:   errors.NotFound(constant.ServiceIdentity, "unknown user"),
			},
		} {
			ctx := setCtx(tc.ReqUser, tc.Tenant.Id)

			var rsp = new(identity.MessageResponse)

			err := handler.DeleteUser(ctx, &identity.DeleteUserRequest{UserId: tc.ID}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetUsers(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var users []*model.User
		var groups []*model.Group
		var roles []*model.Role
		var tenant model.Tenant

		tenant = model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     "GildongHong",
			TenantID: tenant.ID,
		})
		users[0].Department = new(string)
		users[0].Position = new(string)
		*users[0].Department = "Development"
		*users[0].Position = "Developer"

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     "GildongKo",
			TenantID: tenant.ID,
		})
		users[1].Position = new(string)
		*users[1].Position = "Team leader"

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		})
		users[2].Department = new(string)
		*users[2].Department = "Development"

		groups = append(groups, &model.Group{
			Name:     "Team1",
			TenantID: tenant.ID,
			Remarks:  new(string),
		})
		groups = append(groups, &model.Group{
			Name:     "Team2",
			TenantID: tenant.ID,
		})
		*groups[0].Remarks = uuid.New().String()[:30]

		roles = append(roles, &model.Role{
			Solution: "CDM-DisasterRecovery",
			Role:     constant.Manager,
		})
		roles = append(roles, &model.Role{
			Solution: "CDM-DisasterRecovery",
			Role:     "Developer",
		})

		for _, u := range users {
			if err := db.Save(u).Error; err != nil {
				panic(err)
			}
		}

		for _, g := range groups {
			if err := db.Save(g).Error; err != nil {
				panic(err)
			}
		}

		for _, r := range roles {
			if err := db.Save(r).Error; err != nil {
				panic(err)
			}
		}

		if err := db.Save(&model.UserGroup{UserID: users[0].ID, GroupID: groups[0].ID}).Error; err != nil {
			panic(err)
		}

		if err := db.Save(&model.UserGroup{UserID: users[1].ID, GroupID: groups[0].ID}).Error; err != nil {
			panic(err)
		}

		if err := db.Save(&model.UserGroup{UserID: users[2].ID, GroupID: groups[1].ID}).Error; err != nil {
			panic(err)
		}

		if err := db.Save(&model.UserRole{UserID: users[0].ID, RoleID: roles[0].ID}).Error; err != nil {
			panic(err)
		}

		if err := db.Save(&model.UserRole{UserID: users[1].ID, RoleID: roles[1].ID}).Error; err != nil {
			panic(err)
		}

		if err := db.Save(&model.UserRole{UserID: users[2].ID, RoleID: roles[0].ID}).Error; err != nil {
			panic(err)
		}

		for _, u := range users[:2] {
			key := storeKeyPrefix + "." + strconv.FormatUint(u.ID, 10)
			if err := store.Put(key, "session"); err != nil {
				assert.Fail(t, "Could not put session in GetUsers test. cause :%v", err)
			}
		}
		defer func() {
			for _, u := range users[:2] {
				key := storeKeyPrefix + "." + strconv.FormatUint(u.ID, 10)
				_ = store.Delete(key)
			}
		}()

		time.Sleep(2 * time.Second)

		for _, tc := range []struct {
			Desc           string
			Limit          uint64
			Offset         uint64
			Solution       string
			Role           string
			GroupID        uint64
			UserName       string
			Department     string
			Position       string
			Expected       int
			ExcludeGroupID uint64
			LoginOnly      bool
			Error          error
		}{
			{
				Desc:     "all",
				Expected: 3,
				Error:    nil,
			},
			{
				Desc:     "solution",
				Solution: roles[0].Solution,
				Expected: 3,
				Error:    nil,
			},
			{
				Desc:     "solution+role",
				Solution: roles[0].Solution,
				Role:     roles[0].Role,
				Expected: 2,
				Error:    nil,
			},
			{
				Desc:     "group",
				GroupID:  groups[0].ID,
				Expected: 2,
				Error:    nil,
			},
			{
				Desc:     "solution+role+group",
				Solution: roles[0].Solution,
				Role:     roles[0].Role,
				GroupID:  groups[0].ID,
				Expected: 1,
				Error:    nil,
			},
			{
				Desc:     "name",
				UserName: "Gildong",
				Expected: 2,
				Error:    nil,
			},
			{
				Desc:     "position",
				Position: "Developer",
				Expected: 1,
				Error:    nil,
			},
			{
				Desc:       "department",
				Department: "Development",
				Expected:   2,
				Error:      nil,
			},
			{
				Desc:     "pagination",
				Offset:   1,
				Limit:    2,
				Expected: 2,
				Error:    nil,
			},
			{
				Desc:           "exclude group",
				ExcludeGroupID: groups[1].ID,
				Expected:       2,
				Error:          nil,
			},
			{
				Desc:      "login user",
				LoginOnly: true,
				Expected:  2,
				Error:     nil,
			},
			{
				Desc:     "no contents",
				Solution: "unknown",
				Expected: 0,
				Error:    nil,
			},
			{
				Desc:     "abnormal case: length of solution over max size",
				Solution: generateString(solutionLength + 1),
				Expected: 0,
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:     "abnormal case: unknown role",
				Role:     "developer",
				Expected: 0,
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:     "abnormal case: length of solution over max size",
				Role:     generateString(roleLength + 1),
				Expected: 0,
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:     "abnormal case: length of name over max size",
				UserName: generateString(nameLength + 1),
				Expected: 0,
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: length of department over max size",
				Department: generateString(departmentLength + 1),
				Expected:   0,
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:     "abnormal case: length of position over max size",
				Position: generateString(positionLength + 1),
				Expected: 0,
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			admin := identity.User{}
			tmp := Role[constant.Admin]
			r, _ := roleModelToRsp(&tmp)
			admin.Roles = append(admin.Roles, r)

			ctx := setCtx(&admin, tenant.ID)

			req := identity.GetUsersRequest{
				Offset:         &wrappers.UInt64Value{Value: tc.Offset},
				Limit:          &wrappers.UInt64Value{Value: tc.Limit},
				Solution:       tc.Solution,
				Role:           tc.Role,
				GroupId:        tc.GroupID,
				Name:           tc.UserName,
				Position:       tc.Position,
				Department:     tc.Department,
				ExcludeGroupId: tc.ExcludeGroupID,
				LoginOnly:      tc.LoginOnly,
			}

			var rsp = new(identity.UsersResponse)

			err := handler.GetUsers(ctx, &req, rsp)
			if tc.Error == nil {
				if tc.Expected > 0 {
					assert.Equal(t, tc.Expected, len(rsp.Users), tc.Desc)
				} else {
					assert.Equal(t, int32(204), err.(*errors.Error).Code, tc.Desc)
				}
			} else {
				assert.Equal(t, int32(400), err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetUsersWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var users []*model.User
		var tenants []*model.Tenant
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})

		for _, tenant := range tenants {
			if err := db.Save(&tenant).Error; err != nil {
				panic(err)
			}
		}

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     "GildongHong",
			TenantID: tenants[0].ID,
		})

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     "GildongKo",
			TenantID: tenants[0].ID,
		})

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenants[1].ID,
		})

		for _, u := range users {
			if err := db.Save(u).Error; err != nil {
				panic(err)
			}
		}

		var tmp model.Role

		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		var authManagers []*identity.User
		authManagers = append(authManagers, &identity.User{Tenant: &identity.Tenant{Id: tenants[0].ID}})
		authManagers = append(authManagers, &identity.User{Tenant: &identity.Tenant{Id: tenants[1].ID}})
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		authManagers[0].Roles = append(authManagers[0].Roles, r)
		authManagers[1].Roles = append(authManagers[1].Roles, r)

		user := identity.User{}
		user.Roles = append(user.Roles, &identity.Role{Solution: constant.SolutionName, Role: ""})

		for _, tc := range []struct {
			Desc     string
			Limit    uint64
			Offset   uint64
			ReqUser  *identity.User
			TenantID uint64
			Expected int
		}{
			{
				Desc:     "case 1 - expected 2",
				ReqUser:  &admin,
				TenantID: tenants[0].ID,
				Expected: 2,
			},
			{
				Desc:     "case 2 - expected 1",
				ReqUser:  &admin,
				TenantID: tenants[1].ID,
				Expected: 1,
			},
			{
				Desc:     "case 3 - expected 2",
				ReqUser:  authManagers[0],
				TenantID: tenants[0].ID,
				Expected: 2,
			},
			{
				Desc:     "case 4 - expected 1",
				ReqUser:  authManagers[1],
				TenantID: tenants[1].ID,
				Expected: 1,
			},
		} {
			ctx := setCtx(tc.ReqUser, tc.TenantID)

			req := identity.GetUsersRequest{
				Offset: &wrappers.UInt64Value{Value: tc.Offset},
				Limit:  &wrappers.UInt64Value{Value: tc.Limit},
			}

			var rsp = new(identity.UsersResponse)

			err := handler.GetUsers(ctx, &req, rsp)
			if tc.Expected > 0 {
				assert.Equal(t, tc.Expected, len(rsp.Users), tc.Desc)
			} else {
				assert.Equal(t, int32(403), err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestUpdateUserPassword(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		user := model.User{
			Account:            uuid.New().String()[:30],
			Name:               uuid.New().String()[:30],
			TenantID:           tenant.ID,
			PasswordUpdateFlag: new(bool),
		}

		hashed := sha256.Sum256([]byte("Password!1"))
		user.Password = hex.EncodeToString(hashed[:])
		*user.PasswordUpdateFlag = true
		if err := db.Save(&user).Error; err != nil {
			panic(err)
		}

		authUser := identity.User{
			Id:      user.ID,
			Account: user.Account,
			Name:    user.Name,
			Tenant:  &identity.Tenant{Id: user.TenantID},
		}

		unauthUser := identity.User{
			Id:      999999999,
			Account: user.Account,
			Name:    user.Name,
			Tenant:  &identity.Tenant{Id: user.TenantID},
		}

		for _, tc := range []struct {
			Desc            string
			ID              uint64
			CurrentPassword string
			NewPassword     string
			OldPassword     string
			ReqUser         *identity.User
			Error           error
		}{
			{
				Desc:            "normal case",
				ID:              user.ID,
				CurrentPassword: "Password!1",
				NewPassword:     "bbBB22@@",
				ReqUser:         &authUser,
				Error:           nil,
			},
			{
				Desc:            "abnormal case",
				ID:              user.ID,
				CurrentPassword: "Password!1",
				NewPassword:     "bbBB22@@",
				ReqUser:         &unauthUser,
				Error:           errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:            "abnormal case: current password mismatch",
				ID:              user.ID,
				CurrentPassword: "xxxxxxxxxx",
				NewPassword:     "Password!1",
				ReqUser:         &authUser,
				Error:           errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:            "abnormal case: Password conflict",
				ID:              user.ID,
				CurrentPassword: "bbBB22@@",
				NewPassword:     "Password!1",
				ReqUser:         &authUser,
				Error:           errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:            "abnormal case: unknown user id",
				ID:              999999999,
				CurrentPassword: user.Password,
				NewPassword:     "ccCC33##",
				ReqUser:         &unauthUser,
				Error:           errors.NotFound(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			tmp := sha256.Sum256([]byte(tc.CurrentPassword))
			hashedCurrentPassword := hex.EncodeToString(tmp[:])
			tmp = sha256.Sum256([]byte(tc.NewPassword))
			hashedNewPassword := hex.EncodeToString(tmp[:])

			ctx := setCtx(tc.ReqUser, tenant.ID)

			err := handler.UpdateUserPassword(
				ctx,
				&identity.UpdateUserPasswordRequest{
					UserId:          tc.ID,
					CurrentPassword: hashedCurrentPassword,
					NewPassword:     hashedNewPassword,
				},
				&identity.MessageResponse{},
			)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)

				var u model.User
				db.First(&u, tc.ID)

				assert.Equal(t, hashedCurrentPassword, *u.OldPassword, tc.Desc)
				assert.Equal(t, hashedNewPassword, u.Password, tc.Desc)
				assert.Equal(t, false, *u.PasswordUpdateFlag, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestResetUserPassword(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		var users []*model.User
		user1 := &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		}
		hashed := sha256.Sum256([]byte("Password!1"))
		user1.Password = hex.EncodeToString(hashed[:])
		users = append(users, user1)

		user2 := &model.User{
			Account:            uuid.New().String()[:30],
			Name:               uuid.New().String()[:30],
			TenantID:           tenant.ID,
			PasswordUpdateFlag: new(bool),
		}
		hashed = sha256.Sum256([]byte("Password!1"))
		user2.Password = hex.EncodeToString(hashed[:])
		*user2.PasswordUpdateFlag = false
		users = append(users, user2)
		for _, u := range users {
			if err := db.Save(u).Error; err != nil {
				panic(err)
			}
		}

		admin := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		for _, tc := range []struct {
			Desc  string
			ID    uint64
			Error error
		}{
			{
				Desc:  "normal1 case",
				ID:    user1.ID,
				Error: nil,
			},
			{
				Desc:  "normal2 case",
				ID:    user2.ID,
				Error: nil,
			},
			{
				Desc:  "abnormal case: unknown user id",
				ID:    uint64(0),
				Error: errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:  "abnormal case: unknown user id",
				ID:    uint64(999999999),
				Error: errors.NotFound(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			ctx := setCtx(&admin, tenant.ID)

			var rsp = new(identity.UserPasswordResponse)

			err := handler.ResetUserPassword(ctx, &identity.ResetUserPasswordRequest{UserId: tc.ID}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)

				var u model.User
				db.First(&u, tc.ID)

				tmp := sha256.Sum256([]byte("Password!1"))
				hashedCurrentPassword := hex.EncodeToString(tmp[:])
				assert.Equal(t, hashedCurrentPassword, *u.OldPassword, tc.Desc)
				assert.Equal(t, true, *u.PasswordUpdateFlag, tc.Desc)

			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestResetUserPasswordWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		var users []*model.User

		users = append(users, &model.User{
			Account:            uuid.New().String()[:30],
			Name:               uuid.New().String()[:30],
			TenantID:           tenant.ID,
			PasswordUpdateFlag: new(bool),
		})
		hashed := sha256.Sum256([]byte("Password!1"))
		users[0].Password = hex.EncodeToString(hashed[:])
		*users[0].PasswordUpdateFlag = false

		users = append(users, &model.User{
			Account:            uuid.New().String()[:30],
			Name:               uuid.New().String()[:30],
			TenantID:           tenant.ID,
			PasswordUpdateFlag: new(bool),
		})
		hashed = sha256.Sum256([]byte("Password!1"))
		users[1].Password = hex.EncodeToString(hashed[:])
		*users[1].PasswordUpdateFlag = false

		users = append(users, &model.User{
			Account:            uuid.New().String()[:30],
			Name:               uuid.New().String()[:30],
			TenantID:           tenant.ID,
			PasswordUpdateFlag: new(bool),
		})
		hashed = sha256.Sum256([]byte("Password!1"))
		users[2].Password = hex.EncodeToString(hashed[:])
		*users[2].PasswordUpdateFlag = false

		for _, u := range users {
			if err := db.Save(&u).Error; err != nil {
				panic(err)
			}
		}

		var tmp model.Role

		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		authManager := identity.User{
			Tenant: &identity.Tenant{Id: tenant.ID},
		}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		authManager.Roles = append(authManager.Roles, r)

		unauthManager := identity.User{
			Tenant: &identity.Tenant{Id: 999999999},
		}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		unauthManager.Roles = append(unauthManager.Roles, r)

		unauthUser := identity.User{
			Tenant: &identity.Tenant{Id: tenant.ID},
		}
		unauthUser.Roles = append(unauthUser.Roles, &identity.Role{Solution: constant.SolutionName})

		for _, tc := range []struct {
			Desc     string
			ID       uint64
			ReqUser  *identity.User
			TenantID uint64
			Error    error
		}{
			{
				Desc:     "normal case: admin reset user password",
				ID:       users[0].ID,
				ReqUser:  &admin,
				TenantID: tenant.ID,
				Error:    nil,
			},
			{
				Desc:     "normal case: auth manager reset user password",
				ID:       users[1].ID,
				ReqUser:  &authManager,
				TenantID: tenant.ID,
				Error:    nil,
			},
		} {
			ctx := setCtx(tc.ReqUser, tc.TenantID)

			var rsp = new(identity.UserPasswordResponse)
			err := handler.ResetUserPassword(ctx, &identity.ResetUserPasswordRequest{UserId: tc.ID}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)

				var u model.User
				db.First(&u, tc.ID)

				tmp := sha256.Sum256([]byte("Password!1"))
				hashedCurrentPassword := hex.EncodeToString(tmp[:])
				assert.Equal(t, hashedCurrentPassword, *u.OldPassword, tc.Desc)
				assert.Equal(t, true, *u.PasswordUpdateFlag, tc.Desc)

			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetGroups(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		var groups []*model.Group
		groups = append(groups, &model.Group{Name: "Team1", TenantID: tenant.ID})
		groups = append(groups, &model.Group{Name: "Group1", TenantID: tenant.ID})
		groups = append(groups, &model.Group{Name: "Team2", TenantID: tenant.ID})

		for _, g := range groups {
			if err := db.Save(g).Error; err != nil {
				panic(err)
			}
		}

		admin := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		for _, tc := range []struct {
			Desc     string
			Name     string
			Remarks  string
			Expected int
			Error    error
		}{
			{
				Desc:     "normal case : all search",
				Expected: 3,
				Error:    nil,
			},
			{
				Desc:     "normal case : Name filter",
				Name:     "Team",
				Expected: 2,
				Error:    nil,
			},
			{
				Desc:     "abnormal case : unknown Name",
				Name:     "Developer",
				Expected: 0,
				Error:    nil,
			},
			{
				Desc:     "abnormal case : length of group name over max size",
				Name:     generateString(nameLength + 1),
				Expected: 0,
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(&admin)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))
			ctx = metadata.Set(ctx, commonmeta.HeaderTenantID, strconv.Itoa(int(tenant.ID)))

			var rsp = new(identity.GroupsResponse)
			err := handler.GetGroups(
				ctx,
				&identity.GetGroupsRequest{Name: tc.Name},
				rsp)

			if tc.Error == nil {
				if tc.Expected > 0 {
					assert.NoError(t, err, tc.Desc)
					assert.Equal(t, tc.Expected, len(rsp.Groups), tc.Desc)
				} else {
					assert.Equal(t, int32(204), err.(*errors.Error).Code, tc.Desc)
				}
			} else {
				assert.Equal(t, int32(400), err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetGroupsWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var tenants []*model.Tenant
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})
		for _, tenant := range tenants {
			if err := db.Save(&tenant).Error; err != nil {
				panic(err)
			}
		}

		var groups []*model.Group
		groups = append(groups, &model.Group{Name: "Team1", TenantID: tenants[0].ID})
		groups = append(groups, &model.Group{Name: "Group1", TenantID: tenants[0].ID})
		groups = append(groups, &model.Group{Name: "Team2", TenantID: tenants[1].ID})

		for _, g := range groups {
			if err := db.Save(g).Error; err != nil {
				panic(err)
			}
		}

		var tmp model.Role

		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		var managers []*identity.User
		managers = append(managers, &identity.User{Tenant: &identity.Tenant{Id: tenants[0].ID}})
		managers = append(managers, &identity.User{Tenant: &identity.Tenant{Id: tenants[1].ID}})
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		managers[0].Roles = append(managers[0].Roles, r)
		managers[1].Roles = append(managers[0].Roles, r)

		user := identity.User{}
		user.Roles = append(user.Roles, &identity.Role{})

		for _, tc := range []struct {
			Desc     string
			ReqUser  *identity.User
			TenantID uint64
			Expected int
			Error    error
		}{
			{
				Desc:     "normal case : admin get groups",
				ReqUser:  &admin,
				Expected: 2,
				TenantID: tenants[0].ID,
				Error:    nil,
			},
			{
				Desc:     "normal case : admin get groups",
				ReqUser:  &admin,
				Expected: 1,
				TenantID: tenants[1].ID,
				Error:    nil,
			},
			{
				Desc:     "normal case : expected 2",
				ReqUser:  managers[0],
				TenantID: tenants[0].ID,
				Expected: 2,
				Error:    nil,
			},
			{
				Desc:     "normal case : expected 1",
				ReqUser:  managers[1],
				TenantID: tenants[1].ID,
				Expected: 1,
				Error:    nil,
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(tc.ReqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))
			ctx = metadata.Set(ctx, commonmeta.HeaderTenantID, strconv.Itoa(int(tc.TenantID)))

			var rsp = new(identity.GroupsResponse)
			err := handler.GetGroups(ctx, &identity.GetGroupsRequest{}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.Expected, len(rsp.Groups), tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestAddGroup(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		admin := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		var groups []*model.Group
		for _, tc := range []struct {
			Desc       string
			Name       string
			Tenant     *identity.Tenant
			TenantName string
			Remarks    string
			Error      error
		}{
			{
				Desc:       "normal case",
				Name:       "Test",
				Remarks:    uuid.New().String()[:30],
				Tenant:     &identity.Tenant{Id: tenant.ID},
				TenantName: tenant.Name,
				Error:      nil,
			},
			{
				Desc:    "abnormal case: group name conflict",
				Name:    "Test",
				Tenant:  &identity.Tenant{Id: tenant.ID},
				Remarks: uuid.New().String()[:30],
				Error:   errors.Conflict(constant.ServiceIdentity, "conflict parameter"),
			},
			{
				Desc:    "abnormal case: unknown tenant id",
				Name:    "Test2",
				Tenant:  &identity.Tenant{Id: 999999999},
				Remarks: uuid.New().String()[:30],
				Error:   errors.NotFound(constant.ServiceIdentity, "not found tenant"),
			},
			{
				Desc:       "abnormal case: length of group name over max size",
				Name:       generateString(nameLength + 1),
				Tenant:     &identity.Tenant{Id: tenant.ID},
				TenantName: tenant.Name,
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: length of group name is 0",
				Name:       "",
				Tenant:     &identity.Tenant{Id: tenant.ID},
				TenantName: tenant.Name,
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: length of remarks over max size",
				Name:       uuid.New().String()[:30],
				Remarks:    generateString(remarksLength + 1),
				Tenant:     &identity.Tenant{Id: tenant.ID},
				TenantName: tenant.Name,
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(&admin)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))
			ctx = metadata.Set(ctx, commonmeta.HeaderTenantID, strconv.Itoa(int(tc.Tenant.Id)))

			var rsp = new(identity.GroupResponse)
			var req = new(identity.AddGroupRequest)
			req.Group = new(identity.Group)
			req.Group.Tenant = tc.Tenant
			req.Group.Name = tc.Name
			req.Group.Remarks = tc.Remarks

			err := handler.AddGroup(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.Tenant.Id, rsp.Group.Tenant.Id)
				assert.Equal(t, tc.TenantName, rsp.Group.Tenant.Name)
				assert.Equal(t, tc.Name, rsp.Group.Name, tc.Desc)
				assert.Equal(t, tc.Remarks, rsp.Group.Remarks, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
			if err == nil && rsp.Group != nil {
				groups = append(groups, &model.Group{ID: rsp.Group.Id})
			}
		}
	})
}

func TestAddGroupNilPointer(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		admin := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		ctx := context.Background()
		b, _ := json.Marshal(&admin)
		ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))
		ctx = metadata.Set(ctx, commonmeta.HeaderTenantID, strconv.Itoa(int(tenant.ID)))

		var rsp = new(identity.GroupResponse)
		var req = new(identity.AddGroupRequest)
		req.Group = new(identity.Group)
		req.Group.Tenant = nil
		req.Group.Name = "name"

		err := handler.AddGroup(ctx, req, rsp)
		assert.Equal(t, int32(400), err.(*errors.Error).Code)
	})
}

func TestAddGroupWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		var tmp model.Role
		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		authManager := identity.User{
			Tenant: &identity.Tenant{Id: tenant.ID},
		}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		authManager.Roles = append(authManager.Roles, r)

		unauthManager := identity.User{}
		unauthManager.Roles = append(unauthManager.Roles, &identity.Role{Solution: "unknown", Role: constant.Manager})

		user := identity.User{}
		user.Roles = append(user.Roles, &identity.Role{Solution: constant.SolutionName, Role: ""})

		var groups []*model.Group
		for _, tc := range []struct {
			Desc       string
			Name       string
			Tenant     *identity.Tenant
			TenantName string
			ReqUser    *identity.User
			Error      error
		}{
			{
				Desc:       "normal case: admin add group",
				Name:       uuid.New().String()[:30],
				Tenant:     &identity.Tenant{Id: tenant.ID},
				TenantName: tenant.Name,
				ReqUser:    &admin,
				Error:      nil,
			},
			{
				Desc:       "normal case: auth manager add group",
				Name:       uuid.New().String()[:30],
				Tenant:     &identity.Tenant{Id: tenant.ID},
				TenantName: tenant.Name,
				ReqUser:    &authManager,
				Error:      nil,
			},
		} {
			ctx := setCtx(tc.ReqUser, tc.Tenant.Id)

			var rsp = new(identity.GroupResponse)
			var req = new(identity.AddGroupRequest)
			req.Group = new(identity.Group)
			req.Group.Tenant = tc.Tenant
			req.Group.Name = tc.Name

			err := handler.AddGroup(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.Tenant.Id, rsp.Group.Tenant.Id)
				assert.Equal(t, tc.TenantName, rsp.Group.Tenant.Name)
				assert.Equal(t, tc.Name, rsp.Group.Name, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
			if err == nil && rsp.Group != nil {
				groups = append(groups, &model.Group{ID: rsp.Group.Id})
			}
		}
	})
}

func TestAddGroupWithId(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30]}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}
		group := model.Group{Name: uuid.New().String()[:30], TenantID: tenant.ID}
		if err := db.Save(&group).Error; err != nil {
			panic(err)
		}

		admin := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		var groups []*model.Group
		for _, tc := range []struct {
			Desc       string
			ID         uint64
			Name       string
			Tenant     *identity.Tenant
			TenantName string
			Remarks    string
			Error      error
		}{
			{
				Desc:       "normal case",
				Name:       "Test",
				Tenant:     &identity.Tenant{Id: tenant.ID},
				TenantName: tenant.Name,
				Remarks:    uuid.New().String()[:30],
				Error:      nil,
			},
			{
				Desc:       "abnormal case: group id (already exist)",
				ID:         group.ID,
				Name:       "Test2",
				Tenant:     &identity.Tenant{Id: tenant.ID},
				TenantName: tenant.Name,
				Remarks:    uuid.New().String()[:30],
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(&admin)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))
			ctx = metadata.Set(ctx, commonmeta.HeaderTenantID, strconv.Itoa(int(tc.Tenant.Id)))

			var rsp = new(identity.GroupResponse)
			var req = new(identity.AddGroupRequest)
			req.Group = new(identity.Group)
			req.Group.Name = tc.Name
			req.Group.Tenant = tc.Tenant
			req.Group.Id = tc.ID
			req.Group.Remarks = tc.Remarks

			err := handler.AddGroup(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.Tenant.Id, rsp.Group.Tenant.Id)
				assert.Equal(t, tc.TenantName, rsp.Group.Tenant.Name)
				assert.Equal(t, tc.Name, rsp.Group.Name, tc.Desc)
				assert.Equal(t, tc.Remarks, rsp.Group.Remarks, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
			if err == nil && rsp.Group != nil {
				groups = append(groups, &model.Group{ID: rsp.Group.Id})
			}
		}
	})
}

func TestDeletedGroupName(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var tenants []*model.Tenant
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})

		for _, tn := range tenants {
			if err := db.Save(tn).Error; err != nil {
				panic(err)
			}
		}

		var groups []*model.Group
		groups = append(groups, &model.Group{Name: uuid.New().String()[:30], TenantID: tenants[0].ID, DeletedFlag: false})
		groups = append(groups, &model.Group{Name: uuid.New().String()[:30], TenantID: tenants[0].ID, DeletedFlag: true})
		groups = append(groups, &model.Group{Name: uuid.New().String()[:30], TenantID: tenants[0].ID, DeletedFlag: false})
		groups = append(groups, &model.Group{Name: uuid.New().String()[:30], TenantID: tenants[0].ID, DeletedFlag: true})
		groups = append(groups, &model.Group{Name: uuid.New().String()[:30], TenantID: tenants[1].ID, DeletedFlag: false})

		for _, g := range groups {
			if err := db.Save(g).Error; err != nil {
				panic(err)
			}
		}

		admin := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		t.Run("add deleted group name", func(t *testing.T) {
			for _, tc := range []struct {
				Desc            string
				Name            string
				Tenant          *identity.Tenant
				Remarks         string
				ExpectedMessage string
				Error           error
			}{
				{
					Desc:    "add exist group name in same tenant",
					Name:    groups[0].Name,
					Tenant:  &identity.Tenant{Id: tenants[0].ID},
					Remarks: "test remarks",
					Error:   errors.Conflict(constant.ServiceIdentity, "conflict parameter"),
				},
				{
					Desc:            "add deleted group name in same tenant",
					Name:            groups[1].Name,
					Tenant:          &identity.Tenant{Id: tenants[0].ID},
					Remarks:         "test remarks",
					ExpectedMessage: "cdm-cloud.identity.add_group.success",
				},
				{
					Desc:            "add exist group name in different tenant",
					Name:            groups[0].Name,
					Tenant:          &identity.Tenant{Id: tenants[1].ID},
					Remarks:         "test remarks",
					ExpectedMessage: "cdm-cloud.identity.add_group.success",
				},
				{
					Desc:            "add deleted group name in different tenant",
					Name:            groups[1].Name,
					Tenant:          &identity.Tenant{Id: tenants[1].ID},
					Remarks:         "test remarks",
					ExpectedMessage: "cdm-cloud.identity.add_group.success",
				},
			} {
				ctx := context.Background()
				b, _ := json.Marshal(&admin)
				ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))
				ctx = metadata.Set(ctx, commonmeta.HeaderTenantID, strconv.Itoa(int(tc.Tenant.Id)))

				var rsp = new(identity.GroupResponse)
				var req = new(identity.AddGroupRequest)
				req.Group = new(identity.Group)
				req.Group.Tenant = tc.Tenant
				req.Group.Name = tc.Name
				req.Group.Remarks = tc.Remarks

				if err := handler.AddGroup(ctx, req, rsp); err != nil {
					assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code)
				} else {
					assert.Equal(t, tc.ExpectedMessage, rsp.Message.Code, tc.Desc)
				}
			}
		})

		t.Run("update deleted group name", func(t *testing.T) {
			for _, tc := range []struct {
				Desc            string
				ID              uint64
				Tenant          *identity.Tenant
				TenantName      string
				GroupID         uint64
				Name            string
				Remarks         string
				ExpectedMessage string
				Error           error
			}{
				{
					Desc:    "update exist group name in same tenant",
					ID:      groups[0].ID,
					GroupID: groups[0].ID,
					Tenant:  &identity.Tenant{Id: tenants[0].ID},
					Name:    groups[2].Name,
					Error:   errors.Conflict(constant.ServiceIdentity, "conflict parameter"),
				},
				{
					Desc:            "update deleted group name in same tenant",
					ID:              groups[0].ID,
					GroupID:         groups[0].ID,
					Tenant:          &identity.Tenant{Id: tenants[0].ID},
					Name:            groups[3].Name,
					ExpectedMessage: "cdm-cloud.identity.update_group.success",
				},
				{
					Desc:            "update exist group name in different tenant",
					ID:              groups[4].ID,
					GroupID:         groups[4].ID,
					Tenant:          &identity.Tenant{Id: tenants[1].ID},
					Name:            groups[2].Name,
					ExpectedMessage: "cdm-cloud.identity.update_group.success",
				},
				{
					Desc:            "update deleted group name in different tenant",
					ID:              groups[4].ID,
					GroupID:         groups[4].ID,
					Tenant:          &identity.Tenant{Id: tenants[1].ID},
					Name:            groups[3].Name,
					ExpectedMessage: "cdm-cloud.identity.update_group.success",
				},
			} {
				ctx := setCtx(&admin, tc.Tenant.Id)

				var rsp = new(identity.GroupResponse)
				var req = new(identity.UpdateGroupRequest)
				req.Group = new(identity.Group)
				req.GroupId = tc.ID
				req.Group.Id = tc.GroupID
				req.Group.Tenant = tc.Tenant
				req.Group.Name = tc.Name
				req.Group.Remarks = tc.Remarks

				if err := handler.UpdateGroup(ctx, req, rsp); err != nil {
					assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code)
				} else {
					assert.Equal(t, tc.ExpectedMessage, rsp.Message.Code, tc.Desc)
				}
			}
		})
	})
}

func TestUpdateGroup(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		var groups []*model.Group
		groups = append(groups, &model.Group{Name: uuid.New().String()[:30], TenantID: tenant.ID})
		groups = append(groups, &model.Group{Name: uuid.New().String()[:30], TenantID: tenant.ID})

		for _, g := range groups {
			if err := db.Save(g).Error; err != nil {
				panic(err)
			}
		}

		admin := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		for _, tc := range []struct {
			Desc       string
			ID         uint64
			Tenant     *identity.Tenant
			TenantName string
			GroupID    uint64
			Name       string
			Remarks    string
			Error      error
		}{
			{
				Desc:       "normal case",
				ID:         groups[0].ID,
				GroupID:    groups[0].ID,
				Tenant:     &identity.Tenant{Id: tenant.ID},
				TenantName: tenant.Name,
				Name:       "Test",
				Remarks:    uuid.New().String()[:30],
				Error:      nil,
			},
			{
				Desc:    "abnormal case: group name conflict",
				ID:      groups[1].ID,
				GroupID: groups[1].ID,
				Tenant:  &identity.Tenant{Id: tenant.ID},
				Name:    "Test",
				Error:   errors.Conflict(constant.ServiceIdentity, "conflict parameter"),
			},
			{
				Desc:    "abnormal case: group id mismatch",
				ID:      groups[1].ID,
				GroupID: groups[0].ID,
				Tenant:  &identity.Tenant{Id: tenant.ID},
				Name:    "Test",
				Error:   errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:    "abnormal case: unknown group id",
				ID:      uint64(999999999),
				GroupID: uint64(999999999),
				Tenant:  &identity.Tenant{Id: tenant.ID},
				Name:    uuid.New().String()[:30],
				Error:   errors.NotFound(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: length of group name is 0",
				ID:         groups[0].ID,
				GroupID:    groups[0].ID,
				Tenant:     &identity.Tenant{Id: tenant.ID},
				TenantName: tenant.Name,
				Name:       "",
				Remarks:    uuid.New().String()[:30],
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: length of group name over max size",
				ID:         groups[0].ID,
				GroupID:    groups[0].ID,
				Tenant:     &identity.Tenant{Id: tenant.ID},
				TenantName: tenant.Name,
				Name:       generateString(nameLength + 1),
				Remarks:    uuid.New().String()[:30],
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: length of remarks over max size",
				ID:         groups[0].ID,
				GroupID:    groups[0].ID,
				Tenant:     &identity.Tenant{Id: tenant.ID},
				TenantName: tenant.Name,
				Name:       uuid.New().String()[:30],
				Remarks:    generateString(remarksLength + 1),
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: tenant id is mismatch",
				ID:         groups[0].ID,
				GroupID:    groups[0].ID,
				Tenant:     &identity.Tenant{Id: 999999999},
				TenantName: tenant.Name,
				Name:       uuid.New().String()[:30],
				Remarks:    generateString(remarksLength + 1),
				Error:      errors.NotFound(constant.ServiceIdentity, "unchangeable parameter"),
			},
		} {
			{
				ctx := setCtx(&admin, tc.Tenant.Id)

				var rsp = new(identity.GroupResponse)
				var req = new(identity.UpdateGroupRequest)
				req.Group = new(identity.Group)
				req.GroupId = tc.ID
				req.Group.Id = tc.GroupID
				req.Group.Tenant = tc.Tenant
				req.Group.Name = tc.Name
				req.Group.Remarks = tc.Remarks

				err := handler.UpdateGroup(ctx, req, rsp)
				if tc.Error == nil {
					assert.NoError(t, err, tc.Desc)
					assert.Equal(t, tc.Tenant.Id, rsp.Group.Tenant.Id)
					assert.Equal(t, tc.TenantName, rsp.Group.Tenant.Name)
					assert.Equal(t, tc.Name, rsp.Group.Name, tc.Desc)
					assert.Equal(t, tc.Remarks, rsp.Group.Remarks, tc.Desc)
				} else {
					assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
				}
			}
		}
	})
}

func TestUpdateGroupWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		var groups []*model.Group
		groups = append(groups, &model.Group{Name: uuid.New().String()[:30], TenantID: tenant.ID})
		groups = append(groups, &model.Group{Name: uuid.New().String()[:30], TenantID: tenant.ID})

		for _, g := range groups {
			if err := db.Save(g).Error; err != nil {
				panic(err)
			}
		}

		var tmp model.Role

		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		authManager := identity.User{
			Tenant: &identity.Tenant{Id: tenant.ID},
		}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		authManager.Roles = append(authManager.Roles, r)

		unauthManager := identity.User{
			Tenant: &identity.Tenant{Id: 9999999999},
		}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		unauthManager.Roles = append(unauthManager.Roles, r)

		for _, tc := range []struct {
			Desc        string
			ID          uint64
			GroupID     uint64
			Name        string
			GroupTenant *identity.Tenant
			TenantName  string
			ReqUser     *identity.User
			ReqTenantID uint64
			Error       error
		}{
			{
				Desc:        "normal case: admin has sent a request to update a group",
				ID:          groups[0].ID,
				GroupID:     groups[0].ID,
				Name:        "Test",
				GroupTenant: &identity.Tenant{Id: tenant.ID},
				TenantName:  tenant.Name,
				ReqTenantID: tenant.ID,
				ReqUser:     &admin,
				Error:       nil,
			},
			{
				Desc:        "normal case: authorized manager has sent a request to update a group",
				ID:          groups[0].ID,
				GroupID:     groups[0].ID,
				Name:        "Test",
				GroupTenant: &identity.Tenant{Id: tenant.ID},
				TenantName:  tenant.Name,
				ReqTenantID: tenant.ID,
				ReqUser:     &authManager,
				Error:       nil,
			},
			{
				Desc:        "abnormal case: wrong tenant id (unchangeable parameter)",
				ID:          groups[0].ID,
				GroupID:     groups[0].ID,
				Name:        "Test",
				GroupTenant: &identity.Tenant{Id: tenant.ID},
				ReqTenantID: 9999999999,
				ReqUser:     &unauthManager,
				Error:       errors.NotFound(constant.ServiceIdentity, "not found group"),
			},
		} {
			{
				ctx := setCtx(tc.ReqUser, tc.ReqTenantID)

				var rsp = new(identity.GroupResponse)
				var req = new(identity.UpdateGroupRequest)
				req.Group = new(identity.Group)
				req.GroupId = tc.ID
				req.Group.Id = tc.GroupID
				req.Group.Name = tc.Name
				req.Group.Tenant = tc.GroupTenant

				err := handler.UpdateGroup(ctx, req, rsp)
				if tc.Error == nil {
					assert.NoError(t, err, tc.Desc)
					assert.Equal(t, tc.Name, rsp.Group.Name, tc.Desc)
					assert.Equal(t, tc.GroupTenant.Id, rsp.Group.Tenant.Id)
					assert.Equal(t, tc.TenantName, rsp.Group.Tenant.Name)
				} else {
					assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
				}
			}
		}
	})
}

func TestGetGroup(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		group := model.Group{
			Name:     uuid.New().String()[:30],
			Remarks:  new(string),
			TenantID: tenant.ID,
		}
		*group.Remarks = uuid.New().String()[:30]

		if err := db.Save(&group).Error; err != nil {
			panic(err)
		}

		reqUser := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		reqUser.Roles = append(reqUser.Roles, r)

		for _, tc := range []struct {
			Desc  string
			ID    uint64
			Error error
		}{
			{
				Desc:  "normal case",
				ID:    group.ID,
				Error: nil,
			},
			{
				Desc:  "abnormal case: group id is 0",
				ID:    0,
				Error: errors.BadRequest(constant.ServiceIdentity, "required parameter"),
			},
			{
				Desc:  "abnormal case: unknown group id",
				ID:    uint64(999999999),
				Error: errors.NotFound(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			ctx := setCtx(&reqUser, tenant.ID)

			var rsp = new(identity.GroupResponse)

			err := handler.GetGroup(ctx, &identity.GetGroupRequest{GroupId: tc.ID}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, group.ID, rsp.Group.Id, tc.Desc)
				assert.Equal(t, tenant.ID, rsp.Group.Tenant.Id)
				assert.Equal(t, tenant.Name, rsp.Group.Tenant.Name)
				assert.Equal(t, group.Name, rsp.Group.Name, tc.Desc)
				assert.Equal(t, *group.Remarks, rsp.Group.Remarks, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetGroupWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		group := model.Group{
			Name:     uuid.New().String()[:30],
			Remarks:  new(string),
			TenantID: tenant.ID,
		}
		*group.Remarks = uuid.New().String()[:30]
		if err := db.Save(&group).Error; err != nil {
			panic(err)
		}

		var tmp model.Role

		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		authManager := identity.User{
			Tenant: &identity.Tenant{Id: tenant.ID},
		}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		authManager.Roles = append(authManager.Roles, r)

		unauthManager := identity.User{
			Tenant: &identity.Tenant{Id: 99999999},
		}
		unauthManager.Roles = append(unauthManager.Roles, &identity.Role{Solution: "unknown", Role: constant.Manager})

		user := identity.User{}
		user.Roles = append(user.Roles, &identity.Role{Solution: constant.SolutionName, Role: ""})

		for _, tc := range []struct {
			Desc    string
			ID      uint64
			ReqUser *identity.User
			Error   error
		}{
			{
				Desc:    "normal case",
				ID:      group.ID,
				ReqUser: &admin,
				Error:   nil,
			},
			{
				Desc:    "normal case: auth manager get group",
				ID:      group.ID,
				ReqUser: &authManager,
				Error:   nil,
			},
		} {
			ctx := setCtx(tc.ReqUser, tenant.ID)

			var rsp = new(identity.GroupResponse)

			err := handler.GetGroup(ctx, &identity.GetGroupRequest{GroupId: tc.ID}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, group.ID, rsp.Group.Id, tc.Desc)
				assert.Equal(t, group.TenantID, rsp.Group.Tenant.Id)
				assert.Equal(t, tenant.Name, rsp.Group.Tenant.Name)
				assert.Equal(t, group.Name, rsp.Group.Name, tc.Desc)
				assert.Equal(t, *group.Remarks, rsp.Group.Remarks, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetGroupDeleted(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		group := model.Group{
			Name:     uuid.New().String()[:30],
			Remarks:  new(string),
			TenantID: tenant.ID,
		}
		*group.Remarks = uuid.New().String()[:30]

		if err := db.Save(&group).Error; err != nil {
			panic(err)
		}

		reqUser := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		reqUser.Roles = append(reqUser.Roles, r)

		for _, tc := range []struct {
			Desc  string
			ID    uint64
			Error error
		}{
			{
				Desc:  "deleted case",
				ID:    group.ID,
				Error: errors.NotFound(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			ctx := setCtx(&reqUser, tenant.ID)

			var rsp = new(identity.GroupResponse)

			err := handler.DeleteGroup(ctx, &identity.DeleteGroupRequest{GroupId: tc.ID}, &identity.MessageResponse{})
			err = handler.GetGroup(ctx, &identity.GetGroupRequest{GroupId: tc.ID}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, group.ID, rsp.Group.Id, tc.Desc)
				assert.Equal(t, tenant.ID, rsp.Group.Tenant.Id)
				assert.Equal(t, tenant.Name, rsp.Group.Tenant.Name)
				assert.Equal(t, group.Name, rsp.Group.Name, tc.Desc)
				assert.Equal(t, *group.Remarks, rsp.Group.Remarks, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestDeleteGroup(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		group := &model.Group{Name: uuid.New().String()[:30], TenantID: tenant.ID}
		if err := db.Save(group).Error; err != nil {
			panic(err)
		}

		user := model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		}

		if err := db.Save(&user).Error; err != nil {
			panic(err)
		}

		userGroup := model.UserGroup{
			UserID:  user.ID,
			GroupID: group.ID,
		}
		if err := db.Save(&userGroup).Error; err != nil {
			panic(err)
		}

		reqUser := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		reqUser.Roles = append(reqUser.Roles, r)

		for _, tc := range []struct {
			Desc  string
			ID    uint64
			Error error
		}{
			{
				Desc:  "normal case",
				ID:    group.ID,
				Error: nil,
			},
			{
				Desc:  "abnormal case: group id is 0",
				ID:    0,
				Error: errors.BadRequest(constant.ServiceIdentity, "required parameter"),
			},
			{
				Desc:  "abnormal case: unknown group id",
				ID:    uint64(999999999),
				Error: errors.NotFound(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			ctx := setCtx(&reqUser, tenant.ID)

			err := handler.DeleteGroup(ctx, &identity.DeleteGroupRequest{GroupId: tc.ID}, &identity.MessageResponse{})

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestDeleteGroupAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		var groups []*model.Group
		groups = append(groups, &model.Group{Name: uuid.New().String()[:30], TenantID: tenant.ID})
		groups = append(groups, &model.Group{Name: uuid.New().String()[:30], TenantID: tenant.ID})
		groups = append(groups, &model.Group{Name: uuid.New().String()[:30], TenantID: tenant.ID})

		for _, g := range groups {
			if err := db.Save(g).Error; err != nil {
				panic(err)
			}
		}

		var tmp model.Role

		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		authManager := identity.User{
			Tenant: &identity.Tenant{Id: tenant.ID},
		}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		authManager.Roles = append(authManager.Roles, r)

		unauthManager := identity.User{
			Tenant: &identity.Tenant{Id: 99999999},
		}
		unauthManager.Roles = append(unauthManager.Roles, &identity.Role{Solution: "unknown", Role: constant.Manager})

		unauthUser := identity.User{}
		unauthUser.Roles = append(unauthUser.Roles, &identity.Role{Solution: constant.SolutionName, Role: ""})

		for _, tc := range []struct {
			Desc    string
			ID      uint64
			ReqUser *identity.User
			Error   error
		}{
			{
				Desc:    "normal case: Request delete group from admin",
				ID:      groups[0].ID,
				ReqUser: &admin,
				Error:   nil,
			},
			{
				Desc:    "normal case: Request delete group from authorized group",
				ID:      groups[1].ID,
				ReqUser: &authManager,
				Error:   nil,
			},
		} {
			ctx := setCtx(tc.ReqUser, tenant.ID)

			err := handler.DeleteGroup(ctx, &identity.DeleteGroupRequest{GroupId: tc.ID}, &identity.MessageResponse{})
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestSetGroupUsers(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var tenants []*model.Tenant
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30]})
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30]})
		for _, t := range tenants {
			if err := db.Save(t).Error; err != nil {
				panic(err)
			}
		}

		group := model.Group{
			Name:     uuid.New().String()[:30],
			TenantID: tenants[0].ID,
		}
		if err := db.Save(&group).Error; err != nil {
			panic(err)
		}

		var users []*model.User
		hashed := sha256.Sum256([]byte("Password!1"))
		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenants[0].ID,
			Password: hex.EncodeToString(hashed[:]),
		})

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenants[0].ID,
			Password: hex.EncodeToString(hashed[:]),
		})

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenants[1].ID,
			Password: hex.EncodeToString(hashed[:]),
		})
		for _, u := range users {
			if err := db.Save(u).Error; err != nil {
				panic(err)
			}
		}

		if err := db.Save(&model.UserGroup{UserID: users[0].ID, GroupID: group.ID}).Error; err != nil {
			panic(err)
		}

		var rspUsers []*identity.User
		b, _ := json.Marshal(users)
		_ = json.Unmarshal(b, &rspUsers)

		var tmp model.Role
		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		for _, tc := range []struct {
			Desc         string
			GroupID      uint64
			TenantID     uint64
			ReqUsers     []*identity.User
			ExpectdUsers []*identity.User
			Error        error
		}{
			{
				Desc:         "normal case 1",
				GroupID:      group.ID,
				TenantID:     tenants[0].ID,
				ReqUsers:     []*identity.User{{Id: users[0].ID}, {Id: users[1].ID}},
				ExpectdUsers: rspUsers[:2],
				Error:        nil,
			},
			{
				Desc:     "normal case 2",
				GroupID:  group.ID,
				TenantID: tenants[0].ID,
				Error:    nil,
			},
			{
				Desc:     "abnormal case: group id is 0",
				GroupID:  0,
				TenantID: tenants[0].ID,
				Error:    errors.BadRequest(constant.ServiceIdentity, "required parameter"),
			},
			{
				Desc:     "abnormal case not found group",
				GroupID:  999999999,
				TenantID: tenants[0].ID,
				Error:    errors.NotFound(constant.ServiceIdentity, "not found group"),
			},
			{
				Desc:     "abnormal case different group tenant id and metadata tenant id",
				GroupID:  group.ID,
				TenantID: 999999999,
				Error:    errors.Forbidden(constant.ServiceIdentity, "unauthorized user"),
			},
			{
				Desc:     "abnormal case not found user",
				GroupID:  group.ID,
				TenantID: tenants[0].ID,
				ReqUsers: []*identity.User{{Id: 9999999999}},
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:     "abnormal case different user tenant id",
				GroupID:  group.ID,
				TenantID: tenants[0].ID,
				ReqUsers: []*identity.User{{Id: users[0].ID}, {Id: users[1].ID}, {Id: users[2].ID}},
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:     "abnormal case unknown user id",
				GroupID:  group.ID,
				TenantID: tenants[0].ID,
				ReqUsers: []*identity.User{{Id: users[0].ID}, {}},
				Error:    errors.BadRequest(constant.ServiceIdentity, "required parameter"),
			},
			{
				Desc:     "abnormal case user id parameter is missing",
				GroupID:  group.ID,
				TenantID: tenants[0].ID,
				ReqUsers: []*identity.User{{Name: users[0].Name}},
				Error:    errors.BadRequest(constant.ServiceIdentity, "required parameter"),
			},
		} {
			ctx := setCtx(&admin, tc.TenantID)

			var rsp = new(identity.UsersResponse)

			err := handler.SetGroupUsers(ctx, &identity.SetGroupUsersRequest{
				GroupId: tc.GroupID,
				Users:   tc.ReqUsers,
			}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err)
				assert.Equal(t, len(tc.ExpectdUsers), len(rsp.Users))
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestSetGroupUsersWithDeletedGroup(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var tenants []*model.Tenant
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30]})
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30]})
		for _, t := range tenants {
			if err := db.Save(t).Error; err != nil {
				panic(err)
			}
		}

		group := model.Group{
			Name:     uuid.New().String()[:30],
			TenantID: tenants[0].ID,
		}
		if err := db.Save(&group).Error; err != nil {
			panic(err)
		}

		deletedGroup := model.Group{
			Name:        uuid.New().String()[:30],
			TenantID:    tenants[1].ID,
			DeletedFlag: true,
		}
		if err := db.Save(&deletedGroup).Error; err != nil {
			panic(err)
		}

		var users []*model.User
		hashed := sha256.Sum256([]byte("Password!1"))
		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenants[0].ID,
			Password: hex.EncodeToString(hashed[:]),
		})

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenants[1].ID,
			Password: hex.EncodeToString(hashed[:]),
		})
		for _, u := range users {
			if err := db.Save(u).Error; err != nil {
				panic(err)
			}
		}

		if err := db.Save(&model.UserGroup{UserID: users[0].ID, GroupID: group.ID}).Error; err != nil {
			panic(err)
		}
		if err := db.Save(&model.UserGroup{UserID: users[1].ID, GroupID: deletedGroup.ID}).Error; err != nil {
			panic(err)
		}

		var rspUsers []*identity.User
		b, _ := json.Marshal(users)
		_ = json.Unmarshal(b, &rspUsers)

		var tmp model.Role
		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		for _, tc := range []struct {
			Desc         string
			GroupID      uint64
			TenantID     uint64
			ReqUsers     []*identity.User
			ExpectdUsers []*identity.User
			Error        error
		}{
			{
				Desc:         "normal case 1",
				GroupID:      group.ID,
				TenantID:     tenants[0].ID,
				ReqUsers:     []*identity.User{{Id: users[0].ID}},
				ExpectdUsers: rspUsers[:1],
				Error:        nil,
			},
			{
				Desc:     "abnormal case not found group",
				GroupID:  deletedGroup.ID,
				TenantID: tenants[1].ID,
				Error:    errors.NotFound(constant.ServiceIdentity, "not found group"),
			},
		} {
			ctx := setCtx(&admin, tc.TenantID)

			var rsp = new(identity.UsersResponse)

			err := handler.SetGroupUsers(ctx, &identity.SetGroupUsersRequest{
				GroupId: tc.GroupID,
				Users:   tc.ReqUsers,
			}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err)
				assert.Equal(t, len(tc.ExpectdUsers), len(rsp.Users))
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetRoles(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var roles []*model.Role
		var solutions []*model.TenantSolution

		tenant := model.Tenant{
			Name: uuid.New().String()[:30],
		}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		solutions = append(solutions, &model.TenantSolution{TenantID: tenant.ID, Solution: constant.SolutionName})
		solutions = append(solutions, &model.TenantSolution{TenantID: tenant.ID, Solution: "CDM-DisasterRecovery"})

		for _, s := range solutions {
			if err := db.Save(&s).Error; err != nil {
				panic(err)
			}
		}

		roles = append(roles, &model.Role{Solution: constant.SolutionName, Role: "Viewer"})
		roles = append(roles, &model.Role{Solution: "CDM-DisasterRecovery", Role: constant.Manager})
		roles = append(roles, &model.Role{Solution: "CDM-DisasterRecovery", Role: "Viewer"})

		for _, r := range roles {
			if err := db.Save(r).Error; err != nil {
				panic(err)
			}
		}

		for _, tc := range []struct {
			Desc     string
			Role     string
			Solution string
			Expected int
			Error    error
		}{
			{
				Desc:     "normal case : all search",
				Expected: 7, // 미리 추가된 2개의 role 포함
				Error:    nil,
			},
			{
				Desc:     "normal case : Solution filter",
				Solution: "CDM-DisasterRecovery",
				Expected: 2,
				Error:    nil,
			},
			{
				Desc:     "normal case : Role filter",
				Role:     "manager",
				Expected: 2,
				Error:    nil,
			},
			{
				Desc:     "abnormal case : unknown Name",
				Solution: "Operator",
				Expected: 0,
				Error:    errors.New(constant.ServiceIdentity, "no contents", 204),
			},
			{
				Desc:     "abnormal case : unknown Name",
				Solution: generateString(solutionLength + 1),
				Expected: 0,
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:     "abnormal case : invalid role(unknown)",
				Role:     "developer",
				Expected: 0,
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:     "abnormal case : invalid role(length of role over max size)",
				Role:     generateString(roleLength + 1),
				Expected: 0,
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			var rsp = new(identity.RolesResponse)

			admin := identity.User{}
			tmp := Role[constant.Admin]
			r, _ := roleModelToRsp(&tmp)
			admin.Roles = append(admin.Roles, r)

			ctx := context.Background()
			b, _ := json.Marshal(&admin)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))
			ctx = metadata.Set(ctx, commonmeta.HeaderTenantID, strconv.Itoa(int(tenant.ID)))
			err := handler.GetRoles(
				ctx,
				&identity.GetRolesRequest{Solution: tc.Solution, Role: tc.Role},
				rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.Expected, len(rsp.Roles), tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetRolesWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var roles []*model.Role
		var solutions []*model.TenantSolution
		var tenants []*model.Tenant

		tenants = append(tenants, &model.Tenant{
			Name: uuid.New().String()[:30],
		})
		tenants = append(tenants, &model.Tenant{
			Name: uuid.New().String()[:30],
		})

		for _, tenant := range tenants {
			if err := db.Save(&tenant).Error; err != nil {
				panic(err)
			}
		}

		solutions = append(solutions, &model.TenantSolution{TenantID: tenants[0].ID, Solution: constant.SolutionName})
		solutions = append(solutions, &model.TenantSolution{TenantID: tenants[1].ID, Solution: "cdm-dr"})

		for _, s := range solutions {
			if err := db.Save(&s).Error; err != nil {
				panic(err)
			}
		}

		roles = append(roles, &model.Role{Solution: constant.SolutionName, Role: "Viewer"})
		roles = append(roles, &model.Role{Solution: "cdm-dr", Role: constant.Manager})

		for _, r := range roles {
			if err := db.Save(r).Error; err != nil {
				panic(err)
			}
		}

		admin := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		manager := identity.User{
			Tenant: &identity.Tenant{Id: tenants[0].ID},
		}

		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		manager.Roles = append(manager.Roles, r)

		for _, tc := range []struct {
			Desc     string
			Role     string
			Solution string
			ReqUser  *identity.User
			TenantID uint64
			Expected int
			Error    error
		}{
			{
				Desc:     "normal case : all search",
				Expected: 5, // 미리 추가된 2개의 role 포함
				ReqUser:  &admin,
				TenantID: tenants[0].ID,
				Error:    nil,
			},
			{
				Desc:     "normal case : all search",
				Expected: 1, // 미리 추가된 2개의 role 포함
				ReqUser:  &admin,
				TenantID: tenants[1].ID,
				Error:    nil,
			},
			{
				Desc:     "normal case : all search",
				Expected: 5,
				ReqUser:  &manager,
				TenantID: tenants[0].ID,
				Error:    nil,
			},
		} {
			var rsp = new(identity.RolesResponse)

			ctx := setCtx(&admin, tc.TenantID)

			err := handler.GetRoles(
				ctx,
				&identity.GetRolesRequest{Solution: tc.Solution, Role: tc.Role},
				rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.Expected, len(rsp.Roles), tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestLogin(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var sessions []string
		var users []*model.User
		var group *model.Group
		var role *model.Role

		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		if err := db.Save(&model.TenantConfig{TenantID: tenant.ID, Key: config.UserLoginRestrictionEnable, Value: "false"}).Error; err != nil {
			panic(err)
		}

		hashed := sha256.Sum256([]byte("Password!1"))

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
			Password: hex.EncodeToString(hashed[:]),
		})

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
			Password: hex.EncodeToString(hashed[:]),
		})

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
			Password: hex.EncodeToString(hashed[:]),
		})

		group = &model.Group{
			Name:     "Team1",
			TenantID: tenant.ID,
		}

		role = &model.Role{
			Solution: "CDM-DisasterRecovery",
			Role:     "Developer",
		}

		var (
			rspUserGroup identity.Group
			rspUserRole  identity.Role
			b            []byte
		)
		for _, u := range users {
			if err := db.Save(u).Error; err != nil {
				panic(err)
			}
		}

		if err := db.Save(group).Error; err != nil {
			panic(err)
		}
		b, _ = json.Marshal(group)
		json.Unmarshal(b, &rspUserGroup)

		if err := db.Save(role).Error; err != nil {
			panic(err)
		}
		b, _ = json.Marshal(role)
		json.Unmarshal(b, &rspUserRole)

		if err := db.Save(&model.UserGroup{UserID: users[1].ID, GroupID: group.ID}).Error; err != nil {
			panic(err)
		}

		if err := db.Save(&model.UserRole{UserID: users[1].ID, RoleID: role.ID}).Error; err != nil {
			panic(err)
		}

		for _, tc := range []struct {
			Desc               string
			Account            string
			Password           string
			ExpectedUserGroups []*identity.Group
			ExpectedUserRoles  []*identity.Role
			Error              error
		}{
			{
				Desc:     "normal case",
				Account:  users[0].Account,
				Password: users[0].Password,
				Error:    nil,
			},
			{
				Desc:               "normal case",
				Account:            users[1].Account,
				Password:           users[1].Password,
				ExpectedUserGroups: []*identity.Group{&rspUserGroup},
				ExpectedUserRoles:  []*identity.Role{&rspUserRole},
				Error:              nil,
			},
			{
				Desc:     "abnormal case : already logged in",
				Account:  users[0].Account,
				Password: users[0].Password,
				Error:    errors.Conflict(constant.ServiceIdentity, "already logged in"),
			},
			{
				Desc:     "abnormal case: password mismatch",
				Account:  users[2].Account,
				Password: "testestestestestest",
				Error:    errors.Unauthorized(constant.ServiceIdentity, "unauthenticated"),
			},
			{
				Desc:     "abnormal case: unknown user id",
				Account:  "Test",
				Password: users[2].Password,
				Error:    errors.Unauthorized(constant.ServiceIdentity, "unauthenticated"),
			},
			{
				Desc:     "abnormal case: length of account is 0",
				Account:  "",
				Password: users[2].Password,
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:     "abnormal case: length of account over max size",
				Account:  generateString(accountLength + 1),
				Password: users[2].Password,
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:     "abnormal case: length of password is 0",
				Account:  users[2].Account,
				Password: "",
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:     "abnormal case: length of password over max size",
				Account:  users[2].Account,
				Password: generateString(passwordLength + 1),
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			var rsp = new(identity.UserResponse)
			ctx := context.Background()
			ctx = metadata.Set(ctx, commonmeta.HeaderClientIP, "192.168.1.1")

			err := handler.Login(ctx, &identity.LoginRequest{Account: tc.Account, Password: tc.Password, Force: false}, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.Account, rsp.User.Account)
				assert.NotEmpty(t, rsp.User.Session.Key)
				assert.Equal(t, tc.ExpectedUserGroups, rsp.User.Groups)
				assert.Equal(t, tc.ExpectedUserRoles, rsp.User.Roles)
				if err == nil {
					sessions = append(sessions, rsp.User.Session.Key)
				}
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestForceLogin(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var sessions []string
		var err error

		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err = db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		if err = db.Save(&model.TenantConfig{TenantID: tenant.ID, Key: config.UserLoginRestrictionEnable, Value: "false"}).Error; err != nil {
			panic(err)
		}

		user := model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		}
		hashed := sha256.Sum256([]byte("Password!1"))
		user.Password = hex.EncodeToString(hashed[:])

		if err = db.Save(&user).Error; err != nil {
			panic(err)
		}

		var privateKey *rsa.PrivateKey
		if privateKey, err = getPrivateKeyFromFile(defaultPrivateKeyPath + "identity.pem"); err != nil {
			panic(err)
		}

		ctx := context.Background()
		ctx = metadata.Set(ctx, commonmeta.HeaderClientIP, "192.168.1.1")

		if _, err = newSession(ctx, db, user.TenantID, user.ID, privateKey, false); err != nil {
			panic(err)
		}

		for _, tc := range []struct {
			Desc     string
			Account  string
			Password string
			Force    bool
			Error    error
		}{
			{
				Desc:     "already log in",
				Account:  user.Account,
				Password: user.Password,
				Force:    false,
				Error:    errors.Conflict(constant.ServiceIdentity, "already logged in"),
			},
			{
				Desc:     "force logout",
				Account:  user.Account,
				Password: user.Password,
				Force:    true,
				Error:    nil,
			},
		} {
			var rsp = new(identity.UserResponse)

			b, _ := json.Marshal(user)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			err = handler.Login(ctx, &identity.LoginRequest{Account: tc.Account, Password: tc.Password, Force: tc.Force}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.Account, rsp.User.Account)
				assert.NotEmpty(t, rsp.User.Session.Key)
				if err == nil {
					sessions = append(sessions, rsp.User.Session.Key)
				}
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestLoginRestriction(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		user := model.User{
			Account:              uuid.New().String()[:30],
			Name:                 uuid.New().String()[:30],
			LastLoginFailedCount: new(uint),
			LastLoginFailedAt:    new(int64),
			TenantID:             tenant.ID,
		}
		hashed := sha256.Sum256([]byte("Password!1"))
		user.Password = hex.EncodeToString(hashed[:])

		for _, tc := range []struct {
			Desc                 string
			LastLoginFailedCount uint
			LastLoginFailedAt    int64
			RestrictionInfo      map[string]string
			RestrictionEnable    string
			RestrictionTryCount  string
			RestrictionTime      string
			Error                error
		}{
			{
				Desc:                 "normal case 1",
				LastLoginFailedCount: 0,
				LastLoginFailedAt:    0,
				RestrictionInfo:      map[string]string{config.UserLoginRestrictionEnable: "false"},
				Error:                nil,
			},
			{
				Desc:                 "normal case 2",
				LastLoginFailedCount: 0,
				LastLoginFailedAt:    0,
				RestrictionInfo: map[string]string{config.UserLoginRestrictionEnable: "true",
					config.UserLoginRestrictionTryCount: "5", config.UserLoginRestrictionTime: "60"},
				Error: nil,
			},
			{
				Desc:                 "normal case 3",
				LastLoginFailedCount: 5,
				LastLoginFailedAt:    time.Now().Unix(),
				RestrictionInfo:      map[string]string{config.UserLoginRestrictionEnable: "false"},
				Error:                nil,
			},
			{
				Desc:                 "normal case 4",
				LastLoginFailedCount: 5,
				LastLoginFailedAt:    time.Now().Unix(),
				RestrictionInfo: map[string]string{config.UserLoginRestrictionEnable: "false",
					config.UserLoginRestrictionTryCount: "5", config.UserLoginRestrictionTime: "60"},
				Error: nil,
			},
			{
				Desc:                 "abnormal case 5",
				LastLoginFailedCount: 5,
				LastLoginFailedAt:    time.Now().Unix(),
				RestrictionInfo: map[string]string{config.UserLoginRestrictionEnable: "true",
					config.UserLoginRestrictionTryCount: "5", config.UserLoginRestrictionTime: "60"},
				Error: errors.Unauthorized(constant.ServiceIdentity, "restricted account"),
			},
			{
				Desc:                 "abnormal case 6",
				LastLoginFailedCount: 5,
				LastLoginFailedAt:    time.Now().Unix() - 70,
				RestrictionInfo: map[string]string{config.UserLoginRestrictionEnable: "true",
					config.UserLoginRestrictionTryCount: "5", config.UserLoginRestrictionTime: "60"},
				Error: nil,
			},
			{
				Desc:                 "abnormal case 7",
				LastLoginFailedCount: 6,
				LastLoginFailedAt:    time.Now().Unix(),
				RestrictionInfo: map[string]string{config.UserLoginRestrictionEnable: "true",
					config.UserLoginRestrictionTryCount: "5", config.UserLoginRestrictionTime: "60"},
				Error: nil,
			},
			{
				Desc:                 "abnormal case 7",
				LastLoginFailedCount: 10,
				LastLoginFailedAt:    time.Now().Unix(),
				RestrictionInfo: map[string]string{config.UserLoginRestrictionEnable: "true",
					config.UserLoginRestrictionTryCount: "5", config.UserLoginRestrictionTime: "60"},
				Error: errors.Unauthorized(constant.ServiceIdentity, "restricted account"),
			},
		} {
			for k, v := range tc.RestrictionInfo {
				db.Save(&model.TenantConfig{TenantID: tenant.ID, Key: k, Value: v})
			}

			*user.LastLoginFailedCount = tc.LastLoginFailedCount
			*user.LastLoginFailedAt = tc.LastLoginFailedAt
			db.Save(&user)

			ctx := context.Background()
			ctx = metadata.Set(ctx, commonmeta.HeaderClientIP, "192.168.1.1")

			b, _ := json.Marshal(user)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var rsp = new(identity.UserResponse)

			err := handler.Login(ctx, &identity.LoginRequest{Account: user.Account, Password: user.Password, Force: false}, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, user.Account, rsp.User.Account)
				assert.NotEmpty(t, rsp.User.Session.Key)

				storeKey := storeKeyPrefix + "." + strconv.FormatUint(user.ID, 10)
				store.Delete(storeKey)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestPasswordUpdateFlag(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		if err := db.Save(&model.TenantConfig{TenantID: tenant.ID, Key: config.UserLoginRestrictionEnable, Value: "false"}).Error; err != nil {
			panic(err)
		}

		user := model.User{
			Account:              uuid.New().String()[:30],
			Name:                 uuid.New().String()[:30],
			LastLoginFailedCount: new(uint),
			LastLoginFailedAt:    new(int64),
			TenantID:             tenant.ID,
		}
		hashed := sha256.Sum256([]byte("Password!1"))
		user.Password = hex.EncodeToString(hashed[:])
		user.PasswordUpdatedAt = new(int64)
		*user.PasswordUpdatedAt = time.Now().Unix() - int64(time.Hour)*25*defaultUserPasswordChangeCycle
		user.PasswordUpdateFlag = new(bool)
		*user.PasswordUpdateFlag = false

		for _, tc := range []struct {
			Desc        string
			ChangeCycle int64
			Expected    bool
		}{
			{
				Desc:        "normal case 1",
				ChangeCycle: 90,
				Expected:    true,
			}, {
				Desc:        "normal case 2",
				ChangeCycle: 0,
				Expected:    false,
			},
		} {
			if err := db.Save(&user).Error; err != nil {
				panic(err)
			}

			if err := db.Save(&model.TenantConfig{TenantID: tenant.ID, Key: config.UserPasswordChangeCycle, Value: strconv.Itoa(int(tc.ChangeCycle))}).Error; err != nil {
				panic(err)
			}

			var rsp = new(identity.UserResponse)

			ctx := context.Background()
			ctx = metadata.Set(ctx, commonmeta.HeaderClientIP, "192.168.1.1")
			b, _ := json.Marshal(user)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			_ = handler.Login(ctx, &identity.LoginRequest{Account: user.Account, Password: user.Password, Force: false}, rsp)

			if tc.Expected == true {
				assert.True(t, rsp.User.PasswordUpdateFlag)
			} else {
				assert.False(t, rsp.User.PasswordUpdateFlag)
			}

			storeKey := storeKeyPrefix + "." + strconv.FormatUint(user.ID, 10)
			store.Delete(storeKey)
		}
	})
}

func TestLogout(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var err error
		var users []*model.User
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		})

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		})

		for _, u := range users {
			if err = db.Save(&u).Error; err != nil {
				panic(err)
			}
		}

		var privateKey *rsa.PrivateKey
		if privateKey, err = getPrivateKeyFromFile(defaultPrivateKeyPath + "identity.pem"); err != nil {
			panic(err)
		}

		ctx := context.Background()
		ctx = metadata.Set(ctx, commonmeta.HeaderClientIP, "192.168.1.1")

		var normalSessionKey, abnormalSessionKey string
		if normalSessionKey, err = newSession(ctx, db, users[0].TenantID, users[0].ID, privateKey, false); err != nil {
			panic(err)
		}

		if abnormalSessionKey, err = newSession(ctx, db, users[1].TenantID, users[1].ID, privateKey, false); err != nil {
			panic(err)
		}

		abnormalStoreKey := storeKeyPrefix + "." + strconv.FormatUint(users[1].ID, 10)
		store.Delete(abnormalStoreKey)

		for _, tc := range []struct {
			Desc       string
			SessionKey string
			Error      error
		}{
			{
				Desc:       "abnormal case: unknown session",
				SessionKey: abnormalSessionKey,
				Error:      errors.BadRequest(constant.ServiceIdentity, "unknown session"),
			},
			{
				Desc:       "abnormal case: empty session",
				SessionKey: "",
				Error:      errors.InternalServerError(constant.ServiceIdentity, "unknown error"),
			},
			{
				Desc:       "abnormal case: invalid session format",
				SessionKey: "aaaaaaaaa",
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid session"),
			},
			{
				Desc:       "normal case",
				SessionKey: normalSessionKey,
				Error:      nil,
			},
		} {
			ctx := context.Background()
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedSession, tc.SessionKey)
			ctx = metadata.Set(ctx, commonmeta.HeaderTenantID, "1")
			b, _ := json.Marshal(users[0])
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))
			err := handler.Logout(ctx, &identity.Empty{}, &identity.MessageResponse{})

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestRevokeSession(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var err error
		var defaultTenant model.Tenant
		fatalIfError(t, db.First(&defaultTenant, model.Tenant{Name: "default"}).Error)
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		var users []*model.User
		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: defaultTenant.ID,
		})

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: defaultTenant.ID,
		})

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		})

		for _, u := range users {
			if err = db.Save(&u).Error; err != nil {
				panic(err)
			}
		}

		var privateKey *rsa.PrivateKey
		if privateKey, err = getPrivateKeyFromFile(defaultPrivateKeyPath + "identity.pem"); err != nil {
			panic(err)
		}

		ctx := context.Background()
		ctx = metadata.Set(ctx, commonmeta.HeaderClientIP, "192.168.1.1")

		var normalSessionKey, abnormalSessionKey, unknownTenantSessionKey string
		if normalSessionKey, err = newSession(ctx, db, users[0].TenantID, users[0].ID, privateKey, false); err != nil {
			panic(err)
		}

		if abnormalSessionKey, err = newSession(ctx, db, users[1].TenantID, users[1].ID, privateKey, false); err != nil {
			panic(err)
		}

		if unknownTenantSessionKey, err = newSession(ctx, db, users[2].TenantID, users[2].ID, privateKey, false); err != nil {
			panic(err)
		}

		abnormalStoreKey := storeKeyPrefix + "." + strconv.FormatUint(users[1].ID, 10)
		store.Delete(abnormalStoreKey)

		for _, tc := range []struct {
			Desc       string
			SessionKey string
			Error      error
		}{
			{
				Desc:       "abnormal case: unknown session",
				SessionKey: abnormalSessionKey,
				Error:      errors.NotFound(constant.ServiceIdentity, "not found session key"),
			},
			{
				Desc:       "abnormal case: empty session",
				SessionKey: "",
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: invalid session format",
				SessionKey: "aaaaaaa",
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "abnormal case: unknown tenant session key",
				SessionKey: unknownTenantSessionKey,
				Error:      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:       "normal case",
				SessionKey: normalSessionKey,
				Error:      nil,
			},
		} {
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedSession, tc.SessionKey)
			ctx = metadata.Set(ctx, commonmeta.HeaderTenantID, "1")
			b, _ := json.Marshal(users[0])
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))
			err = handler.RevokeSession(ctx, &identity.RevokeSessionRequest{SessionKey: tc.SessionKey}, &identity.MessageResponse{})
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestVerifySession(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var err error
		var users []*model.User

		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		})

		users = append(users, &model.User{
			Account:  uuid.New().String()[:30],
			Name:     uuid.New().String()[:30],
			TenantID: tenant.ID,
		})

		for _, u := range users {
			if err = db.Save(&u).Error; err != nil {
				panic(err)
			}
		}

		var privateKey *rsa.PrivateKey
		if privateKey, err = getPrivateKeyFromFile(defaultPrivateKeyPath + "identity.pem"); err != nil {
			panic(err)
		}

		ctx := context.Background()
		ctx = metadata.Set(ctx, commonmeta.HeaderClientIP, "192.168.1.1")

		var normalSessionKey, abnormalSessionKey string
		if normalSessionKey, err = newSession(ctx, db, users[0].TenantID, users[0].ID, privateKey, false); err != nil {
			panic(err)
		}

		if abnormalSessionKey, err = newSession(ctx, db, users[1].TenantID, 999999999, privateKey, false); err != nil {
			panic(err)
		}

		random.Seed(time.Now().UnixNano())

		b, _ := json.Marshal(SessionPayload{
			ID:          users[1].ID,
			CreateDate:  time.Now().Unix(),
			ExpiryDate:  time.Now().Unix() - (40 * 60),
			MagicNumber: random.Uint64(),
			ClientIP:    "192.168.1.1",
		})

		hashed := sha256.Sum256(b)
		sig, err := rsa.SignPSS(rand.Reader, privateKey, crypto.SHA256, hashed[:], nil)
		if err != nil {
			panic(err)
		}

		timeoutSessionKey := fmt.Sprintf("%s.%s",
			base64.URLEncoding.EncodeToString(b),
			base64.URLEncoding.EncodeToString(sig))

		timeoutStoreKey := storeKeyPrefix + "." + strconv.FormatUint(users[1].ID, 10)
		normalStoreKey := storeKeyPrefix + "." + strconv.FormatUint(users[1].ID, 10)
		abnormalStoreKey := storeKeyPrefix + "." + strconv.FormatUint(999999999, 10)

		normalSessionKeyCut := strings.Split(fmt.Sprintf("%v", normalSessionKey), ".")
		abNormalSessionKeyCut := strings.Split(fmt.Sprintf("%v", abnormalSessionKey), ".")

		store.Put(timeoutStoreKey, timeoutSessionKey, store.PutTTL(time.Duration(defaultUserSessionTimeout)*time.Minute))
		store.Delete(abnormalStoreKey)

		for _, tc := range []struct {
			Desc       string
			User       *model.User
			SessionKey string
			ClientIP   string
			Error      error
		}{
			{
				Desc:       "normal case",
				User:       users[0],
				SessionKey: normalSessionKey,
				ClientIP:   "192.168.1.1",
				Error:      nil,
			},
			{
				Desc:       "abnormal case: unknown session",
				User:       users[1],
				SessionKey: abnormalSessionKey,
				ClientIP:   "192.168.1.1",
				Error:      errors.Unauthorized(constant.ServiceIdentity, "unauthenticated request"),
			},
			{
				Desc:       "abnormal case: unverified session",
				User:       users[1],
				SessionKey: normalSessionKeyCut[0] + "." + abNormalSessionKeyCut[1],
				ClientIP:   "192.168.1.1",
				Error:      errors.Unauthorized(constant.ServiceIdentity, "unverified session"),
			},
			{
				Desc:       "abnormal case: timeout session key",
				User:       users[1],
				SessionKey: timeoutSessionKey,
				ClientIP:   "192.168.1.1",
				Error:      errors.Unauthorized(constant.ServiceIdentity, "session expiry"),
			},
			{
				Desc:       "abnormal case: unauthenticated client",
				User:       users[1],
				SessionKey: normalSessionKey,
				ClientIP:   "192.168.1.2",
				Error:      errors.Unauthorized(constant.ServiceIdentity, "unauthenticated request"),
			},
			{
				Desc:       "abnormal case: empty session",
				User:       users[1],
				SessionKey: "",
				ClientIP:   "192.168.1.1",
				Error:      errors.Unauthorized(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			var rsp = new(identity.UserResponse)

			ctx = metadata.Set(ctx, commonmeta.HeaderTenantID, strconv.Itoa(int(tenant.ID)))
			ctx = metadata.Set(ctx, commonmeta.HeaderClientIP, tc.ClientIP)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedSession, tc.SessionKey)
			b, _ = json.Marshal(tc.User)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			err = handler.VerifySession(ctx, &identity.Empty{}, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, rsp.User.Id, users[0].ID, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
		store.Delete(timeoutStoreKey)
		store.Delete(normalStoreKey)
	})
}

func TestSetConfig(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var tenant = &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}

		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		var tmp model.Role

		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		for _, tc := range []struct {
			Desc                         string
			ReqUser                      *identity.User
			Tenant                       uint64
			GlobalTimezone               *wrappers.StringValue
			GlobalLanguageSet            *wrappers.StringValue
			UserLoginRestrictionEnable   *wrappers.BoolValue
			UserLoginRestrictionTryCount *wrappers.UInt64Value
			UserLoginRestrictionTime     *wrappers.Int64Value
			UserReuseOldPassword         *wrappers.BoolValue
			UserPasswordChangeCycle      *wrappers.UInt64Value
			UserSessionTimeout           *wrappers.UInt64Value
			Error                        error
		}{
			{
				Desc:    "abnormal case 1",
				ReqUser: &admin,
				Tenant:  tenant.ID,
				Error:   errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:           "abnormal case 2",
				ReqUser:        &admin,
				Tenant:         tenant.ID,
				GlobalTimezone: &wrappers.StringValue{Value: "datacommand"},
				Error:          errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:           "abnormal case 3",
				ReqUser:        &admin,
				Tenant:         tenant.ID,
				GlobalTimezone: &wrappers.StringValue{Value: "UTC"},
				Error:          errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:              "abnormal case 4",
				ReqUser:           &admin,
				Tenant:            tenant.ID,
				GlobalTimezone:    &wrappers.StringValue{Value: "UTC"},
				GlobalLanguageSet: &wrappers.StringValue{Value: "chinese"},
				Error:             errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:              "abnormal case 5",
				ReqUser:           &admin,
				Tenant:            tenant.ID,
				GlobalTimezone:    &wrappers.StringValue{Value: "UTC"},
				GlobalLanguageSet: &wrappers.StringValue{Value: "eng"},
				Error:             errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:                       "abnormal case 6",
				ReqUser:                    &admin,
				Tenant:                     tenant.ID,
				GlobalTimezone:             &wrappers.StringValue{Value: "UTC"},
				GlobalLanguageSet:          &wrappers.StringValue{Value: "eng"},
				UserLoginRestrictionEnable: &wrappers.BoolValue{Value: true},
				Error:                      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:                       "abnormal case 7",
				ReqUser:                    &admin,
				Tenant:                     tenant.ID,
				GlobalTimezone:             &wrappers.StringValue{Value: "UTC"},
				GlobalLanguageSet:          &wrappers.StringValue{Value: "eng"},
				UserLoginRestrictionEnable: &wrappers.BoolValue{Value: true},
				UserReuseOldPassword:       &wrappers.BoolValue{Value: true},
				Error:                      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:                       "abnormal case 8",
				ReqUser:                    &admin,
				Tenant:                     tenant.ID,
				GlobalTimezone:             &wrappers.StringValue{Value: "UTC"},
				GlobalLanguageSet:          &wrappers.StringValue{Value: "eng"},
				UserLoginRestrictionEnable: &wrappers.BoolValue{Value: true},
				UserReuseOldPassword:       &wrappers.BoolValue{Value: true},
				UserPasswordChangeCycle:    &wrappers.UInt64Value{Value: 190},
				Error:                      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:                       "abnormal case 9",
				ReqUser:                    &admin,
				Tenant:                     tenant.ID,
				GlobalTimezone:             &wrappers.StringValue{Value: "UTC"},
				GlobalLanguageSet:          &wrappers.StringValue{Value: "eng"},
				UserLoginRestrictionEnable: &wrappers.BoolValue{Value: true},
				UserReuseOldPassword:       &wrappers.BoolValue{Value: true},
				UserPasswordChangeCycle:    &wrappers.UInt64Value{Value: 60},
				Error:                      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:                       "abnormal case 10",
				ReqUser:                    &admin,
				Tenant:                     tenant.ID,
				GlobalTimezone:             &wrappers.StringValue{Value: "UTC"},
				GlobalLanguageSet:          &wrappers.StringValue{Value: "eng"},
				UserLoginRestrictionEnable: &wrappers.BoolValue{Value: true},
				UserReuseOldPassword:       &wrappers.BoolValue{Value: true},
				UserPasswordChangeCycle:    &wrappers.UInt64Value{Value: 60},
				UserSessionTimeout:         &wrappers.UInt64Value{Value: 1500},
				Error:                      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:                       "abnormal case 11",
				ReqUser:                    &admin,
				Tenant:                     tenant.ID,
				GlobalTimezone:             &wrappers.StringValue{Value: "UTC"},
				GlobalLanguageSet:          &wrappers.StringValue{Value: "eng"},
				UserLoginRestrictionEnable: &wrappers.BoolValue{Value: true},
				UserLoginRestrictionTime:   &wrappers.Int64Value{Value: 6000},
				UserReuseOldPassword:       &wrappers.BoolValue{Value: true},
				UserPasswordChangeCycle:    &wrappers.UInt64Value{Value: 60},
				UserSessionTimeout:         &wrappers.UInt64Value{Value: 2},
				Error:                      errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:                         "abnormal case 12",
				ReqUser:                      &admin,
				Tenant:                       tenant.ID,
				GlobalTimezone:               &wrappers.StringValue{Value: "UTC"},
				GlobalLanguageSet:            &wrappers.StringValue{Value: "eng"},
				UserLoginRestrictionEnable:   &wrappers.BoolValue{Value: true},
				UserLoginRestrictionTryCount: &wrappers.UInt64Value{Value: 4},
				UserLoginRestrictionTime:     &wrappers.Int64Value{Value: 6000},
				UserReuseOldPassword:         &wrappers.BoolValue{Value: true},
				UserPasswordChangeCycle:      &wrappers.UInt64Value{Value: 60},
				UserSessionTimeout:           &wrappers.UInt64Value{Value: 2},
				Error:                        errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:                         "abnormal case 13",
				ReqUser:                      &admin,
				Tenant:                       tenant.ID,
				GlobalTimezone:               &wrappers.StringValue{Value: "UTC"},
				GlobalLanguageSet:            &wrappers.StringValue{Value: "eng"},
				UserLoginRestrictionEnable:   &wrappers.BoolValue{Value: true},
				UserLoginRestrictionTryCount: &wrappers.UInt64Value{Value: 6},
				UserReuseOldPassword:         &wrappers.BoolValue{Value: true},
				UserPasswordChangeCycle:      &wrappers.UInt64Value{Value: 60},
				UserSessionTimeout:           &wrappers.UInt64Value{Value: 2},
				Error:                        errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:                         "abnormal case 14",
				ReqUser:                      &admin,
				Tenant:                       tenant.ID,
				GlobalTimezone:               &wrappers.StringValue{Value: "UTC"},
				GlobalLanguageSet:            &wrappers.StringValue{Value: "eng"},
				UserLoginRestrictionEnable:   &wrappers.BoolValue{Value: true},
				UserLoginRestrictionTryCount: &wrappers.UInt64Value{Value: 6},
				UserLoginRestrictionTime:     &wrappers.Int64Value{Value: 8000},
				UserReuseOldPassword:         &wrappers.BoolValue{Value: true},
				UserPasswordChangeCycle:      &wrappers.UInt64Value{Value: 60},
				UserSessionTimeout:           &wrappers.UInt64Value{Value: 2},
				Error:                        errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:                         "normal case 1",
				ReqUser:                      &admin,
				Tenant:                       tenant.ID,
				GlobalTimezone:               &wrappers.StringValue{Value: "UTC"},
				GlobalLanguageSet:            &wrappers.StringValue{Value: "eng"},
				UserLoginRestrictionEnable:   &wrappers.BoolValue{Value: true},
				UserLoginRestrictionTryCount: &wrappers.UInt64Value{Value: 6},
				UserLoginRestrictionTime:     &wrappers.Int64Value{Value: 6000},
				UserReuseOldPassword:         &wrappers.BoolValue{Value: true},
				UserPasswordChangeCycle:      &wrappers.UInt64Value{Value: 60},
				UserSessionTimeout:           &wrappers.UInt64Value{Value: 2},
				Error:                        nil,
			},
			{
				Desc:                       "normal case 2",
				ReqUser:                    &admin,
				Tenant:                     tenant.ID,
				GlobalTimezone:             &wrappers.StringValue{Value: "UTC"},
				GlobalLanguageSet:          &wrappers.StringValue{Value: "kor"},
				UserLoginRestrictionEnable: &wrappers.BoolValue{Value: false},
				UserReuseOldPassword:       &wrappers.BoolValue{Value: true},
				UserPasswordChangeCycle:    &wrappers.UInt64Value{Value: 60},
				UserSessionTimeout:         &wrappers.UInt64Value{Value: 2},
				Error:                      nil,
			},
		} {
			ctx := setCtx(tc.ReqUser, tc.Tenant)

			var req = &identity.ConfigRequest{IdentityConfig: &identity.Config{}}

			req.IdentityConfig.GlobalTimezone = tc.GlobalTimezone
			req.IdentityConfig.GlobalLanguageSet = tc.GlobalLanguageSet
			req.IdentityConfig.UserLoginRestrictionEnable = tc.UserLoginRestrictionEnable
			req.IdentityConfig.UserLoginRestrictionTryCount = tc.UserLoginRestrictionTryCount
			req.IdentityConfig.UserLoginRestrictionTime = tc.UserLoginRestrictionTime
			req.IdentityConfig.UserReuseOldPassword = tc.UserReuseOldPassword
			req.IdentityConfig.UserPasswordChangeCycle = tc.UserPasswordChangeCycle
			req.IdentityConfig.UserSessionTimeout = tc.UserSessionTimeout

			var rsp = new(identity.ConfigResponse)

			err := handler.SetConfig(ctx, req, rsp)

			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, req.IdentityConfig.GetGlobalTimezone().GetValue(), rsp.IdentityConfig.GetGlobalTimezone().GetValue(), tc.Desc)
				assert.Equal(t, req.IdentityConfig.GetGlobalLanguageSet().GetValue(), rsp.IdentityConfig.GetGlobalLanguageSet().GetValue(), tc.Desc)
				assert.Equal(t, req.IdentityConfig.GetUserLoginRestrictionEnable().GetValue(), rsp.IdentityConfig.GetUserLoginRestrictionEnable().GetValue(), tc.Desc)
				assert.Equal(t, req.IdentityConfig.GetUserLoginRestrictionTryCount().GetValue(), rsp.IdentityConfig.GetUserLoginRestrictionTryCount().GetValue(), tc.Desc)
				assert.Equal(t, req.IdentityConfig.GetUserLoginRestrictionTime().GetValue(), rsp.IdentityConfig.GetUserLoginRestrictionTime().GetValue(), tc.Desc)
				assert.Equal(t, req.IdentityConfig.GetUserReuseOldPassword().GetValue(), rsp.IdentityConfig.GetUserReuseOldPassword().GetValue(), tc.Desc)
				assert.Equal(t, req.IdentityConfig.GetUserPasswordChangeCycle().GetValue(), rsp.IdentityConfig.GetUserPasswordChangeCycle().GetValue(), tc.Desc)
				assert.Equal(t, req.IdentityConfig.GetUserSessionTimeout().GetValue(), rsp.IdentityConfig.GetUserSessionTimeout().GetValue(), tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetConfig(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var tenants []*model.Tenant
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30], UseFlag: true})

		for _, tenant := range tenants {
			if err := db.Save(&tenant).Error; err != nil {
				panic(err)
			}
		}

		var configMap = make(map[uint64][]*model.TenantConfig)
		// LoginRestriction true
		configMap[tenants[0].ID] = append(configMap[tenants[0].ID], &model.TenantConfig{
			TenantID: tenants[0].ID,
			Key:      config.GlobalTimeZone,
			Value:    "UTC",
		})

		configMap[tenants[0].ID] = append(configMap[tenants[0].ID], &model.TenantConfig{
			TenantID: tenants[0].ID,
			Key:      config.GlobalLanguageSet,
			Value:    "eng",
		})

		configMap[tenants[0].ID] = append(configMap[tenants[0].ID], &model.TenantConfig{
			TenantID: tenants[0].ID,
			Key:      config.UserPasswordChangeCycle,
			Value:    "90",
		})

		configMap[tenants[0].ID] = append(configMap[tenants[0].ID], &model.TenantConfig{
			TenantID: tenants[0].ID,
			Key:      config.UserReuseOldPassword,
			Value:    "false",
		})

		configMap[tenants[0].ID] = append(configMap[tenants[0].ID], &model.TenantConfig{
			TenantID: tenants[0].ID,
			Key:      config.UserSessionTimeout,
			Value:    "30",
		})

		configMap[tenants[0].ID] = append(configMap[tenants[0].ID], &model.TenantConfig{
			TenantID: tenants[0].ID,
			Key:      config.UserLoginRestrictionEnable,
			Value:    "true",
		})

		configMap[tenants[0].ID] = append(configMap[tenants[0].ID], &model.TenantConfig{
			TenantID: tenants[0].ID,
			Key:      config.UserLoginRestrictionTryCount,
			Value:    "5",
		})

		configMap[tenants[0].ID] = append(configMap[tenants[0].ID], &model.TenantConfig{
			TenantID: tenants[0].ID,
			Key:      config.UserLoginRestrictionTime,
			Value:    "6000",
		})

		// LoginRestriction False
		configMap[tenants[1].ID] = append(configMap[tenants[1].ID], &model.TenantConfig{
			TenantID: tenants[1].ID,
			Key:      config.GlobalTimeZone,
			Value:    "Asisa/Seoul",
		})

		configMap[tenants[1].ID] = append(configMap[tenants[1].ID], &model.TenantConfig{
			TenantID: tenants[1].ID,
			Key:      config.GlobalLanguageSet,
			Value:    "kor",
		})

		configMap[tenants[1].ID] = append(configMap[tenants[1].ID], &model.TenantConfig{
			TenantID: tenants[1].ID,
			Key:      config.UserPasswordChangeCycle,
			Value:    "80",
		})

		configMap[tenants[1].ID] = append(configMap[tenants[1].ID], &model.TenantConfig{
			TenantID: tenants[1].ID,
			Key:      config.UserReuseOldPassword,
			Value:    "true",
		})

		configMap[tenants[1].ID] = append(configMap[tenants[1].ID], &model.TenantConfig{
			TenantID: tenants[1].ID,
			Key:      config.UserSessionTimeout,
			Value:    "20",
		})

		configMap[tenants[1].ID] = append(configMap[tenants[1].ID], &model.TenantConfig{
			TenantID: tenants[1].ID,
			Key:      config.UserLoginRestrictionEnable,
			Value:    "false",
		})

		for _, configs := range configMap {
			for _, config := range configs {
				if err := db.Save(&config).Error; err != nil {
					panic(err)
				}
			}
		}

		var tmp model.Role
		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		for _, tc := range []struct {
			Desc    string
			Limit   uint64
			Offset  uint64
			ReqUser *identity.User
			Tenant  uint64
		}{
			{
				Desc:    "normal case 1",
				ReqUser: &admin,
				Tenant:  tenants[0].ID,
			},
			{
				Desc:    "normal case 2",
				ReqUser: &admin,
				Tenant:  tenants[1].ID,
			},
		} {
			ctx := setCtx(tc.ReqUser, tc.Tenant)

			var rsp = new(identity.ConfigResponse)

			err := handler.GetConfig(ctx, &identity.Empty{}, rsp)

			assert.NoError(t, err)
			assert.Equal(t, configMap[tc.Tenant][0].Value, rsp.IdentityConfig.GetGlobalTimezone().GetValue(), tc.Desc)
			assert.Equal(t, configMap[tc.Tenant][1].Value, rsp.IdentityConfig.GetGlobalLanguageSet().GetValue(), tc.Desc)
			assert.Equal(t, configMap[tc.Tenant][2].Value, strconv.Itoa(int(rsp.IdentityConfig.GetUserPasswordChangeCycle().GetValue())), tc.Desc)
			assert.Equal(t, configMap[tc.Tenant][3].Value, strconv.FormatBool(rsp.IdentityConfig.GetUserReuseOldPassword().GetValue()), tc.Desc)
			assert.Equal(t, configMap[tc.Tenant][4].Value, strconv.Itoa(int(rsp.IdentityConfig.GetUserSessionTimeout().GetValue())), tc.Desc)
			assert.Equal(t, configMap[tc.Tenant][5].Value, strconv.FormatBool(rsp.IdentityConfig.GetUserLoginRestrictionEnable().GetValue()), tc.Desc)
			if configMap[tc.Tenant][5].Value == "false" {
				assert.Nil(t, rsp.IdentityConfig.UserLoginRestrictionTryCount, tc.Desc)
				assert.Nil(t, rsp.IdentityConfig.UserLoginRestrictionTime, tc.Desc)
			} else {
				assert.Equal(t, configMap[tc.Tenant][6].Value, strconv.Itoa(int(rsp.IdentityConfig.GetUserLoginRestrictionTryCount().GetValue())), tc.Desc)
				assert.Equal(t, configMap[tc.Tenant][7].Value, strconv.Itoa(int(rsp.IdentityConfig.GetUserLoginRestrictionTime().GetValue())), tc.Desc)
			}

		}
	})
}

func generateString(len int) string {
	var s string
	for i := 0; i < len; i++ {
		s += "a"
	}
	return s
}

func TestAddTenant(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		for _, tc := range []struct {
			Desc      string
			Name      string
			Solutions []string
			Remarks   string
			UseFlag   *wrappers.BoolValue
			Error     error
		}{
			{
				Desc:      "normal case1: UseFlag is set true",
				Name:      uuid.New().String()[:30],
				Solutions: []string{uuid.New().String()[:30], uuid.New().String()[:30]},
				Remarks:   uuid.New().String()[:30],
				UseFlag:   &wrappers.BoolValue{Value: true},
				Error:     nil,
			},
			{
				Desc:      "normal case2: UseFlag is set false",
				Name:      uuid.New().String()[:30],
				Solutions: []string{uuid.New().String()[:30], uuid.New().String()[:30]},
				Remarks:   uuid.New().String()[:30],
				UseFlag:   &wrappers.BoolValue{Value: false},
				Error:     nil,
			},
			{
				Desc:      "abnormal case1: length of tenant name is 0",
				Name:      "",
				Solutions: []string{uuid.New().String()[:30], uuid.New().String()[:30]},
				Remarks:   uuid.New().String()[:30],
				UseFlag:   &wrappers.BoolValue{Value: true},
				Error:     errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:      "abnormal case2: length of tenant name over max size",
				Name:      generateString(nameLength + 1),
				Solutions: []string{uuid.New().String()[:30], uuid.New().String()[:30]},
				Remarks:   uuid.New().String()[:30],
				UseFlag:   &wrappers.BoolValue{Value: true},
				Error:     errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:      "abnormal case3: UseFlag is empty",
				Name:      uuid.New().String()[:30],
				Solutions: []string{uuid.New().String()[:30], uuid.New().String()[:30]},
				Remarks:   uuid.New().String()[:30],
				Error:     errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:      "abnormal case4: length of tenant solution over max size",
				Name:      uuid.New().String()[:30],
				Solutions: []string{generateString(solutionLength + 1)},
				Remarks:   uuid.New().String()[:30],
				UseFlag:   &wrappers.BoolValue{Value: true},
				Error:     errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:      "abnormal case5: length of tenant remarks over max size",
				Name:      uuid.New().String()[:30],
				Solutions: []string{uuid.New().String()[:30], uuid.New().String()[:30]},
				Remarks:   generateString(remarksLength + 1),
				UseFlag:   &wrappers.BoolValue{Value: true},
				Error:     errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			reqUser := identity.User{}
			tmp := Role[constant.Admin]
			r, _ := roleModelToRsp(&tmp)
			reqUser.Roles = append(reqUser.Roles, r)

			ctx := context.Background()
			b, _ := json.Marshal(&reqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.AddTenantRequest)
			req.Tenant = new(identity.Tenant)
			req.Tenant.Name = tc.Name
			req.Tenant.UseFlag = tc.UseFlag
			for _, SolutionName := range tc.Solutions {
				req.Tenant.Solutions = append(req.Tenant.Solutions, &identity.Solution{Solution: SolutionName})
			}
			req.Tenant.Remarks = tc.Remarks

			var rsp = new(identity.TenantResponse)

			err := handler.AddTenant(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.NotEqual(t, 0, req.Tenant.Id)
				assert.Equal(t, tc.Name, req.Tenant.Name)
				assert.NotEqual(t, 0, rsp.Tenant.CreatedAt)
				assert.NotEqual(t, 0, rsp.Tenant.UpdatedAt)
				assert.Equal(t, tc.UseFlag.Value, rsp.Tenant.GetUseFlag().GetValue(), tc.Desc)

				var solutions []string
				for _, s := range rsp.Tenant.Solutions {
					solutions = append(solutions, s.Solution)
				}
				assert.ElementsMatch(t, tc.Solutions, solutions)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestAddTenantWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var tmp model.Role

		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		manager := identity.User{}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		manager.Roles = append(manager.Roles, r)

		user := identity.User{}
		user.Roles = append(user.Roles, &identity.Role{Solution: constant.SolutionName, Role: ""})

		for _, tc := range []struct {
			Desc      string
			Name      string
			Solutions []string
			Remarks   string
			ReqUser   *identity.User
			Error     error
		}{
			{
				Desc:      "normal case: admin are add Tenant",
				Name:      uuid.New().String()[:30],
				Solutions: []string{uuid.New().String()[:30], uuid.New().String()[:30]},
				Remarks:   uuid.New().String()[:30],
				ReqUser:   &admin,
				Error:     nil,
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(tc.ReqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.AddTenantRequest)
			req.Tenant = new(identity.Tenant)
			req.Tenant.Name = tc.Name
			req.Tenant.UseFlag = &wrappers.BoolValue{Value: false}
			for _, SolutionName := range tc.Solutions {
				req.Tenant.Solutions = append(req.Tenant.Solutions, &identity.Solution{Solution: SolutionName})
			}
			var rsp = new(identity.TenantResponse)

			err := handler.AddTenant(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.NotEqual(t, 0, req.Tenant.Id)
				assert.Equal(t, tc.Name, req.Tenant.Name)
				assert.NotEqual(t, 0, rsp.Tenant.CreatedAt)
				assert.NotEqual(t, 0, rsp.Tenant.UpdatedAt)

				var solutions []string
				for _, s := range rsp.Tenant.Solutions {
					solutions = append(solutions, s.Solution)
				}
				assert.ElementsMatch(t, tc.Solutions, solutions)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestAddTenantWithId(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{Name: uuid.New().String()[:30], UseFlag: true}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		for _, tc := range []struct {
			Desc      string
			ID        uint64
			Name      string
			Solutions []string
			Remarks   string
			UseFlag   *wrappers.BoolValue
			Error     error
		}{
			{
				Desc:      "normal case: UseFlag is set true",
				Name:      uuid.New().String()[:30],
				Solutions: []string{uuid.New().String()[:30], uuid.New().String()[:30]},
				Remarks:   uuid.New().String()[:30],
				UseFlag:   &wrappers.BoolValue{Value: true},
				Error:     nil,
			},
			{
				Desc:      "abnormal case: tenant id (already exist)",
				ID:        tenant.ID,
				Name:      uuid.New().String()[:30],
				Solutions: []string{uuid.New().String()[:30], uuid.New().String()[:30]},
				Remarks:   uuid.New().String()[:30],
				UseFlag:   &wrappers.BoolValue{Value: true},
				Error:     errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			reqUser := identity.User{}
			tmp := Role[constant.Admin]
			r, _ := roleModelToRsp(&tmp)
			reqUser.Roles = append(reqUser.Roles, r)

			ctx := context.Background()
			b, _ := json.Marshal(&reqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.AddTenantRequest)
			req.Tenant = new(identity.Tenant)
			req.Tenant.Id = tc.ID
			req.Tenant.Name = tc.Name
			req.Tenant.UseFlag = tc.UseFlag
			for _, SolutionName := range tc.Solutions {
				req.Tenant.Solutions = append(req.Tenant.Solutions, &identity.Solution{Solution: SolutionName})
			}
			req.Tenant.Remarks = tc.Remarks

			var rsp = new(identity.TenantResponse)

			err := handler.AddTenant(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.NotEqual(t, 0, req.Tenant.Id)
				assert.Equal(t, tc.Name, req.Tenant.Name)
				assert.NotEqual(t, 0, rsp.Tenant.CreatedAt)
				assert.NotEqual(t, 0, rsp.Tenant.UpdatedAt)
				assert.Equal(t, tc.UseFlag.Value, rsp.Tenant.GetUseFlag().GetValue(), tc.Desc)

				var solutions []string
				for _, s := range rsp.Tenant.Solutions {
					solutions = append(solutions, s.Solution)
				}
				assert.ElementsMatch(t, tc.Solutions, solutions)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestAddTenantWithEmptySolution(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		for _, tc := range []struct {
			Desc      string
			Name      string
			Solutions []string
			Remarks   string
			UseFlag   *wrappers.BoolValue
			Error     error
		}{
			{
				Desc:      "normal case1: UseFlag is set true",
				Name:      uuid.New().String()[:30],
				Solutions: []string{uuid.New().String()[:30], uuid.New().String()[:30]},
				Remarks:   uuid.New().String()[:30],
				UseFlag:   &wrappers.BoolValue{Value: true},
				Error:     nil,
			},
			{
				Desc:      "abnormal case1: length of tenant solution is 0",
				Name:      uuid.New().String()[:30],
				Solutions: []string{""},
				Remarks:   uuid.New().String()[:30],
				UseFlag:   &wrappers.BoolValue{Value: true},
				Error:     errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:    "abnormal case2: length of tenant solution is 0",
				Name:    uuid.New().String()[:30],
				Remarks: uuid.New().String()[:30],
				UseFlag: &wrappers.BoolValue{Value: true},
				Error:   errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			reqUser := identity.User{}
			tmp := Role[constant.Admin]
			r, _ := roleModelToRsp(&tmp)
			reqUser.Roles = append(reqUser.Roles, r)

			ctx := context.Background()
			b, _ := json.Marshal(&reqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.AddTenantRequest)
			req.Tenant = new(identity.Tenant)
			req.Tenant.Name = tc.Name
			req.Tenant.UseFlag = tc.UseFlag
			for _, SolutionName := range tc.Solutions {
				req.Tenant.Solutions = append(req.Tenant.Solutions, &identity.Solution{Solution: SolutionName})
			}
			req.Tenant.Remarks = tc.Remarks

			var rsp = new(identity.TenantResponse)

			err := handler.AddTenant(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.NotEqual(t, 0, req.Tenant.Id)
				assert.Equal(t, tc.Name, req.Tenant.Name)
				assert.NotEqual(t, 0, rsp.Tenant.CreatedAt)
				assert.NotEqual(t, 0, rsp.Tenant.UpdatedAt)
				assert.Equal(t, tc.UseFlag.Value, rsp.Tenant.GetUseFlag().GetValue(), tc.Desc)

				var solutions []string
				for _, s := range rsp.Tenant.Solutions {
					solutions = append(solutions, s.Solution)
				}
				assert.ElementsMatch(t, tc.Solutions, solutions)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestUpdateTenant(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var tenant model.Tenant
		var solutions []*model.TenantSolution

		tenant = model.Tenant{
			Name: uuid.New().String()[:30],
		}

		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		solutions = append(solutions, &model.TenantSolution{TenantID: tenant.ID, Solution: uuid.New().String()[:30]})
		solutions = append(solutions, &model.TenantSolution{TenantID: tenant.ID, Solution: uuid.New().String()[:30]})
		for _, s := range solutions {
			if err := db.Save(&s).Error; err != nil {
				panic(err)
			}
		}

		reqUser := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		reqUser.Roles = append(reqUser.Roles, r)

		// update 시간차 확인을 위한 sleep
		time.Sleep(time.Second * 1)

		for _, tc := range []struct {
			Desc        string
			ID          uint64
			ParameterID uint64
			Name        string
			Remarks     string
			Solutions   []string
			Error       error
		}{
			{
				Desc:        "normal case",
				ID:          tenant.ID,
				ParameterID: tenant.ID,
				Name:        uuid.New().String()[:30],
				Remarks:     uuid.New().String()[:30],
				Solutions:   []string{solutions[0].Solution, solutions[1].Solution},
				Error:       nil,
			},
			{
				Desc:        "abnormal case1: length of tenant name is 0",
				ID:          tenant.ID,
				ParameterID: tenant.ID,
				Name:        "",
				Remarks:     "",
				Solutions:   []string{""},
				Error:       errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:        "abnormal case2: length of tenant name over max size",
				ID:          tenant.ID,
				ParameterID: tenant.ID,
				Name:        generateString(nameLength + 1),
				Remarks:     "",
				Error:       errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:        "abnormal case3: length of tenant solution over max size",
				ID:          tenant.ID,
				ParameterID: tenant.ID,
				Name:        uuid.New().String()[:30],
				Solutions:   []string{generateString(solutionLength + 1)},
				Error:       errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:        "abnormal case4: length of remarks over max size",
				ID:          tenant.ID,
				ParameterID: tenant.ID,
				Name:        uuid.New().String()[:30],
				Remarks:     generateString(remarksLength + 1),
				Error:       errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
			{
				Desc:        "abnormal case5: invalid tenant id",
				ID:          9999999,
				ParameterID: 9999999,
				Name:        uuid.New().String()[:30],
				Remarks:     uuid.New().String()[:30],
				Solutions:   []string{solutions[0].Solution, solutions[1].Solution},
				Error:       errors.NotFound(constant.ServiceIdentity, "not found tenant"),
			},
			{
				Desc:        "abnormal case6: missmatch tenant id",
				ID:          tenant.ID,
				ParameterID: 9999999,
				Name:        uuid.New().String()[:30],
				Remarks:     uuid.New().String()[:30],
				Solutions:   []string{""},
				Error:       errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(&reqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.UpdateTenantRequest)
			req.TenantId = tc.ParameterID
			req.Tenant = new(identity.Tenant)
			req.Tenant.Id = tc.ID
			req.Tenant.Name = tc.Name
			req.Tenant.Remarks = tc.Remarks

			for _, SolutionName := range tc.Solutions {
				req.Tenant.Solutions = append(req.Tenant.Solutions, &identity.Solution{Solution: SolutionName})
			}

			var rsp = new(identity.TenantResponse)

			err := handler.UpdateTenant(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.Name, req.Tenant.Name)
				assert.Equal(t, tc.Remarks, req.Tenant.Remarks)
				assert.Equal(t, tenant.CreatedAt, rsp.Tenant.CreatedAt)
				assert.Equal(t, tenant.UseFlag, rsp.Tenant.GetUseFlag().GetValue())
				assert.NotEqual(t, tenant.UpdatedAt, rsp.Tenant.UpdatedAt)

				var solutions []string
				for _, s := range rsp.Tenant.Solutions {
					solutions = append(solutions, s.Solution)
				}
				assert.ElementsMatch(t, tc.Solutions, solutions)

			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestUpdateTenantWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var tenant model.Tenant
		var solutions []*model.TenantSolution

		tenant = model.Tenant{
			Name: uuid.New().String()[:30],
		}

		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		solutions = append(solutions, &model.TenantSolution{TenantID: tenant.ID, Solution: uuid.New().String()[:30]})
		solutions = append(solutions, &model.TenantSolution{TenantID: tenant.ID, Solution: uuid.New().String()[:30]})
		for _, s := range solutions {
			if err := db.Save(&s).Error; err != nil {
				panic(err)
			}
		}

		var tmp model.Role

		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		manager := identity.User{}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		manager.Roles = append(manager.Roles, r)

		user := identity.User{}
		user.Roles = append(user.Roles, &identity.Role{Solution: constant.SolutionName, Role: ""})

		// update 시간차 확인을 위한 sleep
		time.Sleep(time.Second * 1)

		for _, tc := range []struct {
			Desc        string
			ID          uint64
			ParameterID uint64
			Name        string
			Remarks     string
			Solutions   []string
			ReqUser     *identity.User
			Error       error
		}{
			{
				Desc:        "normal case",
				ID:          tenant.ID,
				ParameterID: tenant.ID,
				Name:        uuid.New().String()[:30],
				Remarks:     uuid.New().String()[:30],
				Solutions:   []string{solutions[0].Solution, solutions[1].Solution},
				ReqUser:     &admin,
				Error:       nil,
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(tc.ReqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.UpdateTenantRequest)
			req.TenantId = tc.ParameterID
			req.Tenant = new(identity.Tenant)
			req.Tenant.Id = tc.ID
			req.Tenant.Name = tc.Name
			req.Tenant.Remarks = tc.Remarks

			for _, SolutionName := range tc.Solutions {
				req.Tenant.Solutions = append(req.Tenant.Solutions, &identity.Solution{Solution: SolutionName})
			}

			var rsp = new(identity.TenantResponse)

			err := handler.UpdateTenant(ctx, req, rsp)
			_ = err
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.Name, req.Tenant.Name)
				assert.Equal(t, tc.Remarks, req.Tenant.Remarks)
				assert.Equal(t, tenant.CreatedAt, rsp.Tenant.CreatedAt)
				assert.Equal(t, tenant.UseFlag, rsp.Tenant.GetUseFlag().GetValue())
				assert.NotEqual(t, tenant.UpdatedAt, rsp.Tenant.UpdatedAt)

				var solutions []string
				for _, s := range rsp.Tenant.Solutions {
					solutions = append(solutions, s.Solution)
				}
				assert.ElementsMatch(t, tc.Solutions, solutions)

			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestUpdateTenantWithEmptySolution(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var tenant model.Tenant
		var solutions []*model.TenantSolution

		tenant = model.Tenant{
			Name: uuid.New().String()[:30],
		}

		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		solutions = append(solutions, &model.TenantSolution{TenantID: tenant.ID, Solution: uuid.New().String()[:30]})
		solutions = append(solutions, &model.TenantSolution{TenantID: tenant.ID, Solution: uuid.New().String()[:30]})
		for _, s := range solutions {
			if err := db.Save(&s).Error; err != nil {
				panic(err)
			}
		}

		reqUser := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		reqUser.Roles = append(reqUser.Roles, r)

		// update 시간차 확인을 위한 sleep
		time.Sleep(time.Second * 1)

		for _, tc := range []struct {
			Desc        string
			ID          uint64
			ParameterID uint64
			Name        string
			Remarks     string
			Solutions   []string
			Error       error
		}{
			{
				Desc:        "normal case",
				ID:          tenant.ID,
				ParameterID: tenant.ID,
				Name:        uuid.New().String()[:30],
				Remarks:     uuid.New().String()[:30],
				Solutions:   []string{solutions[0].Solution, solutions[1].Solution},
				Error:       nil,
			},
			{
				Desc:        "abnormal case1: solution length is 0",
				ID:          tenant.ID,
				ParameterID: tenant.ID,
				Name:        uuid.New().String()[:30],
				Remarks:     uuid.New().String()[:30],
				Solutions:   []string{""},
				Error:       errors.BadRequest(constant.ServiceIdentity, "empty parameter"),
			},
			{
				Desc:        "abnormal case2: solution is empty",
				ID:          tenant.ID,
				ParameterID: tenant.ID,
				Name:        uuid.New().String()[:30],
				Remarks:     uuid.New().String()[:30],
				Error:       errors.BadRequest(constant.ServiceIdentity, "empty parameter"),
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(&reqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.UpdateTenantRequest)
			req.TenantId = tc.ParameterID
			req.Tenant = new(identity.Tenant)
			req.Tenant.Id = tc.ID
			req.Tenant.Name = tc.Name
			req.Tenant.Remarks = tc.Remarks

			for _, SolutionName := range tc.Solutions {
				req.Tenant.Solutions = append(req.Tenant.Solutions, &identity.Solution{Solution: SolutionName})
			}

			var rsp = new(identity.TenantResponse)

			err := handler.UpdateTenant(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tc.Name, req.Tenant.Name)
				assert.Equal(t, tc.Remarks, req.Tenant.Remarks)
				assert.Equal(t, tenant.CreatedAt, rsp.Tenant.CreatedAt)
				assert.Equal(t, tenant.UseFlag, rsp.Tenant.GetUseFlag().GetValue())
				assert.NotEqual(t, tenant.UpdatedAt, rsp.Tenant.UpdatedAt)

				var solutions []string
				for _, s := range rsp.Tenant.Solutions {
					solutions = append(solutions, s.Solution)
				}
				assert.ElementsMatch(t, tc.Solutions, solutions)

			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetTenant(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var tenant model.Tenant
		var solutions []*model.TenantSolution

		tenant = model.Tenant{
			Name:    uuid.New().String()[:30],
			UseFlag: true,
		}
		tenant.Remarks = new(string)
		*tenant.Remarks = uuid.New().String()[:30]

		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		solutions = append(solutions, &model.TenantSolution{TenantID: tenant.ID, Solution: uuid.New().String()[:30]})
		solutions = append(solutions, &model.TenantSolution{TenantID: tenant.ID, Solution: uuid.New().String()[:30]})

		for _, s := range solutions {
			if err := db.Save(&s).Error; err != nil {
				panic(err)
			}
		}

		reqUser := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		reqUser.Roles = append(reqUser.Roles, r)

		for _, tc := range []struct {
			Desc  string
			ID    uint64
			Error error
		}{
			{
				Desc:  "normal case",
				ID:    tenant.ID,
				Error: nil,
			},
			{
				Desc:  "abnormal case",
				ID:    999999999,
				Error: errors.NotFound(constant.ServiceIdentity, "not found tenant"),
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(&reqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.TenantRequest)
			req.TenantId = tc.ID

			var rsp = new(identity.TenantResponse)

			err := handler.GetTenant(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tenant.Name, rsp.Tenant.Name)
				assert.Equal(t, *tenant.Remarks, rsp.Tenant.Remarks)
				assert.Equal(t, tenant.CreatedAt, rsp.Tenant.CreatedAt)
				assert.Equal(t, tenant.UpdatedAt, rsp.Tenant.UpdatedAt)
				assert.Equal(t, tenant.UseFlag, rsp.Tenant.GetUseFlag().GetValue())

				var resSolutions []string
				for _, s := range rsp.Tenant.Solutions {
					resSolutions = append(resSolutions, s.Solution)
				}

				var dbSolutions []string
				for _, s := range solutions {
					dbSolutions = append(dbSolutions, s.Solution)
				}

				assert.ElementsMatch(t, dbSolutions, resSolutions, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetTenantWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var tenant model.Tenant
		var solutions []*model.TenantSolution

		tenant = model.Tenant{
			Name:    uuid.New().String()[:30],
			UseFlag: true,
		}
		tenant.Remarks = new(string)
		*tenant.Remarks = uuid.New().String()[:30]

		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		solutions = append(solutions, &model.TenantSolution{TenantID: tenant.ID, Solution: uuid.New().String()[:30]})
		solutions = append(solutions, &model.TenantSolution{TenantID: tenant.ID, Solution: uuid.New().String()[:30]})
		for _, s := range solutions {
			if err := db.Save(&s).Error; err != nil {
				panic(err)
			}
		}

		var tmp model.Role

		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		manager := identity.User{}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		manager.Roles = append(manager.Roles, r)

		user := identity.User{}
		user.Roles = append(user.Roles, &identity.Role{Solution: constant.SolutionName, Role: ""})

		for _, tc := range []struct {
			Desc    string
			ID      uint64
			ReqUser *identity.User
			Error   error
		}{
			{
				Desc:    "normal case",
				ID:      tenant.ID,
				ReqUser: &admin,
				Error:   nil,
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(tc.ReqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.TenantRequest)
			req.TenantId = tc.ID

			var rsp = new(identity.TenantResponse)

			err := handler.GetTenant(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.Equal(t, tenant.Name, rsp.Tenant.Name)
				assert.Equal(t, *tenant.Remarks, rsp.Tenant.Remarks)
				assert.Equal(t, tenant.CreatedAt, rsp.Tenant.CreatedAt)
				assert.Equal(t, tenant.UpdatedAt, rsp.Tenant.UpdatedAt)
				assert.Equal(t, tenant.UseFlag, rsp.Tenant.GetUseFlag().GetValue())

				var resSolutions []string
				for _, s := range rsp.Tenant.Solutions {
					resSolutions = append(resSolutions, s.Solution)
				}

				var dbSolutions []string
				for _, s := range solutions {
					dbSolutions = append(dbSolutions, s.Solution)
				}

				assert.ElementsMatch(t, dbSolutions, resSolutions, tc.Desc)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetTenants(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var tenants []*model.Tenant

		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30]})
		tenants = append(tenants, &model.Tenant{Name: "Tenant"})
		for _, tenant := range tenants {
			if err := db.Save(&tenant).Error; err != nil {
				panic(err)
			}
		}

		reqUser := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		reqUser.Roles = append(reqUser.Roles, r)

		for _, tc := range []struct {
			Desc     string
			Name     string
			Expected int
			Error    error
		}{
			// default tenant , append tennant
			{
				Desc:     "expected : 3",
				Name:     "",
				Expected: 3,
				Error:    nil,
			},
			{
				Desc:     "expected : 1",
				Name:     "Tenant",
				Expected: 1,
				Error:    nil,
			},
			{
				Desc:     "expected : 0",
				Name:     "Test",
				Expected: 0,
				Error:    nil,
			},
			{
				Desc:     "abnormal case: ",
				Name:     generateString(nameLength + 1),
				Expected: 0,
				Error:    errors.BadRequest(constant.ServiceIdentity, "invalid parameter"),
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(&reqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.TenantsRequest)
			req.Name = tc.Name

			var rsp = new(identity.TenantsResponse)

			err := handler.GetTenants(ctx, req, rsp)

			if tc.Error == nil {
				if tc.Expected > 0 {
					assert.Equal(t, tc.Expected, len(rsp.Tenants), tc.Desc)
				} else {
					assert.Equal(t, int32(204), err.(*errors.Error).Code, tc.Desc)
				}
			} else {
				assert.Equal(t, int32(400), err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestGetTenantsWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var tenants []*model.Tenant

		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30]})
		tenants = append(tenants, &model.Tenant{Name: uuid.New().String()[:30]})

		for _, tenant := range tenants {
			if err := db.Save(&tenant).Error; err != nil {
				panic(err)
			}
		}

		var tmp model.Role

		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		manager := identity.User{}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		manager.Roles = append(manager.Roles, r)

		user := identity.User{}
		user.Roles = append(user.Roles, &identity.Role{Solution: constant.SolutionName, Role: ""})

		for _, tc := range []struct {
			Desc     string
			Name     string
			ReqUser  *identity.User
			Expected int
			Error    error
		}{
			// default tenant , append tennant
			{
				Desc:     "expected : 3",
				Name:     "",
				ReqUser:  &admin,
				Expected: 3,
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(tc.ReqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.TenantsRequest)
			req.Name = tc.Name

			var rsp = new(identity.TenantsResponse)

			err := handler.GetTenants(ctx, req, rsp)
			if tc.Expected > 0 {
				assert.Equal(t, tc.Expected, len(rsp.Tenants), tc.Desc)
			} else {
				assert.Equal(t, int32(403), err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestActivateTenant(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{
			Name:    uuid.New().String()[:30],
			UseFlag: false,
		}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		reqUser := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		reqUser.Roles = append(reqUser.Roles, r)

		for _, tc := range []struct {
			Desc  string
			ID    uint64
			Error error
		}{
			{
				Desc:  "normal case",
				ID:    tenant.ID,
				Error: nil,
			},
			{
				Desc:  "abnormal case",
				ID:    999999999,
				Error: errors.NotFound(constant.ServiceIdentity, "not found tenant"),
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(&reqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.TenantRequest)
			req.TenantId = tc.ID

			var rsp = new(identity.TenantResponse)

			err := handler.ActivateTenant(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.True(t, rsp.Tenant.GetUseFlag().GetValue())
				assert.Equal(t, tenant.CreatedAt, rsp.Tenant.CreatedAt)
				assert.Equal(t, tenant.UpdatedAt, rsp.Tenant.UpdatedAt)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestActivateTenantWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{
			Name:    uuid.New().String()[:30],
			UseFlag: false,
		}
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		var tmp model.Role

		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		manager := identity.User{}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		manager.Roles = append(manager.Roles, r)

		user := identity.User{}
		user.Roles = append(user.Roles, &identity.Role{Solution: constant.SolutionName, Role: ""})

		for _, tc := range []struct {
			Desc    string
			ID      uint64
			ReqUser *identity.User
			Error   error
		}{
			{
				Desc:    "normal case",
				ID:      tenant.ID,
				ReqUser: &admin,
				Error:   nil,
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(tc.ReqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.TenantRequest)
			req.TenantId = tc.ID

			var rsp = new(identity.TenantResponse)

			err := handler.ActivateTenant(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.True(t, rsp.Tenant.GetUseFlag().GetValue())
				assert.Equal(t, tenant.CreatedAt, rsp.Tenant.CreatedAt)
				assert.Equal(t, tenant.UpdatedAt, rsp.Tenant.UpdatedAt)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestDeactivateTenant(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{
			Name: uuid.New().String()[:30],
		}
		tenant.UseFlag = true
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		reqUser := identity.User{}
		tmp := Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		reqUser.Roles = append(reqUser.Roles, r)

		for _, tc := range []struct {
			Desc  string
			ID    uint64
			Error error
		}{
			{
				Desc:  "normal case",
				ID:    tenant.ID,
				Error: nil,
			},
			{
				Desc:  "abnormal case",
				ID:    999999999,
				Error: errors.NotFound(constant.ServiceIdentity, "not found tenant"),
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(&reqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.TenantRequest)
			req.TenantId = tc.ID

			var rsp = new(identity.TenantResponse)

			err := handler.DeactivateTenant(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.False(t, rsp.Tenant.GetUseFlag().GetValue())
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestDeactivateTenantWithAuth(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		tenant := model.Tenant{
			Name: uuid.New().String()[:30],
		}
		tenant.UseFlag = true
		if err := db.Save(&tenant).Error; err != nil {
			panic(err)
		}

		var tmp model.Role

		admin := identity.User{}
		tmp = Role[constant.Admin]
		r, _ := roleModelToRsp(&tmp)
		admin.Roles = append(admin.Roles, r)

		manager := identity.User{}
		tmp = Role[constant.Manager]
		r, _ = roleModelToRsp(&tmp)
		manager.Roles = append(manager.Roles, r)

		user := identity.User{}
		user.Roles = append(user.Roles, &identity.Role{Solution: constant.SolutionName, Role: ""})

		for _, tc := range []struct {
			Desc    string
			ID      uint64
			ReqUser *identity.User
			Error   error
		}{
			{
				Desc:    "normal case",
				ID:      tenant.ID,
				ReqUser: &admin,
				Error:   nil,
			},
		} {
			ctx := context.Background()
			b, _ := json.Marshal(tc.ReqUser)
			ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))

			var req = new(identity.TenantRequest)
			req.TenantId = tc.ID

			var rsp = new(identity.TenantResponse)

			err := handler.DeactivateTenant(ctx, req, rsp)
			if tc.Error == nil {
				assert.NoError(t, err, tc.Desc)
				assert.False(t, rsp.Tenant.GetUseFlag().GetValue())
				assert.Equal(t, tenant.CreatedAt, rsp.Tenant.CreatedAt)
				assert.Equal(t, tenant.UpdatedAt, rsp.Tenant.UpdatedAt)
			} else {
				assert.Equal(t, tc.Error.(*errors.Error).Code, err.(*errors.Error).Code, tc.Desc)
			}
		}
	})
}

func TestCheckAuthorization(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var policies = []struct {
			role      string
			solutions string
			endpoint  string
		}{
			//cloud 관련 policy
			{
				role:      constant.Admin,
				solutions: constant.SolutionName,
				endpoint:  cloudAdminEndpoint,
			}, {
				role:      constant.Admin,
				solutions: constant.SolutionName,
				endpoint:  cloudAdminAndManagerEndPoint,
			}, {
				role:      constant.Manager,
				solutions: constant.SolutionName,
				endpoint:  cloudAdminAndManagerEndPoint,
			}, {
				role:      constant.User,
				solutions: constant.SolutionName,
				endpoint:  cloudAllEndpoint,
			},
		}

		for _, tc := range policies {
			ok, err := handler.enforcer.AddPolicy(tc.role, tc.solutions, tc.endpoint)
			assert.Equal(t, true, ok)
			assert.NoError(t, err)
		}

		defer func() {
			for _, tc := range policies {
				_, _ = handler.enforcer.RemovePolicy(tc.role, tc.solutions, tc.endpoint)
			}
		}()

		var tenants []*model.Tenant
		tenants = append(tenants, &model.Tenant{
			Name: uuid.New().String()[:30],
		})
		tenants[0].UseFlag = true
		tenants = append(tenants, &model.Tenant{
			Name: uuid.New().String()[:30],
		})
		tenants[1].UseFlag = true

		for _, tenant := range tenants {
			if err := db.Save(&tenant).Error; err != nil {
				panic(err)
			}
		}

		var (
			tmpAdmin    = Role[constant.Admin]
			tmpManager  = Role[constant.Manager]
			cloudViewer = &identity.Role{Solution: constant.SolutionName, Role: "viewer"}
		)

		adminRole, _ := roleModelToRsp(&tmpAdmin)
		managerRole, _ := roleModelToRsp(&tmpManager)

		for _, tc := range []struct {
			req      *identity.CheckAuthorizationRequest
			user     *identity.User
			success  bool
			tenantID string
			err      error
		}{
			//authorized ok
			{
				tenantID: strconv.FormatUint(tenants[0].ID, 10),
				req:      &identity.CheckAuthorizationRequest{Endpoint: cloudAdminEndpoint},
				user:     &identity.User{Roles: []*identity.Role{adminRole}},
				success:  true,
			}, {
				tenantID: strconv.FormatUint(tenants[0].ID, 10),
				req:      &identity.CheckAuthorizationRequest{Endpoint: cloudAdminAndManagerEndPoint},
				user:     &identity.User{Roles: []*identity.Role{adminRole}},
				success:  true,
			}, {
				tenantID: strconv.FormatUint(tenants[0].ID, 10),
				req:      &identity.CheckAuthorizationRequest{Endpoint: cloudAllEndpoint},
				user:     &identity.User{Roles: []*identity.Role{adminRole}},
				success:  true,
			},
			{
				tenantID: strconv.FormatUint(tenants[0].ID, 10),
				req:      &identity.CheckAuthorizationRequest{Endpoint: cloudAdminAndManagerEndPoint},
				user:     &identity.User{Tenant: &identity.Tenant{Id: tenants[0].ID}, Roles: []*identity.Role{managerRole}},
				success:  true,
			}, {
				tenantID: strconv.FormatUint(tenants[0].ID, 10),
				req:      &identity.CheckAuthorizationRequest{Endpoint: cloudAllEndpoint},
				user:     &identity.User{Tenant: &identity.Tenant{Id: tenants[0].ID}, Roles: []*identity.Role{managerRole}},
				success:  true,
			}, {
				tenantID: strconv.FormatUint(tenants[0].ID, 10),
				req:      &identity.CheckAuthorizationRequest{Endpoint: cloudAllEndpoint},
				user:     &identity.User{Tenant: &identity.Tenant{Id: tenants[0].ID}, Roles: []*identity.Role{cloudViewer}},
				success:  true,
			},
			//unauthorized
			{
				tenantID: strconv.FormatUint(tenants[0].ID, 10),
				req:      &identity.CheckAuthorizationRequest{Endpoint: cloudAdminEndpoint},
				user:     &identity.User{Tenant: &identity.Tenant{Id: tenants[1].ID}, Roles: []*identity.Role{managerRole}},
				success:  false,
				err:      errors.Forbidden(constant.ServiceIdentity, "unauthorized"),
			},
			{
				tenantID: strconv.FormatUint(tenants[0].ID, 10),
				req:      &identity.CheckAuthorizationRequest{Endpoint: cloudAdminEndpoint},
				user:     &identity.User{Tenant: &identity.Tenant{Id: tenants[1].ID}, Roles: []*identity.Role{cloudViewer}},
				success:  false,
				err:      errors.Forbidden(constant.ServiceIdentity, "unauthorized"),
			}, {
				tenantID: strconv.FormatUint(tenants[0].ID, 10),
				req:      &identity.CheckAuthorizationRequest{Endpoint: cloudAdminAndManagerEndPoint},
				user:     &identity.User{Tenant: &identity.Tenant{Id: tenants[0].ID}, Roles: []*identity.Role{cloudViewer}},
				success:  false,
				err:      errors.Forbidden(constant.ServiceIdentity, "unauthorized"),
			},
			//differ header tenant id to user tenant id
			{
				tenantID: strconv.FormatUint(tenants[0].ID, 10),
				req:      &identity.CheckAuthorizationRequest{Endpoint: cloudAllEndpoint},
				user:     &identity.User{Tenant: &identity.Tenant{Id: tenants[1].ID}, Roles: []*identity.Role{cloudViewer}},
				success:  false,
				err:      errors.Forbidden(constant.ServiceIdentity, "unauthorized"),
			},
			//header user info not exsist
			{
				tenantID: strconv.FormatUint(tenants[0].ID, 10),
				req:      &identity.CheckAuthorizationRequest{Endpoint: cloudAllEndpoint},
				success:  false,
				err:      errors.InternalServerError(constant.ServiceIdentity, "unknown error"),
			},
			//header tenant id not exists
			{
				req:     &identity.CheckAuthorizationRequest{Endpoint: cloudAllEndpoint},
				user:    &identity.User{Tenant: &identity.Tenant{Id: 1}, Roles: []*identity.Role{cloudViewer}},
				success: false,
				err:     errors.InternalServerError(constant.ServiceIdentity, "unknown error"),
			},
		} {
			ctx := context.Background()
			switch {
			case tc.user != nil:
				b, _ := json.Marshal(tc.user)
				ctx = metadata.Set(ctx, commonmeta.HeaderAuthenticatedUser, string(b))
				fallthrough

			case tc.tenantID != "":
				ctx = metadata.Set(ctx, commonmeta.HeaderTenantID, tc.tenantID)
			}

			err := handler.CheckAuthorization(ctx, tc.req, &identity.MessageResponse{})
			if tc.success {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tc.err.(*errors.Error).Code, err.(*errors.Error).Code)
			}
		}
	})
}
