package resp

import "github.com/gin-gonic/gin"

func OK(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{"code": 0, "message": "success", "data": data})
}

func Err(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"code": status, "message": msg, "data": nil})
}
