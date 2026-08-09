package ratio_setting

import "strings"

const CompactModelSuffix = "-openai-compact"
const CompactWildcardModelKey = "*" + CompactModelSuffix

func HasCompactModelSuffix(modelName string) bool {
	return strings.HasSuffix(modelName, CompactModelSuffix)
}

func TrimCompactModelSuffix(modelName string) string {
	return strings.TrimSuffix(modelName, CompactModelSuffix)
}

func WithCompactModelSuffix(modelName string) string {
	if HasCompactModelSuffix(modelName) {
		return modelName
	}
	return modelName + CompactModelSuffix
}
