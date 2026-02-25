package sunshine

import "fmt"

// CallbackHooks generates PrepCmd entries that call back to a local HTTP
// server when a stream starts and ends. This is the event-driven mechanism
// that powers the session registry.
//
// callbackBaseURL is the base URL of the callback server (e.g. "http://localhost:47991").
// appName identifies which application the callback is for.
// curlBin is the curl binary to use (e.g. "curl" on Linux, "curl.exe" on Windows).
func CallbackHooks(callbackBaseURL, appName, curlBin string) []PrepCmd {
	return []PrepCmd{{
		Do: fmt.Sprintf(
			`%s -s -X POST %s/api/v1/stream/started -H "Content-Type: application/json" -d "{\"app\":\"%s\"}"`,
			curlBin, callbackBaseURL, appName,
		),
		Undo: fmt.Sprintf(
			`%s -s -X POST %s/api/v1/stream/ended -H "Content-Type: application/json" -d "{\"app\":\"%s\"}"`,
			curlBin, callbackBaseURL, appName,
		),
	}}
}
