package sunshine

import (
	"testing"
	"time"
)

// The production defect this file guards against, measured on cosmic-pretzel-98
// on 2026-10-17: two concurrent streams of one experience shared a single
// session record, so ending the first made the second look unowned and a
// cleanup pass killed it six seconds later.

func TestTwoConcurrentSessionsOfOneApp(t *testing.T) {
	r := NewSessionRegistry(30 * time.Second)

	a := r.OnStreamStartedID("slot-0", "mercator-talks")
	b := r.OnStreamStartedID("slot-1", "mercator-talks")

	if a.ID == b.ID {
		t.Fatalf("two slots shared one session ID: %q", a.ID)
	}
	if got := r.ActiveCount(); got != 2 {
		t.Errorf("ActiveCount = %d, want 2", got)
	}
	if got := len(r.ActiveSessionsForApp("mercator-talks")); got != 2 {
		t.Errorf("ActiveSessionsForApp = %d, want 2", got)
	}
}

func TestEndingOneStreamLeavesTheOtherRunning(t *testing.T) {
	r := NewSessionRegistry(30 * time.Second)

	first := r.OnStreamStartedID("slot-0", "mercator-talks")
	second := r.OnStreamStartedID("slot-1", "mercator-talks")
	first.Metadata["pid"] = "1111"
	second.Metadata["pid"] = "2222"

	r.OnStreamEndedID(first.ID)

	if got := r.ActiveCount(); got != 1 {
		t.Fatalf("ActiveCount after one stream ended = %d, want 1", got)
	}
	active := r.ActiveSessionsForApp("mercator-talks")
	if len(active) != 1 || active[0].ID != second.ID {
		t.Fatalf("surviving session = %+v, want slot-1", active)
	}

	// The heart of it: slot 1 ending must not make slot 2's process look
	// unowned, which is what made the cleanup pass kill it.
	owned := r.OwnedPIDs()
	if !owned["2222"] {
		t.Error("PID of the still-running stream is not owned; cleanup would kill it")
	}
	if !owned["1111"] {
		t.Error("PID of the stream inside its grace period is not owned; cleanup would kill it early")
	}
}

func TestUnkeyedEndPicksOneSessionNotAll(t *testing.T) {
	r := NewSessionRegistry(30 * time.Second)
	r.OnStreamStartedID("slot-0", "mercator-talks")
	r.OnStreamStartedID("slot-1", "mercator-talks")

	// The Sunshine hook carries only an app name. Ending by app name must
	// still retire exactly one session, never the whole app.
	r.OnStreamEnded("mercator-talks")

	if got := r.ActiveCount(); got != 1 {
		t.Errorf("ActiveCount = %d, want 1: an app-name end retired more than one session", got)
	}
}

func TestReconnectWithinGraceReusesTheSession(t *testing.T) {
	r := NewSessionRegistry(30 * time.Second)

	id := r.OnStreamStarted("mercator-talks")
	r.OnStreamEnded("mercator-talks")
	again := r.OnStreamStarted("mercator-talks")

	if again != id {
		t.Errorf("reconnect minted a new session %q, want the existing %q", again, id)
	}
	if got := len(r.ActiveSessionsForApp("mercator-talks")); got != 1 {
		t.Errorf("sessions for app = %d, want 1: a reconnect was counted as a second stream", got)
	}
}

func TestSecondStreamWhileFirstIsActiveMintsANewSession(t *testing.T) {
	r := NewSessionRegistry(30 * time.Second)

	first := r.OnStreamStarted("mercator-talks")
	second := r.OnStreamStarted("mercator-talks")

	if first == second {
		t.Fatal("second concurrent stream reused the first session")
	}
	if got := r.ActiveCount(); got != 2 {
		t.Errorf("ActiveCount = %d, want 2", got)
	}
}

func TestOwnedPIDsDropsExpiredSessions(t *testing.T) {
	r := NewSessionRegistry(time.Millisecond)

	s := r.OnStreamStartedID("slot-0", "mercator-talks")
	s.Metadata["pid"] = "4242"
	r.OnStreamEndedID(s.ID)

	if !r.OwnedPIDs()["4242"] {
		t.Fatal("PID stopped being owned during the grace period")
	}

	r.ExpiredSessions(time.Now().Add(time.Hour))

	if r.OwnedPIDs()["4242"] {
		t.Error("PID is still owned after the grace period expired; the process would never be cleaned up")
	}
}

func TestRemoveByIDLeavesSiblingsAlone(t *testing.T) {
	r := NewSessionRegistry(30 * time.Second)
	a := r.OnStreamStartedID("slot-0", "mercator-talks")
	b := r.OnStreamStartedID("slot-1", "mercator-talks")

	r.RemoveByID(a.ID)

	if r.GetByID(b.ID) == nil {
		t.Error("removing one session removed its sibling")
	}
	if r.GetByID(a.ID) != nil {
		t.Error("session was not removed")
	}
}

func TestRemoveByAppStillClearsEverySession(t *testing.T) {
	r := NewSessionRegistry(30 * time.Second)
	r.OnStreamStartedID("slot-0", "mercator-talks")
	r.OnStreamStartedID("slot-1", "mercator-talks")
	r.OnStreamStartedID("slot-2", "rupelmonde-castle-viewer")

	r.Remove("mercator-talks")

	if got := r.ActiveCount(); got != 1 {
		t.Errorf("ActiveCount = %d, want 1: Remove(app) should clear only that app", got)
	}
}

func TestSessionsOfDifferentAppsAreIndependent(t *testing.T) {
	r := NewSessionRegistry(30 * time.Second)
	r.OnStreamStartedID("slot-0", "mercator-talks")
	b := r.OnStreamStartedID("slot-1", "rupelmonde-castle-viewer")

	r.OnStreamEndedID(b.ID)

	if got := len(r.ActiveSessionsForApp("mercator-talks")); got != 1 {
		t.Errorf("mercator sessions = %d, want 1", got)
	}
	if got := len(r.ActiveSessionsForApp("rupelmonde-castle-viewer")); got != 0 {
		t.Errorf("rupelmonde sessions = %d, want 0", got)
	}
}

func TestUntrackedAppsProtectsSessionsWithNoPID(t *testing.T) {
	r := NewSessionRegistry(30 * time.Second)

	tracked := r.OnStreamStartedID("slot-0", "mercator-talks")
	tracked.Metadata["pid"] = "1111"
	r.OnStreamStartedID("slot-1", "rupelmonde-castle-viewer") // PID lookup failed

	u := r.UntrackedApps()
	if u["mercator-talks"] {
		t.Error("an app whose session has a PID was reported untracked")
	}
	if !u["rupelmonde-castle-viewer"] {
		t.Error("an app whose session has no PID must be reported untracked, or cleanup would kill a live stream")
	}
}

func TestUntrackedAppsIgnoresExpiredSessions(t *testing.T) {
	r := NewSessionRegistry(time.Millisecond)

	s := r.OnStreamStartedID("slot-0", "mercator-talks")
	r.OnStreamEndedID(s.ID)
	r.ExpiredSessions(time.Now().Add(time.Hour))

	if r.UntrackedApps()["mercator-talks"] {
		t.Error("an expired session still blocks cleanup; orphans would never be collected")
	}
}
