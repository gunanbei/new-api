package service

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/QuantumNous/new-api/model"
	"github.com/robfig/cron/v3"
)

const (
	inflightTaskCleanupScheduleModeOptionKey = "InflightTaskCleanupScheduleMode"
	inflightTaskCleanupCronOptionKey         = "InflightTaskCleanupCron"
	InflightTaskCleanupScheduleModeInterval  = "interval"
	InflightTaskCleanupScheduleModeCron      = "cron"
	inflightTaskCleanupCronDefault           = "0 3 * * *"
)

var (
	errInflightTaskCleanupIntervalOutOfRange = errors.New("inflight task cleanup interval out of range")
	errInflightTaskCleanupScheduleMode       = errors.New("invalid inflight task cleanup schedule mode")
	errInflightTaskCleanupCronInvalid        = errors.New("invalid inflight task cleanup cron expression")
)

func InflightTaskCleanupScheduleMode() string {
	common.OptionMapRWMutex.RLock()
	raw := strings.TrimSpace(common.OptionMap[inflightTaskCleanupScheduleModeOptionKey])
	common.OptionMapRWMutex.RUnlock()
	if raw == InflightTaskCleanupScheduleModeCron {
		return InflightTaskCleanupScheduleModeCron
	}
	return InflightTaskCleanupScheduleModeInterval
}

func InflightTaskCleanupCron() string {
	common.OptionMapRWMutex.RLock()
	raw := strings.TrimSpace(common.OptionMap[inflightTaskCleanupCronOptionKey])
	common.OptionMapRWMutex.RUnlock()
	if raw == "" {
		return inflightTaskCleanupCronDefault
	}
	if _, err := cron.ParseStandard(raw); err != nil {
		return inflightTaskCleanupCronDefault
	}
	return raw
}

func ValidateInflightTaskCleanupIntervalMinutes(raw string) error {
	minutes, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return errInflightTaskCleanupIntervalOutOfRange
	}
	if minutes < 1 || minutes > inflightTaskCleanupIntervalMinutesMax {
		return fmt.Errorf(
			"%w: allowed range is 1-%d minutes",
			errInflightTaskCleanupIntervalOutOfRange,
			inflightTaskCleanupIntervalMinutesMax,
		)
	}
	return nil
}

func ValidateInflightTaskCleanupScheduleMode(raw string) error {
	switch strings.TrimSpace(raw) {
	case InflightTaskCleanupScheduleModeInterval, InflightTaskCleanupScheduleModeCron:
		return nil
	default:
		return errInflightTaskCleanupScheduleMode
	}
}

func ValidateInflightTaskCleanupCron(raw string) error {
	expr := strings.TrimSpace(raw)
	if expr == "" {
		return errInflightTaskCleanupCronInvalid
	}
	if _, err := cron.ParseStandard(expr); err != nil {
		return fmt.Errorf("%w: %v", errInflightTaskCleanupCronInvalid, err)
	}
	return nil
}

func IsInflightTaskCleanupCronDue(expr string, lastRunAt int64, now int64) bool {
	schedule, err := cron.ParseStandard(strings.TrimSpace(expr))
	if err != nil {
		return false
	}
	if lastRunAt <= 0 {
		return true
	}
	next := schedule.Next(time.Unix(lastRunAt, 0))
	return now >= next.Unix()
}

type InflightTaskCleanupCronPreview struct {
	Timezone string  `json:"timezone"`
	NextRuns []int64 `json:"next_runs"`
}

func PreviewInflightTaskCleanupCron(expr string, count int) (*InflightTaskCleanupCronPreview, error) {
	if count <= 0 {
		count = 6
	}
	expr = strings.TrimSpace(expr)
	schedule, err := cron.ParseStandard(expr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errInflightTaskCleanupCronInvalid, err)
	}
	nextRuns := make([]int64, 0, count)
	next := time.Now()
	for i := 0; i < count; i++ {
		next = schedule.Next(next)
		nextRuns = append(nextRuns, next.Unix())
	}
	return &InflightTaskCleanupCronPreview{
		Timezone: time.Now().Location().String(),
		NextRuns: nextRuns,
	}, nil
}

func IsInflightTaskCleanupScheduleDue(lastRunAt int64, now int64) bool {
	if InflightTaskCleanupScheduleMode() == InflightTaskCleanupScheduleModeCron {
		return IsInflightTaskCleanupCronDue(InflightTaskCleanupCron(), lastRunAt, now)
	}
	if lastRunAt <= 0 {
		return true
	}
	return now-lastRunAt >= int64(InflightTaskCleanupInterval().Seconds())
}

type InflightTaskCleanupScheduleSummary struct {
	Enabled         bool   `json:"enabled"`
	ScheduleMode    string `json:"schedule_mode"`
	IntervalMinutes int    `json:"interval_minutes"`
	CronExpression  string `json:"cron_expression"`
	Timezone        string `json:"timezone"`
	NextRunAt       int64  `json:"next_run_at"`
	Running         bool   `json:"running"`
	LastRunAt       int64  `json:"last_run_at,omitempty"`
}

func NextInflightTaskCleanupRunAt(lastRunAt int64, now int64) int64 {
	if InflightTaskCleanupScheduleMode() == InflightTaskCleanupScheduleModeCron {
		preview, err := PreviewInflightTaskCleanupCron(InflightTaskCleanupCron(), 1)
		if err != nil || len(preview.NextRuns) == 0 {
			return 0
		}
		return preview.NextRuns[0]
	}
	intervalSec := int64(InflightTaskCleanupInterval().Seconds())
	if lastRunAt <= 0 {
		return now
	}
	next := lastRunAt + intervalSec
	if next < now {
		return now
	}
	return next
}

func GetInflightTaskCleanupScheduleSummary() (*InflightTaskCleanupScheduleSummary, error) {
	summary := &InflightTaskCleanupScheduleSummary{
		Timezone: time.Now().Location().String(),
	}
	if InflightTaskCleanupRule() == InflightTaskCleanupRuleDisabled {
		return summary, nil
	}
	summary.Enabled = true
	summary.ScheduleMode = InflightTaskCleanupScheduleMode()
	if summary.ScheduleMode == InflightTaskCleanupScheduleModeCron {
		summary.CronExpression = InflightTaskCleanupCron()
	} else {
		summary.IntervalMinutes = int(InflightTaskCleanupInterval().Minutes())
	}

	activeTask, err := model.GetActiveSystemTask(model.SystemTaskTypeInflightLogCleanup)
	if err != nil {
		return nil, err
	}
	if activeTask != nil {
		summary.Running = true
		return summary, nil
	}

	latest, err := model.GetLatestSystemTask(model.SystemTaskTypeInflightLogCleanup)
	if err != nil {
		return nil, err
	}
	var lastRunAt int64
	if latest != nil {
		switch latest.Status {
		case model.SystemTaskStatusSucceeded, model.SystemTaskStatusFailed:
			lastRunAt = latest.UpdatedAt
			summary.LastRunAt = lastRunAt
		case model.SystemTaskStatusPending, model.SystemTaskStatusRunning:
			summary.Running = true
			return summary, nil
		}
	}
	summary.NextRunAt = NextInflightTaskCleanupRunAt(lastRunAt, common.GetTimestamp())
	return summary, nil
}
