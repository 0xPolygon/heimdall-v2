package abci

import "testing"

func TestIsFastForwardMilestone(t *testing.T) {
	tests := []struct {
		name                    string
		latestHeaderNumber      uint64
		latestMilestoneEndBlock uint64
		ffMilestoneThreshold    uint64
		expected                bool
	}{
		{
			name:                    "Header equals milestone block",
			latestHeaderNumber:      100,
			latestMilestoneEndBlock: 100,
			ffMilestoneThreshold:    0,
			expected:                false,
		},
		{
			name:                    "Header less than milestone block",
			latestHeaderNumber:      90,
			latestMilestoneEndBlock: 100,
			ffMilestoneThreshold:    0,
			expected:                false,
		},
		{
			name:                    "Difference equals threshold",
			latestHeaderNumber:      105,
			latestMilestoneEndBlock: 100,
			ffMilestoneThreshold:    5,
			expected:                false, // because 105-100 == 5 (not greater than 5)
		},
		{
			name:                    "Difference less than threshold",
			latestHeaderNumber:      110,
			latestMilestoneEndBlock: 100,
			ffMilestoneThreshold:    15,
			expected:                false,
		},
		{
			name:                    "Difference greater than threshold",
			latestHeaderNumber:      110,
			latestMilestoneEndBlock: 100,
			ffMilestoneThreshold:    5,
			expected:                true,
		},
		{
			name:                    "Threshold zero, header greater than milestone",
			latestHeaderNumber:      101,
			latestMilestoneEndBlock: 100,
			ffMilestoneThreshold:    0,
			expected:                true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isFastForwardMilestone(tc.latestHeaderNumber, tc.latestMilestoneEndBlock, tc.ffMilestoneThreshold)
			if result != tc.expected {
				t.Errorf("isFastForwardMilestone(%d, %d, %d) = %v; expected %v",
					tc.latestHeaderNumber, tc.latestMilestoneEndBlock, tc.ffMilestoneThreshold, result, tc.expected)
			}
		})
	}
}

func TestGetFastForwardMilestoneStartBlock(t *testing.T) {
	tests := []struct {
		name                     string
		latestHeaderNumber       uint64
		latestMilestoneEndBlock  uint64
		ffMilestoneBlockInterval uint64
		expected                 uint64
	}{
		{
			name:                     "Exact multiple",
			latestHeaderNumber:       150,
			latestMilestoneEndBlock:  100,
			ffMilestoneBlockInterval: 10,
			expected:                 151, // (150-100)/10=5*10=50, then 100+50+1 = 151
		},
		{
			name:                     "Not an exact multiple",
			latestHeaderNumber:       153,
			latestMilestoneEndBlock:  100,
			ffMilestoneBlockInterval: 10,
			expected:                 151, // (153-100)=53/10=5*10=50, then 100+50+1 = 151
		},
		{
			name:                     "Zero difference",
			latestHeaderNumber:       100,
			latestMilestoneEndBlock:  100,
			ffMilestoneBlockInterval: 10,
			expected:                 101, // 0/10=0, then 100+0+1 = 101
		},
		{
			name:                     "Interval equals 1",
			latestHeaderNumber:       150,
			latestMilestoneEndBlock:  100,
			ffMilestoneBlockInterval: 1,
			expected:                 151, // every block counts; 150-100=50, then 100+50+1 = 151
		},
		{
			name:                     "Interval larger than difference",
			latestHeaderNumber:       105,
			latestMilestoneEndBlock:  100,
			ffMilestoneBlockInterval: 10,
			expected:                 101, // (5/10=0) then 100+0+1 = 101
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getFastForwardMilestoneStartBlock(tc.latestHeaderNumber, tc.latestMilestoneEndBlock, tc.ffMilestoneBlockInterval)
			if result != tc.expected {
				t.Errorf("getFastForwardMilestoneStartBlock(%d, %d, %d) = %d; expected %d",
					tc.latestHeaderNumber, tc.latestMilestoneEndBlock, tc.ffMilestoneBlockInterval, result, tc.expected)
			}
		})
	}
}
