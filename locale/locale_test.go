package locale

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
)

func TestEnglishOnlyNegotiation(t *testing.T) {
	for _, raw := range []string{"en", "EN", "en-US", "en_GB"} {
		if got, ok := Normalize(raw); !ok || got != En {
			t.Fatalf("English tag %q: %q %v", raw, got, ok)
		}
	}
	for _, raw := range []string{"fr", "zh-CN", "", "unsupported"} {
		if got, ok := Normalize(raw); ok || got != En {
			t.Fatalf("Unsupported tag %q: %q %v", raw, got, ok)
		}
	}
	SetServerDefault(Lang("unsupported"))
	if ServerDefault() != En {
		t.Fatal("unsupported server default did not fall back")
	}
	req := httptest.NewRequest("GET", "/test?lang=unsupported", nil)
	req.Header.Set("Accept-Language", "fr,en-US;q=0.5")
	if got := FromRequest(req); got != En {
		t.Fatalf("request fallback: %q", got)
	}
	if got := FromContext(WithLang(context.Background(), Lang("unsupported"))); got != En {
		t.Fatalf("context fallback: %q", got)
	}
}
func TestCatalogAndRawErrors(t *testing.T) {
	Register("test.greeting", "Hello %s")
	if got := T(Lang("unsupported"), "test.greeting", "world"); got != "Hello world" {
		t.Fatal(got)
	}
	if T(En, "unknown") != "unknown" {
		t.Fatal("unknown key changed")
	}
	cause := errors.New("raw evidence %s")
	Register("Operation %s failed: %w", "Operation %s failed: %w")
	err := Errorf("Operation %s failed: %w", "id", cause)
	if !errors.Is(err, cause) {
		t.Fatal("error identity lost")
	}
	if got := ErrorMessage(En, err); got != "Operation id failed: raw evidence %s" {
		t.Fatal(got)
	}
}
