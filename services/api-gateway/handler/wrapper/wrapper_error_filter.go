package wrapper

import (
	"bufio"
	"github.com/datacommand2/cdm-cloud/common/errors"
	"github.com/datacommand2/cdm-cloud/common/logger"
	"github.com/gobwas/ws"
	microError "github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/util/ctx"
	"net"
	"net/http"
	"regexp"
)

// errFilterResponseWriter http.ResponseWriter status , body 값을 가져오기 위한 wrapper 구조체
type errFilterResponseWriter struct {
	http.ResponseWriter
	body     []byte
	status   int
	hijacker http.Hijacker
}

// WriteHeader errFilterResponseWriter status 에 http 상태 코드를 저장 하는 함수
func (e *errFilterResponseWriter) WriteHeader(status int) {
	e.status = status
}

// Write errFilterResponseWriter body 에 buf 를 저장 하는 함수
func (e *errFilterResponseWriter) Write(buf []byte) (int, error) {
	e.body = append(e.body, buf...)
	return len(e.body), nil
}

func (e *errFilterResponseWriter) flush() {
	if e.hijacker != nil {
		return
	}
	e.ResponseWriter.WriteHeader(e.status)
	_, err := e.ResponseWriter.Write(e.body)
	if err != nil && err != http.ErrBodyNotAllowed {
		logger.Debugf("Could not write response body, cause: %v", err)
	}

}

func (e *errFilterResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	var ok bool
	if e.hijacker, ok = e.ResponseWriter.(http.Hijacker); ok {
		return e.hijacker.Hijack()
	}
	return nil, nil, ws.ErrNotHijacker
}

// errorFilterHandler 는 go micro error 필터 처리를 위한 구조체
type errorFilterHandler struct {
	invalidParameterErrorRegExps []*regexp.Regexp
}

// NewErrorFilterHandler errorFilterHandler 구조체 생성
func NewErrorFilterHandler() Wrapper {
	h := errorFilterHandler{}

	// invalid parameter error regular expressions
	for _, exp := range []string{
		"^grpc: failed to unmarshal the received message json: cannot unmarshal .* into Go value of type .*$",
		"^error during request: invalid character .* after top-level value$",
		"^error during request: unexpected end of JSON input$",
		"^error during request: invalid character .* in string escape code$",
		"^grpc: error while marshaling: json: error calling MarshalJSON for type \\*json.RawMessage: invalid character .*$",
		"^grpc: failed to unmarshal the received message unknown field .*$",
		"^grpc: failed to unmarshal the received message unexpected end of JSON input$",
		"^grpc: failed to unmarshal the received message invalid character .*$",
		"^grpc: error while marshaling: json: error calling MarshalJSON for type \\*json.RawMessage: unexpected end of JSON input$",
	} {
		h.invalidParameterErrorRegExps = append(h.invalidParameterErrorRegExps, regexp.MustCompile(exp))
	}

	return &h
}

// Wrap 에러 필터 처리 하는 함수
func (e *errorFilterHandler) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var rsp = &errFilterResponseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		next.ServeHTTP(rsp, r)

		if rsp.status != http.StatusInternalServerError {
			rsp.flush()
			return
		}

		err := microError.Parse(string(rsp.body)).Detail
		if err == registry.ErrNotFound.Error() {
			writeError(w, r, errors.StatusNotFound(ctx.FromRequest(r), "cdm-cloud.api_gateway.error_filter.failure-parse", "not_found_service", err))
			return
		}

		for _, regExp := range e.invalidParameterErrorRegExps {
			if regExp.MatchString(err) {
				writeError(w, r, errors.StatusBadRequest(ctx.FromRequest(r), "cdm-cloud.api_gateway.error_filter.failure-match_string", "invalid_parameter", err))
				return
			}
		}
		rsp.flush()
	})
}
