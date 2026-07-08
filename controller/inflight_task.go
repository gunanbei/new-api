package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetUserInflightTasks(c *gin.Context) {
	tasks, err := service.ListUserInflightTasks(c.Request.Context(), c.GetInt("id"))
	if err != nil {
		if service.IsInflightTaskUnavailable(err) {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    tasks,
	})
}
func GetInflightTaskStats(c *gin.Context) {
	stats, err := service.GetInflightTaskStats(c.Request.Context())
	if err != nil {
		if service.IsInflightTaskUnavailable(err) {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    stats,
	})
}
