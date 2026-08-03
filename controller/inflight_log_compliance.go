package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

type InflightLogComplianceRequest struct {
	Confirmed bool `json:"confirmed"`
}

func ConfirmInflightLogCompliance(c *gin.Context) {
	if c.GetBool("use_access_token") {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "This operation requires dashboard session authentication. API access token is not allowed.",
		})
		return
	}

	var req InflightLogComplianceRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if !req.Confirmed {
		common.ApiErrorMsg(c, "请确认合规声明")
		return
	}

	now := time.Now().Unix()
	userID := c.GetInt("id")
	clientIP := c.ClientIP()
	if err := model.UpdateOptionsBulk(map[string]string{
		"InflightTaskTraceEnabled":                      "true",
		"inflight_log_setting.compliance_confirmed":     "true",
		"inflight_log_setting.compliance_terms_version": operation_setting.CurrentInflightLogComplianceTermsVersion,
		"inflight_log_setting.compliance_confirmed_at":  strconv.FormatInt(now, 10),
		"inflight_log_setting.compliance_confirmed_by":  strconv.Itoa(userID),
		"inflight_log_setting.compliance_confirmed_ip":  clientIP,
	}); err != nil {
		common.ApiError(c, err)
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf(
		"inflight log compliance confirmed user_id=%d ip=%s terms_version=%s confirmed_at=%d",
		userID,
		clientIP,
		operation_setting.CurrentInflightLogComplianceTermsVersion,
		now,
	))

	common.ApiSuccess(c, gin.H{
		"confirmed":     true,
		"terms_version": operation_setting.CurrentInflightLogComplianceTermsVersion,
		"confirmed_at":  now,
		"confirmed_by":  userID,
	})
}
