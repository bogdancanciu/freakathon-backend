package handlers

import (
	"bytes"
	"encoding/json"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestAppWithHooks(hooksBind func(app core.App)) func(t *testing.T) *tests.TestApp {
	return func(t *testing.T) *tests.TestApp {
		testDataDir := "../test_pb_data"
		testApp, err := tests.NewTestApp(testDataDir)
		if err != nil {
			t.Fatal(err)
		}

		hooksBind(testApp)

		return testApp
	}
}

func testRegisterBody(t *testing.T) io.Reader {
	requestBody := registerRequestBody{Username: "test", Password: "test"}
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		t.Errorf("Failed to serialize request body %s", err)
	}

	return bytes.NewReader(jsonBody)
}

func preRegisterTestUser(t *testing.T, _ *tests.TestApp, e *echo.Echo) {
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v2/register", testRegisterBody(t))
	// set default header
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	// execute request
	e.ServeHTTP(recorder, req)
}
