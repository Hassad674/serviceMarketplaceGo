package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSON_SetsContentTypeHeader(t *testing.T) {
	w := httptest.NewRecorder()

	JSON(w, http.StatusOK, map[string]string{"key": "value"})

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestJSON_WritesCorrectStatusCode(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"400 Bad Request", http.StatusBadRequest},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			JSON(w, tt.status, map[string]string{"status": "test"})

			assert.Equal(t, tt.status, w.Code)
		})
	}
}

func TestJSON_WritesValidJSONBody(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]any{
		"name":  "John",
		"age":   float64(30),
		"admin": false,
	}

	JSON(w, http.StatusOK, data)

	var result map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "John", result["name"])
	assert.Equal(t, float64(30), result["age"])
	assert.Equal(t, false, result["admin"])
}

func TestJSON_WritesNullForNilData(t *testing.T) {
	w := httptest.NewRecorder()

	JSON(w, http.StatusOK, nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "null\n", w.Body.String())
}

func TestError_WritesCorrectFormat(t *testing.T) {
	w := httptest.NewRecorder()

	Error(w, http.StatusNotFound, "user_not_found", "user not found")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "user_not_found", result["error"])
	assert.Equal(t, "user not found", result["message"])
}

func TestError_DifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		errCode string
		message string
	}{
		{"bad request", http.StatusBadRequest, "validation_error", "email is required"},
		{"conflict", http.StatusConflict, "email_already_exists", "email already exists"},
		{"forbidden", http.StatusForbidden, "forbidden", "insufficient permissions"},
		{"internal error", http.StatusInternalServerError, "internal_error", "internal server error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			Error(w, tt.status, tt.errCode, tt.message)

			assert.Equal(t, tt.status, w.Code)

			var result map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &result)
			require.NoError(t, err)
			assert.Equal(t, tt.errCode, result["error"])
			assert.Equal(t, tt.message, result["message"])
		})
	}
}

func TestValidationError_WritesCorrectFormat(t *testing.T) {
	w := httptest.NewRecorder()
	details := map[string]string{
		"email":    "email is required",
		"password": "password is too short",
	}

	ValidationError(w, details)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "validation_error", result["error"])
	assert.Equal(t, "one or more fields are invalid", result["message"])

	detailsResult, ok := result["details"].(map[string]any)
	require.True(t, ok, "details should be a map")
	assert.Equal(t, "email is required", detailsResult["email"])
	assert.Equal(t, "password is too short", detailsResult["password"])
}

func TestValidationError_EmptyDetails(t *testing.T) {
	w := httptest.NewRecorder()

	ValidationError(w, map[string]string{})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

	var result map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "validation_error", result["error"])
}

func TestNoContent_WritesCorrectStatusCode(t *testing.T) {
	w := httptest.NewRecorder()

	NoContent(w)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
}
