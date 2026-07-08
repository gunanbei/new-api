package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetUserInflightTasks(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	var isStream *bool
	switch c.Query("is_stream") {
	case "true":
		v := true
		isStream = &v
	case "false":
		v := false
		isStream = &v
	}

	tasks, total, err := service.ListUserInflightTasks(c.Request.Context(), c.GetInt("id"), service.InflightTaskQuery{
		Status:         c.Query("status"),
		Kind:           c.Query("kind"),
		Channel:        c.Query("channel"),
		ModelName:      c.Query("model_name"),
		RequestID:      c.Query("request_id"),
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		IsStream:       isStream,
		StartIdx:       pageInfo.GetStartIdx(),
		Num:            pageInfo.GetPageSize(),
	})
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

	pageInfo.SetTotal(total)
	pageInfo.SetItems(tasks)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": pageInfo})
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
