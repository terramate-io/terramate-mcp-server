//go:build unix || darwin || linux

package terramate

import (
	"fmt"
	"os"
)

// checkCredentialFilePermissions validates that the credential file has secure permissions
// On Unix systems, the file should be 0600 (read/write for owner only)
func checkCredentialFilePermissions(path string, fileInfo os.FileInfo) error {
	mode := fileInfo.Mode()
	if mode.Perm()&0o077 != 0 {
		return fmt.Errorf(
			"credential file has insecure permissions: %s (mode: %o)\n\n"+
				"The file contains sensitive tokens and should only be readable by the owner.\n"+
				"To fix:\n"+
				"  chmod 0600 %s",
			path, mode.Perm(), path,
		)
	}
	return nil
}
