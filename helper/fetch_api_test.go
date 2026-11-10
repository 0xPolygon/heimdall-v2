package helper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type getOnlyClient struct{ url string }

func (c *getOnlyClient) Get(url string) (*http.Response, error) {
	c.url = url
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
}

func TestFetchFromAPIWithContext(t *testing.T) {
	// Not parallel: swaps the global client.
	prev := Client
	t.Cleanup(func() { Client = prev })

	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			<-release
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	defer close(release)
	Client = &http.Client{Timeout: time.Minute}

	body, err := FetchFromAPIWithContext(context.Background(), srv.URL+"/fast")
	require.NoError(t, err)
	require.JSONEq(t, `{"ok":true}`, string(body))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err = FetchFromAPIWithContext(ctx, srv.URL+"/slow")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(started), 5*time.Second, "the context bounds the request")

	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		code := status
		failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"message":"page 2 does not exist"}` + strings.Repeat("x", apiErrorBodyLimit)))
		}))
		_, err = FetchFromAPIWithContext(context.Background(), failing.URL)
		failing.Close()
		require.ErrorContains(t, err, "status: "+strconv.Itoa(code))
		var statusErr *APIStatusError
		require.ErrorAs(t, err, &statusErr)
		require.Equal(t, code, statusErr.Status)
		require.Contains(t, statusErr.Body, "page 2 does not exist")
		require.Len(t, statusErr.Body, apiErrorBodyLimit, "only the start of the body is kept")
	}

	huge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, APIBodyLimit+1))
	}))
	_, err = FetchFromAPIWithContext(context.Background(), huge.URL)
	huge.Close()
	require.Error(t, err, "a body over APIBodyLimit isn't read")

	stub := &getOnlyClient{}
	Client = stub
	_, err = FetchFromAPIWithContext(context.Background(), "http://heimdall/clerk/event-records/count")
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(stub.url, "/count"), "a client without Do falls back to Get")
}

func TestFetchFromAPIWithContextBadURL(t *testing.T) {
	prev := Client
	t.Cleanup(func() { Client = prev })
	Client = &http.Client{Timeout: time.Minute}

	_, err := FetchFromAPIWithContext(context.Background(), "http://\x7f")
	require.Error(t, err)
}
