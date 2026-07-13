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

 Date: 12/07/2026 18:28:05
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

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
-- Records of model_ability
-- ----------------------------
BEGIN;
COMMIT;

SET FOREIGN_KEY_CHECKS = 1;
