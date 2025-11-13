package version

// Version is the semantic version embedded in binaries and used in the user agent.
const Version = "0.0.2"

// UserAgent returns the default HTTP User-Agent string for outbound requests.
func UserAgent() string {
	return "terramate-mcp-server/" + Version
}
