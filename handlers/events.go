package handlers

import (
	"encoding/json"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"net/http"
)

func BindEventsHooks(app core.App) {
	app.OnRecordBeforeCreateRequest("events").Add(func(e *core.RecordCreateEvent) error {
		sessionToken := getSessionToken(e.HttpContext.Request())
		userId, err := UserIdFromSession(sessionToken)
		if err != nil {
			return err
		}

		attendants := []string{userId}

		e.Record.Set("user_id", userId)
		e.Record.Set("attendants", attendants)
		if err := app.Dao().SaveRecord(e.Record); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to create event.", "")
		}

		return nil
	})
	app.OnBeforeServe().Add(AttendEvent(app))
}

func AttendEvent(app core.App) func(e *core.ServeEvent) error {
	return func(e *core.ServeEvent) error {
		e.Router.POST("/api/events/:event_id", func(c echo.Context) error {
			session := getSessionToken(c.Request())
			userId, sessionErr := UserIdFromSession(session)
			if sessionErr != nil {
				return sessionErr
			}

			eventId := c.PathParam("event_id")
			record, err := app.Dao().FindRecordById("events", eventId)
			if err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			var attendantsSlice []string
			attendants := record.Get("attendants").(types.JsonRaw)
			err = json.Unmarshal(attendants, &attendantsSlice)
			if err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			attendantsSlice = append(attendantsSlice, userId)
			record.Set("attendants", attendantsSlice)
			if err := app.Dao().SaveRecord(record); err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			attendingEventsRecord, err := app.Dao().FindFirstRecordByData("attending_events", "user_id", userId)
			if err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			var attendingEvents []string
			attending := attendingEventsRecord.Get("attending_events").(types.JsonRaw)
			err = json.Unmarshal(attending, &attendingEvents)
			if err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			attendingEvents = append(attendingEvents, eventId)
			attendingEventsRecord.Set("attending_events", attendantsSlice)
			if err := app.Dao().SaveRecord(attendingEventsRecord); err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			return c.NoContent(http.StatusOK)
		})
		return nil
	}
}
