package file

import (
	"errors"
	"os"

	types "github.com/0xPolygon/heimdall-v2/types/error"
)

// PermCheck checks the secret key and the keystore files.
// it verifies whether they are stored with the correct permissions.
func PermCheck(filePath string, validPerm os.FileMode) error {
	// get path to keystore files
	f, err := os.Stat(filePath)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return types.InvalidPermissionsError{File: filePath, Perm: validPerm, Err: err}
	}

	filePerm := f.Mode()
	if filePerm != validPerm {
		return types.InvalidPermissionsError{File: filePath, Perm: validPerm}
	}

	return nil
}
