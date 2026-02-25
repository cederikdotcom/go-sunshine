package sunshine

import "testing"

// testCheck is a mock watchdog check for testing.
type testCheck struct {
	name   string
	action WatchdogAction
}

func (c *testCheck) Name() string                          { return c.name }
func (c *testCheck) Check(_ *StreamSession) WatchdogAction { return c.action }

func TestWatchdogNoChecks(t *testing.T) {
	r := NewSessionRegistry(0)
	w := NewWatchdog(r)

	r.OnStreamStarted("game")
	events := w.Tick()
	if len(events) != 0 {
		t.Fatalf("expected 0 events with no checks, got %d", len(events))
	}
}

func TestWatchdogWithCheck(t *testing.T) {
	r := NewSessionRegistry(0)
	w := NewWatchdog(r)
	w.Register(&testCheck{name: "process-alive", action: ActionRelaunch})

	r.OnStreamStarted("game")
	events := w.Tick()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Action != ActionRelaunch {
		t.Errorf("expected relaunch, got %d", events[0].Action)
	}
	if events[0].Check != "process-alive" {
		t.Errorf("expected check 'process-alive', got %q", events[0].Check)
	}
}

func TestWatchdogNoAction(t *testing.T) {
	r := NewSessionRegistry(0)
	w := NewWatchdog(r)
	w.Register(&testCheck{name: "healthy", action: ActionNone})

	r.OnStreamStarted("game")
	events := w.Tick()
	if len(events) != 0 {
		t.Fatalf("expected 0 events for ActionNone, got %d", len(events))
	}
}

func TestWatchdogMultipleSessions(t *testing.T) {
	r := NewSessionRegistry(0)
	w := NewWatchdog(r)
	w.Register(&testCheck{name: "kill-check", action: ActionKill})

	r.OnStreamStarted("game1")
	r.OnStreamStarted("game2")

	events := w.Tick()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestWatchdogSkipsInactive(t *testing.T) {
	r := NewSessionRegistry(0)
	w := NewWatchdog(r)
	w.Register(&testCheck{name: "check", action: ActionKill})

	r.OnStreamStarted("game")
	r.OnStreamEnded("game") // now disconnected, not active

	events := w.Tick()
	if len(events) != 0 {
		t.Fatalf("expected 0 events for disconnected session, got %d", len(events))
	}
}
