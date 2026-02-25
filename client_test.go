package sunshine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := NewClient(srv.URL, "admin", "pass")
	client.HTTP = srv.Client() // use plain HTTP for tests
	return srv, client
}

func TestListApps(t *testing.T) {
	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apps" || r.Method != "GET" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(appsResponse{
			Apps: []App{
				{Name: "desktop", Index: 0},
				{Name: "game", Index: 1, Cmd: "game.exe"},
			},
		})
	})

	apps, err := c.ListApps()
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(apps))
	}
	if apps[1].Name != "game" {
		t.Errorf("expected app name 'game', got %q", apps[1].Name)
	}
}

func TestRegisterApp(t *testing.T) {
	var received App
	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	})

	err := c.RegisterApp(App{Name: "test", Index: -1, Cmd: "test.exe"})
	if err != nil {
		t.Fatal(err)
	}
	if received.Name != "test" {
		t.Errorf("expected name 'test', got %q", received.Name)
	}
}

func TestDeleteApp(t *testing.T) {
	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/api/apps/2" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := c.DeleteApp(2); err != nil {
		t.Fatal(err)
	}
}

func TestCloseRunningApp(t *testing.T) {
	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/apps/close" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := c.CloseRunningApp(); err != nil {
		t.Fatal(err)
	}
}

func TestHTTPError(t *testing.T) {
	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	})

	_, err := c.ListApps()
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

func TestBasicAuth(t *testing.T) {
	var gotUser, gotPass string
	_, c := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, _ = r.BasicAuth()
		json.NewEncoder(w).Encode(appsResponse{})
	})

	c.ListApps()
	if gotUser != "admin" || gotPass != "pass" {
		t.Errorf("expected admin/pass, got %s/%s", gotUser, gotPass)
	}
}
