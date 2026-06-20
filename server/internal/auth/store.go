package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

const SessionCookieName = "gp_session"

type contextKey string

const userContextKey contextKey = "auth.user"

var guestUUIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{8,80}$`)

type Role string

const (
	RoleAdmin  Role = "admin"
	RolePlayer Role = "player"
)

type IdentityKind string

const (
	IdentityGuest IdentityKind = "guest"
	IdentityOIDC  IdentityKind = "oidc"
)

type User struct {
	ID          string       `json:"id"`
	Kind        IdentityKind `json:"kind"`
	Role        Role         `json:"role"`
	DisplayName string       `json:"displayName"`
	Banned      bool         `json:"banned"`
	CreatedAt   time.Time    `json:"createdAt"`
}

type Session struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
}

type Store struct {
	mu          sync.RWMutex
	users       map[string]*User
	sessions    map[string]Session
	guestIndex  map[string]string
	oidcIndex   map[string]string
	adminChosen bool
}

func NewStore() *Store {
	return &Store{
		users:      map[string]*User{},
		sessions:   map[string]Session{},
		guestIndex: map[string]string{},
		oidcIndex:  map[string]string{},
	}
}

func (s *Store) CreateGuestSession(guestUUID string) (*User, Session, error) {
	guestUUID = strings.TrimSpace(guestUUID)
	if !guestUUIDPattern.MatchString(guestUUID) {
		return nil, Session{}, errors.New("invalid_guest_uuid")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	userID, ok := s.guestIndex[guestUUID]
	var user *User
	if ok {
		user = s.users[userID]
	}
	if user == nil {
		user = &User{
			ID:          "usr_" + randomHex(12),
			Kind:        IdentityGuest,
			Role:        RolePlayer,
			DisplayName: "Guest " + strings.ToUpper(randomHex(2)),
			CreatedAt:   time.Now().UTC(),
		}
		s.users[user.ID] = user
		s.guestIndex[guestUUID] = user.ID
	}

	session := s.createSessionLocked(user.ID)
	return cloneUser(user), session, nil
}

func (s *Store) CreateOIDCSession(providerKey string, subject string, displayName string) (*User, Session, error) {
	providerKey = strings.TrimSpace(providerKey)
	subject = strings.TrimSpace(subject)
	displayName = strings.TrimSpace(displayName)
	if providerKey == "" || subject == "" {
		return nil, Session{}, errors.New("invalid_oidc_identity")
	}
	if displayName == "" {
		displayName = "OIDC Player"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	indexKey := providerKey + ":" + subject
	userID, ok := s.oidcIndex[indexKey]
	var user *User
	if ok {
		user = s.users[userID]
	}
	if user == nil {
		role := RolePlayer
		if !s.adminChosen {
			role = RoleAdmin
			s.adminChosen = true
		}

		user = &User{
			ID:          "usr_" + randomHex(12),
			Kind:        IdentityOIDC,
			Role:        role,
			DisplayName: displayName,
			CreatedAt:   time.Now().UTC(),
		}
		s.users[user.ID] = user
		s.oidcIndex[indexKey] = user.ID
	}

	session := s.createSessionLocked(user.ID)
	return cloneUser(user), session, nil
}

func (s *Store) UserBySession(token string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[token]
	if !ok || time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	user := s.users[session.UserID]
	if user == nil {
		return nil, false
	}

	return cloneUser(user), true
}

func (s *Store) ListUsers() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, cloneUser(user))
	}
	return users
}

func (s *Store) SetBanned(userID string, banned bool) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user := s.users[userID]
	if user == nil {
		return nil, errors.New("user_not_found")
	}
	if user.Role == RoleAdmin && banned {
		return nil, errors.New("admin_cannot_be_banned")
	}

	user.Banned = banned
	return cloneUser(user), nil
}

func (s *Store) createSessionLocked(userID string) Session {
	session := Session{
		Token:     "ses_" + randomHex(24),
		UserID:    userID,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour).UTC(),
	}
	s.sessions[session.Token] = session
	return session
}

func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userContextKey).(*User)
	return user, ok
}

func SetSessionCookie(w http.ResponseWriter, session Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
}

func SessionTokenFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie(SessionCookieName); err == nil {
		return cookie.Value
	}

	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}

	return ""
}

func cloneUser(user *User) *User {
	if user == nil {
		return nil
	}
	copy := *user
	return &copy
}

func randomHex(bytes int) string {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		panic(fmt.Errorf("crypto random failed: %w", err))
	}
	return hex.EncodeToString(buffer)
}
