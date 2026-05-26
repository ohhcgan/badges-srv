package dto

const (
	BAD_REQUEST              = "BAD_REQUEST"
	UNAUTHORIZED             = "UNAUTHORIZED"
	FORBIDDEN                = "FORBIDDEN"
	NOT_FOUND                = "NOT_FOUND"
	INTERNAL_SERVER_ERROR    = "INTERNAL_SERVER_ERROR"
	BAD_GATEWAY              = "BAD_GATEWAY"
	SERVICE_TEMP_UNAVAILABLE = "SERVICE_TEMP_UNAVAILABLE"
)

type ErrorResponse struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

type Response[T any] struct {
	Message string         `json:"message"`
	Success bool           `json:"success"`
	Data    *T             `json:"data,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
}
