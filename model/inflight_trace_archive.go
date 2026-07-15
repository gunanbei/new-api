package model

type InflightTraceArchive struct {
	ID               uint64 `json:"id" gorm:"primaryKey"`
	UserID           int    `json:"user_id" gorm:"index"`
	FileName         string `json:"file_name" gorm:"type:varchar(255)"`
	LocalPath        string `json:"-" gorm:"type:text"`
	ObjectKey        string `json:"object_key" gorm:"type:text"`
	RemoteURL        string `json:"remote_url" gorm:"type:text"`
	StorageChannelID uint64 `json:"storage_channel_id" gorm:"index"`
	Status           string `json:"status" gorm:"type:varchar(32);index"`
	RetryCount       int    `json:"retry_count"`
	LastError        string `json:"last_error" gorm:"type:text"`
	CreatedAt        int64  `json:"created_at" gorm:"bigint;index"`
	LatestRecordedAt int64  `json:"latest_recorded_at" gorm:"bigint;index"`
	UploadedAt       int64  `json:"uploaded_at" gorm:"bigint;index"`
}
