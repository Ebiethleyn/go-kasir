package response

import (
	"encoding/json"
	"net/http"
)

type Body struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func JSON(w http.ResponseWriter, status int, success bool, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Body{
		Success: success,
		Message: message,
		Data:    data,
	})
}

func OK(w http.ResponseWriter, message string, data interface{}) {
	JSON(w, http.StatusOK, true, message, data)
}

func Created(w http.ResponseWriter, message string, data interface{}) {
	JSON(w, http.StatusCreated, true, message, data)
}

func BadRequest(w http.ResponseWriter, message string) {
	JSON(w, http.StatusBadRequest, false, message, nil)
}

func Unauthorized(w http.ResponseWriter, message string) {
	JSON(w, http.StatusUnauthorized, false, message, nil)
}

func Forbidden(w http.ResponseWriter, message string) {
	JSON(w, http.StatusForbidden, false, message, nil)
}

func NotFound(w http.ResponseWriter, message string) {
	JSON(w, http.StatusNotFound, false, message, nil)
}

func Internal(w http.ResponseWriter, message string) {
	JSON(w, http.StatusInternalServerError, false, message, nil)
}
