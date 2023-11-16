package handler

import (
	"context"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/event"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/datacommand2/cdm-cloud/common/metadata"
	"github.com/datacommand2/cdm-cloud/common/util"
	identity "github.com/datacommand2/cdm-cloud/services/identity/proto"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/sethvargo/go-password/password"
	"io/ioutil"
	"math/rand"
	"time"
)

// createError 각 서비스의 internal error 를 처리
func createError(ctx context.Context, eventCode string, err error) error {
	if err == nil {
		return nil
	}

	var errorCode string
	switch {
	// not found
	case errors.Equal(err, errNotFoundUser):
		errorCode = "not_found_user"
		return errors.StatusNotFound(ctx, eventCode, errorCode, err)

	case errors.Equal(err, errNotFoundTenant):
		errorCode = "not_found_tenant"
		return errors.StatusNotFound(ctx, eventCode, errorCode, err)

	case errors.Equal(err, errNotFoundGroup):
		errorCode = "not_found_group"
		return errors.StatusNotFound(ctx, eventCode, errorCode, err)

	// user
	case errors.Equal(err, errNotReusableOldPassword):
		errorCode = "not_reusable_old_password"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	case errors.Equal(err, errCurrentPasswordMismatch):
		errorCode = "mismatch_password"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	case errors.Equal(err, errUndeletableUser):
		errorCode = "undeletable_user"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	// group
	case errors.Equal(err, errAlreadyDeletedGroup):
		errorCode = "already_deleted"
		return errors.StatusPreconditionFailed(ctx, eventCode, errorCode, err)

	case errors.Equal(err, errUndeletableGroup):
		errorCode = "undeletable_group"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	// session
	case errors.Equal(err, errAlreadyLogin):
		errorCode = "already_login"
		return errors.StatusConflict(ctx, eventCode, errorCode, err)

	case errors.Equal(err, errLoginRestricted):
		errorCode = "login_restricted"
		return errors.StatusUnauthenticated(ctx, eventCode, errorCode, err)

	case errors.Equal(err, errIncorrectPassword):
		errorCode = "incorrect_password"
		return errors.StatusUnauthenticated(ctx, eventCode, errorCode, err)

	case errors.Equal(err, errUnknownSession):
		errorCode = "unknown_session"
		return errors.StatusNotFound(ctx, eventCode, errorCode, err)

	case errors.Equal(err, errExpiredSession):
		errorCode = "expired_session"
		return errors.StatusUnauthenticated(ctx, eventCode, errorCode, err)

	case errors.Equal(err, errUnverifiedSession):
		errorCode = "unverified_session"
		return errors.StatusUnauthenticated(ctx, eventCode, errorCode, err)

	case errors.Equal(err, errInvalidSession):
		errorCode = "invalid_session"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	// config
	case errors.Equal(err, errNotfoundTenantConfig):
		errorCode = "not_found_tenant_config"
		return errors.StatusNotFound(ctx, eventCode, errorCode, err)

	case errors.Equal(err, errInvalidTenantConfig):
		errorCode = "invalid_tenant_config"
		return errors.StatusInternalServerError(ctx, eventCode, errorCode, err)

	// role
	case errors.Equal(err, errUnassignableRole):
		errorCode = "unassignable_role"
		return errors.StatusBadRequest(ctx, eventCode, errorCode, err)

	default:
		if err := util.CreateError(ctx, eventCode, err); err != nil {
			return err
		}
	}

	return nil
}

func generatePassword() (string, error) {
	var passwd string
	var err error

	//Init random seed
	rand.Seed(time.Now().UnixNano())

	//암호의 길이는 6~16자
	passwdLength := rand.Intn(10) + 6
	per := passwdLength / 6

	for i := 0; i < 10; i++ {
		passwd, err = password.Generate(passwdLength, per, per, false, false)
		if err != nil {
			time.Sleep(1 * time.Second)
			logger.Warnf("Generate password failed(%v)... try again\n", err)
			continue
		}
		return passwd, nil
	}

	return "", errors.Unknown(err)
}

// userRspToModel 은 identity 의 User 를 model 의 User 로 변환 하는함수이다
func userRspToModel(u *identity.User) (*model.User, error) {
	user := model.User{TenantID: u.Tenant.Id}

	b, err := json.Marshal(u)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// userModelToRsp 는 DB model 의 User 객체를 리스폰스의 User 객체로 변환하기위한 함수이다
func userModelToRsp(modelUser *model.User) (*identity.User, error) {
	var rspUser = identity.User{}
	var err error

	b, err := json.Marshal(*modelUser)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &rspUser)
	if err != nil {
		return nil, err
	}

	return &rspUser, nil
}

func simpleUserModelToRsp(modelUser *model.User) (*identity.SimpleUser, error) {
	var rspUser = identity.SimpleUser{}
	var err error

	b, err := json.Marshal(*modelUser)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &rspUser)
	if err != nil {
		return nil, err
	}

	return &rspUser, nil
}

func roleModelToRsp(modelRole *model.Role) (*identity.Role, error) {
	var rspRole = identity.Role{}
	var err error

	b, err := json.Marshal(*modelRole)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &rspRole)
	if err != nil {
		return nil, err
	}

	return &rspRole, nil
}

// groupRspToModel 은 identity 의 Group 을 model 의 Group 으로 변환 하는 함수이다
func groupRspToModel(g *identity.Group) (*model.Group, error) {
	group := model.Group{TenantID: g.Tenant.Id}

	b, err := json.Marshal(g)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &group)
	if err != nil {
		return nil, err
	}

	return &group, nil
}

func groupModelToRsp(modelGroup *model.Group) (*identity.Group, error) {
	var rspGroup = identity.Group{}
	var err error

	b, err := json.Marshal(*modelGroup)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &rspGroup)
	if err != nil {
		return nil, err
	}

	return &rspGroup, nil
}

func solutionModelToRsp(modelSolution *model.TenantSolution) (*identity.Solution, error) {
	var rspSolution = identity.Solution{}

	b, err := json.Marshal(*modelSolution)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &rspSolution)
	if err != nil {
		return nil, err
	}

	return &rspSolution, nil
}

func tenantModelToRsp(modelTenant *model.Tenant) (*identity.Tenant, error) {
	var rspTenant identity.Tenant

	b, err := json.Marshal(modelTenant)
	if err != nil {
		return nil, err
	}
	//TODO: bool -> wrapper.Boolean 에러 해결해야함
	err = json.Unmarshal(b, &rspTenant)

	rspTenant.UseFlag = &wrappers.BoolValue{Value: modelTenant.UseFlag}

	return &rspTenant, nil
}

func tenantRspToModel(t *identity.Tenant) (*model.Tenant, error) {
	var tenant model.Tenant

	b, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &tenant)

	tenant.UseFlag = t.GetUseFlag().GetValue()

	return &tenant, nil
}

func getPrivateKeyFromFile(path string) (*rsa.PrivateKey, error) {
	pemString, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(pemString)
	if block == nil {
		return nil, err
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func isAdmin(userRoles []*identity.Role) bool {
	for _, r := range userRoles {
		if r.Id == Role[constant.Admin].ID {
			return true
		}
	}
	return false
}

func isManager(userRoles []*identity.Role) bool {
	for _, g := range userRoles {
		if g.Id == Role[constant.Manager].ID {
			return true
		}
	}
	return false
}

func reportEvent(ctx context.Context, eventCode, errorCode string, eventContents interface{}) {
	id, err := metadata.GetTenantID(ctx)
	if err != nil {
		logger.Warnf("Could not report event. cause: %v", err)
		return
	}

	err = event.ReportEvent(id, eventCode, errorCode, event.WithContents(eventContents))
	if err != nil {
		logger.Warnf("Could not report event. cause: %v", err)
	}
}

func encodePassword(password string) string {
	b := sha256.Sum256([]byte(password))
	return hex.EncodeToString(b[:])
}
