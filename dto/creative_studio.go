package dto

import "encoding/json"

type CreativeModelRequest struct {
	ModelName   string `json:"model_name"`
	DisplayName string `json:"display_name"`
	Vendor      string `json:"vendor"`
	Description string `json:"description"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sort_order"`
}

type CreativeCapabilityRequest struct {
	Category      string          `json:"category"`
	Operation     string          `json:"operation"`
	AssetKind     string          `json:"asset_kind"`
	Protocol      string          `json:"protocol"`
	ExecutionMode string          `json:"execution_mode"`
	InputSchema   json.RawMessage `json:"input_schema"`
	DefaultParams json.RawMessage `json:"default_params"`
	Enabled       bool            `json:"enabled"`
	SortOrder     int             `json:"sort_order"`
}

type CreativePublicationRequest struct {
	GroupName          string          `json:"group_name"`
	Enabled            bool            `json:"enabled"`
	SortOrder          int             `json:"sort_order"`
	GroupDefaultParams json.RawMessage `json:"group_default_params"`
}

type CreativeBindingRequest struct {
	ChannelID    int    `json:"channel_id"`
	RequestModel string `json:"request_model"`
	Priority     int    `json:"priority"`
	Enabled      bool   `json:"enabled"`
}

type CreativeStudioSettingsRequest struct {
	DefaultFileChannelID uint64 `json:"default_file_channel_id"`
}
