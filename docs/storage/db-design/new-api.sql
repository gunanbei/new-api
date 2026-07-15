/*
 Navicat Premium Dump SQL

 Source Server         : 本机 NEW-Api
 Source Server Type    : MySQL
 Source Server Version : 80044 (8.0.44)
 Source Host           : 127.0.0.1:3306
 Source Schema         : new-api

 Target Server Type    : MySQL
 Target Server Version : 80044 (8.0.44)
 File Encoding         : 65001

 Date: 15/07/2026 14:58:28
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for abilities
-- ----------------------------
DROP TABLE IF EXISTS `abilities`;
CREATE TABLE `abilities` (
  `group` varchar(64) NOT NULL,
  `model` varchar(255) NOT NULL,
  `channel_id` bigint NOT NULL,
  `enabled` tinyint(1) DEFAULT NULL,
  `priority` bigint DEFAULT '0',
  `weight` bigint unsigned DEFAULT '0',
  `tag` varchar(191) DEFAULT NULL,
  PRIMARY KEY (`group`,`model`,`channel_id`),
  KEY `idx_abilities_channel_id` (`channel_id`),
  KEY `idx_abilities_priority` (`priority`),
  KEY `idx_abilities_weight` (`weight`),
  KEY `idx_abilities_tag` (`tag`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for authz_roles
-- ----------------------------
DROP TABLE IF EXISTS `authz_roles`;
CREATE TABLE `authz_roles` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `key` varchar(64) NOT NULL,
  `name` varchar(100) NOT NULL,
  `description` text,
  `built_in` tinyint(1) DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT NULL,
  `sort` bigint DEFAULT NULL,
  `created_at` bigint DEFAULT NULL,
  `updated_at` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_authz_roles_key` (`key`)
) ENGINE=InnoDB AUTO_INCREMENT=91 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for casbin_rule
-- ----------------------------
DROP TABLE IF EXISTS `casbin_rule`;
CREATE TABLE `casbin_rule` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ptype` varchar(100) DEFAULT NULL,
  `v0` varchar(100) DEFAULT NULL,
  `v1` varchar(100) DEFAULT NULL,
  `v2` varchar(100) DEFAULT NULL,
  `v3` varchar(100) DEFAULT NULL,
  `v4` varchar(100) DEFAULT NULL,
  `v5` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_casbin_rule_unique` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`),
  KEY `idx_casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
) ENGINE=InnoDB AUTO_INCREMENT=136 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for channels
-- ----------------------------
DROP TABLE IF EXISTS `channels`;
CREATE TABLE `channels` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `type` bigint DEFAULT '0',
  `key` longtext NOT NULL,
  `open_ai_organization` longtext,
  `test_model` longtext,
  `status` bigint DEFAULT '1',
  `name` varchar(191) DEFAULT NULL,
  `weight` bigint unsigned DEFAULT '0',
  `created_time` bigint DEFAULT NULL,
  `test_time` bigint DEFAULT NULL,
  `response_time` bigint DEFAULT NULL,
  `base_url` varchar(191) DEFAULT '',
  `other` longtext,
  `balance` double DEFAULT NULL,
  `balance_updated_time` bigint DEFAULT NULL,
  `models` longtext,
  `group` varchar(64) DEFAULT 'default',
  `used_quota` bigint DEFAULT '0',
  `model_mapping` text,
  `status_code_mapping` varchar(1024) DEFAULT '',
  `priority` bigint DEFAULT '0',
  `auto_ban` bigint DEFAULT '1',
  `other_info` longtext,
  `tag` varchar(191) DEFAULT NULL,
  `setting` text,
  `param_override` text,
  `header_override` text,
  `remark` varchar(255) DEFAULT NULL,
  `channel_info` json DEFAULT NULL,
  `settings` longtext,
  PRIMARY KEY (`id`),
  KEY `idx_channels_name` (`name`),
  KEY `idx_channels_tag` (`tag`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for checkins
-- ----------------------------
DROP TABLE IF EXISTS `checkins`;
CREATE TABLE `checkins` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL,
  `checkin_date` varchar(10) NOT NULL,
  `quota_awarded` bigint NOT NULL,
  `created_at` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_checkin_date` (`user_id`,`checkin_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for creative_channel_binding
-- ----------------------------
DROP TABLE IF EXISTS `creative_channel_binding`;
CREATE TABLE `creative_channel_binding` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `publication_id` bigint unsigned DEFAULT NULL,
  `channel_id` bigint DEFAULT NULL,
  `request_model` varchar(255) NOT NULL,
  `priority` bigint NOT NULL,
  `enabled` tinyint(1) NOT NULL,
  `validation_status` varchar(16) NOT NULL,
  `validation_message` text NOT NULL,
  `created_by` bigint NOT NULL,
  `updated_by` bigint NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `validation_checked_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_creative_binding` (`publication_id`,`channel_id`),
  KEY `idx_creative_channel_binding_publication_id` (`publication_id`),
  KEY `idx_creative_channel_binding_channel_id` (`channel_id`),
  KEY `idx_creative_channel_binding_enabled` (`enabled`),
  KEY `idx_creative_channel_binding_validation_status` (`validation_status`),
  KEY `idx_creative_channel_binding_validation_checked_at` (`validation_checked_at`)
) ENGINE=InnoDB AUTO_INCREMENT=11 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for creative_model
-- ----------------------------
DROP TABLE IF EXISTS `creative_model`;
CREATE TABLE `creative_model` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `model_name` varchar(255) NOT NULL,
  `model_key` varchar(255) NOT NULL,
  `display_name` varchar(255) NOT NULL,
  `vendor` varchar(255) NOT NULL,
  `description` text NOT NULL,
  `status` varchar(16) NOT NULL,
  `sort_order` bigint NOT NULL,
  `created_by` bigint NOT NULL,
  `updated_by` bigint NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_creative_model_model_key` (`model_key`),
  KEY `idx_creative_model_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for creative_model_capability
-- ----------------------------
DROP TABLE IF EXISTS `creative_model_capability`;
CREATE TABLE `creative_model_capability` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `model_id` bigint unsigned DEFAULT NULL,
  `category` varchar(16) DEFAULT NULL,
  `operation` varchar(32) DEFAULT NULL,
  `asset_kind` varchar(16) NOT NULL,
  `protocol` varchar(64) DEFAULT NULL,
  `execution_mode` varchar(32) NOT NULL,
  `input_schema` text NOT NULL,
  `default_params` text NOT NULL,
  `enabled` tinyint(1) NOT NULL,
  `sort_order` bigint NOT NULL,
  `created_by` bigint NOT NULL,
  `updated_by` bigint NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_creative_capability` (`model_id`,`category`,`operation`,`protocol`),
  KEY `idx_creative_model_capability_model_id` (`model_id`),
  KEY `idx_creative_model_capability_category` (`category`),
  KEY `idx_creative_model_capability_enabled` (`enabled`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for creative_model_publication
-- ----------------------------
DROP TABLE IF EXISTS `creative_model_publication`;
CREATE TABLE `creative_model_publication` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `capability_id` bigint unsigned DEFAULT NULL,
  `group_name` varchar(64) DEFAULT NULL,
  `enabled` tinyint(1) NOT NULL,
  `sort_order` bigint NOT NULL,
  `group_default_params` text NOT NULL,
  `created_by` bigint NOT NULL,
  `updated_by` bigint NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_creative_publication` (`capability_id`,`group_name`),
  KEY `idx_creative_model_publication_group_name` (`group_name`),
  KEY `idx_creative_model_publication_enabled` (`enabled`),
  KEY `idx_creative_model_publication_capability_id` (`capability_id`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for creative_task
-- ----------------------------
DROP TABLE IF EXISTS `creative_task`;
CREATE TABLE `creative_task` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `task_key` varchar(64) NOT NULL,
  `user_id` bigint NOT NULL,
  `category` varchar(16) NOT NULL,
  `capability_id` bigint unsigned NOT NULL,
  `binding_id` bigint unsigned NOT NULL,
  `model_name` varchar(255) NOT NULL,
  `display_name` varchar(255) NOT NULL,
  `group_name` varchar(64) NOT NULL,
  `protocol` varchar(64) NOT NULL,
  `execution_mode` varchar(32) NOT NULL,
  `channel_id` bigint NOT NULL,
  `request_id` varchar(64) DEFAULT NULL,
  `external_task_id` varchar(191) DEFAULT NULL,
  `status` varchar(16) NOT NULL,
  `requested_params` text NOT NULL,
  `resolved_params` text NOT NULL,
  `result_manifest` text NOT NULL,
  `import_expires_at` datetime(3) DEFAULT NULL,
  `error_code` varchar(64) NOT NULL,
  `error_message` text NOT NULL,
  `retry_of_id` bigint unsigned DEFAULT NULL,
  `started_at` datetime(3) DEFAULT NULL,
  `finished_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `operation` varchar(16) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_creative_task_task_key` (`task_key`),
  KEY `idx_creative_task_request_id` (`request_id`),
  KEY `idx_creative_task_external_task_id` (`external_task_id`),
  KEY `idx_creative_task_status_updated` (`status`,`updated_at`),
  KEY `idx_creative_task_retry_of_id` (`retry_of_id`),
  KEY `idx_creative_task_user_created` (`user_id`,`created_at`),
  KEY `idx_creative_task_user_category_status_created` (`user_id`,`category`,`status`)
) ENGINE=InnoDB AUTO_INCREMENT=37 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for creative_task_asset
-- ----------------------------
DROP TABLE IF EXISTS `creative_task_asset`;
CREATE TABLE `creative_task_asset` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `task_id` bigint unsigned DEFAULT NULL,
  `user_file_id` bigint unsigned DEFAULT NULL,
  `file_id` bigint unsigned NOT NULL,
  `role` varchar(16) DEFAULT NULL,
  `position` bigint DEFAULT NULL,
  `mime_type` varchar(128) NOT NULL,
  `file_size` bigint NOT NULL,
  `file_name` varchar(1000) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_creative_task_asset` (`task_id`,`user_file_id`,`role`,`position`),
  KEY `idx_creative_task_asset_task_id` (`task_id`),
  KEY `idx_creative_task_asset_user_file_id` (`user_file_id`),
  KEY `idx_creative_task_asset_file_id` (`file_id`)
) ENGINE=InnoDB AUTO_INCREMENT=12 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for custom_oauth_providers
-- ----------------------------
DROP TABLE IF EXISTS `custom_oauth_providers`;
CREATE TABLE `custom_oauth_providers` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL,
  `slug` varchar(64) NOT NULL,
  `icon` varchar(128) DEFAULT '',
  `enabled` tinyint(1) DEFAULT '0',
  `client_id` varchar(256) DEFAULT NULL,
  `client_secret` varchar(512) DEFAULT NULL,
  `authorization_endpoint` varchar(512) DEFAULT NULL,
  `token_endpoint` varchar(512) DEFAULT NULL,
  `user_info_endpoint` varchar(512) DEFAULT NULL,
  `scopes` varchar(256) DEFAULT 'openid profile email',
  `user_id_field` varchar(128) DEFAULT 'sub',
  `username_field` varchar(128) DEFAULT 'preferred_username',
  `display_name_field` varchar(128) DEFAULT 'name',
  `email_field` varchar(128) DEFAULT 'email',
  `well_known` varchar(512) DEFAULT NULL,
  `auth_style` bigint DEFAULT '0',
  `access_policy` text,
  `access_denied_message` varchar(512) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_custom_oauth_providers_slug` (`slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for file
-- ----------------------------
DROP TABLE IF EXISTS `file`;
CREATE TABLE `file` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 Id',
  `file_channel_id` bigint unsigned NOT NULL DEFAULT '0',
  `channel_type` varchar(1) NOT NULL,
  `file_size` bigint NOT NULL DEFAULT '0',
  `object_key` varchar(512) NOT NULL DEFAULT '',
  `file_url` varchar(1000) NOT NULL DEFAULT '',
  `identifier` varchar(255) NOT NULL,
  `mime_type` varchar(128) NOT NULL DEFAULT '',
  `etag` varchar(128) NOT NULL DEFAULT '',
  `ref_count` bigint NOT NULL DEFAULT '0',
  `status` varchar(1) NOT NULL DEFAULT '1',
  `creater_user_id` bigint NOT NULL DEFAULT '0',
  `create_time` datetime(3) DEFAULT NULL,
  `update_time` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_identifier_channel_type` (`identifier`,`channel_type`),
  KEY `idx_file_channel_id` (`file_channel_id`),
  KEY `idx_status` (`status`),
  KEY `idx_file_file_channel_id` (`file_channel_id`),
  KEY `idx_file_channel_type` (`channel_type`),
  KEY `idx_file_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=16 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='文件存储（物理对象）';

-- ----------------------------
-- Table structure for file_upload_channel
-- ----------------------------
DROP TABLE IF EXISTS `file_upload_channel`;
CREATE TABLE `file_upload_channel` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 Id',
  `name` varchar(255) NOT NULL DEFAULT '',
  `type` varchar(1) NOT NULL,
  `status` varchar(1) NOT NULL DEFAULT '1',
  `is_default` varchar(1) NOT NULL DEFAULT '0',
  `chunk_threshold` bigint NOT NULL DEFAULT '16777216',
  `chunk_size` bigint NOT NULL DEFAULT '8388608',
  `max_size` bigint NOT NULL DEFAULT '0',
  `config_proflle` text NOT NULL,
  `create_user_id` bigint NOT NULL,
  `update_user_id` bigint NOT NULL,
  `create_time` datetime(3) DEFAULT NULL,
  `update_time` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_type_status` (`type`,`status`),
  KEY `idx_is_default` (`is_default`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='文件上传渠道';

-- ----------------------------
-- Table structure for inflight_trace_archives
-- ----------------------------
DROP TABLE IF EXISTS `inflight_trace_archives`;
CREATE TABLE `inflight_trace_archives` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint DEFAULT NULL,
  `file_name` varchar(255) DEFAULT NULL,
  `local_path` text,
  `object_key` text,
  `remote_url` text,
  `storage_channel_id` bigint unsigned DEFAULT NULL,
  `status` varchar(32) DEFAULT NULL,
  `retry_count` bigint DEFAULT NULL,
  `last_error` text,
  `created_at` bigint DEFAULT NULL,
  `uploaded_at` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_inflight_trace_archives_user_id` (`user_id`),
  KEY `idx_inflight_trace_archives_storage_channel_id` (`storage_channel_id`),
  KEY `idx_inflight_trace_archives_status` (`status`),
  KEY `idx_inflight_trace_archives_created_at` (`created_at`),
  KEY `idx_inflight_trace_archives_uploaded_at` (`uploaded_at`)
) ENGINE=InnoDB AUTO_INCREMENT=47 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for logs
-- ----------------------------
DROP TABLE IF EXISTS `logs`;
CREATE TABLE `logs` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint DEFAULT NULL,
  `created_at` bigint DEFAULT NULL,
  `type` bigint DEFAULT NULL,
  `content` longtext,
  `username` varchar(191) DEFAULT '',
  `token_name` varchar(191) DEFAULT '',
  `model_name` varchar(191) DEFAULT '',
  `quota` bigint DEFAULT '0',
  `prompt_tokens` bigint DEFAULT '0',
  `completion_tokens` bigint DEFAULT '0',
  `use_time` bigint DEFAULT '0',
  `is_stream` tinyint(1) DEFAULT NULL,
  `channel_id` bigint DEFAULT NULL,
  `channel_name` longtext,
  `token_id` bigint DEFAULT '0',
  `group` varchar(191) DEFAULT NULL,
  `ip` varchar(191) DEFAULT '',
  `request_id` varchar(64) DEFAULT '',
  `upstream_request_id` varchar(128) DEFAULT '',
  `other` longtext,
  PRIMARY KEY (`id`),
  KEY `idx_logs_ip` (`ip`),
  KEY `idx_logs_user_id` (`user_id`),
  KEY `idx_logs_username` (`username`),
  KEY `index_username_model_name` (`model_name`,`username`),
  KEY `idx_logs_token_name` (`token_name`),
  KEY `idx_logs_channel_id` (`channel_id`),
  KEY `idx_logs_request_id` (`request_id`),
  KEY `idx_logs_upstream_request_id` (`upstream_request_id`),
  KEY `idx_created_at_id` (`created_at`,`id`),
  KEY `idx_user_id_id` (`user_id`,`id`),
  KEY `idx_created_at_type` (`created_at`,`type`),
  KEY `idx_logs_model_name` (`model_name`),
  KEY `idx_logs_token_id` (`token_id`),
  KEY `idx_logs_group` (`group`)
) ENGINE=InnoDB AUTO_INCREMENT=139 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for midjourneys
-- ----------------------------
DROP TABLE IF EXISTS `midjourneys`;
CREATE TABLE `midjourneys` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `code` bigint DEFAULT NULL,
  `user_id` bigint DEFAULT NULL,
  `action` varchar(40) DEFAULT NULL,
  `mj_id` varchar(191) DEFAULT NULL,
  `prompt` longtext,
  `prompt_en` longtext,
  `description` longtext,
  `state` longtext,
  `submit_time` bigint DEFAULT NULL,
  `start_time` bigint DEFAULT NULL,
  `finish_time` bigint DEFAULT NULL,
  `image_url` longtext,
  `video_url` longtext,
  `video_urls` longtext,
  `status` varchar(20) DEFAULT NULL,
  `progress` varchar(30) DEFAULT NULL,
  `fail_reason` longtext,
  `channel_id` bigint DEFAULT NULL,
  `quota` bigint DEFAULT NULL,
  `buttons` longtext,
  `properties` longtext,
  PRIMARY KEY (`id`),
  KEY `idx_midjourneys_submit_time` (`submit_time`),
  KEY `idx_midjourneys_start_time` (`start_time`),
  KEY `idx_midjourneys_finish_time` (`finish_time`),
  KEY `idx_midjourneys_status` (`status`),
  KEY `idx_midjourneys_progress` (`progress`),
  KEY `idx_midjourneys_user_id` (`user_id`),
  KEY `idx_midjourneys_action` (`action`),
  KEY `idx_midjourneys_mj_id` (`mj_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for model_ability
-- ----------------------------
DROP TABLE IF EXISTS `model_ability`;
CREATE TABLE `model_ability` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 Id',
  `model_id` bigint NOT NULL COMMENT 'model Id',
  `desc` varchar(255) NOT NULL COMMENT '描述',
  `abilities` varchar(1000) NOT NULL COMMENT '模型类型列表，不同类型之间以英文逗号 ,分割\n1-图片\n2-向量\n3-视频\n4-对话\n5-重排序',
  `create_user_id` bigint NOT NULL COMMENT '创建者',
  `update_user_id` bigint NOT NULL COMMENT '更新者',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='模型能力表';

-- ----------------------------
-- Table structure for models
-- ----------------------------
DROP TABLE IF EXISTS `models`;
CREATE TABLE `models` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `model_name` varchar(128) NOT NULL,
  `description` text,
  `icon` varchar(128) DEFAULT NULL,
  `tags` varchar(255) DEFAULT NULL,
  `vendor_id` bigint DEFAULT NULL,
  `endpoints` text,
  `status` bigint DEFAULT '1',
  `sync_official` bigint DEFAULT '1',
  `created_time` bigint DEFAULT NULL,
  `updated_time` bigint DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `name_rule` bigint DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_model_name_delete_at` (`model_name`,`deleted_at`),
  KEY `idx_models_vendor_id` (`vendor_id`),
  KEY `idx_models_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for options
-- ----------------------------
DROP TABLE IF EXISTS `options`;
CREATE TABLE `options` (
  `key` varchar(191) NOT NULL,
  `value` longtext,
  PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for passkey_credentials
-- ----------------------------
DROP TABLE IF EXISTS `passkey_credentials`;
CREATE TABLE `passkey_credentials` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL,
  `credential_id` varchar(512) NOT NULL,
  `public_key` text NOT NULL,
  `attestation_type` varchar(255) DEFAULT NULL,
  `aa_guid` varchar(512) DEFAULT NULL,
  `sign_count` int unsigned DEFAULT '0',
  `clone_warning` tinyint(1) DEFAULT NULL,
  `user_present` tinyint(1) DEFAULT NULL,
  `user_verified` tinyint(1) DEFAULT NULL,
  `backup_eligible` tinyint(1) DEFAULT NULL,
  `backup_state` tinyint(1) DEFAULT NULL,
  `transports` text,
  `attachment` varchar(32) DEFAULT NULL,
  `last_used_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_passkey_credentials_credential_id` (`credential_id`),
  UNIQUE KEY `idx_passkey_credentials_user_id` (`user_id`),
  KEY `idx_passkey_credentials_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for perf_metrics
-- ----------------------------
DROP TABLE IF EXISTS `perf_metrics`;
CREATE TABLE `perf_metrics` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `model_name` varchar(128) DEFAULT NULL,
  `group` varchar(64) DEFAULT NULL,
  `bucket_ts` bigint DEFAULT NULL,
  `request_count` bigint DEFAULT '0',
  `success_count` bigint DEFAULT '0',
  `total_latency_ms` bigint DEFAULT '0',
  `ttft_sum_ms` bigint DEFAULT '0',
  `ttft_count` bigint DEFAULT '0',
  `output_tokens` bigint DEFAULT '0',
  `generation_ms` bigint DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_perf_model_group_bucket` (`model_name`,`group`,`bucket_ts`),
  KEY `idx_perf_bucket_ts` (`bucket_ts`)
) ENGINE=InnoDB AUTO_INCREMENT=10 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for prefill_groups
-- ----------------------------
DROP TABLE IF EXISTS `prefill_groups`;
CREATE TABLE `prefill_groups` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL,
  `type` varchar(32) NOT NULL,
  `items` json DEFAULT NULL,
  `description` varchar(255) DEFAULT NULL,
  `created_time` bigint DEFAULT NULL,
  `updated_time` bigint DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_prefill_name` (`name`),
  KEY `idx_prefill_groups_type` (`type`),
  KEY `idx_prefill_groups_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for quota_data
-- ----------------------------
DROP TABLE IF EXISTS `quota_data`;
CREATE TABLE `quota_data` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint DEFAULT NULL,
  `username` varchar(64) DEFAULT '',
  `model_name` varchar(64) DEFAULT '',
  `created_at` bigint DEFAULT NULL,
  `use_group` varchar(64) DEFAULT '',
  `token_id` bigint DEFAULT '0',
  `channel_id` bigint DEFAULT '0',
  `node_name` varchar(64) DEFAULT '',
  `token_used` bigint DEFAULT '0',
  `count` bigint DEFAULT '0',
  `quota` bigint DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `idx_qdt_created_at` (`created_at`),
  KEY `idx_quota_data_use_group` (`use_group`),
  KEY `idx_quota_data_token_id` (`token_id`),
  KEY `idx_quota_data_channel_id` (`channel_id`),
  KEY `idx_quota_data_node_name` (`node_name`),
  KEY `idx_quota_data_user_id` (`user_id`),
  KEY `idx_qdt_model_user_name` (`model_name`,`username`)
) ENGINE=InnoDB AUTO_INCREMENT=12 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for redemptions
-- ----------------------------
DROP TABLE IF EXISTS `redemptions`;
CREATE TABLE `redemptions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint DEFAULT NULL,
  `key` char(32) DEFAULT NULL,
  `status` bigint DEFAULT '1',
  `name` varchar(191) DEFAULT NULL,
  `quota` bigint DEFAULT '100',
  `created_time` bigint DEFAULT NULL,
  `redeemed_time` bigint DEFAULT NULL,
  `used_user_id` bigint DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `expired_time` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_redemptions_key` (`key`),
  KEY `idx_redemptions_name` (`name`),
  KEY `idx_redemptions_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for setups
-- ----------------------------
DROP TABLE IF EXISTS `setups`;
CREATE TABLE `setups` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `version` varchar(50) NOT NULL,
  `initialized_at` bigint NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for subscription_orders
-- ----------------------------
DROP TABLE IF EXISTS `subscription_orders`;
CREATE TABLE `subscription_orders` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint DEFAULT NULL,
  `plan_id` bigint DEFAULT NULL,
  `money` double DEFAULT NULL,
  `trade_no` varchar(255) DEFAULT NULL,
  `payment_method` varchar(50) DEFAULT NULL,
  `payment_provider` varchar(50) DEFAULT '',
  `status` longtext,
  `create_time` bigint DEFAULT NULL,
  `complete_time` bigint DEFAULT NULL,
  `provider_payload` text,
  PRIMARY KEY (`id`),
  UNIQUE KEY `trade_no` (`trade_no`),
  KEY `idx_subscription_orders_plan_id` (`plan_id`),
  KEY `idx_subscription_orders_trade_no` (`trade_no`),
  KEY `idx_subscription_orders_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for subscription_plans
-- ----------------------------
DROP TABLE IF EXISTS `subscription_plans`;
CREATE TABLE `subscription_plans` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `title` varchar(128) NOT NULL,
  `subtitle` varchar(255) DEFAULT '',
  `price_amount` decimal(10,6) NOT NULL DEFAULT '0.000000',
  `currency` varchar(8) NOT NULL DEFAULT 'USD',
  `duration_unit` varchar(16) NOT NULL DEFAULT 'month',
  `duration_value` bigint NOT NULL DEFAULT '1',
  `custom_seconds` bigint NOT NULL DEFAULT '0',
  `enabled` tinyint(1) DEFAULT '1',
  `sort_order` bigint DEFAULT '0',
  `allow_balance_pay` tinyint(1) DEFAULT NULL,
  `allow_wallet_overflow` tinyint(1) DEFAULT NULL,
  `stripe_price_id` varchar(128) DEFAULT '',
  `creem_product_id` varchar(128) DEFAULT '',
  `waffo_pancake_product_id` varchar(128) DEFAULT '',
  `max_purchase_per_user` bigint DEFAULT '0',
  `upgrade_group` varchar(64) DEFAULT '',
  `downgrade_group` varchar(64) DEFAULT '',
  `total_amount` bigint NOT NULL DEFAULT '0',
  `quota_reset_period` varchar(16) DEFAULT 'never',
  `quota_reset_custom_seconds` bigint DEFAULT '0',
  `created_at` bigint DEFAULT NULL,
  `updated_at` bigint DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for subscription_pre_consume_records
-- ----------------------------
DROP TABLE IF EXISTS `subscription_pre_consume_records`;
CREATE TABLE `subscription_pre_consume_records` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `request_id` varchar(64) DEFAULT NULL,
  `user_id` bigint DEFAULT NULL,
  `user_subscription_id` bigint DEFAULT NULL,
  `pre_consumed` bigint NOT NULL DEFAULT '0',
  `status` varchar(32) DEFAULT NULL,
  `created_at` bigint DEFAULT NULL,
  `updated_at` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_subscription_pre_consume_records_request_id` (`request_id`),
  KEY `idx_subscription_pre_consume_records_user_id` (`user_id`),
  KEY `idx_subscription_pre_consume_records_user_subscription_id` (`user_subscription_id`),
  KEY `idx_subscription_pre_consume_records_status` (`status`),
  KEY `idx_subscription_pre_consume_records_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for system_instances
-- ----------------------------
DROP TABLE IF EXISTS `system_instances`;
CREATE TABLE `system_instances` (
  `node_name` varchar(128) NOT NULL,
  `info` text,
  `started_at` bigint DEFAULT NULL,
  `last_seen_at` bigint DEFAULT NULL,
  `created_at` bigint DEFAULT NULL,
  `updated_at` bigint DEFAULT NULL,
  PRIMARY KEY (`node_name`),
  KEY `idx_system_instances_last_seen_at` (`last_seen_at`),
  KEY `idx_system_instances_created_at` (`created_at`),
  KEY `idx_system_instances_updated_at` (`updated_at`),
  KEY `idx_system_instances_started_at` (`started_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for system_task_locks
-- ----------------------------
DROP TABLE IF EXISTS `system_task_locks`;
CREATE TABLE `system_task_locks` (
  `type` varchar(64) NOT NULL,
  `task_id` varchar(64) DEFAULT NULL,
  `locked_by` varchar(128) DEFAULT NULL,
  `locked_until` bigint DEFAULT NULL,
  `updated_at` bigint DEFAULT NULL,
  PRIMARY KEY (`type`),
  KEY `idx_system_task_locks_task_id` (`task_id`),
  KEY `idx_system_task_locks_locked_by` (`locked_by`),
  KEY `idx_system_task_locks_locked_until` (`locked_until`),
  KEY `idx_system_task_locks_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for system_tasks
-- ----------------------------
DROP TABLE IF EXISTS `system_tasks`;
CREATE TABLE `system_tasks` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `task_id` varchar(64) DEFAULT NULL,
  `type` varchar(64) DEFAULT NULL,
  `status` varchar(32) DEFAULT NULL,
  `active_key` varchar(64) DEFAULT NULL,
  `payload` text,
  `state` text,
  `result` text,
  `error` text,
  `locked_by` varchar(128) DEFAULT NULL,
  `created_at` bigint DEFAULT NULL,
  `updated_at` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_system_tasks_task_id` (`task_id`),
  UNIQUE KEY `idx_system_tasks_active_key` (`active_key`),
  KEY `idx_system_tasks_created_at` (`created_at`),
  KEY `idx_system_tasks_updated_at` (`updated_at`),
  KEY `idx_system_tasks_type` (`type`),
  KEY `idx_system_tasks_status` (`status`),
  KEY `idx_system_tasks_locked_by` (`locked_by`)
) ENGINE=InnoDB AUTO_INCREMENT=373 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for tasks
-- ----------------------------
DROP TABLE IF EXISTS `tasks`;
CREATE TABLE `tasks` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `created_at` bigint DEFAULT NULL,
  `updated_at` bigint DEFAULT NULL,
  `task_id` varchar(191) DEFAULT NULL,
  `platform` varchar(30) DEFAULT NULL,
  `user_id` bigint DEFAULT NULL,
  `group` varchar(50) DEFAULT NULL,
  `channel_id` bigint DEFAULT NULL,
  `quota` bigint DEFAULT NULL,
  `action` varchar(40) DEFAULT NULL,
  `status` varchar(20) DEFAULT NULL,
  `fail_reason` longtext,
  `submit_time` bigint DEFAULT NULL,
  `start_time` bigint DEFAULT NULL,
  `finish_time` bigint DEFAULT NULL,
  `progress` varchar(20) DEFAULT NULL,
  `properties` json DEFAULT NULL,
  `private_data` json DEFAULT NULL,
  `data` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_tasks_action` (`action`),
  KEY `idx_tasks_submit_time` (`submit_time`),
  KEY `idx_tasks_task_id` (`task_id`),
  KEY `idx_tasks_platform` (`platform`),
  KEY `idx_tasks_status` (`status`),
  KEY `idx_tasks_start_time` (`start_time`),
  KEY `idx_tasks_finish_time` (`finish_time`),
  KEY `idx_tasks_progress` (`progress`),
  KEY `idx_tasks_created_at` (`created_at`),
  KEY `idx_tasks_user_id` (`user_id`),
  KEY `idx_tasks_channel_id` (`channel_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for tokens
-- ----------------------------
DROP TABLE IF EXISTS `tokens`;
CREATE TABLE `tokens` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint DEFAULT NULL,
  `key` varchar(128) DEFAULT NULL,
  `status` bigint DEFAULT '1',
  `name` varchar(191) DEFAULT NULL,
  `created_time` bigint DEFAULT NULL,
  `accessed_time` bigint DEFAULT NULL,
  `expired_time` bigint DEFAULT '-1',
  `remain_quota` bigint DEFAULT '0',
  `unlimited_quota` tinyint(1) DEFAULT NULL,
  `model_limits_enabled` tinyint(1) DEFAULT NULL,
  `model_limits` text,
  `allow_ips` varchar(191) DEFAULT '',
  `used_quota` bigint DEFAULT '0',
  `group` varchar(191) DEFAULT '',
  `cross_group_retry` tinyint(1) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_tokens_key` (`key`),
  KEY `idx_tokens_user_id` (`user_id`),
  KEY `idx_tokens_name` (`name`),
  KEY `idx_tokens_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for top_ups
-- ----------------------------
DROP TABLE IF EXISTS `top_ups`;
CREATE TABLE `top_ups` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint DEFAULT NULL,
  `amount` bigint DEFAULT NULL,
  `money` double DEFAULT NULL,
  `trade_no` varchar(255) DEFAULT NULL,
  `payment_method` varchar(50) DEFAULT NULL,
  `payment_provider` varchar(50) DEFAULT '',
  `create_time` bigint DEFAULT NULL,
  `complete_time` bigint DEFAULT NULL,
  `status` longtext,
  PRIMARY KEY (`id`),
  UNIQUE KEY `trade_no` (`trade_no`),
  KEY `idx_top_ups_user_id` (`user_id`),
  KEY `idx_top_ups_trade_no` (`trade_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for two_fa_backup_codes
-- ----------------------------
DROP TABLE IF EXISTS `two_fa_backup_codes`;
CREATE TABLE `two_fa_backup_codes` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL,
  `code_hash` varchar(255) NOT NULL,
  `is_used` tinyint(1) DEFAULT NULL,
  `used_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_two_fa_backup_codes_user_id` (`user_id`),
  KEY `idx_two_fa_backup_codes_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for two_fas
-- ----------------------------
DROP TABLE IF EXISTS `two_fas`;
CREATE TABLE `two_fas` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL,
  `secret` varchar(255) NOT NULL,
  `is_enabled` tinyint(1) DEFAULT NULL,
  `failed_attempts` bigint DEFAULT '0',
  `locked_until` datetime(3) DEFAULT NULL,
  `last_used_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_id` (`user_id`),
  KEY `idx_two_fas_user_id` (`user_id`),
  KEY `idx_two_fas_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for user_file
-- ----------------------------
DROP TABLE IF EXISTS `user_file`;
CREATE TABLE `user_file` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 Id',
  `file_id` bigint unsigned NOT NULL,
  `file_channel_id` bigint unsigned NOT NULL,
  `file_name` varchar(1000) NOT NULL,
  `file_suffix` varchar(255) NOT NULL DEFAULT '',
  `user_id` bigint NOT NULL,
  `create_time` datetime(3) DEFAULT NULL,
  `update_time` datetime(3) DEFAULT NULL,
  `source` varchar(64) NOT NULL DEFAULT '',
  `status` varchar(1) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_file` (`file_id`,`user_id`),
  KEY `idx_user_file_file_id` (`file_id`),
  KEY `idx_user_file_file_channel_id` (`file_channel_id`),
  KEY `idx_user_suffix` (`user_id`,`file_suffix`),
  KEY `idx_user_status` (`user_id`,`status`),
  KEY `idx_user_id_create_time` (`user_id`,`create_time`)
) ENGINE=InnoDB AUTO_INCREMENT=14 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户文件';

-- ----------------------------
-- Table structure for user_oauth_bindings
-- ----------------------------
DROP TABLE IF EXISTS `user_oauth_bindings`;
CREATE TABLE `user_oauth_bindings` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint NOT NULL,
  `provider_id` bigint NOT NULL,
  `provider_user_id` varchar(256) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ux_user_provider` (`user_id`,`provider_id`),
  UNIQUE KEY `ux_provider_userid` (`provider_id`,`provider_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for user_subscriptions
-- ----------------------------
DROP TABLE IF EXISTS `user_subscriptions`;
CREATE TABLE `user_subscriptions` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `user_id` bigint DEFAULT NULL,
  `plan_id` bigint DEFAULT NULL,
  `amount_total` bigint NOT NULL DEFAULT '0',
  `amount_used` bigint NOT NULL DEFAULT '0',
  `start_time` bigint DEFAULT NULL,
  `end_time` bigint DEFAULT NULL,
  `status` varchar(32) DEFAULT NULL,
  `source` varchar(32) DEFAULT 'order',
  `last_reset_time` bigint DEFAULT '0',
  `next_reset_time` bigint DEFAULT '0',
  `upgrade_group` varchar(64) DEFAULT '',
  `prev_user_group` varchar(64) DEFAULT '',
  `downgrade_group` varchar(64) DEFAULT '',
  `allow_wallet_overflow` tinyint(1) DEFAULT NULL,
  `created_at` bigint DEFAULT NULL,
  `updated_at` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_user_subscriptions_plan_id` (`plan_id`),
  KEY `idx_user_subscriptions_end_time` (`end_time`),
  KEY `idx_user_subscriptions_status` (`status`),
  KEY `idx_user_subscriptions_next_reset_time` (`next_reset_time`),
  KEY `idx_user_subscriptions_user_id` (`user_id`),
  KEY `idx_user_sub_active` (`user_id`,`status`,`end_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for users
-- ----------------------------
DROP TABLE IF EXISTS `users`;
CREATE TABLE `users` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `username` varchar(191) DEFAULT NULL,
  `password` longtext NOT NULL,
  `display_name` varchar(191) DEFAULT NULL,
  `role` bigint DEFAULT '1',
  `status` bigint DEFAULT '1',
  `email` varchar(191) DEFAULT NULL,
  `github_id` varchar(191) DEFAULT NULL,
  `discord_id` varchar(191) DEFAULT NULL,
  `oidc_id` varchar(191) DEFAULT NULL,
  `wechat_id` varchar(191) DEFAULT NULL,
  `telegram_id` varchar(191) DEFAULT NULL,
  `access_token` char(32) DEFAULT NULL,
  `quota` bigint DEFAULT '0',
  `used_quota` bigint DEFAULT '0',
  `request_count` bigint DEFAULT '0',
  `group` varchar(64) DEFAULT 'default',
  `aff_code` varchar(32) DEFAULT NULL,
  `aff_count` bigint DEFAULT '0',
  `aff_quota` bigint DEFAULT '0',
  `aff_history` bigint DEFAULT '0',
  `inviter_id` bigint DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  `linux_do_id` varchar(191) DEFAULT NULL,
  `setting` text,
  `remark` varchar(255) DEFAULT NULL,
  `stripe_customer` varchar(64) DEFAULT NULL,
  `created_at` bigint DEFAULT NULL,
  `last_login_at` bigint DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `username` (`username`),
  UNIQUE KEY `idx_users_access_token` (`access_token`),
  UNIQUE KEY `idx_users_aff_code` (`aff_code`),
  KEY `idx_users_we_chat_id` (`wechat_id`),
  KEY `idx_users_telegram_id` (`telegram_id`),
  KEY `idx_users_stripe_customer` (`stripe_customer`),
  KEY `idx_users_username` (`username`),
  KEY `idx_users_email` (`email`),
  KEY `idx_users_discord_id` (`discord_id`),
  KEY `idx_users_inviter_id` (`inviter_id`),
  KEY `idx_users_deleted_at` (`deleted_at`),
  KEY `idx_users_linux_do_id` (`linux_do_id`),
  KEY `idx_users_display_name` (`display_name`),
  KEY `idx_users_git_hub_id` (`github_id`),
  KEY `idx_users_oidc_id` (`oidc_id`)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- ----------------------------
-- Table structure for vendors
-- ----------------------------
DROP TABLE IF EXISTS `vendors`;
CREATE TABLE `vendors` (
  `id` bigint NOT NULL AUTO_INCREMENT,
  `name` varchar(128) NOT NULL,
  `description` text,
  `icon` varchar(128) DEFAULT NULL,
  `status` bigint DEFAULT '1',
  `created_time` bigint DEFAULT NULL,
  `updated_time` bigint DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_vendor_name_delete_at` (`name`,`deleted_at`),
  KEY `idx_vendors_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

SET FOREIGN_KEY_CHECKS = 1;
