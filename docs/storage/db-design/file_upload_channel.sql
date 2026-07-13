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
-- Table structure for file_upload_channel
-- 渠道类型 type 码值（本表为准）：
--   0-本地存储
--   1-Webdav
--   2-Cloudflare-ImageBed
--   3-S3存储
-- config_proflle：按 type 存放 JSON，密钥仅服务端使用、管理端读出脱敏
--   type=0 local:  {"root_path":"/data/files","public_base_url":""}
--   type=1 webdav: {"base_url":"https://dav.example.com/.../","username":"...","password":"...","auth_type":"basic","path_prefix":"playground/","public_base_url":"","timeout_ms":600000}
--   type=2 imgbed: {"base_url":"https://your.imgbed.domain","api_token":"...","upload_channel":"cfr2","upload_folder":"playground","return_format":"full"}
--   type=3 s3:     {"endpoint":"...","region":"auto","bucket":"...","access_key_id":"...","secret_access_key":"...","force_path_style":true,"public_base_url":"...","key_prefix":"files/"}
-- ----------------------------
DROP TABLE IF EXISTS `file_upload_channel`;
CREATE TABLE `file_upload_channel` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 Id',
  `name` varchar(255) NOT NULL DEFAULT '' COMMENT '文件上传渠道名称',
  `type` varchar(1) NOT NULL COMMENT '文件上传渠道类型\n0-本地存储\n1-Webdav\n2-Cloudflare-ImageBed\n3-S3存储',
  `status` varchar(1) NOT NULL DEFAULT '1' COMMENT '状态\n0-停用\n1-启用',
  `is_default` varchar(1) NOT NULL DEFAULT '0' COMMENT '是否默认渠道（全局同时仅一条为 1）\n0-否\n1-是',
  `chunk_threshold` bigint NOT NULL DEFAULT '16777216' COMMENT '超过该字节数则分片上传；Webdav/本地可与 max_size 相同表示不走通用分片',
  `chunk_size` bigint NOT NULL DEFAULT '8388608' COMMENT '分片大小（字节）',
  `max_size` bigint NOT NULL DEFAULT '0' COMMENT '单文件上限（字节），0 表示不在此限制',
  `config_proflle` text NOT NULL COMMENT '当前渠道的 Json 配置，理论上同一类型渠道的 key 一致',
  `create_user_id` bigint NOT NULL COMMENT '创建者 Id',
  `update_user_id` bigint NOT NULL COMMENT '更新者用户 Id',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_type_status` (`type`, `status`),
  KEY `idx_is_default` (`is_default`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='文件上传渠道';

-- ----------------------------
-- Records of file_upload_channel
-- ----------------------------
BEGIN;
COMMIT;

SET FOREIGN_KEY_CHECKS = 1;
