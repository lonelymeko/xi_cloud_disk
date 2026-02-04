-- =============================================
-- 玺云盘 - 分布式云存储系统 数据库脚本
-- 数据库名：cloud_disk
-- 编码格式：utf8mb4（兼容 emoji 等特殊字符，比 utf8 更全面）
-- 存储引擎：InnoDB（支持事务、外键，适合业务系统）
-- =============================================

-- 1. 创建数据库（如果不存在），并指定编码和排序规则
CREATE DATABASE IF NOT EXISTS `cloud_disk` 
DEFAULT CHARACTER SET utf8mb4 
DEFAULT COLLATE utf8mb4_general_ci;

-- 2. 切换到该数据库
USE `cloud_disk`;

-- 3. 创建用户基础信息表（user_basic）
DROP TABLE IF EXISTS `user_basic`;
CREATE TABLE `user_basic` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键ID',
  `identity` varchar(36) DEFAULT NULL COMMENT '用户唯一标识（UUID，避免主键暴露）',
  `name` varchar(60) DEFAULT NULL COMMENT '用户名',
  `password` varchar(32) DEFAULT NULL COMMENT '用户密码（建议加密存储，如MD5、BCrypt）',
  `email` varchar(100) DEFAULT NULL COMMENT '用户邮箱（用于登录、找回密码）',
  `created_at` datetime DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
  `deleted_at` datetime DEFAULT NULL COMMENT '删除时间（软删除，标记而非物理删除）',
  PRIMARY KEY (`id`) COMMENT '主键索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户基础信息表';

-- 4. 创建文件存储池表（repository_pool）
DROP TABLE IF EXISTS `repository_pool`;
CREATE TABLE `repository_pool` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键ID',
  `identity` varchar(36) DEFAULT NULL COMMENT '文件存储唯一标识（UUID）',
  `hash` varchar(32) DEFAULT NULL COMMENT '文件唯一哈希值（用于去重，避免相同文件重复存储）',
  `name` varchar(255) DEFAULT NULL COMMENT '文件原始名称',
  `ext` varchar(30) DEFAULT NULL COMMENT '文件扩展名（如 mp4、jpg、docx）',
  `size` double DEFAULT NULL COMMENT '文件大小（单位：字节）',
  `path` varchar(255) DEFAULT NULL COMMENT '文件存储的物理路径（服务器本地或对象存储地址）',
  `created_at` datetime DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
  `deleted_at` datetime DEFAULT NULL COMMENT '删除时间（软删除）',
  PRIMARY KEY (`id`) COMMENT '主键索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文件存储池表（存储文件核心物理信息，用于去重）';

-- 5. 创建用户文件关联表（user_repository）
DROP TABLE IF EXISTS `user_repository`;
CREATE TABLE `user_repository` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键ID',
  `identity` varchar(36) DEFAULT NULL COMMENT '用户文件关联唯一标识（UUID）',
  `user_identity` varchar(36) DEFAULT NULL COMMENT '关联的用户唯一标识（对应 user_basic.identity）',
  `parent_id` int(11) DEFAULT NULL COMMENT '父级文件夹ID（0 表示根目录，用于构建文件目录结构）',
  `repository_identity` varchar(36) DEFAULT NULL COMMENT '关联的文件存储唯一标识（对应 repository_pool.identity）',
  `ext` varchar(255) DEFAULT NULL COMMENT '类型标识（文件/文件夹，可填 file/folder）',
  `name` varchar(255) DEFAULT NULL COMMENT '用户显示的文件/文件夹名称（可重命名，与原始文件名无关）',
  `created_at` datetime DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
  `deleted_at` datetime DEFAULT NULL COMMENT '删除时间（软删除）',
  PRIMARY KEY (`id`) COMMENT '主键索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户文件关联表（构建用户的个人文件目录，支持重命名、文件夹结构）';

-- 6. 创建文件分享表（share_basic）
DROP TABLE IF EXISTS `share_basic`;
CREATE TABLE `share_basic` (
  `id` int(11) unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键ID',
  `identity` varchar(36) DEFAULT NULL COMMENT '分享唯一标识（UUID）',
  `user_identity` varchar(36) DEFAULT NULL COMMENT '分享者用户唯一标识（对应 user_basic.identity）',
  `repository_identity` varchar(36) DEFAULT NULL COMMENT '关联的文件存储唯一标识（对应 repository_pool.identity）',
  `expired_time` int(11) DEFAULT NULL COMMENT '失效时间（单位：秒，如 86400 表示 24 小时后失效，0 表示永久有效）',
  `created_at` datetime DEFAULT NULL COMMENT '创建时间',
  `updated_at` datetime DEFAULT NULL COMMENT '更新时间',
  `deleted_at` datetime DEFAULT NULL COMMENT '删除时间（软删除）',
  PRIMARY KEY (`id`) COMMENT '主键索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='文件分享表（存储文件分享的相关信息）';

-- =============================================
-- 脚本执行完成提示
-- =============================================
SELECT 'cloud_disk 数据库及 4 张表创建成功！' AS `执行结果`;