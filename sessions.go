package sunshine

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// SessionRegistry tracks active streaming sessions with grace period support.
// It is safe for concurrent use.
//
// Sessions are keyed by a registry-minted session ID, NOT by app name. One app
// can therefore have several concurrent sessions, which is what a body serving
// more than one head at a time actually does. Keying by app name collapsed them
// into a single record, so ending one stream made every other stream of the
// same app look like it had no session, and callers that clean up unowned
// processes killed all of them.
type SessionRegistry struct {
	mu       sync.Mutex
	sessions map[string]*StreamSession
	grace    time.Duration
	seq      uint64
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

// newIDLocked mints a session ID. The caller must hold r.mu. The app name is
// included so a log line or a map dump says which experience a session belongs
// to without a second lookup.
func (r *SessionRegistry) newIDLocked(appName string) string {
	r.seq++
	return fmt.Sprintf("%s#%d", appName, r.seq)
}

// OnStreamStarted starts a session for appName and returns its session ID.
//
// A disconnected session for the same app that is still inside its grace period
// is reactivated rather than duplicated, because that is a client reconnecting.
// Otherwise a NEW session is created, so a second concurrent stream of the same
// app gets its own record.
//
// Sunshine's stream hooks carry only the app name, so this call cannot tell
// "slot 2 is starting" from "slot 1 reconnected" when a disconnected session is
// pending. A caller that knows its own identity should use OnStreamStartedID
// instead, which is unambiguous.
func (r *SessionRegistry) OnStreamStarted(appName string) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if s := r.newestLocked(appName, SessionDisconnected); s != nil {
		s.State = SessionActive
		s.EndedAt = nil
		s.GraceExpiry = nil
		return s.ID
	}

	return r.startLocked(r.newIDLocked(appName), appName).ID
}

// OnStreamStartedID starts or reactivates the session with the given ID. Use it
// when the caller has its own stable identity for the stream, such as a slot on
// a multi-stream body. Reactivation is then exact: a reconnect reuses its own
// session and never adopts another slot's.
func (r *SessionRegistry) OnStreamStartedID(sessionID, appName string) *StreamSession {
	r.mu.Lock()
	defer r.mu.Unlock()

	if s, ok := r.sessions[sessionID]; ok {
		s.State = SessionActive
		s.EndedAt = nil
		s.GraceExpiry = nil
		return s
	}
	return r.startLocked(sessionID, appName)
}

func (r *SessionRegistry) startLocked(sessionID, appName string) *StreamSession {
	s := &StreamSession{
		ID:        sessionID,
		AppName:   appName,
		StartedAt: time.Now(),
		State:     SessionActive,
		Metadata:  make(map[string]string),
	}
	r.sessions[sessionID] = s
	return s
}

// OnStreamEnded moves the NEWEST active session for appName into the
// disconnected state and starts its grace timer. Other concurrent sessions for
// the same app are left alone.
//
// Prefer OnStreamEndedID when the caller knows which session ended. With only
// an app name there is no way to tell which of several concurrent streams the
// hook referred to, and ending the newest is a guess.
func (r *SessionRegistry) OnStreamEnded(appName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s := r.newestLocked(appName, SessionActive)
	if s == nil {
		return
	}
	r.endLocked(s)
}

// OnStreamEndedID moves one specific session into the disconnected state.
func (r *SessionRegistry) OnStreamEndedID(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	s, ok := r.sessions[sessionID]
	if !ok || s.State != SessionActive {
		return
	}
	r.endLocked(s)
}

func (r *SessionRegistry) endLocked(s *StreamSession) {
	now := time.Now()
	expiry := now.Add(r.grace)
	s.State = SessionDisconnected
	s.EndedAt = &now
	s.GraceExpiry = &expiry
}

// newestLocked returns the most recently started session for appName in the
// given state, or nil. The caller must hold r.mu.
func (r *SessionRegistry) newestLocked(appName string, state SessionState) *StreamSession {
	var best *StreamSession
	for _, s := range r.sessions {
		if s.AppName != appName || s.State != state {
			continue
		}
		if best == nil || s.StartedAt.After(best.StartedAt) {
			best = s
		}
	}
	return best
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
	sortByStart(expired)
	return expired
}

// ActiveSessions returns all sessions in the active state, oldest first.
func (r *SessionRegistry) ActiveSessions() []*StreamSession {
	r.mu.Lock()
	defer r.mu.Unlock()

	var active []*StreamSession
	for _, s := range r.sessions {
		if s.State == SessionActive {
			active = append(active, s)
		}
	}
	sortByStart(active)
	return active
}

// ActiveSessionsForApp returns the active sessions for one app, oldest first.
func (r *SessionRegistry) ActiveSessionsForApp(appName string) []*StreamSession {
	r.mu.Lock()
	defer r.mu.Unlock()

	var active []*StreamSession
	for _, s := range r.sessions {
		if s.State == SessionActive && s.AppName == appName {
			active = append(active, s)
		}
	}
	sortByStart(active)
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

// Get returns the newest session for the given app name, preferring an active
// one, or nil. It exists for callers that predate concurrent sessions; use
// GetByID or ActiveSessionsForApp when more than one session can be in play.
func (r *SessionRegistry) Get(appName string) *StreamSession {
	r.mu.Lock()
	defer r.mu.Unlock()

	if s := r.newestLocked(appName, SessionActive); s != nil {
		return s
	}
	var best *StreamSession
	for _, s := range r.sessions {
		if s.AppName != appName {
			continue
		}
		if best == nil || s.StartedAt.After(best.StartedAt) {
			best = s
		}
	}
	return best
}

// GetByID returns one session by its ID, or nil.
func (r *SessionRegistry) GetByID(sessionID string) *StreamSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessions[sessionID]
}

// Remove deletes EVERY session for the given app name. Callers that clean up a
// single finished stream should use RemoveByID, or they will discard the
// records of other streams of the same app that are still running.
func (r *SessionRegistry) Remove(appName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, s := range r.sessions {
		if s.AppName == appName {
			delete(r.sessions, id)
		}
	}
}

// RemoveByID deletes one session from the registry.
func (r *SessionRegistry) RemoveByID(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, sessionID)
}

// OwnedPIDs returns the PIDs recorded in the metadata of every session that is
// not expired, as a set. A cleanup pass uses it to avoid killing a process that
// belongs to a stream which is still running or still inside its grace period.
//
// The metadata key is "pid" and is written by the caller, so a session with no
// recorded PID simply contributes nothing.
func (r *SessionRegistry) OwnedPIDs() map[string]bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	owned := make(map[string]bool)
	for _, s := range r.sessions {
		if s.State == SessionExpired || s.Metadata == nil {
			continue
		}
		if pid := s.Metadata["pid"]; pid != "" {
			owned[pid] = true
		}
	}
	return owned
}

// UntrackedApps returns the app names that have at least one live session
// (active, or disconnected inside its grace period) whose PID was never
// recorded.
//
// A cleanup pass cannot tell such a session's processes from a genuinely
// orphaned one, so it must leave that app alone rather than guess. Without
// this, a single failed PID lookup at stream start would make a running
// stream look like an orphan and get it killed.
func (r *SessionRegistry) UntrackedApps() map[string]bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	untracked := make(map[string]bool)
	for _, s := range r.sessions {
		if s.State == SessionExpired {
			continue
		}
		if s.Metadata == nil || s.Metadata["pid"] == "" {
			untracked[s.AppName] = true
		}
	}
	return untracked
}

func sortByStart(ss []*StreamSession) {
	sort.Slice(ss, func(i, j int) bool { return ss[i].StartedAt.Before(ss[j].StartedAt) })
}
