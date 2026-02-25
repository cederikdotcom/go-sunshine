package sunshine

import (
	"strings"
	"testing"
)

func TestCallbackHooks(t *testing.T) {
	hooks := CallbackHooks("http://localhost:47991", "my-game", "curl.exe")

	if len(hooks) != 1 {
		t.Fatalf("expected 1 prep-cmd, got %d", len(hooks))
	}

	h := hooks[0]
	if !strings.Contains(h.Do, "/api/v1/stream/started") {
		t.Errorf("Do should contain stream/started, got: %s", h.Do)
	}
	if !strings.Contains(h.Undo, "/api/v1/stream/ended") {
		t.Errorf("Undo should contain stream/ended, got: %s", h.Undo)
	}
	if !strings.Contains(h.Do, "my-game") {
		t.Errorf("Do should contain app name, got: %s", h.Do)
	}
	if !strings.Contains(h.Do, "curl.exe") {
		t.Errorf("Do should use curl.exe, got: %s", h.Do)
	}
}

func TestCallbackHooksLinux(t *testing.T) {
	hooks := CallbackHooks("http://localhost:8080", "app", "curl")
	if !strings.Contains(hooks[0].Do, "curl -s") {
		t.Errorf("expected 'curl' binary, got: %s", hooks[0].Do)
	}
}
