package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
	db          *pgxpool.Pool
}

func NewStore() *Store {
	return &Store{
		users:      map[string]*User{},
		sessions:   map[string]Session{},
		guestIndex: map[string]string{},
		oidcIndex:  map[string]string{},
	}
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*Store, error) {
	databaseURL = strings.TrimSpace(databaseURL)
	if databaseURL == "" {
		return nil, errors.New("database_url_required")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("auth database connection failed: %w", err)
	}
	store := NewStore()
	store.db = pool
	if err := store.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *Store) CreateGuestSession(guestUUID string) (*User, Session, error) {
	guestUUID = strings.TrimSpace(guestUUID)
	if !guestUUIDPattern.MatchString(guestUUID) {
		return nil, Session{}, errors.New("invalid_guest_uuid")
	}
	if s.db != nil {
		return s.createGuestSessionPostgres(guestUUID)
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
	if s.db != nil {
		return s.createOIDCSessionPostgres(providerKey, subject, displayName)
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
	if s.db != nil {
		return s.userBySessionPostgres(token)
	}

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

func (s *Store) DeleteSession(token string) {
	if token == "" {
		return
	}
	if s.db != nil {
		s.deleteSessionPostgres(token)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *Store) ListUsers() []*User {
	if s.db != nil {
		return s.listUsersPostgres()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, cloneUser(user))
	}
	return users
}

func (s *Store) SetBanned(userID string, banned bool) (*User, error) {
	if s.db != nil {
		return s.setBannedPostgres(userID, banned)
	}

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

func (s *Store) migrate(ctx context.Context) error {
	migrateCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	const schema = `
CREATE TABLE IF NOT EXISTS auth_users (
	id TEXT PRIMARY KEY,
	kind TEXT NOT NULL,
	role TEXT NOT NULL,
	display_name TEXT NOT NULL,
	banned BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS auth_guest_identities (
	guest_uuid TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS auth_oidc_identities (
	provider_key TEXT NOT NULL,
	subject TEXT NOT NULL,
	user_id TEXT NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
	PRIMARY KEY (provider_key, subject)
);

CREATE TABLE IF NOT EXISTS auth_sessions (
	token TEXT PRIMARY KEY,
	user_id TEXT NOT NULL REFERENCES auth_users(id) ON DELETE CASCADE,
	expires_at TIMESTAMPTZ NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS auth_sessions_expires_at_idx ON auth_sessions(expires_at);
CREATE INDEX IF NOT EXISTS auth_sessions_user_id_idx ON auth_sessions(user_id);
`
	if _, err := s.db.Exec(migrateCtx, schema); err != nil {
		return fmt.Errorf("auth database migration failed: %w", err)
	}
	return nil
}

func (s *Store) createGuestSessionPostgres(guestUUID string) (*User, Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, Session{}, err
	}
	defer rollbackQuietly(ctx, tx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(582104212)`); err != nil {
		return nil, Session{}, err
	}

	user, err := postgresGuestUser(ctx, tx, guestUUID)
	if err != nil {
		return nil, Session{}, err
	}
	if user == nil {
		user = &User{
			ID:          "usr_" + randomHex(12),
			Kind:        IdentityGuest,
			Role:        RolePlayer,
			DisplayName: "Guest " + strings.ToUpper(randomHex(2)),
			CreatedAt:   time.Now().UTC(),
		}
		if err := insertPostgresUser(ctx, tx, user); err != nil {
			return nil, Session{}, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO auth_guest_identities (guest_uuid, user_id) VALUES ($1, $2)`, guestUUID, user.ID); err != nil {
			return nil, Session{}, err
		}
	}

	session, err := insertPostgresSession(ctx, tx, user.ID)
	if err != nil {
		return nil, Session{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, Session{}, err
	}
	return cloneUser(user), session, nil
}

func (s *Store) createOIDCSessionPostgres(providerKey string, subject string, displayName string) (*User, Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, Session{}, err
	}
	defer rollbackQuietly(ctx, tx)

	// Serialize identity bootstrap so the first OIDC user is the only automatic admin.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(582104211)`); err != nil {
		return nil, Session{}, err
	}

	user, err := postgresOIDCUser(ctx, tx, providerKey, subject)
	if err != nil {
		return nil, Session{}, err
	}
	if user == nil {
		role := RolePlayer
		var adminExists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM auth_users WHERE role = $1)`, RoleAdmin).Scan(&adminExists); err != nil {
			return nil, Session{}, err
		}
		if !adminExists {
			role = RoleAdmin
		}

		user = &User{
			ID:          "usr_" + randomHex(12),
			Kind:        IdentityOIDC,
			Role:        role,
			DisplayName: displayName,
			CreatedAt:   time.Now().UTC(),
		}
		if err := insertPostgresUser(ctx, tx, user); err != nil {
			return nil, Session{}, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO auth_oidc_identities (provider_key, subject, user_id) VALUES ($1, $2, $3)`, providerKey, subject, user.ID); err != nil {
			return nil, Session{}, err
		}
	}

	session, err := insertPostgresSession(ctx, tx, user.ID)
	if err != nil {
		return nil, Session{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, Session{}, err
	}
	return cloneUser(user), session, nil
}

func (s *Store) userBySessionPostgres(token string) (*User, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	user := &User{}
	err := s.db.QueryRow(ctx, `
SELECT u.id, u.kind, u.role, u.display_name, u.banned, u.created_at
FROM auth_sessions s
JOIN auth_users u ON u.id = s.user_id
WHERE s.token = $1 AND s.expires_at > NOW()
`, token).Scan(&user.ID, &user.Kind, &user.Role, &user.DisplayName, &user.Banned, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false
	}
	if err != nil {
		slog.Warn("auth session lookup failed", "error", err)
		return nil, false
	}
	return user, true
}

func (s *Store) deleteSessionPostgres(token string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := s.db.Exec(ctx, `DELETE FROM auth_sessions WHERE token = $1`, token); err != nil {
		slog.Warn("auth session delete failed", "error", err)
	}
}

func (s *Store) listUsersPostgres() []*User {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := s.db.Query(ctx, `SELECT id, kind, role, display_name, banned, created_at FROM auth_users ORDER BY created_at ASC, id ASC`)
	if err != nil {
		slog.Warn("auth list users failed", "error", err)
		return nil
	}
	defer rows.Close()

	users := []*User{}
	for rows.Next() {
		user := &User{}
		if err := rows.Scan(&user.ID, &user.Kind, &user.Role, &user.DisplayName, &user.Banned, &user.CreatedAt); err != nil {
			slog.Warn("auth scan user failed", "error", err)
			return users
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("auth list users rows failed", "error", err)
	}
	return users
}

func (s *Store) setBannedPostgres(userID string, banned bool) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	user := &User{}
	err := s.db.QueryRow(ctx, `
UPDATE auth_users
SET banned = $2
WHERE id = $1 AND NOT (role = $3 AND $2 = TRUE)
RETURNING id, kind, role, display_name, banned, created_at
`, userID, banned, RoleAdmin).Scan(&user.ID, &user.Kind, &user.Role, &user.DisplayName, &user.Banned, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		if existsErr := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM auth_users WHERE id = $1)`, userID).Scan(&exists); existsErr == nil && exists {
			return nil, errors.New("admin_cannot_be_banned")
		}
		return nil, errors.New("user_not_found")
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func postgresGuestUser(ctx context.Context, tx pgx.Tx, guestUUID string) (*User, error) {
	user := &User{}
	err := tx.QueryRow(ctx, `
SELECT u.id, u.kind, u.role, u.display_name, u.banned, u.created_at
FROM auth_guest_identities g
JOIN auth_users u ON u.id = g.user_id
WHERE g.guest_uuid = $1
`, guestUUID).Scan(&user.ID, &user.Kind, &user.Role, &user.DisplayName, &user.Banned, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func postgresOIDCUser(ctx context.Context, tx pgx.Tx, providerKey string, subject string) (*User, error) {
	user := &User{}
	err := tx.QueryRow(ctx, `
SELECT u.id, u.kind, u.role, u.display_name, u.banned, u.created_at
FROM auth_oidc_identities i
JOIN auth_users u ON u.id = i.user_id
WHERE i.provider_key = $1 AND i.subject = $2
`, providerKey, subject).Scan(&user.ID, &user.Kind, &user.Role, &user.DisplayName, &user.Banned, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

func insertPostgresUser(ctx context.Context, tx pgx.Tx, user *User) error {
	_, err := tx.Exec(ctx, `
INSERT INTO auth_users (id, kind, role, display_name, banned, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
`, user.ID, user.Kind, user.Role, user.DisplayName, user.Banned, user.CreatedAt)
	return err
}

func insertPostgresSession(ctx context.Context, tx pgx.Tx, userID string) (Session, error) {
	session := Session{
		Token:     "ses_" + randomHex(24),
		UserID:    userID,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour).UTC(),
	}
	_, err := tx.Exec(ctx, `
INSERT INTO auth_sessions (token, user_id, expires_at, created_at)
VALUES ($1, $2, $3, NOW())
`, session.Token, session.UserID, session.ExpiresAt)
	return session, err
}

func rollbackQuietly(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userContextKey).(*User)
	return user, ok
}

func SetSessionCookie(w http.ResponseWriter, session Session) {
	setSessionCookie(w, session, false)
}

func setSessionCookie(w http.ResponseWriter, session Session, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    session.Token,
		Path:     "/",
		Expires:  session.ExpiresAt,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
}

func ClearSessionCookie(w http.ResponseWriter) {
	clearSessionCookie(w, false)
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   secure,
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
