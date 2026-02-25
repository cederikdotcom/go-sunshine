# go-sunshine

Go SDK for the [Sunshine](https://github.com/LizardByte/Sunshine) game streaming server.

Provides a REST API client, session lifecycle management with grace periods, and an extensible watchdog framework for health checks.

## Features

- **REST API Client** - Full coverage of the Sunshine API (apps, clients, config, logs, restart)
- **Session Registry** - Track active streaming sessions with thread-safe grace period support
- **Lifecycle Manager** - Detect expired sessions after disconnect grace periods
- **Watchdog** - Pluggable health check framework for active sessions
- **Callback Hooks** - Generate `prep-cmd` entries for event-driven session tracking via Sunshine's app hooks

## Install

```bash
go get github.com/cederikdotcom/go-sunshine
```

## Usage

### API Client

```go
client := sunshine.NewClient("https://localhost:47990", "admin", "password")

apps, _ := client.ListApps()
client.RegisterApp(sunshine.App{Name: "game", Index: -1, Cmd: "game.exe"})
client.CloseRunningApp()
```

### Session Tracking

```go
registry := sunshine.NewSessionRegistry(30 * time.Second)

// Called by prep-cmd hooks when streams start/end.
registry.OnStreamStarted("game")
registry.OnStreamEnded("game")

// Check for expired sessions.
expired := registry.ExpiredSessions(time.Now())
```

### Lifecycle Management

```go
lm := sunshine.NewLifecycleManager(registry, client)

// Call periodically (e.g. every 30s).
actions := lm.Tick(time.Now())
for _, a := range actions {
    // Kill process, clean up resources, etc.
    log.Printf("cleanup: %s - %s", a.Session.AppName, a.Reason)
}
```

### Watchdog

```go
wd := sunshine.NewWatchdog(registry)
wd.Register(myProcessAliveCheck)

events := wd.Tick()
for _, e := range events {
    // Handle relaunch/kill based on e.Action
}
```

### Callback Hooks

```go
hooks := sunshine.CallbackHooks("http://localhost:47991", "game", "curl.exe")
app := sunshine.App{
    Name:    "game",
    Index:   -1,
    Cmd:     "game.exe",
    PrepCmd: hooks,
}
client.RegisterApp(app)
```

## License

MIT
