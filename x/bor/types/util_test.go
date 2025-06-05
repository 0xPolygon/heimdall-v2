package types

import (
	math "math"
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
			name:           "Valid span length 16, at start block",
			latestBorBlock: 100,
			span:           &Span{Id: 1, StartBlock: 100, EndBlock: 115}, // len = 16
			want:           1,
			wantErr:        false,
		},
		{
			name:           "Valid span length 16, at end block",
			latestBorBlock: 115,
			span:           &Span{Id: 1, StartBlock: 100, EndBlock: 115},
			want:           1,
			wantErr:        false,
		},
		{
			name:           "Valid span length 16, just after end block",
			latestBorBlock: 116,
			span:           &Span{Id: 1, StartBlock: 100, EndBlock: 115},
			want:           2,
			wantErr:        false,
		},
		{
			name:           "Multiple spans exact division",
			latestBorBlock: 196,
			span:           &Span{Id: 5, StartBlock: 100, EndBlock: 115}, // len = 16, (196-100)/16 = 6
			want:           11,
			wantErr:        false,
		},
		{
			name:           "Multiple spans with remainder",
			latestBorBlock: 198,
			span:           &Span{Id: 5, StartBlock: 100, EndBlock: 115}, // len = 16, (198-100)/16 = 6.125
			want:           11,
			wantErr:        false,
		},
		{
			name:           "Underflow latestBorBlock < start",
			latestBorBlock: 50,
			span:           &Span{Id: 1, StartBlock: 100, EndBlock: 115},
			want:           0,
			wantErr:        true,
		},
		{
			name:           "Single span length 16 with large Id",
			latestBorBlock: 115,
			span:           &Span{Id: 123456, StartBlock: 100, EndBlock: 115},
			want:           123456,
			wantErr:        false,
		},
		{
			name:           "Overflow when computing span ID",
			latestBorBlock: 100,                                                       // offset = 0
			span:           &Span{Id: math.MaxUint64, StartBlock: 100, EndBlock: 115}, // spanId = MaxUint64
			want:           math.MaxUint64,
			wantErr:        false,
		},
		{
			name:           "Overflow wrap-around with actual overflow",
			latestBorBlock: 132,                                                           // offset = 32 -> 2 spans, spanId = MaxUint64 - 1 + 2 = wrap
			span:           &Span{Id: math.MaxUint64 - 1, StartBlock: 100, EndBlock: 115}, // len = 16
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
