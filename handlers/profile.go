package handlers

import (
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
)

func BindProfileHooks(app core.App) {
	app.OnBeforeServe().Add(GetProfile(app))
}

func GetProfile(app core.App) func(e *core.ServeEvent) error {
	return func(e *core.ServeEvent) error {
		e.Router.GET("/api/profile", func(c echo.Context) error {
			session := getSessionToken(c.Request())
			userId, sessionErr := UserIdFromSession(session)
			if sessionErr != nil {
				return sessionErr
			}

			userRecord, err := app.Dao().FindRecordById("users", userId)
			if err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			return c.JSON(http.StatusOK, userRecord)
		})
		return nil
	}
}
