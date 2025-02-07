package engine

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
)

type jwtTransport struct {
	underlyingTransport http.RoundTripper
	jwtSecret           []byte
}

// RoundTrip is an HTTP filter that injects a generated JWT token for authentication
func (t *jwtTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tokenString, err := generateTokenString(t.jwtSecret)
	if err != nil {
		return nil, errors.Wrap(err, "could not produce signed JWT token")
	}
	req.Header.Set("Authorization", "Bearer "+tokenString)
	return t.underlyingTransport.RoundTrip(req)
}

func parseJWTSecretFromFile(path string) ([]byte, error) {
	enc, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	strData := strings.TrimSpace(string(enc))
	if len(strData) == 0 {
		return nil, fmt.Errorf("%s is empty", path)
	}
	secret, err := hex.DecodeString(strings.TrimPrefix(strData, "0x"))
	if err != nil {
		return nil, err
	}
	if len(secret) < 32 {
		return nil, errors.New("JWT secret must be a hex string of at least 32 bytes")
	}
	return secret, nil
}

func generateTokenString(jwtSecret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat": time.Now().Unix(), // "is issued at" is required by engine API
	})
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
