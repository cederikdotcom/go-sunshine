package sunshine

import "time"

// CleanupAction describes a session that needs to be cleaned up.
type CleanupAction struct {
	Session *StreamSession
	Reason  string
}

// LifecycleManager detects expired sessions and signals them for cleanup.
// It does NOT kill processes directly. The caller handles platform-specific cleanup.
type LifecycleManager struct {
	sessions *SessionRegistry
	client   *Client
}

// NewLifecycleManager creates a lifecycle manager.
func NewLifecycleManager(sessions *SessionRegistry, client *Client) *LifecycleManager {
	return &LifecycleManager{
		sessions: sessions,
		client:   client,
	}
}

// Tick checks for expired sessions and returns cleanup actions.
func (lm *LifecycleManager) Tick(now time.Time) []CleanupAction {
	expired := lm.sessions.ExpiredSessions(now)
	if len(expired) == 0 {
		return nil
	}

	var actions []CleanupAction
	for _, s := range expired {
		actions = append(actions, CleanupAction{
			Session: s,
			Reason:  "grace period expired",
		})
	}
	return actions
}
