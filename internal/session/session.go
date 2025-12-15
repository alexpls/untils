package session

import (
	"crypto/rand"
	"time"
)

type Flash string

var FlashTypeAlert Flash = "Alert"

const sessionExpirySecs = 86400 * 30 // 30 days

type Session struct {
	ID        string
	CreatedAt time.Time
	ExpiresAt time.Time
	Data      SessionData
}

func (s *Session) Reset() {
	s.ID = rand.Text()
	s.CreatedAt = time.Now()
	s.ExpiresAt = s.CreatedAt.Add(sessionExpirySecs * time.Second)
	s.Data = SessionData{} // make sure empty data when creating
}

type SessionData struct {
	UserID int64            `json:"user_id"`
	Flash  map[Flash]string `json:"flash"`
}

func (sd *SessionData) IsSignedIn() bool {
	return sd.UserID > 0
}
