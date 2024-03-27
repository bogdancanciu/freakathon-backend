package handlers

import (
	"database/sql"
	"errors"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
)

func BindSearchFriendsHooks(app core.App) {
	app.OnBeforeServe().Add(SearchFriend(app))
}

func SearchFriend(app core.App) func(e *core.ServeEvent) error {
	return func(e *core.ServeEvent) error {
		e.Router.POST("/api/search/:user_tag", func(c echo.Context) error {
			session := getSessionToken(c.Request())
			_, sessionErr := UserIdFromSession(session)
			if sessionErr != nil {
				return sessionErr
			}

			userTag := c.PathParam("user_tag")
			_, err := app.Dao().FindFirstRecordByData("users", "tag", userTag)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return c.NoContent(http.StatusNotFound)
				}

				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			return c.NoContent(http.StatusOK)
		})
		return nil
	}
}
