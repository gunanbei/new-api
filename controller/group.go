package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func GetGroups(c *gin.Context) {
	groupNames := make([]string, 0)
	for groupName := range ratio_setting.GetGroupRatioCopy() {
		groupNames = append(groupNames, groupName)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    groupNames,
	})
}

func GetUserGroups(c *gin.Context) {
	usableGroups := make(map[string]map[string]interface{})
	crossGroupRetryStatusCodes := make(map[string]string)
	userId := c.GetInt("id")
	if userId > 0 {
		common.OptionMapRWMutex.RLock()
		for _, rule := range operation_setting.GetRoutingReliabilitySetting().CrossGroupRetryRules {
			if rule.HTTPStatusCodes != "" {
				crossGroupRetryStatusCodes[rule.Group] = rule.HTTPStatusCodes
			}
		}
		common.OptionMapRWMutex.RUnlock()
	}
	userGroup := ""
	userGroup, _ = model.GetUserGroup(userId, false)
	userUsableGroups := service.GetUserUsableGroups(userGroup)
	for groupName, _ := range ratio_setting.GetGroupRatioCopy() {
		// UserUsableGroups contains the groups that the user can use
		if desc, ok := userUsableGroups[groupName]; ok {
			groupInfo := map[string]interface{}{
				"ratio": service.GetUserGroupRatio(userGroup, groupName),
				"desc":  desc,
			}
			if statusCodes := crossGroupRetryStatusCodes[groupName]; statusCodes != "" {
				groupInfo["cross_group_retry_status_codes"] = statusCodes
			}
			usableGroups[groupName] = groupInfo
		}
	}
	if _, ok := userUsableGroups["auto"]; ok {
		usableGroups["auto"] = map[string]interface{}{
			"ratio": "自动",
			"desc":  setting.GetUsableGroupDescription("auto"),
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    usableGroups,
	})
}
