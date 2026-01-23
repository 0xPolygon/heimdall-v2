package app

import "net/http"

// HealthStatus represents the health status with level, code, and message
type HealthStatus struct {
	Level   HealthStatusLevel `json:"level"`
	Code    int               `json:"code"`
	Message string            `json:"message"`
}

// HealthStatusLevel represents the health status level.
type HealthStatusLevel int

const (
	StatusOK HealthStatusLevel = iota
	StatusWarn
	StatusCritical
)

// String returns the string representation of the health status level.
func (h HealthStatusLevel) String() string {
	switch h {
	case StatusOK:
		return "OK"
	case StatusWarn:
		return "WARN"
	case StatusCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Code returns the numeric code for the health status level.
func (h HealthStatusLevel) Code() int {
	switch h {
	case StatusOK:
		return 0
	case StatusWarn:
		return 1
	case StatusCritical:
		return 2
	default:
		return -1
	}
}

// MarshalJSON implements json.Marshaler interface to return the string representation of the health status level.
func (h HealthStatusLevel) MarshalJSON() ([]byte, error) {
	return []byte(`"` + h.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler interface to parse the string representation back to the health status level.
func (h *HealthStatusLevel) UnmarshalJSON(data []byte) error {
	// Remove quotes from the JSON string
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	switch str {
	case "OK":
		*h = StatusOK
	case "WARN":
		*h = StatusWarn
	case "CRITICAL":
		*h = StatusCritical
	default:
		*h = StatusOK // Default to OK for unknown values
	}

	return nil
}

// ResponseRecorder captures the response from the health-go handler.
type ResponseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (r *ResponseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func (r *ResponseRecorder) Write(data []byte) (int, error) {
	r.body = append(r.body, data...)
	return len(data), nil
}
