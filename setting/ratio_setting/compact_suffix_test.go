package ratio_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompactModelSuffixHelpers(t *testing.T) {
	t.Parallel()

	assert.True(t, HasCompactModelSuffix("gpt-5.6-terra-openai-compact"))
	assert.False(t, HasCompactModelSuffix("gpt-5.6-terra"))
	assert.Equal(t, "gpt-5.6-terra", TrimCompactModelSuffix("gpt-5.6-terra-openai-compact"))
	assert.Equal(t, "gpt-5.6-terra-openai-compact", WithCompactModelSuffix("gpt-5.6-terra-openai-compact"))
	assert.Equal(t, "gpt-5.6-terra-openai-compact", WithCompactModelSuffix("gpt-5.6-terra"))
}
