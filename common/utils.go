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
	pyl := map[string]any{
		"error":       err.Error(),
		"status_code": status,
		"timestamp":   ts,
	}
	if appErr, ok := err.(*AppError); ok {
		pyl["error"] = appErr.Message
		pyl["status_code"] = appErr.Code
	}
	json.NewEncoder(w).Encode(pyl)
}
