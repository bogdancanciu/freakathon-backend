package handlers

import (
	"database/sql"
	"errors"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"net/http"
)

//type messagesRecord struct {
//	ID                   string                  `db:"id" json:"id"`
//	UserID               string                  `db:"user_id" json:"user_id"`
//	ActiveAnonymousChats types.JsonArray[string] `db:"active_anon_chats" json:"active_anon_chats"`
//}

type FindChatAvailable struct {
	CanFindChat bool `json:"canFindChat"`
}

func BindChatFinderHooks(app core.App) {
	app.OnBeforeServe().Add(FindChat(app))
	app.OnBeforeServe().Add(CanFindChat(app))
}

func FindChat(app core.App) func(e *core.ServeEvent) error {
	return func(e *core.ServeEvent) error {
		e.Router.POST("/api/find-chat", func(c echo.Context) error {
			session := getSessionToken(c.Request())
			userId, sessionErr := UserIdFromSession(session)
			if sessionErr != nil {
				return sessionErr
			}

			userRecord, err := app.Dao().FindRecordById("users", userId)
			if err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			interests := userRecord.Get("interests")

			chatFinderCollection, err := app.Dao().FindCollectionByNameOrId("chat_finder")
			if err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			chatFinderRecord := models.NewRecord(chatFinderCollection)

			chatFinderRecord.Set("user_id", userId)
			chatFinderRecord.Set("interests", interests)

			if err := app.Dao().SaveRecord(chatFinderRecord); err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			return nil
		})
		return nil
	}
}

func CanFindChat(app core.App) func(e *core.ServeEvent) error {
	return func(e *core.ServeEvent) error {
		e.Router.GET("/api/find-chat", func(c echo.Context) error {
			session := getSessionToken(c.Request())
			userId, sessionErr := UserIdFromSession(session)
			if sessionErr != nil {
				return sessionErr
			}

			_, err := app.Dao().FindFirstRecordByData("chat_finder", "user_id", userId)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return c.JSON(http.StatusOK, &FindChatAvailable{CanFindChat: true})
				}

				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			return c.JSON(http.StatusOK, &FindChatAvailable{CanFindChat: false})
		})
		return nil
	}
}
