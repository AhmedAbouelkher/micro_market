package common

import (
	"encoding/json"
	"net/http"
	"os"
	"time"
)

func BoolPtr(b bool) *bool       { return &b }
func IntPtr(i int) *int          { return &i }
func Int32Ptr(i int32) *int32    { return &i }
func StringPtr(s string) *string { return &s }

func EnvOrDef(env string, defaultValue string) string {
	if value := os.Getenv(env); value != "" {
		return value
	}
	return defaultValue
}

func SendJsonError(w http.ResponseWriter, status int, err error) {
	ts := time.Now().Format(time.RFC3339)
	statusCode := status
	pyl := map[string]any{
		"error":     err.Error(),
		"timestamp": ts,
	}
	if appErr, ok := err.(*AppError); ok {
		statusCode = appErr.Code
		pyl["error"] = appErr.Message
	}
	pyl["status_code"] = statusCode
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(pyl)
}
