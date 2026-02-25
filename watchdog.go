package sunshine

// WatchdogEvent describes a health check result that requires action.
type WatchdogEvent struct {
	Session *StreamSession
	Check   string // name of the check that triggered
	Action  WatchdogAction
}

// Watchdog runs pluggable health checks against active sessions.
// go-sunshine ships with zero built-in checks. The caller registers
// platform-specific checks (e.g. ProcessAliveCheck).
type Watchdog struct {
	checks   []WatchdogCheck
	sessions *SessionRegistry
}

// NewWatchdog creates a watchdog for the given session registry.
func NewWatchdog(sessions *SessionRegistry) *Watchdog {
	return &Watchdog{
		sessions: sessions,
	}
}

// Register adds a health check to the watchdog.
func (w *Watchdog) Register(check WatchdogCheck) {
	w.checks = append(w.checks, check)
}

// Tick runs all registered checks against active sessions and returns
// events for any sessions that need action.
func (w *Watchdog) Tick() []WatchdogEvent {
	active := w.sessions.ActiveSessions()
	if len(active) == 0 || len(w.checks) == 0 {
		return nil
	}

	var events []WatchdogEvent
	for _, s := range active {
		for _, check := range w.checks {
			action := check.Check(s)
			if action != ActionNone {
				events = append(events, WatchdogEvent{
					Session: s,
					Check:   check.Name(),
					Action:  action,
				})
			}
		}
	}
	return events
}
