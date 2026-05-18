package testhelper

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type HTTPRequest struct {
	Method string
	Target string
	Body   io.Reader
	Header http.Header
}

func ServeHTTP(t testing.TB, handler http.Handler, request HTTPRequest) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(request.Method, request.Target, request.Body)
	for name, values := range request.Header {
		for _, value := range values {
			req.Header.Add(name, value)
		}
	}

	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	return res
}

func ServeJSON(t testing.TB, handler http.Handler, request HTTPRequest, status int, response any) *httptest.ResponseRecorder {
	t.Helper()

	res := ServeHTTP(t, handler, request)
	require.Equal(t, status, res.Code)
	DecodeJSONResponse(t, res, response)
	return res
}

func DecodeJSONResponse(t testing.TB, res *httptest.ResponseRecorder, response any) {
	t.Helper()

	require.NoError(t, json.NewDecoder(res.Body).Decode(response))
}
