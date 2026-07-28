package controller

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

func GetPerfMetricsSummary(c *gin.Context) {
	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}

	activeGroups := activePerfMetricGroups(ratio_setting.GetGroupRatioCopy())
	result, err := perfmetrics.QuerySummaryAll(hours, activeGroups)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func GetPerfMetrics(c *gin.Context) {
	modelName := c.Query("model")
	if modelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "model is required",
		})
		return
	}

	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}

	result, err := perfmetrics.Query(perfmetrics.QueryParams{
		Model: modelName,
		Group: c.Query("group"),
		Hours: hours,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	result.Groups = filterActiveGroups(result.Groups)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func GetPerfMetricsGroups(c *gin.Context) {
	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}

	ratios := ratio_setting.GetGroupRatioCopy()
	activeGroups := activePerfMetricGroups(ratios)
	summaries, err := perfmetrics.QueryGroupSummaryAll(hours, activeGroups)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	channelStatuses, err := model.GetGroupChannelStatuses(activeGroups)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	type groupMonitor struct {
		perfmetrics.GroupSummary
		Ratio *float64 `json:"ratio,omitempty"`
	}
	byGroup := make(map[string]perfmetrics.GroupSummary, len(summaries))
	for _, summary := range summaries {
		byGroup[summary.Group] = summary
	}
	result := make([]groupMonitor, 0, len(activeGroups))
	for _, group := range activeGroups {
		summary, ok := byGroup[group]
		if !ok {
			summary = perfmetrics.GroupSummary{
				Group:  group,
				Status: "unknown",
				Series: []perfmetrics.BucketPoint{},
			}
		}
		channelStatus := channelStatuses[group]
		perfmetrics.ApplyGroupStatus(&summary, channelStatus.ChannelCount, channelStatus.DisabledChannelCount)
		var ratio *float64
		if value, ok := ratios[group]; ok {
			ratio = &value
		}
		result = append(result, groupMonitor{GroupSummary: summary, Ratio: ratio})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func activePerfMetricGroups(ratios map[string]float64) []string {
	delete(ratios, "auto")
	groups := append(lo.Keys(ratios), "auto")
	sort.Strings(groups)
	return groups
}

func filterActiveGroups(groups []perfmetrics.GroupResult) []perfmetrics.GroupResult {
	activeRatios := ratio_setting.GetGroupRatioCopy()
	return lo.Filter(groups, func(g perfmetrics.GroupResult, _ int) bool {
		_, ok := activeRatios[g.Group]
		return ok || g.Group == "auto"
	})
}
