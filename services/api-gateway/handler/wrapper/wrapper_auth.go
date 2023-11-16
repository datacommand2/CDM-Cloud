package wrapper

import (
	"context"
	"encoding/json"
	"github.com/datacommand2/cdm-cloud/common/constant"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/logger"
	commonMetadata "github.com/datacommand2/cdm-cloud/common/metadata"
	identity "github.com/datacommand2/cdm-cloud/services/identity/proto"
	"github.com/micro/go-micro/v2/api/router"
	apiRegistry "github.com/micro/go-micro/v2/api/router/registry"
	"github.com/micro/go-micro/v2/client/grpc"
	"github.com/micro/go-micro/v2/metadata"
	"github.com/micro/go-micro/v2/registry"
	"net/http"
	"strings"
)

type authHandler struct {
	identityClient identity.IdentityService
	router         router.Router
}

// NewAuthHandler authHandler 구조체 생성
func NewAuthHandler() Wrapper {
	return &authHandler{
		identityClient: identity.NewIdentityService(constant.ServiceIdentity, grpc.NewClient()),
		router:         apiRegistry.NewRouter(),
	}
}

func (auth *authHandler) route(ctx context.Context, r *http.Request) (string, error) {
	service, err := auth.router.Route(r)
	switch {
	case err != nil && err == registry.ErrNotFound:
		return "", errors.StatusNotFound(ctx, "cdm-cloud.api_gateway.route_auth.failure-route", "not_found_service", err.Error())

	case err != nil:
		return "", errors.StatusInternalServerError(ctx, "cdm-cloud.api_gateway.route_auth.failure-route", "unknown", err.Error())

	default:
		return service.Endpoint.Name, nil
	}

}

// Wrap 인증,인가 요청 및 확인을 위한 함수
// login 과 session 확인 인 경우 인증, 인가 과정을 걸치지 않는다.
func (auth *authHandler) Wrap(next http.Handler) http.Handler {
	services := make(map[string]string)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := strings.Split(r.RemoteAddr, ":")[0]
		logger.Infof("[%v:%v] received request from [%v].", r.Method, r.URL.Path, ip)
		switch {
		case (r.URL.Path == sessionCheckPath && r.Method != http.MethodDelete) ||
			(r.URL.Path == loginPath && r.Method != http.MethodDelete):
			r.Header.Set(commonMetadata.HeaderClientIP, ip)

		default:
			sessionKey := r.Header.Get(commonMetadata.HeaderAuthenticatedSession)
			ctx := metadata.Set(context.Background(), commonMetadata.HeaderClientIP, ip)
			ctx = metadata.Set(ctx, commonMetadata.HeaderAuthenticatedSession, sessionKey)

			var tenantID string
			if tenantID = r.Header.Get(commonMetadata.HeaderTenantID); tenantID == "" {
				tenantID = "0"
			}
			ctx = metadata.Set(ctx, commonMetadata.HeaderTenantID, tenantID)

			var err error
			var rsp *identity.UserResponse
			if rsp, err = auth.identityClient.VerifySession(ctx, &identity.Empty{}); err != nil {
				writeError(w, r, err)
				return
			}
			w.Header().Set(commonMetadata.HeaderAuthenticatedSession, rsp.User.Session.Key)

			var b []byte
			if b, err = json.Marshal(rsp.User); err != nil {
				writeError(w, r, errors.StatusInternalServerError(ctx, "cdm-cloud.api_gateway.new_auth_handler.failure-marshal", "unknown", err.Error()))
				return
			}
			ctx = metadata.Set(context.Background(), commonMetadata.HeaderAuthenticatedUser, string(b))
			ctx = metadata.Set(ctx, commonMetadata.HeaderTenantID, tenantID)

			var endpoint string
			if endpoint, err = auth.route(ctx, r); err != nil {
				if errors.Equal(err, registry.ErrNotFound) && services[r.Method+":"+r.URL.Path] != "" {
					logger.Warnf("Not found service and try to get cached data from local variable.")
					endpoint = services[r.Method+":"+r.URL.Path]
				} else {
					writeError(w, r, err)
					return
				}
			} else {
				services[r.Method+":"+r.URL.Path] = endpoint
			}

			var req *identity.CheckAuthorizationRequest
			req = &identity.CheckAuthorizationRequest{Endpoint: endpoint}
			if _, err = auth.identityClient.CheckAuthorization(ctx, req); err != nil {
				writeError(w, r, err)
				return
			}

			r.Header.Set(commonMetadata.HeaderAuthenticatedUser, string(b))
			reqID := commonMetadata.GenRequestID()
			r.Header.Set(commonMetadata.HeaderRequestID, reqID)
			logger.Infof("Request id[%v] was issued for the [%v:%v] request of [%v]",
				reqID, r.Method, r.URL, rsp.User.Account)
		}
		next.ServeHTTP(w, r)
	})
}
