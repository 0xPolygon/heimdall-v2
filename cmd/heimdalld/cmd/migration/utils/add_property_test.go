package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddPropertyIfMissing(t *testing.T) {
	t.Run("sets value when key is missing", func(t *testing.T) {
		data := map[string]interface{}{"a": map[string]interface{}{}}
		require.NoError(t, AddPropertyIfMissing(data, "a", "k", "v"))
		require.Equal(t, "v", data["a"].(map[string]interface{})["k"])
	})

	t.Run("preserves existing value", func(t *testing.T) {
		data := map[string]interface{}{"a": map[string]interface{}{"k": "orig"}}
		require.NoError(t, AddPropertyIfMissing(data, "a", "k", "new"))
		require.Equal(t, "orig", data["a"].(map[string]interface{})["k"])
	})

	t.Run("invalid path errors", func(t *testing.T) {
		data := map[string]interface{}{}
		require.Error(t, AddPropertyIfMissing(data, "nope", "k", "v"))
	})
}
