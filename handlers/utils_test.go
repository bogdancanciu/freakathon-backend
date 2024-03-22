package handlers

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTokenPayload(t *testing.T) {
	t.Run("should deserialize payload from valid token", func(t *testing.T) {
		token := testJWT(t, jwt.MapClaims{}, 0)

		_, err := tokenPayload(token)
		assert.Nil(t, err)
	})

	t.Run("should return error when token is malformed", func(t *testing.T) {
		token := "malformed_token"
		_, err := tokenPayload(token)
		assert.Error(t, err)
		assert.Equal(t, errMalformedToken, err.Message)
	})
}

func TestValidateSessionToken(t *testing.T) {
	t.Run("should return session expired error", func(t *testing.T) {
		token := testJWT(t, jwt.MapClaims{}, 0)

		err := validateSessionToken(token)
		assert.Error(t, err)
		assert.Equal(t, errExpiredToken, err.Message)

	})

	t.Run("should validate session token that's not expired", func(t *testing.T) {
		token := testJWT(t, jwt.MapClaims{}, 10)

		err := validateSessionToken(token)
		assert.Nil(t, err)
	})
}

func TestUserIdFromSession(t *testing.T) {
	t.Run("should return malformed token error when token doesn't embed an ID", func(t *testing.T) {
		token := testJWT(t, jwt.MapClaims{}, 10)

		_, err := UserIdFromSession(token)
		assert.Error(t, err)
		assert.Equal(t, errMalformedToken, err.Message)
	})

	t.Run("should return malformed token error when token doesn't embed an ID", func(t *testing.T) {
		claims := jwt.MapClaims{"id": "test_id"}
		token := testJWT(t, claims, 10)

		userId, err := UserIdFromSession(token)
		assert.Nil(t, err)
		assert.Equal(t, claims["id"], userId)
	})
}

func testJWT(t *testing.T, claims jwt.MapClaims, tokenDuration int64) string {
	token, err := security.NewJWT(
		claims,
		"",
		tokenDuration,
	)
	assert.NoError(t, err)

	return token
}
