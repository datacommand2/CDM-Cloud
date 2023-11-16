package wrapper

import (
	"github.com/datacommand2/cdm-cloud/common/metadata"
	"net/http"
	"strings"
)

type corsHandler struct {
}

// NewCORSHandler corsHandler 구조체 생성
func NewCORSHandler() Wrapper {
	return &corsHandler{}
}

// Wrap 인증,인가 요청 및 확인을 위한 함수
// login과 session 확인 인 경우 인증, 인가 과정을 걸치지 않는다.
func (auth *corsHandler) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Expose-Headers", metadata.HeaderAuthenticatedSession)

		if strings.ToUpper(r.Method) == "OPTIONS" {
			return
		}

		next.ServeHTTP(w, r)
	})
}
