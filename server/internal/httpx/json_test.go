package httpx

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestDecodeJSONRejectsLargeBody(t *testing.T) {
	body := `{"value":"` + strings.Repeat("x", int(MaxJSONBodyBytes)) + `"}`
	request, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	var payload map[string]string
	err = DecodeJSON(request, &payload)
	if !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("expected ErrBodyTooLarge, got %v", err)
	}
}

func TestDecodeJSONRejectsMultipleDocuments(t *testing.T) {
	request, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ok":true} {"extra":true}`))
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	var payload map[string]bool
	if err := DecodeJSON(request, &payload); err == nil {
		t.Fatal("expected trailing JSON document to be rejected")
	}
}
