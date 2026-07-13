package model

import (
	"time"
)

type FileUploadChannel struct {
	Id             uint64    `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name           string    `json:"name" gorm:"column:name;type:varchar(255);not null;default:''"`
	Type           string    `json:"type" gorm:"column:type;type:varchar(1);not null"`
	Status         string    `json:"status" gorm:"column:status;type:varchar(1);not null;default:'1'"`
	IsDefault      string    `json:"is_default" gorm:"column:is_default;type:varchar(1);not null;default:'0'"`
	ChunkThreshold int64     `json:"chunk_threshold" gorm:"column:chunk_threshold;not null;default:16777216"`
	ChunkSize      int64     `json:"chunk_size" gorm:"column:chunk_size;not null;default:8388608"`
	MaxSize        int64     `json:"max_size" gorm:"column:max_size;not null;default:0"`
	ConfigProflle  string    `json:"config_proflle" gorm:"column:config_proflle;type:text;not null"`
	CreateUserId   int64     `json:"create_user_id" gorm:"column:create_user_id;not null"`
	UpdateUserId   int64     `json:"update_user_id" gorm:"column:update_user_id;not null"`
	CreateTime     time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`
	UpdateTime     time.Time `json:"update_time" gorm:"column:update_time;autoUpdateTime"`
}

func (FileUploadChannel) TableName() string {
	return "file_upload_channel"
}
