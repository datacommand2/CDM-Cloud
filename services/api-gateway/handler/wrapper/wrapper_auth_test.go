package wrapper

import (
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/metadata"
	"github.com/datacommand2/cdm-cloud/services/identity/handler"
	identity "github.com/datacommand2/cdm-cloud/services/identity/proto"
	"github.com/micro/go-micro/v2/server"
	"github.com/micro/go-micro/v2/server/grpc"
	"github.com/stretchr/testify/assert"

	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	identityServer = grpc.NewServer(server.Name(constant.ServiceIdentity))
	emptyHandler   = NewAuthHandler().Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { return }))
)

var (
	errNotSessionCheckClientIP = errors.New("session check, client ip not exists")
	errNotSessionCheckKey      = errors.New("session check, Session key not exists")
	errNotSessionCheckTenantID = errors.New("session check, Session key not exists")

	errNotAuthUser     = errors.New("authorization, User info not exists")
	errNotAuthTenantID = errors.New("authorization, Tenant id not exists")
)

type identityHandler struct {
	*handler.IdentityHandler
}

func (i *identityHandler) VerifySession(ctx context.Context, _ *identity.Empty, out *identity.UserResponse) error {
	if _, err := metadata.GetTenantID(ctx); err != nil {
		return errors.StatusInternalServerError(ctx, "code.not.defined", "not_session_check", errNotSessionCheckTenantID)
	}

	if _, err := metadata.GetClientIP(ctx); err != nil {
		return errors.StatusInternalServerError(ctx, "code.not.defined", "not_session_check", errNotSessionCheckClientIP)
	}

	if _, err := metadata.GetAuthenticatedSession(ctx); err != nil {
		return errors.StatusInternalServerError(ctx, "code.not.defined", "not_session_check", errNotSessionCheckKey)
	}

	out.User = &identity.User{Session: &identity.Session{Key: "session-key"}}
	return nil
}

func (i *identityHandler) CheckAuthorization(ctx context.Context, _ *identity.CheckAuthorizationRequest, _ *identity.MessageResponse) error {
	if _, err := metadata.GetAuthenticatedUser(ctx); err != nil {
		return errors.StatusInternalServerError(ctx, "code.not.defined", "not_auth_user", errNotAuthUser)
	}

	if _, err := metadata.GetTenantID(ctx); err != nil {
		return errors.StatusInternalServerError(ctx, "code.not.defined", "not_auth_user", errNotAuthTenantID)
	}
	return nil
}

func (i *identityHandler) GetRoles(_ context.Context, _ *identity.GetRolesRequest, _ *identity.RolesResponse) error {
	return nil
}

func TestAuthHandler(t *testing.T) {
	for _, tcData := range []struct {
		path   string
		method string
		header map[string]string
		status int
		desc   string
	}{
		{
			path:   "/identity/auth",
			method: http.MethodPost,
			header: map[string]string{metadata.HeaderClientIP: "my-ip"},
			status: http.StatusOK,
			desc:   "normal case1",
		}, {
			path:   "/identity/sessions/check",
			method: http.MethodGet,
			header: map[string]string{metadata.HeaderClientIP: "why", metadata.HeaderAuthenticatedSession: "data"},
			status: http.StatusOK,
			desc:   "normal case2",
		},
		{
			path:   "/identity/roles",
			method: http.MethodGet,
			header: map[string]string{metadata.HeaderClientIP: "my-ip", metadata.HeaderAuthenticatedSession: "data",
				metadata.HeaderTenantID: "1"},
			status: http.StatusOK,
			desc:   "normal case3",
		},
		{
			path:   "/identity/roles",
			method: http.MethodGet,
			header: map[string]string{metadata.HeaderAuthenticatedSession: "data", metadata.HeaderTenantID: "1"},
			status: http.StatusInternalServerError,
			desc:   "abnormal case1, client ip not set",
		},
		{
			path:   "/identity/roles",
			method: http.MethodGet,
			header: map[string]string{metadata.HeaderClientIP: "my-ip", metadata.HeaderTenantID: "1"},
			status: http.StatusInternalServerError,
			desc:   "abnormal case2, session key not set",
		},
		{
			path:   "/identity/roles",
			method: http.MethodGet,
			header: map[string]string{metadata.HeaderClientIP: "my-ip", metadata.HeaderAuthenticatedSession: "data"},
			status: http.StatusOK,
			desc:   "abnormal case3, tenant id not set",
		},
		{
			path:   "/invalid-endpoint",
			method: http.MethodGet,
			header: map[string]string{metadata.HeaderClientIP: "my-ip", metadata.HeaderAuthenticatedSession: "data", metadata.HeaderTenantID: "1"},
			status: http.StatusNotFound,
			desc:   "abnormal case4, invalid url",
		},
	} {
		req, err := http.NewRequest(
			tcData.method,
			fmt.Sprintf("%s", tcData.path),
			nil,
		)
		if err != nil {
			assert.Fail(t, "http make request error(%v)", err)
			continue
		}

		for key, value := range tcData.header {
			if key == metadata.HeaderClientIP {
				req.RemoteAddr = value
			} else {
				req.Header.Set(key, value)
			}

		}

		rsp := httptest.NewRecorder()
		emptyHandler.ServeHTTP(rsp, req)

		assert.Equal(t, tcData.status, rsp.Code, tcData.desc)
	}
}
