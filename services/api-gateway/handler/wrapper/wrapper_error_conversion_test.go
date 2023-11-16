package wrapper

import (
	"encoding/json"
	"fmt"
	"github.com/micro/go-micro/v2/errors"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func createEventMessage(code, contents string) string {
	var m = &struct {
		Code     string `json:"code,omitempty"`
		Contents string `json:"contents,omitempty"`
	}{
		Code:     code,
		Contents: contents,
	}
	bytes, _ := json.Marshal(m)
	return string(bytes)
}

var errorConversionHTTPHandler = NewErrorConversionHandler().Wrap(http.HandlerFunc(testHTTPHandler))

func testHTTPHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	switch r.URL.Path {
	case "403":
		err = errors.Forbidden("test", createEventMessage(r.Method, r.URL.Path))

	case "400":
		err = errors.BadRequest("test", createEventMessage(r.Method, r.URL.Path))

	case "401":
		err = errors.Unauthorized("test", createEventMessage(r.Method, r.URL.Path))

	case "500":
		err = errors.InternalServerError("test", createEventMessage(r.Method, r.URL.Path))

	default:
		err = errors.InternalServerError("test", "unknown error")
	}

	writeError(w, r, err)
}

func TestErrorConversionHandler(t *testing.T) {
	for _, tc := range []struct {
		url            string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			url:            "403",
			method:         http.MethodGet,
			expectedStatus: http.StatusForbidden,
		},
		{
			url:            "401",
			method:         http.MethodPut,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			url:            "400",
			method:         http.MethodPatch,
			expectedStatus: http.StatusBadRequest,
		},
		{
			url:            "500",
			method:         http.MethodDelete,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			url:            "default",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   fmt.Sprint("{\"message\":{\"code\":\"unknown error\"}}"),
		},
	} {
		req, err := http.NewRequest(
			tc.method,
			tc.url,
			nil,
		)
		if err != nil {
			assert.Fail(t, err.Error())
		}

		if tc.expectedBody == "" {
			tc.expectedBody = fmt.Sprintf("{\"message\":{\"code\":\"%s\",\"contents\":\"%s\"}}", tc.method, tc.url)
		}

		rsp := httptest.NewRecorder()
		errorConversionHTTPHandler.ServeHTTP(rsp, req)

		body, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			assert.Fail(t, err.Error())
			continue
		}

		assert.Equal(t, tc.expectedStatus, rsp.Code)
		assert.Equal(t, tc.expectedBody, string(body))
	}
}
