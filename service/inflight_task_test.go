package service

import (
	"testing"

	relayconstant "github.com/QuantumNous/new-api/relay/constant"

	"github.com/stretchr/testify/assert"
)

func TestInflightTaskKindFromRelayMode(t *testing.T) {
	assert.Equal(t, "chat", inflightTaskKindFromRelayMode(relayconstant.RelayModeChatCompletions))
	assert.Equal(t, "image", inflightTaskKindFromRelayMode(relayconstant.RelayModeImagesGenerations))
	assert.Equal(t, "audio", inflightTaskKindFromRelayMode(relayconstant.RelayModeAudioSpeech))
}

func TestInflightTaskRetentionTTL(t *testing.T) {
	assert.Equal(t, inflightTaskRetentionTTL(InflightTaskStatusCompleted), inflightTaskRetentionTTL(InflightTaskStatusFailed))
	assert.NotZero(t, inflightTaskRetentionTTL(InflightTaskStatusAccepted))
}
