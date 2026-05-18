package user

import (
	"time"
)

type User struct {
	ID             string
	GithubID       int64
	GithubLogin    string
	Name           string
	Email          string
	AvatarURL      string
	EncryptedToken string /* AES-256-GCM encrypted JSON of oauth2.Token */
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
