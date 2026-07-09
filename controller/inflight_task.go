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
	pageInfo.SetMeta(service.InflightTaskListMeta{
		TraceMenuVisible: service.InflightTaskTraceMenuVisible(),
	})
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": pageInfo})
}

func GetUserInflightTaskTrace(c *gin.Context) {
	requestID := c.Param("request_id")
	trace, err := service.GetInflightTaskTrace(c.Request.Context(), c.GetInt("id"), requestID)
	if err != nil {
		switch {
		case service.IsInflightTaskUnavailable(err):
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"message": err.Error(),
			})
		case service.IsInflightTaskTraceNotFound(err):
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"message": "Debug log not found",
			})
		case service.IsInflightTaskTraceForbidden(err):
			c.JSON(http.StatusForbidden, gin.H{
				"success": false,
				"message": err.Error(),
			})
		case service.IsInflightTaskTraceInProgress(err):
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"message": "Debug log is not available for in-progress requests",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    trace,
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
