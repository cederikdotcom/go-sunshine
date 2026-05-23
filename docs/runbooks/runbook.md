# go-sunshine runbook

Go SDK for the Sunshine game streaming server HTTP API. Used by hydrabody to manage Sunshine lifecycle from Go.

## Installation

This is a Go module — import it as a dependency:

```
go get github.com/cederikdotcom/go-sunshine@latest
```

## Configuration

Create a client pointing at the local Sunshine instance:

```go
client := sunshine.NewClient("https://localhost:47990", sunshine.ClientOptions{
    Username:   "admin",
    Password:   "password",
    HTTPClient: &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}},
})
```

Sunshine runs on port 47990 by default with a self-signed TLS cert — `InsecureSkipVerify` is required on the local machine.

## Common operations

### List registered apps

```go
apps, err := client.ListApps()
```

### Register or update an app

```go
err := client.RegisterApp(sunshine.App{
    Name:        "my-experience",
    Command:     `"C:\path\to\experience.exe"`,
    PrepCmd:     []sunshine.PrepCmd{{Do: "setup.bat", Undo: "teardown.bat"}},
})
```

### Close the currently running app

```go
err := client.CloseRunningApp()
```

This sends `POST /api/apps/close` with `Content-Type: application/json`. Sunshine runs the prep-cmd Undo hook as part of teardown — prefer this over killing the process directly.

### Check active clients

```go
clients, err := client.ListClients()
```

## Troubleshooting

### CloseRunningApp returns HTTP 400 "Content type not provided"

Sunshine requires `Content-Type: application/json` on all POST requests, even when the body is empty. The SDK handles this by passing `struct{}{}` (marshals to `{}`) rather than `nil`. If you see this error, check that you are on `v0.1.1` or later — earlier versions passed `nil` and triggered this.

### TLS errors connecting to Sunshine

Sunshine uses a self-signed certificate. The client must be configured with `InsecureSkipVerify: true` or a custom cert pool containing the Sunshine cert. There is no way to disable TLS on Sunshine's API port.

### Sunshine API unreachable

Sunshine takes 10–30 seconds to start after a service restart. The SDK does not retry — callers should poll with a backoff until `ListApps` returns without error before issuing other calls.

## Releasing

Tag the module with a semver tag — the Go module proxy picks it up automatically:

```
git tag v0.1.x
git push --tags
```

Consumers run `go get github.com/cederikdotcom/go-sunshine@v0.1.x` to update.
