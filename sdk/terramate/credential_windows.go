//go:build windows

package terramate

import (
	"os"
)

// checkCredentialFilePermissions validates that the credential file has secure permissions
// On Windows, we skip Unix-style permission checks as Windows uses ACLs instead
// The Terramate CLI handles setting appropriate Windows ACLs when creating the credential file
func checkCredentialFilePermissions(path string, fileInfo os.FileInfo) error {
	// Windows uses ACLs, not Unix permissions
	// The Terramate CLI sets appropriate permissions when creating the file
	// For now, we trust the Windows file system security model
	return nil
}
