package apierrors

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Error string `json:"error"`
}

func Write(writer http.ResponseWriter, statusCode int, message string) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(Response{Error: message})
}
