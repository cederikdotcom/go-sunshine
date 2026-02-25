package sunshine

import (
	"sync"
	"time"
)

// SessionRegistry tracks active streaming sessions with grace period support.
// It is safe for concurrent use.
type SessionRegistry struct {
	mu       sync.Mutex
	sessions map[string]*StreamSession
	grace    time.Duration
}

// NewSessionRegistry creates a registry with the given grace period.
// When a stream ends, the session enters a disconnected state for this
// duration before being marked as expired.
func NewSessionRegistry(gracePeriod time.Duration) *SessionRegistry {
	return &SessionRegistry{
		sessions: make(map[string]*StreamSession),
		grace:    gracePeriod,
	}
}

// OnStreamStarted creates a new session or reactivates a disconnected one.
func (r *SessionRegistry) OnStreamStarted(appName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if s, ok := r.sessions[appName]; ok {
		// Reactivate disconnected session (client reconnected within grace).
		s.State = SessionActive
		s.EndedAt = nil
		s.GraceExpiry = nil
		return
	}

	r.sessions[appName] = &StreamSession{
		AppName:   appName,
		StartedAt: time.Now(),
		State:     SessionActive,
		Metadata:  make(map[string]string),
	}
}

// OnStreamEnded transitions an active session to the disconnected state
// and starts the grace period timer.
func (r *SessionRegistry) OnStreamEnded(appName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.sessions[appName]
	if !ok {
		return
	}

	now := time.Now()
	expiry := now.Add(r.grace)
	s.State = SessionDisconnected
	s.EndedAt = &now
	s.GraceExpiry = &expiry
}

// ExpiredSessions returns sessions whose grace period has elapsed.
func (r *SessionRegistry) ExpiredSessions(now time.Time) []*StreamSession {
	r.mu.Lock()
	defer r.mu.Unlock()

	var expired []*StreamSession
	for _, s := range r.sessions {
		if s.State == SessionDisconnected && s.GraceExpiry != nil && now.After(*s.GraceExpiry) {
			s.State = SessionExpired
			expired = append(expired, s)
		}
	}
	return expired
}

// ActiveSessions returns all sessions in the active state.
func (r *SessionRegistry) ActiveSessions() []*StreamSession {
	r.mu.Lock()
	defer r.mu.Unlock()

	var active []*StreamSession
	for _, s := range r.sessions {
		if s.State == SessionActive {
			active = append(active, s)
		}
	}
	return active
}

// ActiveCount returns the number of active sessions.
func (r *SessionRegistry) ActiveCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0
	for _, s := range r.sessions {
		if s.State == SessionActive {
			count++
		}
	}
	return count
}

// IsStreaming returns true if any session is in the active state.
func (r *SessionRegistry) IsStreaming() bool {
	return r.ActiveCount() > 0
}

// Get returns the session for the given app name, or nil.
func (r *SessionRegistry) Get(appName string) *StreamSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessions[appName]
}

// Remove deletes a session from the registry.
func (r *SessionRegistry) Remove(appName string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, appName)
}
