/*
MySQL: Database - user_management
*********************************************************************
*/
use user_management;

CREATE TABLE IF NOT EXISTS `t_group` (
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_group_id` char(40) NOT NULL COMMENT '用户组唯一标识',
    `f_group_name` varchar(512) NOT NULL COMMENT '用户组名',
    `f_created_time` bigint(40) NOT NULL COMMENT '用户组创建时间',
    `f_notes` varchar(1200) NOT NULL COMMENT '备注',
    PRIMARY KEY (`f_primary_id`)
) ENGINE=InnoDB COMMENT='AnyShare用户组信息表';


CREATE TABLE IF NOT EXISTS `t_group_member` (
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_group_id` char(40) NOT NULL COMMENT '用户组唯一标识',
    `f_member_id` char(40) NOT NULL COMMENT '用户唯一标识,可以是用户，可以是部门',
    `f_member_type` tinyint(4) NOT NULL COMMENT '用户类型,0：部门：1：用户',
    `f_added_time` bigint(40) NOT NULL COMMENT '用户组成员添加时间',
    PRIMARY KEY (`f_primary_id`)
) ENGINE=InnoDB COMMENT='AnyShare用户组成员信息表';

CREATE TABLE IF NOT EXISTS `t_anonymity` (
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_anonymity_id` char(40) NOT NULL COMMENT '匿名账户id',
    `f_password` varchar(100) NOT NULL COMMENT '访问密码',
    `f_expires_at` bigint(40) NOT NULL COMMENT '到期时间, 0为永久有效',
    `f_limited_times` bigint(40) NOT NULL COMMENT '访问限制次数, -1为无限制',
    `f_accessed_times` bigint(40) NOT NULL DEFAULT '0' COMMENT '已访问次数',
    `f_created_at` bigint(40) NOT NULL COMMENT '生成时间',
    `f_type`  char(40) NOT NULL COMMENT '匿名账户类型 example:document 文档匿名用户',
    `f_verify_mobile` tinyint(4) NOT NULL DEFAULT '0' COMMENT '是否手机验证 , 0: 否，1: 是',
    PRIMARY KEY (`f_primary_id`),
    UNIQUE KEY `idx_id` (`f_anonymity_id`),
    KEY `idx_expires_at` (`f_expires_at`)
) ENGINE=InnoDB COMMENT='匿名账户表';

CREATE TABLE IF NOT EXISTS `t_outbox` (
    `f_id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '自增主键',
    `f_business_type` tinyint(4) NOT NULL COMMENT '业务类型',
    `f_message` longtext NOT NULL COMMENT '消息内容，json格式字符串',
    `f_create_time` bigint(20) NOT NULL COMMENT '消息创建时间',
    PRIMARY KEY (`f_id`),
    KEY `idx_business_type_and_create_time` (`f_business_type`, `f_create_time`)
) ENGINE=InnoDB COMMENT='outbox信息表';

CREATE TABLE IF NOT EXISTS `t_outbox_lock` (
    `f_business_type` tinyint(4) NOT NULL COMMENT '业务类型',
    PRIMARY KEY (`f_business_type`)
) ENGINE=InnoDB COMMENT='outbox分布式锁表';

INSERT INTO t_outbox_lock(f_business_type) SELECT 1 FROM DUAL WHERE NOT EXISTS(SELECT f_business_type FROM t_outbox_lock WHERE f_business_type = 1);
INSERT INTO t_outbox_lock(f_business_type) SELECT 2 FROM DUAL WHERE NOT EXISTS(SELECT f_business_type FROM t_outbox_lock WHERE f_business_type = 2);
INSERT INTO t_outbox_lock(f_business_type) SELECT 3 FROM DUAL WHERE NOT EXISTS(SELECT f_business_type FROM t_outbox_lock WHERE f_business_type = 3);
INSERT INTO t_outbox_lock(f_business_type) SELECT 4 FROM DUAL WHERE NOT EXISTS(SELECT f_business_type FROM t_outbox_lock WHERE f_business_type = 4);
INSERT INTO t_outbox_lock(f_business_type) SELECT 5 FROM DUAL WHERE NOT EXISTS(SELECT f_business_type FROM t_outbox_lock WHERE f_business_type = 5);
INSERT INTO t_outbox_lock(f_business_type) SELECT 6 FROM DUAL WHERE NOT EXISTS(SELECT f_business_type FROM t_outbox_lock WHERE f_business_type = 6);
INSERT INTO t_outbox_lock(f_business_type) SELECT 7 FROM DUAL WHERE NOT EXISTS(SELECT f_business_type FROM t_outbox_lock WHERE f_business_type = 7);
INSERT INTO t_outbox_lock(f_business_type) SELECT 8 FROM DUAL WHERE NOT EXISTS(SELECT f_business_type FROM t_outbox_lock WHERE f_business_type = 8);
INSERT INTO t_outbox_lock(f_business_type) SELECT 9 FROM DUAL WHERE NOT EXISTS(SELECT f_business_type FROM t_outbox_lock WHERE f_business_type = 9);

CREATE TABLE IF NOT EXISTS `t_app` (
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_id` char(40) NOT NULL COMMENT '应用账户ID',
    `f_name` varchar(512) NOT NULL COMMENT '应用账户名称',
    `f_password` varchar(100) NOT NULL COMMENT '应用账户密码',           -- 使用BCrypt散列后固定占60位，暂定为varcher(100)
    `f_type` tinyint(4) NOT NULL COMMENT '应用账户类型',
    `f_created_time` bigint(40) NOT NULL COMMENT '应用账户创建时间',
    `f_credential_type` tinyint(4) NOT NULL DEFAULT '1' COMMENT '凭证类型,1: 密码,2: 令牌',
    PRIMARY KEY (`f_primary_id`),
    UNIQUE KEY `f_id` (`f_id`),
    UNIQUE KEY `f_name` (`f_name`)
) ENGINE=InnoDB COMMENT='应用账户信息表';

CREATE TABLE IF NOT EXISTS `t_org_perm_app` (                                                   -- 此表记录应用账户对组织架构管理的权限
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,                                          -- 自增主键
    `f_app_id` char(40) NOT NULL,                                                               -- 应用账户id
    `f_app_name` varchar(150) NOT NULL,                                                         -- 应用账户名称
    `f_org_type` tinyint(4) NOT NULL,                                                           -- 组织架构对象类型，1：用户，2：部门，3：用户组
    `f_perm_value` int(11) NOT NULL DEFAULT '0',                                                -- 权限值
    `f_end_time` bigint(20) DEFAULT '-1',                                                       -- 权限结束时间, 微秒的时间戳, -1标识永久有效
    `f_modify_time` bigint(20) NOT NULL DEFAULT '0',                                            -- 记录修改时间, 微秒的时间戳
    `f_create_time` bigint(20) NOT NULL,                                                        -- 记录创建时间, 微秒的时间戳
    PRIMARY KEY (`f_primary_id`),
    KEY `idx_f_app_id` (`f_app_id`),
    KEY `idx_f_end_time` (`f_end_time`),
    KEY `idx_f_org_type` (`f_org_type`)
) ENGINE=InnoDB COMMENT='应用账户组织架构管理权限表';

CREATE TABLE IF NOT EXISTS `t_avatar` (                                                   -- 此表记录用户头像信息
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,                                    -- 自增主键
    `f_user_id` char(40) NOT NULL,                                                        -- 用户ID
    `f_oss_id` char(40) NOT NULL,                                                         -- 对象存储ID
    `f_key` char(80) NOT NULL,                                                            -- 对象存储内文件KEY值
    `f_type` varchar(50) NOT NULL,                                                        -- 文件类型
    `f_status` tinyint(4) NOT NULL,                                                       -- 文件状态类型，0：未使用，1：已使用
    `f_time` bigint(20) NOT NULL,                                                         -- 记录创建时间, 微秒的时间戳
    PRIMARY KEY (`f_primary_id`),
    KEY `idx_f_user_id` (`f_user_id`),
    UNIQUE KEY `idx_f_key` (`f_key`),
    KEY `idx_f_time` (`f_time`)
) ENGINE=InnoDB COMMENT='用户头像信息表';


CREATE TABLE IF NOT EXISTS `t_internal_group` (
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_id` char(40) NOT NULL COMMENT '内部组唯一标识',
    `f_created_time` bigint(40) NOT NULL COMMENT '内部组创建时间',
    PRIMARY KEY (`f_primary_id`),
    KEY `idx_id` (`f_id`)
) ENGINE=InnoDB COMMENT='AnyShare内部组信息表';

CREATE TABLE IF NOT EXISTS `t_internal_group_member` (
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_internal_group_id` char(40) NOT NULL COMMENT '内部组唯一标识',
    `f_member_id` char(40) NOT NULL COMMENT '成员唯一标识',
    `f_member_type` tinyint(4) NOT NULL COMMENT '成员类型,1：用户',
    `f_added_time` bigint(40) NOT NULL COMMENT '内部组成员添加时间',
    PRIMARY KEY (`f_primary_id`),
    KEY `idx_id` (`f_internal_group_id`),
    KEY `idx_f_member_id` (`f_member_id`)
) ENGINE=InnoDB COMMENT='AnyShare内部组成员信息表';

CREATE TABLE IF NOT EXISTS `option` (
    `key` varchar(40) NOT NULL COMMENT '配置关键字',
    `value` varchar(150) NOT NULL COMMENT '配置值',
    PRIMARY KEY (`key`)
) ENGINE=InnoDB COMMENT='配置表';

CREATE TABLE IF NOT EXISTS `t_org_perm` (
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_id` char(40) NOT NULL COMMENT '账户id',
    `f_name` varchar(150) NOT NULL COMMENT '账户名称',
    `f_type` tinyint(4) NOT NULL COMMENT '账户类型，1：实名用户',
    `f_org_type` tinyint(4) NOT NULL COMMENT '组织架构对象类型，1：用户，2：部门，3：用户组',
    `f_perm_value` int(11) NOT NULL DEFAULT '0' COMMENT '权限值',
    `f_end_time` bigint(20) DEFAULT '-1' COMMENT '权限结束时间, 微秒的时间戳, -1标识永久有效',
    `f_modify_time` bigint(20) NOT NULL DEFAULT '0' COMMENT '记录修改时间, 微秒的时间戳',
    `f_create_time` bigint(20) NOT NULL COMMENT '记录创建时间, 微秒的时间戳',
    PRIMARY KEY (`f_primary_id`),
    KEY `idx_f_id` (`f_id`),
    KEY `idx_f_end_time` (`f_end_time`),
    KEY `idx_f_org_type` (`f_org_type`),
    KEY `idx_f_type` (`f_type`)
) ENGINE=InnoDB COMMENT='组织架构管理权限表';

INSERT INTO `option`(`key`,`value`) SELECT 'user_defalut_des_password','4SLXQjA5JbE=' FROM DUAL WHERE NOT EXISTS(SELECT `value` FROM `option` WHERE `key` = 'user_defalut_des_password');
INSERT INTO `option`(`key`,`value`) SELECT 'user_defalut_ntlm_password','32ed87bdb5fdc5e9cba88547376818d4' FROM DUAL WHERE NOT EXISTS(SELECT `value` FROM `option` WHERE `key` = 'user_defalut_ntlm_password');
INSERT INTO `option`(`key`,`value`) SELECT 'user_defalut_sha2_password','8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92' FROM DUAL WHERE NOT EXISTS(SELECT `value` FROM `option` WHERE `key` = 'user_defalut_sha2_password');
INSERT INTO `option`(`key`,`value`) SELECT 'user_defalut_md5_password','e10adc3949ba59abbe56e057f20f883e' FROM DUAL WHERE NOT EXISTS(SELECT `value` FROM `option` WHERE `key` = 'user_defalut_md5_password');

use sharemgnt_db;

CREATE TABLE IF NOT EXISTS `t_reserved_name` (
  `f_id` char(40) NOT NULL COMMENT 'id',
  `f_name` char(150) NOT NULL COMMENT '名称',
  `f_create_time` bigint(20) NOT NULL COMMENT '创建时间',
  `f_update_time` bigint(20) NOT NULL COMMENT '修改时间',
  PRIMARY KEY (`f_id`),
  KEY `idx_name` (`f_name`)
) ENGINE=InnoDB COMMENT='保留名称表';

INSERT INTO `t_sharemgnt_config`(`f_key`, `f_value`) SELECT 'user_expired_disable_lock', 'locked' FROM DUAL WHERE NOT EXISTS (SELECT `f_key` FROM `t_sharemgnt_config` WHERE `f_key` = 'user_expired_disable_lock');
INSERT INTO `t_sharemgnt_config`(`f_key`, `f_value`) SELECT 'user_not_login_disable_lock', 'locked' FROM DUAL WHERE NOT EXISTS (SELECT `f_key` FROM `t_sharemgnt_config` WHERE `f_key` = 'user_not_login_disable_lock');

