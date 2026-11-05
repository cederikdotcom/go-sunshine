package sunshine

import (
	"errors"
	"net/http"
	"testing"
)

func TestStatusErrorCarriesTheCode(t *testing.T) {
	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Unauthorized","status":false,"status_code":401}`))
	})

	_, err := c.ListApps()
	if err == nil {
		t.Fatal("expected an error for a 401 response")
	}

	var se *StatusError
	if !errors.As(err, &se) {
		t.Fatalf("expected a *StatusError, got %T: %v", err, err)
	}
	if se.Code != http.StatusUnauthorized {
		t.Errorf("Code = %d, want 401", se.Code)
	}
	if !se.Unauthorized() {
		t.Error("Unauthorized() = false, want true for a 401")
	}

	// The message must stay byte-identical to the untyped error it replaced,
	// so existing log lines and runbooks still match.
	want := `HTTP 401: {"error":"Unauthorized","status":false,"status_code":401}`
	if se.Error() != want {
		t.Errorf("Error() = %q, want %q", se.Error(), want)
	}
}

func TestStatusErrorNotUnauthorizedForOtherCodes(t *testing.T) {
	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	_, err := c.ListApps()
	var se *StatusError
	if !errors.As(err, &se) {
		t.Fatalf("expected a *StatusError, got %T: %v", err, err)
	}
	if se.Unauthorized() {
		t.Error("Unauthorized() = true for a 500, want false")
	}
}

// A transport failure must NOT be a StatusError: that is the distinction the
// type exists to make.
func TestTransportFailureIsNotAStatusError(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", "admin", "pass")

	_, err := c.ListApps()
	if err == nil {
		t.Fatal("expected an error dialing a closed port")
	}
	var se *StatusError
	if errors.As(err, &se) {
		t.Errorf("transport failure reported as a StatusError: %v", err)
	}
}
