/*
 Navicat Premium Data Transfer

 Source Server         : 10.0.0.200
 Source Server Type    : MariaDB
 Source Server Version : 100523 (10.5.23-MariaDB)
 Source Host           : 10.0.0.200:3306
 Source Schema         : iptv_spider

 Target Server Type    : MariaDB
 Target Server Version : 100523 (10.5.23-MariaDB)
 File Encoding         : 65001

 Date: 09/12/2025 14:32:40
*/

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ----------------------------
-- Table structure for m3u8_mappings
-- ----------------------------
DROP TABLE IF EXISTS `m3u8_mappings`;
CREATE TABLE `m3u8_mappings` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime DEFAULT NULL,
  `updated_at` datetime DEFAULT NULL,
  `deleted_at` datetime DEFAULT NULL,
  `comm_name` varchar(191) DEFAULT NULL COMMENT '节目通用名称',
  `logo` varchar(191) DEFAULT NULL COMMENT 'Logo 地址',
  `auto_groups` varchar(191) DEFAULT NULL COMMENT '自动分组',
  `custom_groups` varchar(191) DEFAULT NULL COMMENT '自定义组分类',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_m3u8_mappings_comm_name` (`comm_name`),
  KEY `idx_m3u8_mappings_deleted_at` (`deleted_at`)
) ENGINE=InnoDB AUTO_INCREMENT=104582 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

-- ----------------------------
-- Records of m3u8_mappings
-- ----------------------------
BEGIN;
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (1, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '新闻综合', 'https://epg.deny.vip/df/新闻综合.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (2, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '东方卫视', 'https://epg.deny.vip/ws/dongfang.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (3, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '都市频道', 'https://epg.deny.vip/df/都市频道.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (4, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '纪实人文', 'https://epg.deny.vip/df/纪实人文.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (5, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '东方影视', 'https://epg.deny.vip/df/东方影视.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (6, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '第一财经', 'https://epg.deny.vip/df/第一财经.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (7, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '体育频道', 'https://epg.deny.vip/df/五星体育.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (8, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '哈哈炫动', 'https://epg.deny.vip/df/哈哈炫动.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (9, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '外语频道', 'https://epg.deny.vip/df/外语频道.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (10, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '东方购物-1', '', '购物', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (11, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '东方购物-2', '', '购物', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (12, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '上海教育', 'https://epg.deny.vip/df/上海教育.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (13, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '法治天地', 'https://epg.deny.vip/df/法治天地.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (14, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '游戏风云', 'https://epg.deny.vip/df/游戏风云.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (15, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '金色学堂', 'https://epg.deny.vip/df/金色学堂.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (16, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '央广购物', '', '购物', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (17, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '家有购物', '', '购物', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (18, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '聚鲨环球精选', '', '', '购物');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (19, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '家家购物', '', '购物', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (20, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '快乐购物', '', '购物', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (21, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-1', 'https://epg.deny.vip/cctv/CCTV-1.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (22, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-2', 'https://epg.deny.vip/cctv/CCTV-2.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (23, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-3', 'https://epg.deny.vip/cctv/CCTV-3.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (24, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-4', 'https://epg.deny.vip/cctv/CCTV-4.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (25, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-5', 'https://epg.deny.vip/cctv/CCTV-5.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (26, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-6', 'https://epg.deny.vip/cctv/CCTV-6.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (27, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-7', 'https://epg.deny.vip/cctv/CCTV-7.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (28, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-8', 'https://epg.deny.vip/cctv/CCTV-8.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (29, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CGTN', 'https://epg.deny.vip/cctv/CGTN.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (30, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-10', 'https://epg.deny.vip/cctv/CCTV-10.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (31, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-11', 'https://epg.deny.vip/cctv/CCTV-11.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (32, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-12', 'https://epg.deny.vip/cctv/CCTV-12.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (33, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-13', 'https://epg.deny.vip/cctv/CCTV-13.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (34, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-14', 'https://epg.deny.vip/cctv/CCTV-14.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (35, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-15', 'https://epg.deny.vip/cctv/CCTV-15.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (36, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-17', 'https://epg.deny.vip/cctv/CCTV-17.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (37, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '优购物', '', '购物', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (38, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '好享购物', '', '购物', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (39, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '中视购物', '', '购物', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (40, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '风尚购物', '', '购物', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (41, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-9', 'https://epg.deny.vip/cctv/CCTV-9.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (42, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '财富天下', 'https://epg.deny.vip/df/财富天下.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (43, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '快乐垂钓', 'https://epg.deny.vip/df/快乐垂钓.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (44, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '茶频道', 'https://epg.deny.vip/df/茶频道.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (45, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '足球频道', 'https://epg.deny.vip/df/足球频道.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (47, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '生活时尚', 'https://epg.deny.vip/df/生活时尚.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (48, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '动漫秀场', 'https://epg.deny.vip/df/动漫秀场.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (49, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '乐游', 'https://epg.deny.vip/df/乐游.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (50, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '都市剧场', 'https://epg.deny.vip/df/都市剧场.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (52, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '欢笑剧场', 'https://epg.deny.vip/df/欢笑剧场4K.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (59, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '五星体育', 'https://epg.deny.vip/df/五星体育.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (66, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '七彩戏剧', 'https://epg.deny.vip/df/七彩戏剧.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (67, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '东方财经', 'https://epg.deny.vip/df/东方财经.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (69, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-5+', 'https://epg.deny.vip/cctv/CCTV-5+.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (84, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, 'CCTV-16', 'https://epg.deny.vip/cctv/CCTV-16.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (86, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '浙江卫视', 'https://epg.deny.vip/ws/zhejiang.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (87, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '江苏卫视', 'https://epg.deny.vip/ws/jiangsu.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (88, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '湖南卫视', 'https://epg.deny.vip/ws/hunan.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (89, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '北京卫视', 'https://epg.deny.vip/ws/beijing.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (90, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '广东卫视', 'https://epg.deny.vip/ws/guangdong.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (91, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '深圳卫视', 'https://epg.deny.vip/ws/shenzhen.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (92, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '黑龙江卫视', 'https://epg.deny.vip/ws/heilongjiang.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (93, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '山东卫视', 'https://epg.deny.vip/ws/shandong.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (94, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '湖北卫视', 'https://epg.deny.vip/ws/hubei.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (95, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '安徽卫视', 'https://epg.deny.vip/ws/anhui.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (96, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '东南卫视', 'https://epg.deny.vip/ws/dongnan.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (97, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '江西卫视', 'https://epg.deny.vip/ws/jiangxi.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (98, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '辽宁卫视', 'https://epg.deny.vip/ws/liaoning.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (99, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '天津卫视', 'https://epg.deny.vip/ws/tianjin.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (100, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '中国教育-1', 'https://epg.deny.vip/cctv/CETV-1.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (101, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '四川卫视', 'https://epg.deny.vip/ws/sichuan.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (102, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '重庆卫视', 'https://epg.deny.vip/ws/chongqing.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (103, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '贵州卫视', 'https://epg.deny.vip/ws/guizhou.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (104, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '海南卫视', 'https://epg.deny.vip/ws/hainan.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (105, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '河北卫视', 'https://epg.deny.vip/ws/hebei.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (106, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '金鹰纪实', 'https://epg.deny.vip/df/金鹰纪实.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (107, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '纪实科教', 'https://epg.deny.vip/df/BTV-科教.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (108, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '河南卫视', 'https://epg.deny.vip/ws/henan.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (109, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '云南卫视', 'https://epg.deny.vip/ws/yunnan.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (110, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '广西卫视', 'https://epg.deny.vip/ws/guangxi.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (111, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '吉林卫视', 'https://epg.deny.vip/ws/jilin.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (112, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '卡酷少儿', 'https://epg.deny.vip/df/卡酷少儿.png', '', '卡通');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (113, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '甘肃卫视', 'https://epg.deny.vip/ws/gansu.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (114, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '中国教育-4', 'https://epg.deny.vip/cctv/CETV-4.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (117, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '高清导视频道', '', '', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (118, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '宁夏卫视', 'https://epg.deny.vip/ws/ningxia.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (133, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '山西卫视', 'https://epg.deny.vip/ws/shanxi_.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (134, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '青海卫视', 'https://epg.deny.vip/ws/qinghai.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (135, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '西藏卫视', 'https://epg.deny.vip/ws/xizang.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (136, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '陕西卫视', 'https://epg.deny.vip/ws/shanxi.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (142, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '内蒙古卫视', 'https://epg.deny.vip/ws/neimeng.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (148, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '新疆卫视', 'https://epg.deny.vip/ws/xinjiang.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (149, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '兵团卫视', 'https://epg.deny.vip/ws/bingtuan.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (150, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '三沙卫视', 'https://epg.deny.vip/ws/sansha.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (151, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '金鹰卡通', 'https://epg.deny.vip/df/金鹰卡通.png', '', '卡通');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (152, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '嘉佳卡通', 'https://epg.deny.vip/df/嘉佳卡通.png', '', '卡通');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (153, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '卡酷卡通', 'https://epg.deny.vip/df/卡酷少儿.png', '', '卡通');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (155, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '中国教育-2', 'https://epg.deny.vip/cctv/CETV-2.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (156, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '中国天气', 'https://epg.deny.vip/df/中国天气.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (157, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '家庭理财', 'https://epg.deny.vip/df/家庭理财.png', '', '地方');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (158, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '一年级', '', '空中课堂', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (159, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '二年级', '', '空中课堂', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (160, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '三年级', '', '空中课堂', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (161, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '四年级', '', '空中课堂', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (162, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '五年级', '', '空中课堂', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (163, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '六年级', '', '空中课堂', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (164, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '七年级', '', '空中课堂', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (165, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '八年级', '', '空中课堂', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (166, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '九年级', '', '空中课堂', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (167, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '高一年级', '', '空中课堂', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (168, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '高二年级', '', '空中课堂', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (169, '2023-03-09 05:48:35', '2023-03-09 05:48:35', NULL, '高三年级', '', '空中课堂', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (170, '2023-08-16 16:00:00', '2023-08-16 16:00:00', NULL, '延边卫视', 'https://epg.deny.vip/ws/yanbian.png', '卫视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (171, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, 'CHC家庭影院', 'https://epg.deny.vip/szpd/CHC2.png', '', 'CHC');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (172, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, 'CHC动作电影', 'https://epg.deny.vip/szpd/CHC3.png', '', 'CHC');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (173, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, 'CHC高清电影', 'https://epg.deny.vip/szpd/CHC1.png', '', 'CHC');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (174, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, '风云足球', 'https://epg.deny.vip/cctv/CCTV-FYZQ.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (175, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, '央视台球', 'https://epg.deny.vip/cctv/CCTV-YSTQ.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (176, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, '兵器科技', 'https://epg.deny.vip/cctv/CCTV-BQKJ.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (177, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, '世界地理', 'https://epg.deny.vip/cctv/CCTV-SJDL.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (178, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, '女性时尚', 'https://epg.deny.vip/cctv/CCTV-NXSS.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (179, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, '高尔夫网球', 'https://epg.deny.vip/cctv/CCTV-GEFWQ.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (180, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, '怀旧剧场', 'https://epg.deny.vip/cctv/CCTV-HJJC.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (181, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, '风云剧场', 'https://epg.deny.vip/cctv/CCTV-FYJC.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (182, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, '第一剧场', 'https://epg.deny.vip/cctv/CCTV-DYJC.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (183, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, '风云音乐', 'https://epg.deny.vip/cctv/CCTV-FYYY.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (184, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, '央视文化精品', 'https://epg.deny.vip/cctv/CCTV-YSWHJP.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (185, '2024-03-26 15:05:57', '2024-03-26 15:05:57', NULL, '早期教育', 'https://epg.deny.vip/szpd/CETV-ZQJY.png', '', '央视');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (186, '2024-03-26 15:05:58', '2024-03-26 15:05:58', NULL, 'CCTV-4K', 'https://epg.deny.vip/cctv/CCTV-4K.png', '央视', '');
INSERT INTO `m3u8_mappings` (`id`, `created_at`, `updated_at`, `deleted_at`, `comm_name`, `logo`, `auto_groups`, `custom_groups`) VALUES (187, '2024-04-29 16:00:00', '2024-04-29 16:00:00', NULL, 'CHC影迷电影', 'https://epg.deny.vip/szpd/CHC1.png', '', 'CHC');
COMMIT;

SET FOREIGN_KEY_CHECKS = 1;
