// Package vectors holds the test cases every Bugnet implementation reads from one file
package vectors

import "embed"

// X402 holds paths.json, service-ids.json and shared-hosting-suffixes.json
//
//go:embed x402/*.json
var X402 embed.FS
