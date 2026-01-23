package app_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xPolygon/heimdall-v2/app"
	"github.com/stretchr/testify/require"
)

func TestHealthStatusLevel_String(t *testing.T) {
	tests := []struct {
		name     string
		level    app.HealthStatusLevel
		expected string
	}{
		{
			name:     "StatusOK",
			level:    app.StatusOK,
			expected: "OK",
		},
		{
			name:     "StatusWarn",
			level:    app.StatusWarn,
			expected: "WARN",
		},
		{
			name:     "StatusCritical",
			level:    app.StatusCritical,
			expected: "CRITICAL",
		},
		{
			name:     "Unknown status",
			level:    app.HealthStatusLevel(99),
			expected: "UNKNOWN",
		},
		{
			name:     "Negative status",
			level:    app.HealthStatusLevel(-1),
			expected: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.level.String()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestHealthStatusLevel_Code(t *testing.T) {
	tests := []struct {
		name     string
		level    app.HealthStatusLevel
		expected int
	}{
		{
			name:     "StatusOK",
			level:    app.StatusOK,
			expected: 0,
		},
		{
			name:     "StatusWarn",
			level:    app.StatusWarn,
			expected: 1,
		},
		{
			name:     "StatusCritical",
			level:    app.StatusCritical,
			expected: 2,
		},
		{
			name:     "Unknown status",
			level:    app.HealthStatusLevel(99),
			expected: -1,
		},
		{
			name:     "Negative status",
			level:    app.HealthStatusLevel(-1),
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.level.Code()
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestHealthStatusLevel_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		level    app.HealthStatusLevel
		expected string
	}{
		{
			name:     "StatusOK",
			level:    app.StatusOK,
			expected: `"OK"`,
		},
		{
			name:     "StatusWarn",
			level:    app.StatusWarn,
			expected: `"WARN"`,
		},
		{
			name:     "StatusCritical",
			level:    app.StatusCritical,
			expected: `"CRITICAL"`,
		},
		{
			name:     "Unknown status",
			level:    app.HealthStatusLevel(99),
			expected: `"UNKNOWN"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.level.MarshalJSON()
			require.NoError(t, err)
			require.Equal(t, tt.expected, string(data))
		})
	}
}

func TestHealthStatus(t *testing.T) {
	status := app.HealthStatus{
		Level:   app.StatusOK,
		Code:    200,
		Message: "Service is healthy",
	}

	require.Equal(t, app.StatusOK, status.Level)
	require.Equal(t, 200, status.Code)
	require.Equal(t, "Service is healthy", status.Message)
}

func TestHealthStatusJSON(t *testing.T) {
	status := app.HealthStatus{
		Level:   app.StatusWarn,
		Code:    503,
		Message: "Service degraded",
	}

	// Marshal to JSON
	data, err := json.Marshal(status)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Verify JSON contains expected fields
	require.Contains(t, string(data), `"level":"WARN"`)
	require.Contains(t, string(data), `"code":503`)
	require.Contains(t, string(data), `"message":"Service degraded"`)

	// Unmarshal from JSON
	var decoded app.HealthStatus
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Equal(t, status.Code, decoded.Code)
	require.Equal(t, status.Message, decoded.Message)
}

func TestResponseRecorder_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	recorder := &app.ResponseRecorder{
		ResponseWriter: w,
	}

	recorder.WriteHeader(http.StatusOK)
	// statusCode is a private field, but we can test that the method doesn't panic
	require.NotPanics(t, func() {
		recorder.WriteHeader(http.StatusOK)
	})
}

func TestResponseRecorder_Write(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "write simple data",
			data: []byte("test data"),
		},
		{
			name: "write JSON data",
			data: []byte(`{"key": "value"}`),
		},
		{
			name: "write empty data",
			data: []byte{},
		},
		{
			name: "write binary data",
			data: []byte{0x00, 0x01, 0x02, 0x03},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			recorder := &app.ResponseRecorder{
				ResponseWriter: w,
			}

			n, err := recorder.Write(tt.data)
			require.NoError(t, err)
			require.Equal(t, len(tt.data), n)
		})
	}
}

func TestResponseRecorder_MultipleWrites(t *testing.T) {
	w := httptest.NewRecorder()
	recorder := &app.ResponseRecorder{
		ResponseWriter: w,
	}

	// Write multiple times
	n1, err1 := recorder.Write([]byte("first "))
	require.NoError(t, err1)
	require.Equal(t, 6, n1)

	n2, err2 := recorder.Write([]byte("second "))
	require.NoError(t, err2)
	require.Equal(t, 7, n2)

	n3, err3 := recorder.Write([]byte("third"))
	require.NoError(t, err3)
	require.Equal(t, 5, n3)
}

func TestHealthStatusConstants(t *testing.T) {
	// Verify the constants are defined and have expected values
	require.Equal(t, app.HealthStatusLevel(0), app.StatusOK)
	require.Equal(t, app.HealthStatusLevel(1), app.StatusWarn)
	require.Equal(t, app.HealthStatusLevel(2), app.StatusCritical)

	// Verify they're distinct
	require.NotEqual(t, app.StatusOK, app.StatusWarn)
	require.NotEqual(t, app.StatusWarn, app.StatusCritical)
	require.NotEqual(t, app.StatusOK, app.StatusCritical)
}

func TestHealthStatusLevelZeroValue(t *testing.T) {
	var level app.HealthStatusLevel
	require.Equal(t, app.StatusOK, level)
	require.Equal(t, "OK", level.String())
	require.Equal(t, 0, level.Code())
}

func TestHealthStatusZeroValue(t *testing.T) {
	var status app.HealthStatus
	require.Equal(t, app.StatusOK, status.Level)
	require.Equal(t, 0, status.Code)
	require.Equal(t, "", status.Message)
}

func TestResponseRecorderZeroValue(t *testing.T) {
	var recorder app.ResponseRecorder
	require.Nil(t, recorder.ResponseWriter)

	n, err := recorder.Write([]byte("test"))
	require.NoError(t, err)
	require.Equal(t, 4, n)
}
