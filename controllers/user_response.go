package controllers

import (
	"blog-backend/models"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type userResponse struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	EmailSHA256 string `json:"emailSha256,omitempty"`
}

func newUserResponse(user models.User) *userResponse {
	if user.ID == 0 {
		return nil
	}
	return &userResponse{
		ID:          user.ID,
		Username:    user.Username,
		EmailSHA256: emailSHA256(user.Email),
	}
}

func emailSHA256(email string) string {
	normalized := strings.TrimSpace(strings.ToLower(email))
	if normalized == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}
