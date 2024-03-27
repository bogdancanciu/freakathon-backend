package handlers

import (
	"crypto/rand"
	"fmt"
	"github.com/Pallinder/go-randomdata"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"math/big"
	"net/http"
	"strings"
)

func BindRegisterHooks(app core.App) {
	app.OnRecordAfterCreateRequest("users").Add(func(e *core.RecordCreateEvent) error {
		setNewTag(e.Record)
		initializeInterests(e.Record)
		if err := app.Dao().SaveRecord(e.Record); err != nil {
			return err
		}
		friendsCollection, err := app.Dao().FindCollectionByNameOrId("friends")
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
		}

		record := models.NewRecord(friendsCollection)

		record.Set("user_id", e.Record.Id)
		record.Set("friend_list", []Friend{})
		record.Set("pending_list", []Friend{})
		record.Set("sent_invites", []Friend{})

		if err := app.Dao().SaveRecord(record); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
		}

		messagesCollection, err := app.Dao().FindCollectionByNameOrId("messages")
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
		}

		messagesRecord := models.NewRecord(messagesCollection)

		messagesRecord.Set("user_id", e.Record.Id)
		messagesRecord.Set("active_anon_chats", []string{})
		messagesRecord.Set("messages", [][]byte{})

		if err := app.Dao().SaveRecord(messagesRecord); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Server error", "")
		}

		return nil

	})
}

func setNewTag(r *models.Record) {
	r.Set("tag", generateUniqueTag(generateSillyName()))
}

func initializeInterests(r *models.Record) {
	r.Set("interests", []string{})
}

func generateSillyName() string {
	return fmt.Sprintf("%s %s", randomdata.SillyName(), randomdata.SillyName())
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charsetLength := big.NewInt(int64(len(charset)))
	var randomString strings.Builder

	for i := 0; i < length; i++ {
		randomIndex, _ := rand.Int(rand.Reader, charsetLength)
		randomCharacter := charset[randomIndex.Int64()]
		randomString.WriteByte(randomCharacter)
	}

	return randomString.String()
}

func generateUniqueTag(name string) string {
	baseTag := strings.ReplaceAll(strings.ToLower(name), " ", "_")

	randomString := generateRandomString(5)
	uniqueTag := fmt.Sprintf("%s.%s", baseTag, randomString)

	return uniqueTag
}
