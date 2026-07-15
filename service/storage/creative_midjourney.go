package storage

import (
	"context"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

func ObserveCreativeMidjourneyTasks(ctx context.Context, limit int) error {
	if limit <= 0 {
		limit = 100
	}
	var tasks []model.CreativeTask
	if err := model.DB.Where("protocol = ? AND status = ? AND external_task_id <> ?", "midjourney_image", model.CreativeTaskStatusProcessing, "").Order("updated_at asc").Limit(limit).Find(&tasks).Error; err != nil {
		return err
	}
	for index := range tasks {
		task := &tasks[index]
		var upstream model.Midjourney
		if err := model.DB.Where("user_id = ? AND mj_id = ?", task.UserID, task.ExternalTaskID).First(&upstream).Error; err != nil {
			continue
		}
		if upstream.Status == "SUCCESS" && upstream.ImageUrl != "" {
			manifest, err := common.Marshal(map[string]any{"items": []map[string]string{{"url": upstream.ImageUrl, "file_name": "midjourney.png"}}})
			if err != nil {
				return err
			}
			result := model.DB.Model(&model.CreativeTask{}).Where("id = ? AND status = ?", task.ID, model.CreativeTaskStatusProcessing).Updates(map[string]any{"status": model.CreativeTaskStatusImporting, "result_manifest": string(manifest), "import_expires_at": time.Now().Add(time.Hour)})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 1 {
				task.Status, task.ResultManifest = model.CreativeTaskStatusImporting, string(manifest)
				if err := ImportCreativeTaskManifest(ctx, task); err != nil {
					service.MarkCreativeTaskImportFailed(task.ID, err)
				}
			}
			continue
		}
		if upstream.Status == "FAILURE" || upstream.Status == "FAILED" {
			_ = model.DB.Model(&model.CreativeTask{}).Where("id = ? AND status = ?", task.ID, model.CreativeTaskStatusProcessing).Updates(map[string]any{"status": model.CreativeTaskStatusFailed, "error_code": "upstream_failed", "error_message": "image generation failed", "finished_at": time.Now()}).Error
		}
	}
	return nil
}
