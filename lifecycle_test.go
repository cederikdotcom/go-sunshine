package sunshine

import (
	"testing"
	"time"
)

func TestLifecycleTick(t *testing.T) {
	r := NewSessionRegistry(10 * time.Second)
	lm := NewLifecycleManager(r, nil)

	// No sessions, no actions.
	actions := lm.Tick(time.Now())
	if len(actions) != 0 {
		t.Fatalf("expected 0 actions, got %d", len(actions))
	}

	// Start and end a session.
	r.OnStreamStarted("game")
	r.OnStreamEnded("game")

	// Not expired yet.
	actions = lm.Tick(time.Now())
	if len(actions) != 0 {
		t.Fatalf("expected 0 actions before grace, got %d", len(actions))
	}

	// Expired.
	actions = lm.Tick(time.Now().Add(11 * time.Second))
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Session.AppName != "game" {
		t.Errorf("expected app 'game', got %q", actions[0].Session.AppName)
	}
	if actions[0].Reason != "grace period expired" {
		t.Errorf("unexpected reason: %s", actions[0].Reason)
	}
}

func TestLifecycleNoDoubleExpire(t *testing.T) {
	r := NewSessionRegistry(5 * time.Second)
	lm := NewLifecycleManager(r, nil)

	r.OnStreamStarted("game")
	r.OnStreamEnded("game")

	// First tick expires.
	actions := lm.Tick(time.Now().Add(6 * time.Second))
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}

	// Second tick should not re-expire (already in expired state).
	actions = lm.Tick(time.Now().Add(7 * time.Second))
	if len(actions) != 0 {
		t.Fatalf("expected 0 actions on second tick, got %d", len(actions))
	}
}
