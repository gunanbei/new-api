package setting

import (
	"fmt"
	"github.com/QuantumNous/new-api/common"
	"strconv"
	"sync/atomic"
)

const DefaultMaxTokenAutoGroups = 5

var autoGroups = []string{
	"default",
}

var DefaultUseAutoGroup = false
var maxTokenAutoGroups atomic.Int64

func init() { maxTokenAutoGroups.Store(DefaultMaxTokenAutoGroups) }

func ContainsAutoGroup(group string) bool {
	for _, autoGroup := range autoGroups {
		if autoGroup == group {
			return true
		}
	}
	return false
}

func UpdateAutoGroupsByJsonString(jsonString string) error {
	autoGroups = make([]string, 0)
	return common.Unmarshal([]byte(jsonString), &autoGroups)
}

func AutoGroups2JsonString() string {
	jsonBytes, err := common.Marshal(autoGroups)
	if err != nil {
		return "[]"
	}
	return string(jsonBytes)
}

func GetAutoGroups() []string {
	return autoGroups
}

func GetMaxTokenAutoGroups() int { return int(maxTokenAutoGroups.Load()) }
func ValidateMaxTokenAutoGroups(value string) error {
	v, err := strconv.Atoi(value)
	if err != nil || v <= 0 {
		return fmt.Errorf("MaxTokenAutoGroups must be a positive integer")
	}
	return nil
}
func UpdateMaxTokenAutoGroups(value string) error {
	if err := ValidateMaxTokenAutoGroups(value); err != nil {
		return err
	}
	v, _ := strconv.Atoi(value)
	maxTokenAutoGroups.Store(int64(v))
	return nil
}
