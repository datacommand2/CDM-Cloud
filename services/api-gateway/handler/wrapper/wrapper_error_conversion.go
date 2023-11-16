package wrapper

import (
	"bufio"
	"encoding/json"
	"github.com/gobwas/ws"
	"github.com/micro/go-micro/v2/errors"
	"net"
	"net/http"
)

// errConversionResponseWriter 필드 변환을 위해 write 을 구현한 http.ResponseWriter
type errConversionResponseWriter struct {
	http.ResponseWriter
}

func (r *errConversionResponseWriter) parseError(detail string) interface{} {
	type message struct {
		Code     string `json:"code,omitempty"`
		Contents string `json:"contents,omitempty"`
	}

	var ret struct {
		Message message `json:"message,omitempty"`
	}
	var msg message
	if json.Unmarshal([]byte(detail), &msg) == nil {
		ret.Message = msg
	} else {
		ret.Message = message{Code: detail}
	}

	return &ret
}

// Hijack websocket 을 위한 hijacker 임베딩
func (r *errConversionResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := r.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, ws.ErrNotHijacker
}

func (r *errConversionResponseWriter) Write(buf []byte) (len int, err error) {
	var e errors.Error
	if json.Unmarshal(buf, &e) == nil && e.Detail != "" {
		if buf, err = json.Marshal(r.parseError(e.Detail)); err != nil {
			return 0, err
		}
	}

	return r.ResponseWriter.Write(buf)
}

// errorConversionHandler 는 error 필드 변환을 위한 구조체
type errorConversionHandler struct {
}

// NewErrorConversionHandler errorConversionHandler 구조체 생성
func NewErrorConversionHandler() Wrapper {
	return &errorConversionHandler{}
}

func (e *errorConversionHandler) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rsp = &errConversionResponseWriter{
			ResponseWriter: w,
		}
		next.ServeHTTP(rsp, r)

	})
}
