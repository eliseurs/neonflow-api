package api

import "github.com/gin-gonic/gin"

type HttpResponse struct {
	Status int `json:"status"`
	Data   any `json:"data"`
}

func Respond(c *gin.Context, status int, data any) {
	c.JSON(status, HttpResponse{Status: status, Data: data})
}

func RespondError(c *gin.Context, status int, message string) {
	Respond(c, status, gin.H{"error": message})
}
