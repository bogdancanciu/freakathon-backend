package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"log"
	"net/http"
)

func BindInterestsHooks(app core.App) {
	app.OnBeforeServe().Add(SetInterests(app))
}

func SetInterests(app core.App) func(e *core.ServeEvent) error {
	return func(e *core.ServeEvent) error {
		e.Router.POST("/api/interests", func(c echo.Context) error {
			session := getSessionToken(c.Request())
			userId, sessionErr := UserIdFromSession(session)
			if sessionErr != nil {
				return sessionErr
			}

			reqBody, err := readBody(c.Request())
			if err != nil {
				log.Println("Failed to read request body", err)
				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			interests, err := decodeInterests(reqBody)
			if err != nil {
				return apis.NewApiError(http.StatusInternalServerError, err.Error(), "")
			}

			userRecord, err := app.Dao().FindRecordById("users", userId)
			userRecord.Set("interests", interests)

			if err := app.Dao().SaveRecord(userRecord); err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			return nil
		})
		return nil
	}
}

func decodeInterests(body []byte) ([]string, error) {
	var bodyData map[string][]string
	err := json.Unmarshal(body, &bodyData)
	if err != nil {
		return nil, err
	}

	interests, found := bodyData["interests"]
	if !found {
		return nil, fmt.Errorf("Malformed body")
	}

	return interests, nil
}
