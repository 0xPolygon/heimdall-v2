package app

import (
	"testing"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"
)

func TestDecideLastBlockRerun(t *testing.T) {
	const self = "v1@abc"

	tests := []struct {
		name        string
		appHeight   int64
		committedBy string
		pending     *PendingRerun
		wantRerun   bool
		wantTarget  int64
	}{
		{name: "fresh node", appHeight: 0, committedBy: "", wantRerun: false},
		{name: "only genesis height", appHeight: 1, committedBy: "", wantRerun: false},
		{name: "same binary", appHeight: 100, committedBy: self, wantRerun: false},
		{name: "binary without marker", appHeight: 100, committedBy: "", wantRerun: true, wantTarget: 99},
		{name: "different binary", appHeight: 100, committedBy: "v0@old", wantRerun: true, wantTarget: 99},
		{
			name: "pending re-run wins over a same-binary marker", appHeight: 100, committedBy: self,
			pending: &PendingRerun{TargetHeight: 99, Attempts: 1}, wantRerun: true, wantTarget: 99,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := DecideLastBlockRerun(tc.appHeight, tc.committedBy, self, tc.pending)
			require.Equal(t, tc.wantRerun, d.Rerun, d.Reason)
			if tc.wantRerun {
				require.Equal(t, tc.wantTarget, d.TargetHeight)
			}
		})
	}
}

func TestPendingRerunRoundTrip(t *testing.T) {
	db := dbm.NewMemDB()

	_, ok, err := ReadPendingRerun(db)
	require.NoError(t, err)
	require.False(t, ok)

	want := PendingRerun{TargetHeight: 54627194, Attempts: 2}
	require.NoError(t, WritePendingRerun(db, want))

	got, ok, err := ReadPendingRerun(db)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, want, got)

	_, err = decodePendingRerun([]byte{1, 2, 3})
	require.Error(t, err)
}

func TestRecordCommitByThisBinary(t *testing.T) {
	db := dbm.NewMemDB()
	require.NoError(t, WritePendingRerun(db, PendingRerun{TargetHeight: 9, Attempts: 1}))

	hApp := &HeimdallApp{rerunDB: db}
	hApp.recordCommitByThisBinaryWith(log.NewNopLogger(), 10)

	committedBy, err := ReadLastCommitBinary(db)
	require.NoError(t, err)
	require.Equal(t, BinaryIdentity(), committedBy)

	_, ok, err := ReadPendingRerun(db)
	require.NoError(t, err)
	require.False(t, ok, "a successful commit must clear the pending re-run")
	require.True(t, hApp.rerunMarkerWritten)
}
