package sunshine

import "time"

// StreamSession represents a single active streaming connection.
type StreamSession struct {
	AppName     string
	StartedAt   time.Time
	EndedAt     *time.Time // nil if still active
	GraceExpiry *time.Time // nil if not in grace period
	State       SessionState
	Metadata    map[string]string // caller can attach arbitrary data (e.g. PID)
}

// SessionState represents the lifecycle state of a stream session.
type SessionState int

const (
	SessionActive       SessionState = iota
	SessionDisconnected              // client gone, in grace period
	SessionExpired                   // grace period over, ready for cleanup
)

// App represents a Sunshine registered application.
type App struct {
	Name    string    `json:"name"`
	Index   int       `json:"index"`
	Cmd     string    `json:"cmd,omitempty"`
	PrepCmd []PrepCmd `json:"prep-cmd,omitempty"`
}

// PrepCmd represents a preparation command that Sunshine executes on
// stream start (Do) and stream end (Undo).
type PrepCmd struct {
	Do   string `json:"do"`
	Undo string `json:"undo"`
}

// ClientInfo represents a paired Moonlight client.
type ClientInfo struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// WatchdogAction tells the caller what to do about a session.
type WatchdogAction int

const (
	ActionNone WatchdogAction = iota
	ActionRelaunch
	ActionKill
)

// WatchdogCheck is a pluggable health check for active sessions.
type WatchdogCheck interface {
	Name() string
	Check(session *StreamSession) WatchdogAction
}
