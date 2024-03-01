package handlers

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pocketbase/pocketbase/apis"
	"net/http"
	"strings"
	"time"
)

var (
	expectedTokenParts = 3
	errMalformedToken  = "Malformed session token."
	errExpiredToken    = "Session token is expired."
)

func getSessionToken(r *http.Request) string {
	return r.Header.Get("session-token")
}

func tokenPayload(sessionToken string) (map[string]interface{}, *apis.ApiError) {
	tokenParts := strings.Split(sessionToken, ".")
	if len(tokenParts) != expectedTokenParts {
		return nil, apis.NewUnauthorizedError(errMalformedToken, "")
	}
	payload, err := base64.RawURLEncoding.DecodeString(tokenParts[1])
	if err != nil {
		return nil, apis.NewUnauthorizedError(errMalformedToken, "")
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, apis.NewUnauthorizedError(errMalformedToken, "")
	}

	return claims, nil
}

func validateSessionToken(sessionToken string) *apis.ApiError {
	payload, err := tokenPayload(sessionToken)
	if err != nil {
		return err
	}

	currentTime := time.Now()
	exp := payload["exp"].(float64)
	tokenExpiryDate := time.Unix(int64(exp), 0)

	if currentTime.After(tokenExpiryDate) {
		return apis.NewUnauthorizedError(errExpiredToken, "")
	}

	return nil
}

func userIdFromSession(sessionToken string) (string, *apis.ApiError) {
	err := validateSessionToken(sessionToken)
	if err != nil {
		return "", err
	}

	payload, err := tokenPayload(sessionToken)
	if err != nil {
		return "", err
	}
	userId, ok := payload["id"].(string)
	if !ok {
		return "", apis.NewUnauthorizedError(errMalformedToken, "")
	}

	return userId, nil
}
