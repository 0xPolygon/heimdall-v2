//go:build tools

// This file is meant to track developer tools as dependencies in a go module
// See: https://github.com/go-modules-by-example/index/blob/master/010_tools/README.md

package tools

import (
	_ "github.com/bufbuild/buf/cmd/buf"
)
