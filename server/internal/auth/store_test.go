package auth

import "testing"

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
