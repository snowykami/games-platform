package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGuestSessionNeverBootstrapsAdmin(t *testing.T) {
	store := NewStore()

	guest, _, err := store.CreateGuestSession("guest-user-1")
	if err != nil {
		t.Fatalf("create guest session: %v", err)
	}
	if guest.Role != RolePlayer {
		t.Fatalf("expected guest role %q, got %q", RolePlayer, guest.Role)
	}

	oidc, _, err := store.CreateOIDCSession("oidc", "subject-1", "Admin")
	if err != nil {
		t.Fatalf("create oidc session: %v", err)
	}
	if oidc.Role != RoleAdmin {
		t.Fatalf("expected first OIDC user to bootstrap admin, got %q", oidc.Role)
	}
}

func TestSessionCookieSecureFlagIsConfigurable(t *testing.T) {
	session := Session{Token: "ses_test", ExpiresAt: time.Now().Add(time.Hour)}

	secureRecorder := httptest.NewRecorder()
	setSessionCookie(secureRecorder, session, true)
	secureCookie := secureRecorder.Result().Cookies()[0]
	if !secureCookie.Secure || !secureCookie.HttpOnly || secureCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected secure HttpOnly SameSite=Lax cookie, got %+v", secureCookie)
	}

	localRecorder := httptest.NewRecorder()
	setSessionCookie(localRecorder, session, false)
	localCookie := localRecorder.Result().Cookies()[0]
	if localCookie.Secure {
		t.Fatalf("expected local cookie to allow non-HTTPS development, got %+v", localCookie)
	}
}
