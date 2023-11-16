package wrapper

import (
	"net/http"
)

// recordResponseWriter http.ResponseWriter status , body 값을 가져오기 위한 wrapper 구조체
type recordResponseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader recordResponseWriter status 에 http 상태 코드를 저장 하는 함수
func (r *recordResponseWriter) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// recordHandler notification service에 접근 기록 요청 하기 위한 rpc client로 구성된 구조체
// TODO
type recordHandler struct {
}

// NewRecordHandler recordHandler 구조체 생성
func NewRecordHandler() Wrapper {
	return &recordHandler{}
}

// Wrap TODO notification service에 접근 기록 요청
func (r *recordHandler) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var recordResponse = &recordResponseWriter{w, http.StatusOK}
		next.ServeHTTP(recordResponse, r)
	})
}
