package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bogdancanciu/frekathon-backend/strategy"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/types"
	"golang.org/x/exp/rand"
	"log"
	"net/http"
	"strings"
	"time"
)

type chatFinderRec struct {
	ID        string                  `db:"user_id" json:"user_id"`
	Interests types.JsonArray[string] `db:"interests" json:"interests"`
}

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

			return matchExistingUsers(app)
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

func matchExistingUsers(app core.App) error {
	var chatFinderRecords []chatFinderRec
	err := app.Dao().DB().NewQuery("SELECT * FROM chat_finder").All(&chatFinderRecords)
	if err != nil {
		log.Println("Error fetching from chat_finder", err)
		return nil
	}

	var chatUsers []*strategy.User
	for _, record := range chatFinderRecords {
		chatUsers = append(chatUsers, &strategy.User{
			ID:        record.ID,
			Interests: record.Interests,
		})
	}

	matchingStrategy := strategy.NewMatchingStrategy(chatUsers)
	matches, commonInterests := matchingStrategy.FindMatchingGroups()
	if len(matches) > 0 {
		var groupParticipatingUsers []string
		for _, user := range matches[0] {
			groupParticipatingUsers = append(groupParticipatingUsers, user.ID)
		}

		chatsCollection, err := app.Dao().FindCollectionByNameOrId("chats")
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
		}

		chatRecord := models.NewRecord(chatsCollection)

		chatRecord.Set("participants", groupParticipatingUsers)
		chatRecord.Set("type", "group")
		chatRecord.Set("description", randomGroupDescription())
		chatRecord.Set("common_interests", commonInterests[0])

		if err := app.Dao().SaveRecord(chatRecord); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
		}

		err = deleteUsersFromChatFinder(app, groupParticipatingUsers)
		if err != nil {
			log.Println("Failed to delete users from chat finder", err)
			return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
		}

		err = updateUsersActiveChats(app, groupParticipatingUsers, chatRecord)
		if err != nil {
			log.Println("Failed to delete users from chat finder", err)
			return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
		}
	}

	return nil
}

func randomGroupDescription() string {
	descriptionsPool := []string{"In The Forest", "At The Store", "In The Mighty Jungle", "At The Bar"}
	rand.Seed(uint64(time.Now().UnixNano()))

	return descriptionsPool[rand.Int()%len(descriptionsPool)]
}

func deleteUsersFromChatFinder(app core.App, users []string) error {
	var quotedUserIds []string
	for _, id := range users {
		quotedUserIds = append(quotedUserIds, fmt.Sprintf("'%s'", id))
	}
	userIdsStr := strings.Join(quotedUserIds, ",")
	query := fmt.Sprintf("DELETE FROM chat_finder WHERE user_id IN (%s)", userIdsStr)

	_, err := app.Dao().DB().NewQuery(query).Execute()
	if err != nil {
		return err
	}

	return nil
}

func updateUsersActiveChats(app core.App, users []string, chatRecord *models.Record) error {
	for _, user := range users {
		messagesRecord, err := app.Dao().FindFirstRecordByData("messages", "user_id", user)
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
		}

		var activeChats []string
		chats := messagesRecord.Get("active_anon_chats").(types.JsonRaw)
		err = json.Unmarshal(chats, &activeChats)
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
		}

		activeChats = append(activeChats, chatRecord.Id)
		messagesRecord.Set("active_anon_chats", activeChats)
		if err := app.Dao().SaveRecord(messagesRecord); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
		}

	}

	return nil
}
