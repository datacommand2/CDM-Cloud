package wrapper

import (
	"github.com/datacommand2/cdm-cloud/common/logger"

	"github.com/micro/go-micro/v2/errors"

	"net/http"
	"strings"
)

const (
	loginPath        = "/identity/auth"
	sessionCheckPath = "/identity/sessions/check"
)

// Wrapper http.Handler wrapper를 위한 인터페이스
type Wrapper interface {
	Wrap(handler http.Handler) http.Handler
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	ce := errors.Parse(err.Error())

	switch ce.Code {
	case 0:
		ce.Code = 500
		ce.Status = http.StatusText(500)
		ce.Detail = "error during request: " + ce.Detail
		w.WriteHeader(500)
	default:
		w.WriteHeader(int(ce.Code))
	}

	// response content type
	w.Header().Set("Content-Type", "application/json")

	// Set trailers
	if strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
		w.Header().Set("Trailer", "grpc-status")
		w.Header().Set("Trailer", "grpc-message")
		w.Header().Set("grpc-status", "13")
		w.Header().Set("grpc-message", ce.Detail)
	}

	_, werr := w.Write([]byte(ce.Error()))
	if werr != nil {
		logger.Error(werr)
	}
}
