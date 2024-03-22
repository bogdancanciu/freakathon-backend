package handlers

import (
	"encoding/json"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"golang.org/x/exp/slices"
	"log"
	"net/http"
)

type Friend struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	ChatId string `json:"chat_id"`
}

func BindFriendsHooks(app core.App) {
	app.OnBeforeServe().Add(AddFriend(app))
	app.OnBeforeServe().Add(AcceptFriend(app))
}

func AddFriend(app core.App) func(e *core.ServeEvent) error {
	return func(e *core.ServeEvent) error {
		e.Router.POST("/api/friends/:friend_id", func(c echo.Context) error {
			session := getSessionToken(c.Request())
			userId, sessionErr := UserIdFromSession(session)
			if sessionErr != nil {
				return sessionErr
			}

			err := registerCurrentUserSentInvite(app, c, userId)
			if err != nil {
				return err
			}

			err = updateFriendPendingList(app, c, userId)
			if err != nil {
				return err
			}

			return c.NoContent(http.StatusOK)
		})
		return nil
	}
}

func AcceptFriend(app core.App) func(e *core.ServeEvent) error {
	return func(e *core.ServeEvent) error {
		e.Router.PUT("/api/friends/:friend_id", func(c echo.Context) error {
			session := getSessionToken(c.Request())
			userId, sessionErr := UserIdFromSession(session)
			if sessionErr != nil {
				return sessionErr
			}

			friendId := c.PathParam("friend_id")
			chatParticipants := []string{userId, friendId}
			chatId, err := createChat(app, chatParticipants, "dm", "")
			if err != nil {
				log.Println("Error while creating friend chat", err)
			}

			err = updateCurrentUserPendingList(app, c, userId)
			if err != nil {
				return err
			}

			err = updateFriendSentInvites(app, c, userId)
			if err != nil {
				return err
			}

			err = addFriendToFriendList(app, c, userId, chatId)
			if err != nil {
				return err
			}

			return c.NoContent(http.StatusOK)
		})
		return nil
	}
}

func registerCurrentUserSentInvite(app core.App, c echo.Context, userId string) *apis.ApiError {
	friendId := c.PathParam("friend_id")

	record, err := app.Dao().FindFirstRecordByData("friends", "user_id", userId)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	sentInvites, dbErr := getSentInvites(record)
	if dbErr != nil {
		return dbErr
	}

	for _, i := range sentInvites {
		if i.ID == friendId {
			return apis.NewApiError(http.StatusConflict, "Invitation already sent.", "")
		}
	}

	friendRecord, err := app.Dao().FindFirstRecordByData("users", "id", friendId)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	sentInvites = append(sentInvites, Friend{ID: friendRecord.GetId(), Name: friendRecord.Get("name").(string)})
	record.Set("sent_invites", sentInvites)
	if err := app.Dao().SaveRecord(record); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	return nil
}

func updateCurrentUserPendingList(app core.App, c echo.Context, userId string) *apis.ApiError {
	friendId := c.PathParam("friend_id")

	record, err := app.Dao().FindFirstRecordByData("friends", "user_id", userId)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	pendingList, dbErr := getPendingList(record)
	if dbErr != nil {
		return dbErr
	}

	pendingList = slices.DeleteFunc(pendingList, func(f Friend) bool {
		return f.ID == friendId
	})
	record.Set("pending_list", pendingList)
	if err := app.Dao().SaveRecord(record); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	return nil
}

func updateFriendPendingList(app core.App, c echo.Context, userId string) *apis.ApiError {
	friendId := c.PathParam("friend_id")

	friendRecord, err := app.Dao().FindFirstRecordByData("friends", "user_id", friendId)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	pendingList, dbErr := getPendingList(friendRecord)
	if dbErr != nil {
		return dbErr
	}

	for _, p := range pendingList {
		if p.ID == userId {
			return apis.NewApiError(http.StatusConflict, "Invitation already pending.", "")
		}
	}

	currentUserRecord, err := app.Dao().FindFirstRecordByData("users", "id", userId)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	pendingList = append(pendingList, Friend{ID: currentUserRecord.GetId(), Name: currentUserRecord.Get("name").(string)})
	friendRecord.Set("pending_list", pendingList)
	if err := app.Dao().SaveRecord(friendRecord); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	return nil
}

func updateFriendSentInvites(app core.App, c echo.Context, userId string) *apis.ApiError {
	friendId := c.PathParam("friend_id")

	record, err := app.Dao().FindFirstRecordByData("friends", "user_id", friendId)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	sentInvites, dbErr := getSentInvites(record)
	if dbErr != nil {
		return dbErr
	}

	sentInvites = slices.DeleteFunc(sentInvites, func(f Friend) bool {
		return f.ID == userId
	})
	record.Set("sent_invites", sentInvites)
	if err := app.Dao().SaveRecord(record); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	return nil
}

func addFriendToFriendList(app core.App, c echo.Context, userId, chatId string) *apis.ApiError {
	friendId := c.PathParam("friend_id")

	currentUserRecord, err := app.Dao().FindFirstRecordByData("friends", "user_id", userId)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	friendUserRecord, err := app.Dao().FindFirstRecordByData("friends", "user_id", friendId)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	currentUserFriendList, dbErr := getFriendList(currentUserRecord)
	if dbErr != nil {
		return dbErr
	}

	peerFriendList, dbErr := getFriendList(friendUserRecord)
	if dbErr != nil {
		return dbErr
	}

	currentUser, err := app.Dao().FindFirstRecordByData("users", "id", userId)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	friendRecord, err := app.Dao().FindFirstRecordByData("users", "id", friendId)
	if err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	currentUserFriendList = append(currentUserFriendList, Friend{ID: friendId, Name: friendRecord.Get("name").(string), ChatId: chatId})
	currentUserRecord.Set("friend_list", currentUserFriendList)

	peerFriendList = append(peerFriendList, Friend{ID: userId, Name: currentUser.Get("name").(string), ChatId: chatId})
	friendUserRecord.Set("friend_list", peerFriendList)

	if err := app.Dao().SaveRecord(currentUserRecord); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}
	if err := app.Dao().SaveRecord(friendUserRecord); err != nil {
		return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	return nil
}

func getFriendList(record *models.Record) ([]Friend, *apis.ApiError) {
	friendList := record.Get("friend_list")
	jsonBytes, err := json.Marshal(friendList)
	if err != nil {
		return nil, apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	var friends []Friend
	err = json.Unmarshal(jsonBytes, &friends)
	if err != nil {
		return nil, apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	return friends, nil
}

func getPendingList(record *models.Record) ([]Friend, *apis.ApiError) {
	pendingList := record.Get("pending_list")
	jsonBytes, err := json.Marshal(pendingList)
	if err != nil {
		return nil, apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	var pending []Friend
	err = json.Unmarshal(jsonBytes, &pending)
	if err != nil {
		return nil, apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	return pending, nil
}

func getSentInvites(record *models.Record) ([]Friend, *apis.ApiError) {
	sentInvites := record.Get("sent_invites")
	jsonBytes, err := json.Marshal(sentInvites)
	if err != nil {
		return nil, apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	var invited []Friend
	err = json.Unmarshal(jsonBytes, &invited)
	if err != nil {
		return nil, apis.NewApiError(http.StatusInternalServerError, "Server error", "")
	}

	return invited, nil
}
