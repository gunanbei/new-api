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
-- Table structure for user_file
-- 用户逻辑文件：跨用户秒传时多人可指向同一 file.id
-- status：0-上传中 1-可用
-- uk_user_file：同一用户不重复挂载同一物理文件
-- ----------------------------
DROP TABLE IF EXISTS `user_file`;
CREATE TABLE `user_file` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键 Id',
  `file_id` bigint NOT NULL COMMENT '文件 Id（file.id）',
  `file_channel_id` bigint NOT NULL COMMENT '文件上传渠道 Id（file_upload_channel.id）',
  `file_name` varchar(1000) NOT NULL COMMENT '文件名称,由流自动获取，不包含后缀',
  `file_suffix` varchar(255) NOT NULL DEFAULT '' COMMENT '文件后缀（不含点，如 png）',
  `user_id` bigint NOT NULL COMMENT '用户 Id',
  `source` varchar(64) NOT NULL DEFAULT '' COMMENT '来源标识，如 playground / manual',
  `status` varchar(1) NOT NULL DEFAULT '1' COMMENT '状态\n0-上传中\n1-可用',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_file` (`user_id`, `file_id`),
  KEY `idx_user_id_create_time` (`user_id`, `create_time`),
  KEY `idx_file_id` (`file_id`),
  KEY `idx_file_channel_id` (`file_channel_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户文件';

-- ----------------------------
-- Records of user_file
-- ----------------------------
BEGIN;
COMMIT;

SET FOREIGN_KEY_CHECKS = 1;
