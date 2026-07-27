package tencent

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relay/channel/openai"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatchAdaptorInit(t *testing.T) {
	for _, tt := range []struct {
		name, key, baseURL string
		tc3                bool
		wantBaseURL        string
	}{
		{"legacy key", "1300000000|AKID|secret", constant.ChannelBaseURLs[constant.ChannelTypeTencent], true, constant.ChannelBaseURLs[constant.ChannelTypeTencent]},
		{"tokenhub default", "sk-tokenhub", constant.ChannelBaseURLs[constant.ChannelTypeTencent], false, tokenHubBaseURL},
		{"tokenhub custom", "sk-tokenhub", "https://proxy.example", false, "https://proxy.example"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeTencent, ApiKey: tt.key, ChannelBaseUrl: tt.baseURL}}
			dispatch := &DispatchAdaptor{}
			dispatch.Init(info)
			require.NotNil(t, dispatch.Adaptor)
			if tt.tc3 {
				assert.IsType(t, &Adaptor{}, dispatch.Adaptor)
			} else {
				assert.IsType(t, &openai.Adaptor{}, dispatch.Adaptor)
			}
			assert.Equal(t, tt.wantBaseURL, info.ChannelBaseUrl)
		})
	}
}
