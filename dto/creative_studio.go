package dto

import "encoding/json"

type CreativeModelRequest struct {
	CopyFromID  uint64 `json:"copy_from_id,omitempty"`
	ModelName   string `json:"model_name"`
	DisplayName string `json:"display_name"`
	Vendor      string `json:"vendor"`
	Description string `json:"description"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sort_order"`
}

type CreativeCapabilityRequest struct {
	CopyFromID    uint64          `json:"copy_from_id,omitempty"`
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
	CopyFromID         uint64          `json:"copy_from_id,omitempty"`
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

type CreativeReorderRequest struct {
	Kind     string   `json:"kind"`
	ParentID uint64   `json:"parent_id"`
	IDs      []uint64 `json:"ids"`
}

type CreativeStudioSettingsRequest struct {
	DefaultFileChannelID         uint64   `json:"default_file_channel_id"`
	AllowedImageMIMETypes        []string `json:"allowed_image_mime_types"`
	MaxRasterBytes               int64    `json:"max_raster_bytes"`
	MaxSVGBytes                  int64    `json:"max_svg_bytes"`
	ReferenceUploadQueueSize     int      `json:"reference_upload_queue_size"`
	ReferenceUploadMaxConcurrent int      `json:"reference_upload_max_concurrent"`
}

type CreativeTaskCreateRequest struct {
	CapabilityID uint64                     `json:"capability_id"`
	GroupName    string                     `json:"group_name"`
	Params       map[string]json.RawMessage `json:"params"`
}
