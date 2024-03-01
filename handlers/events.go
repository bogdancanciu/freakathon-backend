package handlers

import (
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
)

func BindEventsHooks(app core.App) {
	app.OnRecordBeforeCreateRequest("events").Add(func(e *core.RecordCreateEvent) error {
		sessionToken := getSessionToken(e.HttpContext.Request())
		userId, err := userIdFromSession(sessionToken)
		if err != nil {
			return err
		}

		e.Record.Set("user_id", userId)
		if err := app.Dao().SaveRecord(e.Record); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to create event.", "")
		}

		return nil
	})
}
