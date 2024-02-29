package handlers

import (
	"github.com/pocketbase/pocketbase/tests"
	"net/http"
	"testing"
)

func TestRegisterEndpoint(t *testing.T) {
	// API hooks are required by ApiScenario by default. However, we do not need them for our scenarios.
	eventsMap := map[string]int{"OnBeforeApiError": 0, "OnAfterApiError": 0}
	apiScenario := setupTestAppWithHooks(bindAuthHooks)

	scenarios := []tests.ApiScenario{
		{
			Name:           "issue request with no body",
			Method:         http.MethodPost,
			Url:            "/api/v2/register",
			ExpectedStatus: http.StatusBadRequest,
			ExpectedEvents: eventsMap,
			TestAppFactory: apiScenario,
		},
		{
			Name:           "register a valid user",
			Method:         http.MethodPost,
			Url:            "/api/v2/register",
			Body:           testRegisterBody(t),
			ExpectedStatus: http.StatusOK,
			ExpectedEvents: eventsMap,
			TestAppFactory: apiScenario,
		},
		{
			Name:           "register already existing user",
			Method:         http.MethodPost,
			Url:            "/api/v2/register",
			Body:           testRegisterBody(t),
			ExpectedStatus: http.StatusConflict,
			BeforeTestFunc: preRegisterTestUser,
			ExpectedEvents: eventsMap,
			TestAppFactory: apiScenario,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
