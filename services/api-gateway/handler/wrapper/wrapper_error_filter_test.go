package wrapper

import (
	"github.com/datacommand2/cdm-cloud/common/errors"
	testp "github.com/datacommand2/cdm-cloud/services/api-gateway/handler/test/proto"
	"github.com/google/uuid"
	microError "github.com/micro/go-micro/v2/errors"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/server"
	"github.com/micro/go-micro/v2/server/grpc"
	"github.com/stretchr/testify/assert"

	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"
)

var (
	rpcServer = grpc.NewServer(server.Name("api-gateway-test"))
	rspUUID   = uuid.New().String()[:30]
)

type testRPC struct {
}

func (t *testRPC) PutTest(ctx context.Context, in *testp.TestMessage, out *testp.Empty) error {
	switch {
	case in.Name == "StatusInternalServerError":
		return errors.StatusInternalServerError(ctx, "code.not.defined", "internal_server_error", errors.New("internal server error").Error())
	}
	return nil
}

func (t *testRPC) GetTest(ctx context.Context, in *testp.Empty, out *testp.Empty) error {
	return nil
}

func (t *testRPC) DeleteTest(ctx context.Context, in *testp.Empty, out *testp.Empty) error {
	return nil
}

func (t *testRPC) PostTest(ctx context.Context, in *testp.Empty, out *testp.Empty) error {
	return nil
}

func (t *testRPC) StreamTest(ctx context.Context, in *testp.TestMessage, stream testp.Test_StreamTestStream) error {
	for {
		if err := stream.Send(&testp.TestMessage{Name: rspUUID}); err != nil {
			panic(err)
		}
		time.Sleep(time.Second * 3)
	}
}

func TestErrorFilterHandler(t *testing.T) {
	s, err := registry.GetService(serviceName)
	if err != nil || len(s[0].Nodes) == 0 {
		assert.FailNow(t, "service discovery error(%v)", err)
	}

	address := s[0].Nodes[0].Address
	address = strings.Split(address, ":")[0]
	address = fmt.Sprintf("%s:%s", address, port)

	for _, tc := range []struct {
		body         string
		status       int
		expectedBody string
		desc         string
		path         string
	}{
		{
			body:         `{"ids":[{"id":1}],"flag":true}`,
			status:       http.StatusOK,
			expectedBody: "",
			path:         "/put",
			desc:         "normal 1",
		},
		{
			body:         `{"name":"StatusInternalServerError"}`,
			status:       http.StatusInternalServerError,
			expectedBody: "internal server error",
			path:         "/put",
			desc:         "normal 2 not error filter",
		},
		{
			body:         `{"ids":[{"id":1}],"flag":true}`,
			status:       http.StatusNotFound,
			expectedBody: registry.ErrNotFound.Error(),
			path:         "/service-not-found",
			desc:         "abnormal 2 invalid url",
		},
		{
			body:         `{"ids":[{"id":1}],"flag":"true"}`,
			status:       http.StatusBadRequest,
			expectedBody: "grpc: failed to unmarshal the received message json: cannot unmarshal string into Go value of type bool",
			path:         "/put",
			desc:         "abnormal 3 diff value type",
		},
		{
			body:         `{"ids":[{"id":"abc"}],"flag":"true"}`,
			status:       http.StatusBadRequest,
			expectedBody: "grpc: failed to unmarshal the received message invalid character 'a' looking for beginning of value",
			path:         "/put",
			desc:         "abnormal 4 invalid character value",
		},
		{
			body:         `{"ids":[{"id":1],"flag":true}`,
			status:       http.StatusBadRequest,
			expectedBody: "grpc: error while marshaling: json: error calling MarshalJSON for type *json.RawMessage: invalid character ']' after object key:value pair",
			path:         "/put",
			desc:         "abnormal 5 object key:value pair error",
		},
		{
			body:         `{"flag":true, "datacommmand"}`,
			status:       http.StatusBadRequest,
			expectedBody: "grpc: error while marshaling: json: error calling MarshalJSON for type *json.RawMessage: invalid character '}' after object key",
			path:         "/put",
			desc:         "abnormal 6 not key",
		},
		{
			body:         `{"ids":[{"id":1}}],"flag":true}`,
			status:       http.StatusBadRequest,
			expectedBody: "grpc: error while marshaling: json: error calling MarshalJSON for type *json.RawMessage: invalid character '}' after array element",
			path:         "/put",
			desc:         "abnormal 8 invalid array element",
		},
		{
			body:         `{"flag":true,"name":, }`,
			status:       http.StatusBadRequest,
			expectedBody: "grpc: error while marshaling: json: error calling MarshalJSON for type *json.RawMessage: invalid character ',' looking for beginning of value",
			path:         "/put",
			desc:         "abnormal 9 invalid begin value",
		},
		{
			body:         `{"flag":true, "gid":"", "name":"datacommmand"}`,
			status:       http.StatusBadRequest,
			expectedBody: "grpc: failed to unmarshal the received message unexpected end of JSON input",
			path:         "/put",
			desc:         "abnormal 10 unexpected end of JSON input",
		},
		{
			body:         `{"flag":true,"name":"datacommmand","id2":111}`,
			status:       http.StatusBadRequest,
			expectedBody: "grpc: failed to unmarshal the received message unknown field \"id2\" in testp.TestMessage",
			path:         "/put",
			desc:         "abnormal 11 unknown field",
		},
		{
			body:         `{`,
			status:       http.StatusBadRequest,
			expectedBody: "grpc: error while marshaling: json: error calling MarshalJSON for type *json.RawMessage: unexpected end of JSON input",
			path:         "/put",
			desc:         "abnormal 12 unexpected end of JSON input",
		},
	} {
		req, err := http.NewRequest(
			http.MethodPut,
			fmt.Sprintf("http://%s%s", address, tc.path),
			bytes.NewBuffer([]byte(tc.body)),
		)
		if err != nil {
			assert.Fail(t, "http make request error(%v)", err)
			continue
		}

		rsp, err := (&http.Client{}).Do(req)
		if err != nil {
			assert.Fail(t, err.Error())
			continue
		}

		body, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			assert.Fail(t, err.Error())
			continue
		}

		if tc.expectedBody != "" {
			var (
				e map[string]interface{}
				d string
			)
			assert.NoError(t, json.Unmarshal([]byte(microError.Parse(string(body)).Detail), &e), tc.desc)
			assert.NoError(t, json.Unmarshal([]byte(e["contents"].(string)), &d), tc.desc)
			assert.Equal(t, tc.expectedBody, d, tc.desc)
		}
		assert.Equal(t, tc.status, rsp.StatusCode, tc.desc)
	}
}

func TestStreamErrorFilterHandler(t *testing.T) {
	s, err := registry.GetService(serviceName)
	if err != nil || len(s[0].Nodes) == 0 {
		assert.FailNow(t, "service discovery error(%v)", err)
	}

	address := s[0].Nodes[0].Address
	address = strings.Split(address, ":")[0]
	address = fmt.Sprintf("%s:%s", address, port)

	ws, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%s%s/1", address, "/stream"), http.Header{
		"Content-Type": []string{"application/json"},
	})

	assert.NoError(t, err)
	defer ws.Close()
	_, p, err := ws.ReadMessage()
	assert.NoError(t, err)

	mapInterface := make(map[string]interface{})
	_ = json.Unmarshal(p, &mapInterface)
	assert.Equal(t, mapInterface["name"], rspUUID)
}
