package sunshine

import (
	"testing"
	"time"
)

func TestSessionLifecycle(t *testing.T) {
	r := NewSessionRegistry(30 * time.Second)

	// Start a session.
	r.OnStreamStarted("game")
	if !r.IsStreaming() {
		t.Fatal("expected streaming after start")
	}
	if r.ActiveCount() != 1 {
		t.Fatalf("expected 1 active, got %d", r.ActiveCount())
	}

	s := r.Get("game")
	if s == nil {
		t.Fatal("expected session")
	}
	if s.State != SessionActive {
		t.Errorf("expected active, got %d", s.State)
	}

	// End the session, should enter grace period.
	r.OnStreamEnded("game")
	s = r.Get("game")
	if s.State != SessionDisconnected {
		t.Errorf("expected disconnected, got %d", s.State)
	}
	if s.GraceExpiry == nil {
		t.Fatal("expected grace expiry to be set")
	}

	// Not yet expired.
	expired := r.ExpiredSessions(time.Now())
	if len(expired) != 0 {
		t.Fatalf("expected 0 expired, got %d", len(expired))
	}

	// Expired after grace period.
	expired = r.ExpiredSessions(time.Now().Add(31 * time.Second))
	if len(expired) != 1 {
		t.Fatalf("expected 1 expired, got %d", len(expired))
	}
	if expired[0].State != SessionExpired {
		t.Errorf("expected expired state, got %d", expired[0].State)
	}
}

func TestSessionReactivation(t *testing.T) {
	r := NewSessionRegistry(30 * time.Second)

	r.OnStreamStarted("game")
	r.OnStreamEnded("game")

	s := r.Get("game")
	if s.State != SessionDisconnected {
		t.Fatal("expected disconnected")
	}

	// Reconnect within grace period.
	r.OnStreamStarted("game")
	s = r.Get("game")
	if s.State != SessionActive {
		t.Errorf("expected active after reactivation, got %d", s.State)
	}
	if s.GraceExpiry != nil {
		t.Error("expected grace expiry to be cleared")
	}
	if s.EndedAt != nil {
		t.Error("expected EndedAt to be cleared")
	}
}

func TestSessionRemove(t *testing.T) {
	r := NewSessionRegistry(30 * time.Second)

	r.OnStreamStarted("game")
	r.Remove("game")

	if r.Get("game") != nil {
		t.Error("expected nil after remove")
	}
	if r.ActiveCount() != 0 {
		t.Error("expected 0 active after remove")
	}
}

func TestMultipleSessions(t *testing.T) {
	r := NewSessionRegistry(30 * time.Second)

	r.OnStreamStarted("game1")
	r.OnStreamStarted("game2")
	r.OnStreamStarted("game3")

	if r.ActiveCount() != 3 {
		t.Fatalf("expected 3 active, got %d", r.ActiveCount())
	}

	r.OnStreamEnded("game2")
	if r.ActiveCount() != 2 {
		t.Fatalf("expected 2 active, got %d", r.ActiveCount())
	}

	active := r.ActiveSessions()
	if len(active) != 2 {
		t.Fatalf("expected 2 active sessions, got %d", len(active))
	}
}

func TestEndNonexistent(t *testing.T) {
	r := NewSessionRegistry(30 * time.Second)
	// Should not panic.
	r.OnStreamEnded("nonexistent")
}
