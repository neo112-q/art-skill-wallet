package response

import "github.com/gin-gonic/gin"

// APIResponse is the standard response envelope for all API endpoints.
type APIResponse struct {
	Status       string      `json:"status"`
	ErrorMessage *string     `json:"error_message"`
	Data         interface{} `json:"data"`
}

func Success(c *gin.Context, code int, data interface{}) {
	c.JSON(code, APIResponse{
		Status:       "success",
		ErrorMessage: nil,
		Data:         data,
	})
}

func Error(c *gin.Context, code int, msg string) {
	c.JSON(code, APIResponse{
		Status:       "error",
		ErrorMessage: &msg,
		Data:         nil,
	})
}
