package main

import (
	"fmt"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AppError struct {
	Code    int
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}

func NewAppError(code int, message string, args ...any) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(message, args...),
	}
}

func (e *AppError) GRPCStatus() *status.Status {
	return status.New(mapCodeToGRPCCode(e.Code), e.Message)
}

func mapCodeToGRPCCode(code int) codes.Code {
	switch code {
	case http.StatusUnprocessableEntity:
		return codes.InvalidArgument
	case http.StatusNotFound, http.StatusGone:
		return codes.NotFound
	case http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return codes.Internal
	default:
		return codes.Unknown
	}
}
