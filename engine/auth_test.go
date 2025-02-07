package engine

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cometbft/cometbft/libs/rand"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
)

func TestParseJWTSecret(t *testing.T) {
	const path = "test/jwt.hex"

	bytes, err := parseJWTSecretFromFile(path)
	require.NoError(t, err)

	token, err := generateTokenString(bytes)
	require.NoError(t, err)

	parts := strings.Split(token, ".")
	require.Equal(t, 3, len(parts))

	// first part is always same
	require.Equal(t, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", parts[0])
}

func TestJWTAuthTransport(t *testing.T) {
	jwtData := []byte("0x1d2021329264e9b4430bba1e5f6133447a94b74d5150dd17c93bdab52d5d3b64")
	jwtFile := fmt.Sprintf("%s/jwt.hex.%d", os.TempDir(), rand.Int32())
	defer func(t *testing.T, name string) {
		err := os.Remove(name)
		require.NoError(t, err)
	}(t, jwtFile)

	err := os.WriteFile(jwtFile, jwtData, 0666)
	require.NoError(t, err)

	secret, err := parseJWTSecretFromFile(jwtFile)
	require.NoError(t, err)

	authTransport := &jwtTransport{
		underlyingTransport: http.DefaultTransport,
		jwtSecret:           secret,
	}
	client := &http.Client{
		Timeout:   DefaultRPCTimeout,
		Transport: authTransport,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqToken := r.Header.Get("Authorization")
		splitToken := strings.Split(reqToken, "Bearer")
		// The format should be `Bearer ${token}`.
		require.Equal(t, 2, len(splitToken))
		reqToken = strings.TrimSpace(splitToken[1])
		token, err := jwt.Parse(reqToken, func(token *jwt.Token) (interface{}, error) {
			// We should be doing HMAC signing.
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			require.Equal(t, true, ok)
			return secret, nil
		})
		require.NoError(t, err)
		require.Equal(t, true, token.Valid)
		claims, ok := token.Claims.(jwt.MapClaims)
		require.Equal(t, true, ok)
		item, ok := claims["iat"]
		require.Equal(t, true, ok)
		iat, ok := item.(float64)
		require.Equal(t, true, ok)
		issuedAt := time.Unix(int64(iat), 0)
		// The claims should have an "iat" field (issued at) that is at most, 5 seconds ago.
		since := time.Since(issuedAt)
		require.Equal(t, true, since <= time.Second*5)
	}))
	defer srv.Close()
	_, err = client.Get(srv.URL)
	require.NoError(t, err)
}
