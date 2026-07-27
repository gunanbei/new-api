package openai

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealtimeHeadersUseBetaOnlyForPreviewModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name, model string
		wantBeta    bool
	}{
		{name: "GA", model: "gpt-realtime-2"},
		{name: "preview", model: "gpt-4o-realtime-preview", wantBeta: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/realtime", nil)
			info := &relaycommon.RelayInfo{
				RelayMode:       relayconstant.RelayModeRealtime,
				OriginModelName: tt.model,
				ChannelMeta:     &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeOpenAI, ApiKey: "test-key", UpstreamModelName: tt.model},
			}
			header := http.Header{}
			require.NoError(t, (&Adaptor{}).SetupRequestHeader(ctx, &header, info))
			assert.Equal(t, tt.wantBeta, header.Get("openai-beta") != "")
			assert.Equal(t, "Bearer test-key", header.Get("Authorization"))
		})
	}
}
