package main

import (
	"github.com/bogdancanciu/frekathon-backend/handlers"
	"github.com/bogdancanciu/frekathon-backend/handlers/protocol"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"log"
)

func main() {
	app := pocketbase.New()
	hub := protocol.NewHub(app)
	go hub.Run()

	// Define the middleware function to handle WebSocket upgrade
	wsUpgradeMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := c.Request()
			w := c.Response().Writer
			if r.URL.Path == "/ws" {
				// Handle WebSocket upgrade
				protocol.ServeWs(app, hub, w, r)
				return nil
			}
			return next(c)
		}
	}

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.Use(wsUpgradeMiddleware)
		return nil
	})

	handlers.BindRegisterHooks(app)
	handlers.BindEventsHooks(app)
	handlers.BindFriendsHooks(app)
	handlers.BindInterestsHooks(app)
	handlers.BindChatFinderHooks(app)
	handlers.BindSearchFriendsHooks(app)

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
