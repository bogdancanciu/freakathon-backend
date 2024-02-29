package handlers

import (
	"database/sql"
	"errors"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"log"
	"net/http"
)

func bindAuthHooks(app core.App) {
	app.OnBeforeServe().Add(Register(app))
}

type registerRequestBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Register(app core.App) func(e *core.ServeEvent) error {
	return func(e *core.ServeEvent) error {
		e.Router.POST("/api/v2/register", func(c echo.Context) error {
			var requestBody registerRequestBody
			if err := c.Bind(&requestBody); err != nil {
				return c.NoContent(http.StatusInternalServerError)
			}

			_, err := app.Dao().FindFirstRecordByData("users", "username", requestBody.Username)
			if !errors.Is(err, sql.ErrNoRows) {
				return c.NoContent(http.StatusConflict)
			}

			usersCollection, err := app.Dao().FindCollectionByNameOrId("users")
			if err != nil {
				return c.NoContent(http.StatusInternalServerError)
			}

			record := models.NewRecord(usersCollection)
			err = record.RefreshTokenKey()
			if err != nil {
				return c.NoContent(http.StatusInternalServerError)
			}

			record.Set("username", requestBody.Username)
			record.Set("password", requestBody.Password)

			if err := app.Dao().SaveRecord(record); err != nil {
				log.Println("Failed to save record with username: ", record.Username(), "err ", err)
				return c.NoContent(http.StatusBadRequest)
			}

			return c.NoContent(http.StatusOK)
		})
		return nil
	}
}
