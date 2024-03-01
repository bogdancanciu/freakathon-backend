package handlers

import (
	"crypto/rand"
	"fmt"
	"github.com/Pallinder/go-randomdata"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"math/big"
	"strings"
)

func BindRegisterHooks(app core.App) {
	app.OnRecordAfterCreateRequest("users").Add(func(e *core.RecordCreateEvent) error {
		setNewTag(e.Record)
		if err := app.Dao().SaveRecord(e.Record); err != nil {
			return err
		}
		return nil

	})
}

func setNewTag(r *models.Record) {
	r.Set("tag", generateUniqueTag(generateSillyName()))
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
