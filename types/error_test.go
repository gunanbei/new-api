package types

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIErrorDetailedErrorPreservesInternalCause(t *testing.T) {
	apiErr := NewError(errors.New("EOF"), ErrorCodeEmptyResponse,
		ErrOptionWithStatusCode(http.StatusBadGateway),
		ErrOptionWithInternalError(errors.New("upstream reset the HTTP/2 stream")))

	require.Equal(t, "EOF", apiErr.Error())
	assert.Contains(t, apiErr.DetailedError(), "EOF")
	assert.Contains(t, apiErr.DetailedError(), "upstream reset the HTTP/2 stream")
	assert.Contains(t, apiErr.ErrorWithStatusCode(), "status_code=502")
	assert.NotContains(t, apiErr.ToOpenAIError().Message, "upstream reset the HTTP/2 stream")
}

func TestNewAPIErrorDetailedErrorUnwrapsNestedCause(t *testing.T) {
	upstream := NewError(errors.New("provider stream failed"), ErrorCodeUpstreamStreamError,
		ErrOptionWithInternalError(errors.New("RST_STREAM")))
	apiErr := NewError(errors.New("EOF"), ErrorCodeEmptyResponse,
		ErrOptionWithInternalError(upstream))

	assert.Contains(t, apiErr.DetailedError(), "RST_STREAM")
}
