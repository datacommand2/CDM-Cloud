package handler

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/datacommand2/cdm-cloud/common/errors"
	random "math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/datacommand2/cdm-cloud/common/config"
	"github.com/datacommand2/cdm-cloud/common/database/model"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/datacommand2/cdm-cloud/common/metadata"
	"github.com/datacommand2/cdm-cloud/common/store"
	identity "github.com/datacommand2/cdm-cloud/services/identity/proto"
	"github.com/jinzhu/gorm"
)

// SessionPayload 는 세션을 생성하기위해 필요한 데이터를 저장하는 구조체이다
type SessionPayload struct {
	ID          uint64 `json:"id"`
	CreateDate  int64  `json:"create_date"`
	ExpiryDate  int64  `json:"expiry_date"`
	MagicNumber uint64 `json:"magic_number"`
	ClientIP    string `json:"client_ip"`
}

const (
	storeKeyPrefix                 = "cdm.cloud.user.sessions"
	defaultPrivateKeyPath          = "/etc/ssl/cdm-cloud/"
	defaultUserSessionTimeout      = 30
	defaultUserPasswordChangeCycle = 90
)

func generatePayload(ctx context.Context, userID uint64, magicNumber uint64, timeout int64) SessionPayload {
	ip, err := metadata.GetClientIP(ctx)
	if err != nil {
		logger.Warn("Unknown client IP address")
	}

	return SessionPayload{
		ID:          userID,
		CreateDate:  time.Now().Unix(),
		ExpiryDate:  time.Now().Unix() + (timeout * 60),
		MagicNumber: magicNumber,
		ClientIP:    ip,
	}
}

// generateSessionKey 는 세션 키를 생성하는 함수이다
func generateSessionKey(payload SessionPayload, privateKey *rsa.PrivateKey) (string, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return "", errors.Unknown(err)
	}

	hashed := sha256.Sum256(b)
	sig, err := rsa.SignPSS(rand.Reader, privateKey, crypto.SHA256, hashed[:], nil)
	if err != nil {
		return "", errors.Unknown(err)
	}

	return fmt.Sprintf("%s.%s",
		base64.URLEncoding.EncodeToString(b),
		base64.URLEncoding.EncodeToString(sig),
	), nil
}

func newSession(ctx context.Context, db *gorm.DB, tenantID, userID uint64, privateKey *rsa.PrivateKey, force bool) (string, error) {
	var err error
	var sessionKey string

	storeKey := storeKeyPrefix + "." + strconv.FormatUint(userID, 10)
	_, err = store.Get(storeKey)
	switch {
	case err == nil:
		if force {
			// Todo : 기존 접속한 유저에게 메시지 보내줘야함.
			logger.Warnf("Force logout user(%d).", userID)
			err = store.Delete(storeKey)
			if err != nil {
				return "", errors.UnusableStore(err)
			}
		} else {
			return "", alreadyLogin(userID)
		}

	case err != store.ErrNotFoundKey:
		return "", errors.UnusableStore(err)
	}

	//Init random seed
	random.Seed(time.Now().UnixNano())
	magicNumber := random.Uint64()

	var timeout int64
	var cfg *config.Config
	if cfg = config.TenantConfig(db, tenantID, config.UserSessionTimeout); cfg != nil {
		timeout, err = cfg.Value.Int64()
	}

	if cfg == nil || err != nil {
		timeout = defaultUserSessionTimeout
	}

	payload := generatePayload(ctx, userID, magicNumber, timeout)
	logger.Infof("User(ID: %d, IP: %v) logged in at %v. Session is going to expire at %v.", userID, payload.ClientIP, payload.CreateDate, payload.ExpiryDate)
	sessionKey, err = generateSessionKey(payload, privateKey)
	if err != nil {
		return "", err
	}

	if err = store.Put(storeKey, sessionKey, store.PutTTL(time.Duration(timeout)*time.Minute)); err != nil {
		return "", errors.UnusableStore(err)
	}

	return sessionKey, nil
}

func updateSession(ctx context.Context, db *gorm.DB, userID, tenantID, magicNumber uint64, privateKey *rsa.PrivateKey) (string, error) {
	var timeout int64
	var cfg *config.Config
	var err error
	if cfg = config.TenantConfig(db, tenantID, config.UserSessionTimeout); cfg != nil {
		timeout, err = cfg.Value.Int64()
	}

	if cfg == nil || err != nil {
		timeout = defaultUserSessionTimeout
	}

	payload := generatePayload(ctx, userID, magicNumber, timeout)
	logger.Infof("User(ID: %d, IP:%v) session updated at %v. Session is going to expire at %v.", userID, payload.ClientIP, payload.CreateDate, payload.ExpiryDate)
	sessionKey, err := generateSessionKey(payload, privateKey)
	if err != nil {
		return "", err
	}

	if err = store.Put(storeKeyPrefix+"."+strconv.FormatUint(userID, 10), sessionKey, store.PutTTL(time.Duration(timeout)*time.Minute)); err != nil {
		return "", errors.UnusableStore(err)
	}
	return sessionKey, nil
}

func validateDeleteSession(reqUser *identity.User, session string) (uint64, error) {
	p, _, err := validateSession(session)
	if err != nil {
		return 0, err
	}

	var payload *SessionPayload
	if err = json.Unmarshal(p, &payload); err != nil {
		return 0, errors.Unknown(err)
	}

	if reqUser.Id != payload.ID {
		return 0, errors.InvalidParameterValue("session", session, "unknown session")
	}

	return payload.ID, nil
}

func validateRevokeSession(db *gorm.DB, reqUser *identity.User, tid uint64, session string) (uint64, error) {
	p, _, err := validateSession(session)
	if err != nil {
		return 0, err
	}

	var payload *SessionPayload
	if err = json.Unmarshal(p, &payload); err != nil {
		return 0, errors.Unknown(err)
	}

	// 최고 관리자는 강제로 로그아웃 시킬 수 없음.
	if !isAdmin(reqUser.Roles) && adminUser.ID == payload.ID {
		return 0, errors.InvalidParameterValue("session", session, "invalid session")
	}

	var u model.User
	err = db.Where("id = ? ANd tenant_id = ?", payload.ID, tid).First(&u).Error
	switch {
	case errors.Equal(err, gorm.ErrRecordNotFound):
		return 0, errors.InvalidParameterValue("session", session, "invalid session")

	case err != nil:
		return 0, errors.UnusableDatabase(err)
	}

	return payload.ID, nil
}

func deleteSession(id uint64) error {
	storeKey := storeKeyPrefix + "." + strconv.FormatUint(id, 10)
	_, err := store.Get(storeKey)
	switch {
	case err == store.ErrNotFoundKey:
		return unknownSession(id)

	case err != nil:
		return errors.UnusableStore(err)
	}

	if err = store.Delete(storeKeyPrefix + "." + strconv.FormatUint(id, 10)); err != nil {
		return errors.UnusableStore(err)
	}
	return nil
}

func checkLoginRestricted(db *gorm.DB, user *model.User) error {
	if user.LastLoginFailedCount == nil || uint64(*user.LastLoginFailedCount) == 0 {
		return nil
	}

	var cfg *config.Config
	if cfg = config.TenantConfig(db, user.TenantID, config.UserLoginRestrictionEnable); cfg == nil {
		return errors.UnusableDatabase(nil)
	}

	if enable, err := cfg.Value.Bool(); err != nil {
		return errors.Unknown(err)
	} else if !enable {
		return nil
	}

	if cfg = config.TenantConfig(db, user.TenantID, config.UserLoginRestrictionTryCount); cfg == nil {
		return errors.UnusableDatabase(nil)
	}

	tryCount, err := cfg.Value.Uint64()
	if err != nil {
		return errors.Unknown(err)
	}

	if cfg = config.TenantConfig(db, user.TenantID, config.UserLoginRestrictionTime); cfg == nil {
		return errors.UnusableDatabase(nil)
	}

	restrictionTime, err := cfg.Value.Int64()
	if err != nil {
		return errors.Unknown(err)
	}

	if tryCount == 0 {
		logger.Warnf("user_login_restriction_try_count is zero. tenant: %d", user.TenantID)
		return nil
	}

	if uint64(*user.LastLoginFailedCount)%tryCount == 0 && *user.LastLoginFailedAt+restrictionTime > time.Now().Unix() {
		return loginRestricted(
			user.Account,
			*user.LastLoginFailedCount,
			*user.LastLoginFailedAt,
			*user.LastLoginFailedAt+restrictionTime,
		)
	}

	return nil
}

func updateUserLoginFailedInfo(db *gorm.DB, req *identity.LoginRequest) error {
	user := model.User{}

	err := db.Where("account = ?", req.Account).First(&user).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return errors.Unknown(err)

	case err != nil:
		return errors.UnusableDatabase(err)
	}

	if user.LastLoginFailedCount == nil {
		user.LastLoginFailedCount = new(uint)
	}

	if user.LastLoginFailedAt == nil {
		user.LastLoginFailedAt = new(int64)
	}

	*user.LastLoginFailedAt = time.Now().Unix()
	*user.LastLoginFailedCount = *user.LastLoginFailedCount + 1

	if err := db.Save(&user).Error; err != nil {
		return errors.UnusableDatabase(err)
	}

	return nil
}

func validateUserLogin(db *gorm.DB, req *identity.LoginRequest) (*model.User, error) {
	if len(req.Account) == 0 {
		return nil, errors.RequiredParameter("account")
	}

	if len(req.Account) > accountLength {
		return nil, errors.LengthOverflowParameterValue("account", req.Account, accountLength)
	}

	if len(req.Password) == 0 {
		return nil, errors.RequiredParameter("password")
	}

	if len(req.Password) > passwordLength {
		return nil, errors.LengthOverflowParameterValue("password", "******", passwordLength)
	}

	// find user
	user := model.User{}
	err := db.Where("account = ?", req.Account).First(&user).Error
	switch {
	case err == gorm.ErrRecordNotFound:
		return nil, notFoundUser(req.Account)
	case err != nil:
		return nil, errors.UnusableDatabase(err)
	}

	// check login restricted
	if err := checkLoginRestricted(db, &user); err != nil {
		return nil, err
	}

	// check password
	if user.Password != req.Password {
		return nil, incorrectPassword()
	}

	return &user, nil
}

func loginUser(ctx context.Context, db *gorm.DB, req *identity.LoginRequest) (*identity.User, error) {
	var err error
	var user *model.User
	user, err = validateUserLogin(db, req)
	if err != nil {
		return nil, err
	}

	// update user
	if user.LastLoggedInAt == nil {
		user.LastLoggedInAt = new(int64)
	}

	if user.LastLoggedInIP == nil {
		user.LastLoggedInIP = new(string)
	}

	ip, err := metadata.GetClientIP(ctx)
	if err != nil {
		return nil, errors.InvalidRequest(ctx)
	}

	*user.LastLoggedInAt = time.Now().Unix()
	*user.LastLoggedInIP = ip

	if user.LastLoginFailedCount != nil {
		*user.LastLoginFailedCount = 0
	}

	var cfg *config.Config
	var userPasswordChangeCycle int64
	if cfg = config.TenantConfig(db, user.TenantID, config.UserPasswordChangeCycle); cfg != nil {
		userPasswordChangeCycle, err = cfg.Value.Int64()
	}

	if cfg == nil || err != nil {
		userPasswordChangeCycle = defaultUserPasswordChangeCycle
	}

	if user.PasswordUpdatedAt == nil {
		user.PasswordUpdatedAt = new(int64)
	}

	if user.PasswordUpdateFlag == nil {
		user.PasswordUpdateFlag = new(bool)
	}

	if userPasswordChangeCycle != 0 && time.Now().Unix()-*user.PasswordUpdatedAt >= (int64(time.Hour)*24*userPasswordChangeCycle/int64(time.Second)) {
		*user.PasswordUpdateFlag = true
	}

	if err := db.Save(&user).Error; err != nil {
		return nil, errors.UnusableDatabase(err)
	}

	rspUser, err := userModelToRsp(user)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	rspUser.Tenant, err = getTenant(db, user.TenantID)
	if err != nil {
		return nil, err
	}

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
		rspUser.Groups, err = getGroups(db, user.TenantID)
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

func validateSession(session string) ([]byte, []byte, error) {
	cut := strings.Split(session, ".")
	if len(cut) != 2 {
		return nil, nil, invalidSession(session)
	}

	p, err := base64.URLEncoding.DecodeString(cut[0])
	if err != nil {
		return nil, nil, invalidSession(session)
	}

	sig, err := base64.URLEncoding.DecodeString(cut[1])
	if err != nil {
		return nil, nil, invalidSession(session)
	}

	return p, sig, nil
}

// verifySession 은 세션유효성을 확인하는 함수이다
func verifySession(ctx context.Context, session string, privateKey *rsa.PrivateKey) (*SessionPayload, error) {
	// check session validity
	p, sig, err := validateSession(session)
	if err != nil {
		logger.Errorf("Could not verify session. cause: %+v", err)
		return nil, errors.UnauthenticatedRequest(ctx)
	}

	// VerifyPSS
	hashed := sha256.Sum256(p)
	err = rsa.VerifyPSS(&privateKey.PublicKey, crypto.SHA256, hashed[:], sig, nil)
	if err != nil {
		return nil, unverifiedSession(session, err)
	}

	// request payload
	var reqPayload *SessionPayload
	if err = json.Unmarshal(p, &reqPayload); err != nil {
		return nil, errors.Unknown(err)
	}

	// Is exist in store
	value, err := store.Get(storeKeyPrefix + "." + strconv.FormatUint(reqPayload.ID, 10))
	switch {
	case err == store.ErrNotFoundKey:
		logger.Errorf("Could not verify session. cause: %+v", err)
		return nil, errors.UnauthenticatedRequest(ctx)

	case err != nil:
		return nil, errors.UnusableStore(err)
	}

	p, _, err = validateSession(value)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	var storedPayload SessionPayload
	err = json.Unmarshal(p, &storedPayload)
	if err != nil {
		return nil, errors.Unknown(err)
	}

	// Check client ip
	ip, err := metadata.GetClientIP(ctx)
	if err != nil {
		return nil, errors.InvalidRequest(ctx)
	}

	if ip != storedPayload.ClientIP || reqPayload.MagicNumber != storedPayload.MagicNumber {
		return nil, errors.UnauthenticatedRequest(ctx)
	}

	// Check expiryDate
	logger.Infof("expiry date : %v, now : %v", reqPayload.ExpiryDate, time.Now().Unix())
	if reqPayload.ExpiryDate < time.Now().Unix() {
		logger.Error("expiredSession : %v, now : %v", reqPayload.ExpiryDate, time.Now().Unix())
		return nil, expiredSession(session)
	}

	return reqPayload, nil
}

func getSession(userID uint64) (*identity.Session, error) {
	var storeKey = storeKeyPrefix + "." + strconv.FormatUint(userID, 10)

	sessionKey, err := store.Get(storeKey)
	switch {
	case err == store.ErrNotFoundKey:
		return nil, nil

	case err != nil:
		return nil, errors.UnusableStore(err)

	default:
		return &identity.Session{Key: sessionKey}, nil
	}
}
