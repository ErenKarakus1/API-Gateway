package response

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func Error(c *gin.Context, status int, code string, message string) {
	requestID, _ := c.Get("request_id")
	c.AbortWithStatusJSON(status, ErrorBody{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			RequestID: requestIDString(requestID),
		},
	})
}

func WriteError(w http.ResponseWriter, requestID string, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorBody{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			RequestID: requestID,
		},
	})
}

func requestIDString(value interface{}) string {
	requestID, ok := value.(string)
	if !ok {
		return ""
	}
	return requestID
}
