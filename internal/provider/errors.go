package provider

import "fmt"

// ProviderError is a typed upstream error carrying the HTTP status and an
// OpenAI-compatible error code so clients receive consistent error shapes.
type ProviderError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("upstream %d [%s]: %s", e.StatusCode, e.Code, e.Message)
}

// ClientStatus returns the HTTP status Veil should return to the client.
// Upstream 5xx become 502 Bad Gateway; auth/rate errors are forwarded as-is.
func (e *ProviderError) ClientStatus() int {
	switch e.StatusCode {
	case 401, 403, 413, 429:
		return e.StatusCode
	}
	if e.StatusCode >= 500 {
		return 502
	}
	return 400
}

func newProviderError(statusCode int, body string) *ProviderError {
	return &ProviderError{
		StatusCode: statusCode,
		Code:       errorCode(statusCode),
		Message:    body,
	}
}

func errorCode(status int) string {
	switch status {
	case 401:
		return "invalid_api_key"
	case 403:
		return "permission_denied"
	case 429:
		return "rate_limit_exceeded"
	case 504:
		return "provider_timeout"
	}
	if status >= 500 {
		return "provider_error"
	}
	return "invalid_request_error"
}
