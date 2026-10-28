// Package dto holds the request and response shapes of the HTTP API.
package dto

// ErrorResponse is the body of every error.
type ErrorResponse struct {
	Error     string `json:"error" example:"we couldn't find what you were looking for"`
	RequestID string `json:"request_id,omitempty" example:"9f8c2a41d3b7e605"`
}

// HealthResponse is returned by the health check.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}

// PingResponse is returned by the API ping.
type PingResponse struct {
	Message string `json:"message" example:"pong"`
}
