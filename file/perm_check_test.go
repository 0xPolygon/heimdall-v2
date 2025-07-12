package file

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	cmtos "github.com/cometbft/cometbft/libs/os"
	"github.com/stretchr/testify/assert"

	types "github.com/0xPolygon/heimdall-v2/types/error"
)

func TestPermCheck(t *testing.T) {
	t.Parallel()

	tc := []struct {
		filePath  string
		perm      os.FileMode
		validPerm os.FileMode
		expErr    error
		msg       string
	}{
		{
			filePath:  "/tmp/heimdall_test/test.json",
			perm:      0o777,
			validPerm: 0o600,
			expErr:    types.InvalidPermissionsError{File: "/tmp/heimdall_test/test.json", Perm: 0o600},
			msg:       "test for invalid permission",
		},
		{
			filePath:  "/tmp/heimdall_test/test.json",
			perm:      0o600,
			validPerm: 0o600,
			msg:       "success",
		},
	}

	for i, c := range tc {
		// get the path to the UAT secrets file
		caseMsg := fmt.Sprintf("for i: %v, case: %v", i, c.msg)
		// set files for perm

		err := cmtos.EnsureDir(filepath.Dir(c.filePath), 0o777)
		assert.Nil(t, err, caseMsg)
		_, err = os.OpenFile(c.filePath, os.O_CREATE, c.perm) // os.OpenFile creates the file if it is missing
		assert.Nil(t, err, caseMsg)

		// check file perm for the secret file
		err = PermCheck(c.filePath, c.validPerm)
		assert.Equal(t, c.expErr, err)

		err = os.Remove(c.filePath) // clean up
		assert.Nil(t, err)
	}
}
