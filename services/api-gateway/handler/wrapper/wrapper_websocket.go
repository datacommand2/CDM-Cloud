package wrapper

import (
	commonMetadata "github.com/datacommand2/cdm-cloud/common/metadata"
	"net/http"
)

type websocketHandler struct {
}

// NewWebSocketHandler websocketHandler 구조체 생성
func NewWebSocketHandler() Wrapper {
	return &websocketHandler{}
}

// Wrap websocket protocol 에서 header 에 값을 넘겨주지 못해
// 세션키와 테넌트를 query parameter 넘기기로 함
func (h *websocketHandler) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// websocket protocol 에서 header 에 값을 넘겨주지 못해
		// 세션키와 테넌트를 query parameter 넘기기로 함
		params := r.URL.Query()

		if v, ok := params[commonMetadata.HeaderAuthenticatedSession]; ok {
			r.Header.Set(commonMetadata.HeaderAuthenticatedSession, v[0])
		}

		if v, ok := params[commonMetadata.HeaderTenantID]; ok {
			r.Header.Set(commonMetadata.HeaderTenantID, v[0])
		}

		// query 에서는 세션키와 테넌트를 제거한다.
		delete(params, commonMetadata.HeaderAuthenticatedSession)
		delete(params, commonMetadata.HeaderTenantID)
		r.URL.RawQuery = params.Encode()

		next.ServeHTTP(w, r)
	})
}
