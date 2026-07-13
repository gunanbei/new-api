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

 Date: 12/07/2026 18:40:00
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for file
-- 物理文件（跨用户秒传实体）
-- channel_type 码值与 file_upload_channel.type 一致：
--   0-本地存储 1-Webdav 2-Cloudflare-ImageBed 3-S3存储
-- 秒传唯一键：uk_identifier_channel_type (identifier, channel_type)
-- status：0-上传中 1-可用 2-待删除
-- object_key：各 Provider 稳定路径（S3 key / ImgBed path / WebDAV path / 本地相对路径）
-- file_url：可访问 URL（公开前缀拼接或稳定直链；私有读可为空，运行时签发）
-- ----------------------------
DROP TABLE IF EXISTS `file`;
CREATE TABLE `file` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 Id',
  `file_channel_id` bigint NOT NULL DEFAULT '0' COMMENT '实际上传使用的 file_upload_channel.id',
  `channel_type` varchar(1) NOT NULL COMMENT '渠道类型（与 file_upload_channel.type 码值一致）\n0-本地存储\n1-Webdav\n2-Cloudflare-ImageBed\n3-S3存储',
  `file_size` bigint NOT NULL DEFAULT '0' COMMENT '文件大小（字节）',
  `object_key` varchar(512) NOT NULL DEFAULT '' COMMENT '对象存储路径/键（稳定定位，删改物理文件用）',
  `file_url` varchar(1000) NOT NULL DEFAULT '' COMMENT '文件访问 URL（可为空，由 public_base_url + object_key 运行时拼接）',
  `identifier` varchar(255) NOT NULL COMMENT '文件 MD5，用于跨用户秒传',
  `mime_type` varchar(128) NOT NULL DEFAULT '' COMMENT 'MIME 类型，如 image/png',
  `etag` varchar(128) NOT NULL DEFAULT '' COMMENT '存储方 ETag（S3 等 complete/校验用）',
  `ref_count` int NOT NULL DEFAULT '0' COMMENT '被 user_file 引用次数；归零后可删物理对象',
  `status` varchar(1) NOT NULL DEFAULT '1' COMMENT '状态\n0-上传中\n1-可用\n2-待删除',
  `creater_user_id` bigint NOT NULL DEFAULT '0' COMMENT '首个上传者 Id（秒传引用不改此字段）',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_identifier_channel_type` (`identifier`, `channel_type`),
  KEY `idx_file_channel_id` (`file_channel_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='文件存储（物理对象）';

-- ----------------------------
-- Records of file
-- ----------------------------
BEGIN;
COMMIT;

SET FOREIGN_KEY_CHECKS = 1;
