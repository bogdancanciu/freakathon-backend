package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"golang.org/x/exp/slices"
	"net/http"
	"strings"
)

type eventResponse struct {
	AllEvents       []eventRecord `json:"all_events"`
	YourEvents      []eventRecord `json:"your_events"`
	AttendingEvents []eventRecord `json:"attending_events"`
}

type eventRecord struct {
	ID              string                  `db:"id" json:"id"`
	Title           string                  `db:"name" json:"title"`
	Location        string                  `db:"location" json:"location"`
	Date            string                  `db:"date" json:"date"`
	Attendants      types.JsonArray[string] `db:"attendants" json:"attendants"`
	AttendantsCount int                     `json:"attendants_count"`
	CanAttend       bool                    `json:"can_attend"`
}

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
	app.OnBeforeServe().Add(GetEvents(app))
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

			attendingEventsRecord.Set("attending_events", attendingEvents)
			if err := app.Dao().SaveRecord(attendingEventsRecord); err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
			}

			return c.NoContent(http.StatusOK)
		})
		return nil
	}
}

func GetEvents(app core.App) func(e *core.ServeEvent) error {
	return func(e *core.ServeEvent) error {
		e.Router.GET("/api/events", func(c echo.Context) error {
			session := getSessionToken(c.Request())
			userId, sessionErr := UserIdFromSession(session)
			if sessionErr != nil {
				return sessionErr
			}

			friendsEvents, err := getFriendsEvents(app, userId)
			if err != nil {
				return err
			}

			yourEvents, err := getYourEvents(app, userId)
			if err != nil {
				return err
			}

			attendingEvents, err := getAttendingEvents(app, userId)
			if err != nil {
				return err
			}

			response := eventResponse{
				AllEvents:       append(friendsEvents, yourEvents...),
				YourEvents:      yourEvents,
				AttendingEvents: attendingEvents,
			}

			return c.JSON(http.StatusOK, response)
		})
		return nil
	}
}

func getFriendsEvents(app core.App, userId string) ([]eventRecord, error) {
	friendsRecord, err := app.Dao().FindFirstRecordByData("friends", "user_id", userId)
	if err != nil {
		return nil, apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	var friendList []Friend
	friends := friendsRecord.Get("friend_list").(types.JsonRaw)
	err = json.Unmarshal(friends, &friendList)
	if err != nil {
		return nil, apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	var friendsEvents []eventRecord
	var quotedFriendsIds []string
	for _, friend := range friendList {
		quotedFriendsIds = append(quotedFriendsIds, fmt.Sprintf("'%s'", friend.ID))
	}
	friendIdsStr := strings.Join(quotedFriendsIds, ",")
	queryString := fmt.Sprintf("SELECT * FROM events WHERE user_id IN (%s)", friendIdsStr)

	err = app.Dao().DB().NewQuery(queryString).All(&friendsEvents)
	if err != nil {
		return nil, apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	for i := range friendsEvents {
		if !slices.Contains(friendsEvents[i].Attendants, userId) {
			friendsEvents[i].CanAttend = true
		}
		friendsEvents[i].AttendantsCount = len(friendsEvents[i].Attendants)
	}

	return friendsEvents, nil
}

func getYourEvents(app core.App, userId string) ([]eventRecord, error) {
	var yourEvents []eventRecord
	queryString := fmt.Sprintf("SELECT * FROM events WHERE user_id IN (%s)", fmt.Sprintf("'%s'", userId))

	err := app.Dao().DB().NewQuery(queryString).All(&yourEvents)
	if err != nil {
		return nil, apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	for i := range yourEvents {
		yourEvents[i].AttendantsCount = len(yourEvents[i].Attendants)
	}

	return yourEvents, nil
}

func getAttendingEvents(app core.App, userId string) ([]eventRecord, error) {
	attendingEventsRecord, err := app.Dao().FindFirstRecordByData("attending_events", "user_id", userId)
	if err != nil {
		return nil, apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	var attendingEventsIds []string
	attEvs := attendingEventsRecord.Get("attending_events").(types.JsonRaw)
	err = json.Unmarshal(attEvs, &attendingEventsIds)
	if err != nil {
		return nil, apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	var attendingEvents []eventRecord
	var quotedEventsIds []string
	for _, event := range attendingEventsIds {
		quotedEventsIds = append(quotedEventsIds, fmt.Sprintf("'%s'", event))
	}
	eventsIdsStr := strings.Join(quotedEventsIds, ",")
	queryString := fmt.Sprintf("SELECT * FROM events WHERE id IN (%s)", eventsIdsStr)

	err = app.Dao().DB().NewQuery(queryString).All(&attendingEvents)
	if err != nil {
		return nil, apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	for i := range attendingEvents {
		attendingEvents[i].AttendantsCount = len(attendingEvents[i].Attendants)
	}

	return attendingEvents, nil
}
