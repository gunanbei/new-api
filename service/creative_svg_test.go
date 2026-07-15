package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeCreativeSVG(t *testing.T) {
	clean, err := SanitizeCreativeSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><path d="M0 0h10v10z" fill="#000"/></svg>`), maxCreativeSVGBytes)
	require.NoError(t, err)
	assert.Contains(t, string(clean), "<svg")
	clean, err = SanitizeCreativeSVG([]byte("```svg\n<svg><path d=\"M0 0h10v10z\"/></svg>\n```"), maxCreativeSVGBytes)
	require.NoError(t, err)
	assert.Contains(t, string(clean), "<svg")
	clean, err = SanitizeCreativeSVG([]byte(`<svg><defs><linearGradient id="sky"><stop offset="0%" stop-color="#000"/></linearGradient><filter id="blur"><feGaussianBlur stdDeviation="3"/></filter></defs><rect width="10" height="10" fill="url(#sky)" filter="url(#blur)"/></svg>`), maxCreativeSVGBytes)
	require.NoError(t, err)
	assert.Contains(t, string(clean), `filter="url(#blur)"`)
	for _, sample := range []string{
		`<svg><script>alert(1)</script></svg>`,
		`<!DOCTYPE svg><svg/>`,
		`<svg><foreignObject/></svg>`,
		`<svg><path onclick="x()"/></svg>`,
		`<svg><path fill="url(https://example.com/a)"/></svg>`,
	} {
		_, err := SanitizeCreativeSVG([]byte(sample), maxCreativeSVGBytes)
		assert.Error(t, err, sample)
	}
}
