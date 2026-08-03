package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type tokenAutoGroupsInput struct {
	Set    bool
	Groups []string
}

func (v *tokenAutoGroupsInput) UnmarshalJSON(data []byte) error {
	v.Set = true
	if strings.TrimSpace(string(data)) == "null" {
		return nil
	}
	return common.Unmarshal(data, &v.Groups)
}

type tokenRequest struct {
	model.Token
	AutoGroups tokenAutoGroupsInput `json:"auto_groups"`
}

type tokenResponse struct {
	*model.Token
	AutoGroups []string `json:"auto_groups"`
}

func buildMaskedTokenResponse(token *model.Token) *tokenResponse {
	if token == nil {
		return nil
	}
	masked := *token
	masked.Key = token.GetMaskedKey()
	groups, err := token.GetAutoGroups()
	if err != nil {
		common.SysError(fmt.Sprintf("failed to parse auto groups for token %d: %v", token.Id, err))
		groups = nil
	}
	return &tokenResponse{Token: &masked, AutoGroups: groups}
}

func setTokenAutoGroups(c *gin.Context, token *model.Token, groups []string) bool {
	if len(groups) == 0 {
		_ = token.SetAutoGroups(nil)
		return true
	}
	maxGroups := setting.GetMaxTokenAutoGroups()
	if len(groups) > maxGroups {
		common.ApiErrorI18n(c, i18n.MsgTokenAutoGroupsTooMany, map[string]any{"Max": maxGroups})
		return false
	}
	userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	if userGroup == "" {
		userGroup = c.GetString("group")
	}
	if userGroup == "" {
		var err error
		userGroup, err = model.GetUserGroup(c.GetInt("id"), false)
		if err != nil {
			common.ApiError(c, err)
			return false
		}
	}
	seen := map[string]struct{}{}
	for _, group := range groups {
		if _, ok := seen[group]; ok {
			common.ApiErrorI18n(c, i18n.MsgTokenAutoGroupsDuplicate, map[string]any{"Group": group})
			return false
		}
		seen[group] = struct{}{}
		if !service.IsUserSelectableGroup(userGroup, group) {
			common.ApiErrorI18n(c, i18n.MsgTokenAutoGroupsInvalid, map[string]any{"Group": group})
			return false
		}
	}
	if err := token.SetAutoGroups(groups); err != nil {
		common.ApiError(c, err)
		return false
	}
	return true
}

// normalizeTokenFailover validates the failover settings submitted by the user and
// rewrites them into their canonical stored form. Failover is mutually exclusive with
// the auto group cross-group retry, and its retry budget can never exceed the system
// wide RetryTimes, otherwise a single key could multiply upstream load at will.
// token.Group is deliberately left untouched so turning failover off restores it.
func normalizeTokenFailover(c *gin.Context, token *model.Token) error {
	if !token.FailoverEnabled {
		token.FailoverGroups = ""
		token.FailoverStrategy = ""
		token.FailoverMaxRetry = 0
		token.FailoverRules = ""
		return nil
	}
	if common.RetryTimes <= 0 {
		return errors.New("系统未开启重试，无法启用故障转移模式，请联系管理员配置最大重试次数")
	}
	if !constant.IsValidFailoverStrategy(token.FailoverStrategy) {
		return errors.New("故障转移策略无效")
	}
	if token.FailoverMaxRetry < 1 || token.FailoverMaxRetry > common.RetryTimes {
		return fmt.Errorf("故障转移的最大重试次数必须在 1 到 %d 之间", common.RetryTimes)
	}
	if token.FailoverRules != "" {
		rules := types.DefaultTokenFailoverRules()
		if err := common.UnmarshalJsonStr(token.FailoverRules, &rules); err != nil {
			return errors.New("故障转移触发规则格式错误")
		}
		for _, timeout := range []struct {
			name  string
			value *int
		}{
			{name: "响应头超时", value: &rules.ResponseHeaderTimeoutMS},
			{name: "首个有效内容超时", value: &rules.FirstContentTimeoutMS},
			{name: "流式响应头超时", value: rules.StreamResponseHeaderTimeoutMS},
			{name: "流式首个有效内容超时", value: rules.StreamFirstContentTimeoutMS},
			{name: "非流式响应头超时", value: rules.NonStreamResponseHeaderTimeoutMS},
			{name: "非流式首个有效内容超时", value: rules.NonStreamFirstContentTimeoutMS},
		} {
			if timeout.value == nil {
				continue
			}
			if *timeout.value < 0 {
				return fmt.Errorf("%s不能小于 0 毫秒", timeout.name)
			}
			if common.RelayTimeout > 0 &&
				(*timeout.value/1000 > common.RelayTimeout ||
					(*timeout.value/1000 == common.RelayTimeout && *timeout.value%1000 > 0)) {
				return fmt.Errorf("%s不能超过全局超时 %d 秒", timeout.name, common.RelayTimeout)
			}
		}
		if _, err := operation_setting.ParseHTTPStatusCodeRanges(rules.HTTPStatusCodes); err != nil {
			return errors.New("HTTP 状态码触发规则无效")
		}
		rules.HTTPStatusCodes = strings.TrimSpace(rules.HTTPStatusCodes)
		encoded, err := common.Marshal(rules)
		if err != nil {
			return errors.New("故障转移触发规则序列化失败")
		}
		token.FailoverRules = string(encoded)
	}

	var groups []string
	if token.FailoverGroups != "" {
		if err := common.UnmarshalJsonStr(token.FailoverGroups, &groups); err != nil {
			return errors.New("故障转移分组格式错误")
		}
	}

	userGroup, err := model.GetUserGroup(c.GetInt("id"), false)
	if err != nil {
		return err
	}
	usableGroups := service.GetUserUsableGroups(userGroup)
	seen := make(map[string]struct{}, len(groups))
	cleaned := make([]string, 0, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" || group == "auto" {
			return errors.New("故障转移分组不能为空，且不能选择 auto 分组")
		}
		if _, duplicate := seen[group]; duplicate {
			continue
		}
		if _, ok := usableGroups[group]; !ok {
			return fmt.Errorf("无权访问 %s 分组", group)
		}
		seen[group] = struct{}{}
		cleaned = append(cleaned, group)
	}
	if len(cleaned) < constant.MinFailoverGroups || len(cleaned) > constant.MaxFailoverGroups {
		return fmt.Errorf("故障转移分组数量必须在 %d 到 %d 之间", constant.MinFailoverGroups, constant.MaxFailoverGroups)
	}

	encoded, err := common.Marshal(cleaned)
	if err != nil {
		return errors.New("故障转移分组序列化失败")
	}
	token.FailoverGroups = string(encoded)
	token.CrossGroupRetry = false
	return nil
}

func buildMaskedTokenResponses(tokens []*model.Token) []*tokenResponse {
	maskedTokens := make([]*tokenResponse, 0, len(tokens))
	for _, token := range tokens {
		maskedTokens = append(maskedTokens, buildMaskedTokenResponse(token))
	}
	return maskedTokens
}

func GetAllTokens(c *gin.Context) {
	userId := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	tokens, err := model.GetAllUserTokens(userId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	total, _ := model.CountUserTokens(userId)
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(buildMaskedTokenResponses(tokens))
	common.ApiSuccess(c, pageInfo)
}

func SearchTokens(c *gin.Context) {
	userId := c.GetInt("id")
	keyword := c.Query("keyword")
	token := c.Query("token")

	pageInfo := common.GetPageQuery(c)

	tokens, total, err := model.SearchUserTokens(userId, keyword, token, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(buildMaskedTokenResponses(tokens))
	common.ApiSuccess(c, pageInfo)
}

func GetToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	token, err := model.GetTokenByIds(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, buildMaskedTokenResponse(token))
}

func GetTokenAutoGroups(c *gin.Context) {
	userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	if userGroup == "" {
		userGroup = c.GetString("group")
	}
	if userGroup == "" {
		var err error
		userGroup, err = model.GetUserGroup(c.GetInt("id"), false)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}
	common.ApiSuccess(c, gin.H{"groups": service.GetUserAutoGroup(userGroup), "max_count": setting.GetMaxTokenAutoGroups()})
}

func GetTokenKey(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	token, err := model.GetTokenByIds(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"key": token.GetFullKey(),
	})
}

func GetTokenStatus(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	userId := c.GetInt("id")
	token, err := model.GetTokenByIds(tokenId, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0
	}
	c.JSON(http.StatusOK, gin.H{
		"object":          "credit_summary",
		"total_granted":   token.RemainQuota,
		"total_used":      0, // not supported currently
		"total_available": token.RemainQuota,
		"expires_at":      expiredAt * 1000,
	})
}

func GetTokenUsage(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "No Authorization header",
		})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Invalid Bearer token",
		})
		return
	}
	tokenKey := parts[1]

	token, err := model.GetTokenByKey(strings.TrimPrefix(tokenKey, "sk-"), false)
	if err != nil {
		common.SysError("failed to get token by key: " + err.Error())
		common.ApiErrorI18n(c, i18n.MsgTokenGetInfoFailed)
		return
	}

	expiredAt := token.ExpiredTime
	if expiredAt == -1 {
		expiredAt = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    true,
		"message": "ok",
		"data": gin.H{
			"object":               "token_usage",
			"name":                 token.Name,
			"total_granted":        token.RemainQuota + token.UsedQuota,
			"total_used":           token.UsedQuota,
			"total_available":      token.RemainQuota,
			"unlimited_quota":      token.UnlimitedQuota,
			"model_limits":         token.GetModelLimitsMap(),
			"model_limits_enabled": token.ModelLimitsEnabled,
			"expires_at":           expiredAt,
		},
	})
}

func AddToken(c *gin.Context) {
	request := tokenRequest{}
	err := c.ShouldBindJSON(&request)
	token := request.Token
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(token.Name) > 50 {
		common.ApiErrorI18n(c, i18n.MsgTokenNameTooLong)
		return
	}
	// 非无限额度时，检查额度值是否超出有效范围
	if !token.UnlimitedQuota {
		if token.RemainQuota < 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaNegative)
			return
		}
		maxQuotaValue := int((1000000000 * common.QuotaPerUnit))
		if token.RemainQuota > maxQuotaValue {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaExceedMax, map[string]any{"Max": maxQuotaValue})
			return
		}
	}
	// 检查用户令牌数量是否已达上限
	maxTokens := operation_setting.GetMaxUserTokens()
	count, err := model.CountUserTokens(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if int(count) >= maxTokens {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("已达到最大令牌数量限制 (%d)", maxTokens),
		})
		return
	}
	if err := normalizeTokenFailover(c, &token); err != nil {
		common.ApiError(c, err)
		return
	}
	if token.Group == "auto" {
		if !setTokenAutoGroups(c, &token, request.AutoGroups.Groups) {
			return
		}
	} else {
		_ = token.SetAutoGroups(nil)
	}
	key, err := common.GenerateKey()
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgTokenGenerateFailed)
		common.SysLog("failed to generate token key: " + err.Error())
		return
	}
	cleanToken := model.Token{
		UserId:             c.GetInt("id"),
		Name:               token.Name,
		Key:                key,
		CreatedTime:        common.GetTimestamp(),
		AccessedTime:       common.GetTimestamp(),
		ExpiredTime:        token.ExpiredTime,
		RemainQuota:        token.RemainQuota,
		UnlimitedQuota:     token.UnlimitedQuota,
		ModelLimitsEnabled: token.ModelLimitsEnabled,
		ModelLimits:        token.ModelLimits,
		AllowIps:           token.AllowIps,
		Group:              token.Group,
		CrossGroupRetry:    token.CrossGroupRetry,
		AutoGroups:         token.AutoGroups,
		FailoverEnabled:    token.FailoverEnabled,
		FailoverGroups:     token.FailoverGroups,
		FailoverStrategy:   token.FailoverStrategy,
		FailoverMaxRetry:   token.FailoverMaxRetry,
		FailoverRules:      token.FailoverRules,
	}
	err = cleanToken.Insert()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func DeleteToken(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	err := model.DeleteTokenById(id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func UpdateToken(c *gin.Context) {
	userId := c.GetInt("id")
	statusOnly := c.Query("status_only")
	request := tokenRequest{}
	err := c.ShouldBindJSON(&request)
	token := request.Token
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(token.Name) > 50 {
		common.ApiErrorI18n(c, i18n.MsgTokenNameTooLong)
		return
	}
	if !token.UnlimitedQuota {
		if token.RemainQuota < 0 {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaNegative)
			return
		}
		maxQuotaValue := int((1000000000 * common.QuotaPerUnit))
		if token.RemainQuota > maxQuotaValue {
			common.ApiErrorI18n(c, i18n.MsgTokenQuotaExceedMax, map[string]any{"Max": maxQuotaValue})
			return
		}
	}
	cleanToken, err := model.GetTokenByIds(token.Id, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if token.Status == common.TokenStatusEnabled {
		if cleanToken.Status == common.TokenStatusExpired && cleanToken.ExpiredTime <= common.GetTimestamp() && cleanToken.ExpiredTime != -1 {
			common.ApiErrorI18n(c, i18n.MsgTokenExpiredCannotEnable)
			return
		}
		if cleanToken.Status == common.TokenStatusExhausted && cleanToken.RemainQuota <= 0 && !cleanToken.UnlimitedQuota {
			common.ApiErrorI18n(c, i18n.MsgTokenExhaustedCannotEable)
			return
		}
	}
	if statusOnly != "" {
		cleanToken.Status = token.Status
	} else {
		if err := normalizeTokenFailover(c, &token); err != nil {
			common.ApiError(c, err)
			return
		}
		// If you add more fields, please also update token.Update()
		cleanToken.Name = token.Name
		cleanToken.ExpiredTime = token.ExpiredTime
		cleanToken.RemainQuota = token.RemainQuota
		cleanToken.UnlimitedQuota = token.UnlimitedQuota
		cleanToken.ModelLimitsEnabled = token.ModelLimitsEnabled
		cleanToken.ModelLimits = token.ModelLimits
		cleanToken.AllowIps = token.AllowIps
		cleanToken.Group = token.Group
		cleanToken.CrossGroupRetry = token.CrossGroupRetry
		if token.Group != "auto" {
			cleanToken.CrossGroupRetry = false
			_ = cleanToken.SetAutoGroups(nil)
		} else if request.AutoGroups.Set && !setTokenAutoGroups(c, cleanToken, request.AutoGroups.Groups) {
			return
		}
		cleanToken.FailoverEnabled = token.FailoverEnabled
		cleanToken.FailoverGroups = token.FailoverGroups
		cleanToken.FailoverStrategy = token.FailoverStrategy
		cleanToken.FailoverMaxRetry = token.FailoverMaxRetry
		cleanToken.FailoverRules = token.FailoverRules
	}
	err = cleanToken.Update()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    buildMaskedTokenResponse(cleanToken),
	})
}

type TokenBatch struct {
	Ids []int `json:"ids"`
}

func DeleteTokenBatch(c *gin.Context) {
	tokenBatch := TokenBatch{}
	if err := c.ShouldBindJSON(&tokenBatch); err != nil || len(tokenBatch.Ids) == 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	userId := c.GetInt("id")
	count, err := model.BatchDeleteTokens(tokenBatch.Ids, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    count,
	})
}

func GetTokenKeysBatch(c *gin.Context) {
	tokenBatch := TokenBatch{}
	if err := c.ShouldBindJSON(&tokenBatch); err != nil || len(tokenBatch.Ids) == 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if len(tokenBatch.Ids) > 100 {
		common.ApiErrorI18n(c, i18n.MsgBatchTooMany, map[string]any{"Max": 100})
		return
	}
	userId := c.GetInt("id")
	tokens, err := model.GetTokenKeysByIds(tokenBatch.Ids, userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	keysMap := make(map[int]string)
	for _, t := range tokens {
		keysMap[t.Id] = t.GetFullKey()
	}
	common.ApiSuccess(c, gin.H{"keys": keysMap})
}
