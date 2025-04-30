package types

import (
	"testing"
)

func TestCalcCurrentBorSpanId(t *testing.T) {
	tests := []struct {
		name           string
		latestBorBlock uint64
		span           *Span
		want           uint64
		wantErr        bool
	}{
		{
			name:           "Nil span pointer",
			latestBorBlock: 100,
			span:           nil,
			want:           0,
			wantErr:        true,
		},
		{
			name:           "Zero-length span",
			latestBorBlock: 100,
			span:           &Span{Id: 1, StartBlock: 100, EndBlock: 100},
			want:           0,
			wantErr:        true,
		},
		{
			name:           "Underflow latestBorBlock < start",
			latestBorBlock: 50,
			span:           &Span{Id: 1, StartBlock: 100, EndBlock: 110},
			want:           0,
			wantErr:        true,
		},
		{
			name:           "At start block",
			latestBorBlock: 100,
			span:           &Span{Id: 1, StartBlock: 100, EndBlock: 200},
			want:           1,
			wantErr:        false,
		},
		{
			name:           "Within span",
			latestBorBlock: 150,
			span:           &Span{Id: 1, StartBlock: 100, EndBlock: 200},
			want:           1,
			wantErr:        false,
		},
		{
			name:           "At end block",
			latestBorBlock: 200,
			span:           &Span{Id: 1, StartBlock: 100, EndBlock: 200},
			want:           1,
			wantErr:        false,
		},
		{
			name:           "Just after end block",
			latestBorBlock: 201,
			span:           &Span{Id: 1, StartBlock: 100, EndBlock: 200},
			want:           3,
			wantErr:        false,
		},
		{
			name:           "Multiple spans exact",
			latestBorBlock: 500,
			span:           &Span{Id: 10, StartBlock: 100, EndBlock: 200},
			want:           14, // (500-100)/100 = 4
			wantErr:        false,
		},
		{
			name:           "Multiple spans with remainder",
			latestBorBlock: 550,
			span:           &Span{Id: 10, StartBlock: 100, EndBlock: 200},
			want:           15, // (550-100)/100 = 4 + 1 remainder
			wantErr:        false,
		},
		{
			name:           "Span length one multiple",
			latestBorBlock: 105,
			span:           &Span{Id: 5, StartBlock: 100, EndBlock: 101},
			want:           10, // (105-100)/1 = 5
			wantErr:        false,
		},
		{
			name:           "Max values within range",
			latestBorBlock: ^uint64(0),
			span:           &Span{Id: 0, StartBlock: 0, EndBlock: ^uint64(0)},
			want:           0,
			wantErr:        false,
		},
		{
			name:           "Overflow wrap-around",
			latestBorBlock: 3,
			span:           &Span{Id: ^uint64(0) - 2, StartBlock: 0, EndBlock: 1},
			want:           0,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalcCurrentBorSpanId(tt.latestBorBlock, tt.span)
			if tt.wantErr {
				if err == nil {
					t.Errorf("%q: expected error, got none", tt.name)
				}
				return
			}
			if err != nil {
				t.Errorf("%q: unexpected error: %v", tt.name, err)
				return
			}
			if got != tt.want {
				t.Errorf("%q: got %d, want %d", tt.name, got, tt.want)
			}
		})
	}
}
