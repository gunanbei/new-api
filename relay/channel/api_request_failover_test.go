package channel

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoRequestFailoverTimeouts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service.InitHttpClient()

	t.Run("response headers", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer server.Close()

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		req, err := http.NewRequest(http.MethodPost, server.URL, http.NoBody)
		require.NoError(t, err)
		info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}, FailoverRules: types.TokenFailoverRules{Enabled: true, ResponseHeaderTimeoutMS: 10}}

		_, err = doRequest(c, req, info)
		var apiErr *types.NewAPIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, types.ErrorCodeUpstreamResponseHeaderTimeout, apiErr.GetErrorCode())
	})

	t.Run("first content", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		}))
		defer server.Close()

		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
		info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}, FailoverRules: types.TokenFailoverRules{Enabled: true, FirstContentTimeoutMS: 10}}
		info.AttemptResponse = relaycommon.NewAttemptResponseWriter(c.Writer)
		req, err := http.NewRequest(http.MethodPost, server.URL, http.NoBody)
		require.NoError(t, err)

		resp, err := doRequest(c, req, info)
		require.NoError(t, err)
		defer resp.Body.Close()
		_, err = io.ReadAll(resp.Body)
		assert.True(t, errors.Is(err, relaycommon.ErrFirstContentTimeout))
	})
}

func TestTokenFailoverRulesSelectsTimeoutByStreamMode(t *testing.T) {
	streamHeader := 1000
	nonStreamHeader := 2000
	streamContent := 3000
	nonStreamContent := 4000
	rules := types.TokenFailoverRules{
		ResponseHeaderTimeoutMS:          99,
		FirstContentTimeoutMS:            98,
		StreamResponseHeaderTimeoutMS:    &streamHeader,
		NonStreamResponseHeaderTimeoutMS: &nonStreamHeader,
		StreamFirstContentTimeoutMS:      &streamContent,
		NonStreamFirstContentTimeoutMS:   &nonStreamContent,
	}

	assert.Equal(t, 1000, rules.ResponseHeaderTimeout(true))
	assert.Equal(t, 2000, rules.ResponseHeaderTimeout(false))
	assert.Equal(t, 3000, rules.FirstContentTimeout(true))
	assert.Equal(t, 4000, rules.FirstContentTimeout(false))

	legacy := types.TokenFailoverRules{ResponseHeaderTimeoutMS: 5000, FirstContentTimeoutMS: 6000}
	assert.Equal(t, 5000, legacy.ResponseHeaderTimeout(true))
	assert.Equal(t, 5000, legacy.ResponseHeaderTimeout(false))
	assert.Equal(t, 6000, legacy.FirstContentTimeout(true))
	assert.Equal(t, 6000, legacy.FirstContentTimeout(false))
}
