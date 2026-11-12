package util

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	"github.com/stretchr/testify/require"

	"github.com/0xPolygon/heimdall-v2/helper"
)

func TestGetLatestRecordID(t *testing.T) {
	// Not parallel: points the global helper config and client at a stub.
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	tests := []struct {
		name    string
		status  int
		body    string
		want    uint64
		wantErr bool
	}{
		{name: "latest id", status: http.StatusOK, body: `{"latest_record_id":"49992","is_processed_by_heimdall":false}`, want: 49992},
		{name: "query failed", status: http.StatusInternalServerError, body: `{"code":13}`, wantErr: true},
		{name: "malformed body", status: http.StatusOK, body: `{`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, ClerkLatestRecordIDURL, r.URL.Path)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			server := *serverconfig.DefaultConfig()
			server.API.Address = srv.URL
			helper.SetTestConfig(helper.CustomAppConfig{Config: server, Custom: helper.GetDefaultHeimdallConfig()})
			prev := helper.Client
			helper.Client = &http.Client{}
			t.Cleanup(func() { helper.Client = prev })

			got, err := GetLatestRecordID(cdc)
			if tc.wantErr {
				require.Error(t, err)
				require.Zero(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestGetRecordCount(t *testing.T) {
	// Not parallel: points the global helper config and client at a stub.
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	tests := []struct {
		name    string
		status  int
		body    string
		want    uint64
		wantErr bool
	}{
		{name: "count", status: http.StatusOK, body: `{"count":"49991"}`, want: 49991},
		{name: "query failed", status: http.StatusInternalServerError, body: `{"code":13}`, wantErr: true},
		{name: "malformed body", status: http.StatusOK, body: `{`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, ClerkRecordCountURL, r.URL.Path)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			server := *serverconfig.DefaultConfig()
			server.API.Address = srv.URL
			helper.SetTestConfig(helper.CustomAppConfig{Config: server, Custom: helper.GetDefaultHeimdallConfig()})
			prev := helper.Client
			helper.Client = &http.Client{}
			t.Cleanup(func() { helper.Client = prev })

			got, err := GetRecordCount(context.Background(), cdc)
			if tc.wantErr {
				require.Error(t, err)
				require.Zero(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestGetRecordCountHonoursContext(t *testing.T) {
	// Not parallel: points the global helper config and client at a stub.
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(time.Minute):
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()

	server := *serverconfig.DefaultConfig()
	server.API.Address = srv.URL
	helper.SetTestConfig(helper.CustomAppConfig{Config: server, Custom: helper.GetDefaultHeimdallConfig()})
	prev := helper.Client
	helper.Client = &http.Client{}
	t.Cleanup(func() { helper.Client = prev })

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := GetRecordCount(ctx, cdc)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestGetRecordIDsPage(t *testing.T) {
	// Not parallel: points the global helper config and client at a stub.
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	tests := []struct {
		name    string
		status  int
		body    string
		want    []uint64
		wantErr bool
	}{
		{name: "records", status: http.StatusOK, body: `{"event_records":[{"id":"49951"},{"id":"49953"}]}`, want: []uint64{49951, 49953}},
		{name: "no records on the page", status: http.StatusOK, body: `{"event_records":[]}`, want: []uint64{}},
		{name: "query failed", status: http.StatusInternalServerError, body: `{"code":13}`, wantErr: true},
		{name: "empty page", status: http.StatusInternalServerError, body: `{"code":13,"message":"page 1000 does not exist","details":[]}`, want: []uint64{}},
		{name: "another page's answer", status: http.StatusInternalServerError, body: `{"code":13,"message":"page 999 does not exist","details":[]}`, wantErr: true},
		{name: "malformed body", status: http.StatusOK, body: `{`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/clerk/event-records/list", r.URL.Path)
				require.Equal(t, "page=1000&limit=50", r.URL.RawQuery)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			server := *serverconfig.DefaultConfig()
			server.API.Address = srv.URL
			helper.SetTestConfig(helper.CustomAppConfig{Config: server, Custom: helper.GetDefaultHeimdallConfig()})
			prev := helper.Client
			helper.Client = &http.Client{}
			t.Cleanup(func() { helper.Client = prev })

			got, err := GetRecordIDsPage(context.Background(), cdc, 1000, 50)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestClerkRecordStored(t *testing.T) {
	// Not parallel: points the global helper config and client at a stub.
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	tests := []struct {
		name       string
		status     int
		body       string
		wantStored bool
		wantErr    bool
	}{
		{name: "stored", status: http.StatusOK, body: `{"record":{"id":"49992"}}`, wantStored: true},
		{name: "another record", status: http.StatusOK, body: `{"record":{"id":"49991"}}`, wantErr: true},
		{name: "empty record", status: http.StatusOK, body: `{"record":{}}`, wantErr: true},
		{name: "not found", status: http.StatusNotFound},
		{name: "heimdall's not-found answer", status: http.StatusInternalServerError, body: `{"code":13,"message":"collections: not found: key '49992' of type uint64"}`},
		{name: "other query failure", status: http.StatusInternalServerError, body: `{"code":13,"message":"database closed"}`, wantErr: true},
		{name: "unavailable", status: http.StatusServiceUnavailable, wantErr: true},
		{name: "malformed body", status: http.StatusOK, body: `{`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/clerk/event-records/49992", r.URL.Path)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			server := *serverconfig.DefaultConfig()
			server.API.Address = srv.URL
			helper.SetTestConfig(helper.CustomAppConfig{Config: server, Custom: helper.GetDefaultHeimdallConfig()})
			prev := helper.Client
			helper.Client = &http.Client{}
			t.Cleanup(func() { helper.Client = prev })

			stored, err := ClerkRecordStored(context.Background(), cdc, 49992)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantStored, stored)
		})
	}
}
