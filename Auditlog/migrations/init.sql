/*
MySQL: Database - anyshare
*********************************************************************
*/
use anyshare;

CREATE TABLE IF NOT EXISTS `t_log_login` (
  `f_log_id` bigint(20) NOT NULL COMMENT '日志id',
  `f_user_id` char(40) NOT NULL COMMENT '用户id',
  `f_user_name` char(128) NOT NULL COMMENT '用户显示名',
  `f_user_type` varchar(32) NOT NULL DEFAULT 'authenticated_user' COMMENT '用户类型',
  `f_obj_id` char(40) NOT NULL COMMENT '对象id',
  `f_additional_info` text NOT NULL COMMENT '附加信息',
  `f_level` tinyint(4) NOT NULL COMMENT '日志级别, 1: 信息, 2: 警告',
  `f_op_type` tinyint(4) NOT NULL COMMENT '操作类型',
  `f_date` bigint(20) NOT NULL COMMENT '日志记录时间, 微秒的时间戳',
  `f_ip` char(40) NOT NULL COMMENT '访问者的IP',
  `f_mac` char(40) NOT NULL DEFAULT '' COMMENT '文档入口属于哪个站点',
  `f_msg` text NOT NULL COMMENT '日志描述',
  `f_exmsg` text NOT NULL COMMENT '日志附加描述',
  `f_user_agent` varchar(1024) NOT NULL DEFAULT '' COMMENT '用户代理',
  `f_user_paths` text COMMENT '用户所属部门信息',
  `f_obj_name` char(128) NOT NULL DEFAULT '' COMMENT '对象名称',
  `f_obj_type` tinyint(4) NOT NULL DEFAULT 0 COMMENT '对象类型',
  PRIMARY KEY (`f_log_id`),
  KEY `t_log_f_user_id_index` (`f_user_id`) USING BTREE,
  KEY `t_log_f_user_name_index` (`f_user_name`) USING BTREE,
  KEY `t_log_f_op_type_index` (`f_op_type`) USING BTREE,
  KEY `t_log_f_date_index` (`f_date`) USING BTREE,
  KEY `t_log_f_ip_index` (`f_ip`) USING BTREE,
  KEY `t_log_f_mac_index` (`f_mac`) USING BTREE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_log_management` (
  `f_log_id` bigint(20) NOT NULL COMMENT '日志id',
  `f_user_id` char(40) NOT NULL COMMENT '用户id',
  `f_user_name` char(128) NOT NULL COMMENT '用户显示名',
  `f_user_type` varchar(32) NOT NULL DEFAULT 'authenticated_user' COMMENT '用户类型',
  `f_obj_id` char(40) NOT NULL COMMENT '对象id',
  `f_additional_info` text NOT NULL COMMENT '附加信息',
  `f_level` tinyint(4) NOT NULL COMMENT '日志级别, 1: 信息, 2: 警告',
  `f_op_type` tinyint(4) NOT NULL COMMENT '操作类型',
  `f_date` bigint(20) NOT NULL COMMENT '日志记录时间',
  `f_ip` char(40) NOT NULL COMMENT '访问者IP',
  `f_mac` char(40) NOT NULL DEFAULT '' COMMENT '文档入口所属站点',
  `f_msg` text NOT NULL COMMENT '日志描述',
  `f_exmsg` text NOT NULL COMMENT '日志附加描述',
  `f_user_agent` varchar(1024) NOT NULL DEFAULT '' COMMENT '用户代理',
  `f_user_paths` text COMMENT '用户所属部门信息',
  `f_obj_name` char(128) NOT NULL DEFAULT '' COMMENT '对象名称',
  `f_obj_type` tinyint(4) NOT NULL DEFAULT 0 COMMENT '对象类型',
  PRIMARY KEY (`f_log_id`),
  KEY `t_log_f_user_id_index` (`f_user_id`) USING BTREE,
  KEY `t_log_f_user_name_index` (`f_user_name`) USING BTREE,
  KEY `t_log_f_op_type_index` (`f_op_type`) USING BTREE,
  KEY `t_log_f_date_index` (`f_date`) USING BTREE,
  KEY `t_log_f_ip_index` (`f_ip`) USING BTREE,
  KEY `t_log_f_mac_index` (`f_mac`) USING BTREE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_log_operation` (
  `f_log_id` bigint(20) NOT NULL COMMENT '日志id',
  `f_user_id` char(40) NOT NULL COMMENT '用户id',
  `f_user_name` char(128) NOT NULL COMMENT '用户显示名',
  `f_user_type` varchar(32) NOT NULL DEFAULT 'authenticated_user' COMMENT '用户类型',
  `f_obj_id` char(40) NOT NULL COMMENT '对象id',
  `f_additional_info` text NOT NULL COMMENT '附加信息',
  `f_level` tinyint(4) NOT NULL COMMENT '日志级别, 1: 信息, 2: 警告',
  `f_op_type` tinyint(4) NOT NULL COMMENT '日志类型',
  `f_date` bigint(20) NOT NULL COMMENT '日志记录时间',
  `f_ip` char(40) NOT NULL COMMENT '访问者IP',
  `f_mac` char(40) NOT NULL DEFAULT '' COMMENT '文档入口所属站点',
  `f_msg` text NOT NULL COMMENT '日志描述',
  `f_exmsg` text NOT NULL COMMENT '日志附加描述',
  `f_user_agent` varchar(1024) NOT NULL DEFAULT '' COMMENT '用户代理',
  `f_user_paths` text COMMENT '用户所属部门信息',
  `f_obj_name` char(128) NOT NULL DEFAULT '' COMMENT '对象名称',
  `f_obj_type` tinyint(4) NOT NULL DEFAULT 0 COMMENT '对象类型',
  PRIMARY KEY (`f_log_id`),
  KEY `t_log_f_user_id_index` (`f_user_id`) USING BTREE,
  KEY `t_log_f_user_name_index` (`f_user_name`) USING BTREE,
  KEY `t_log_f_op_type_index` (`f_op_type`) USING BTREE,
  KEY `t_log_f_date_index` (`f_date`) USING BTREE,
  KEY `t_log_f_ip_index` (`f_ip`) USING BTREE,
  KEY `t_log_f_mac_index` (`f_mac`) USING BTREE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_history_log_info` (
  `f_id` char(128) NOT NULL COMMENT '唯一标识',
  `f_name` char(128) NOT NULL COMMENT '日志记录名',
  `f_size` bigint(20) NOT NULL COMMENT '日志大小',
  `f_type` tinyint(4) NOT NULL COMMENT '记录类型, 10: 登录日志, 11: 管理日志, 12: 操作日志',
  `f_date` bigint(20) NOT NULL COMMENT '记录时间',
  `f_dump_date` bigint(20) NOT NULL COMMENT '转存时间',
  `f_oss_id` char(40) NOT NULL COMMENT '历史日志所属的对象存储ID',
  PRIMARY KEY (`f_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_log_config` (
  `f_key` char(40) NOT NULL,
  `f_value` char(40) NOT NULL,
  PRIMARY KEY (`f_key`)
) ENGINE=InnoDB COMMENT='日志配置';

-- 转存周期
INSERT INTO t_log_config (f_key, f_value) SELECT 'retention_period', 1 FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM t_log_config WHERE f_key = 'retention_period');
-- 转存周期单位
INSERT INTO t_log_config (f_key, f_value) SELECT 'retention_period_unit', 'year' FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM t_log_config WHERE f_key = 'retention_period_unit');
-- 转存时间
INSERT INTO t_log_config (f_key, f_value) SELECT 'dump_time', '03:00:00' FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM t_log_config WHERE f_key = 'dump_time');
-- 转存格式
INSERT INTO t_log_config (f_key, f_value) SELECT 'dump_format', 'csv' FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM t_log_config WHERE f_key = 'dump_format');
-- 历史日志导出是否加密
INSERT INTO t_log_config (f_key, f_value) SELECT 'history_log_export_with_pwd', 0 FROM DUAL WHERE NOT EXISTS (SELECT 1 FROM t_log_config WHERE f_key = 'history_log_export_with_pwd');

CREATE TABLE IF NOT EXISTS `t_auditlog_outbox` (
    `f_id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '自增主键',
    `f_business_type` varchar(20) NOT NULL COMMENT '业务类型',
    `f_message` longtext NOT NULL COMMENT '消息内容，json格式字符串',
    `f_create_time` bigint(20) NOT NULL COMMENT '消息创建时间',
    PRIMARY KEY (`f_id`),
    KEY `idx_business_type_and_create_time` (`f_business_type`, `f_create_time`)
  ) ENGINE=InnoDB COMMENT='outbox信息表';

CREATE TABLE IF NOT EXISTS `t_auditlog_outbox_lock` (
    `f_business_type` varchar(20) NOT NULL COMMENT '业务类型',
    PRIMARY KEY (`f_business_type`)
) ENGINE=InnoDB COMMENT='outbox分布式锁表';

INSERT INTO t_auditlog_outbox_lock(f_business_type) SELECT 'client_log' FROM DUAL WHERE NOT EXISTS(SELECT f_business_type FROM t_auditlog_outbox_lock WHERE f_business_type = 'client_log');

CREATE TABLE IF NOT EXISTS `t_log_scope_strategy` (
  `f_id` bigint(20) NOT NULL,
  `f_created_at` bigint(20) NOT NULL DEFAULT 0 COMMENT '创建时间',
  `f_created_by` varchar(64) NOT NULL DEFAULT '' COMMENT '创建人员',
  `f_updated_at` bigint(20) NOT NULL DEFAULT 0 COMMENT '更新时间',
  `f_updated_by` varchar(64) NOT NULL DEFAULT '' COMMENT '更新人员',
  `f_log_type` tinyint(4) NOT NULL COMMENT '日志类型',
  `f_log_category` tinyint(4) NOT NULL COMMENT '日志分类',
  `f_role` char(128) NOT NULL COMMENT '查看者角色名',
  `f_scope` varchar(1024) NOT NULL COMMENT '查看范围',
  PRIMARY KEY (`f_id`),
  KEY `idx_log_type` (`f_log_type`),
  KEY `idx_log_category` (`f_log_category`),
  KEY `idx_role` (`f_role`)
) ENGINE=InnoDB COMMENT='日志查看范围策略';

-- 安全管理员查看活跃访问日志
INSERT INTO t_log_scope_strategy(f_id, f_log_type, f_log_category, f_role, f_scope)
SELECT 111000222000333000, 10, 1, 'sec_admin', 'audit_admin,normal_user'
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM t_log_scope_strategy
  WHERE f_id = 111000222000333000 OR (f_log_type = 10 AND f_log_category = 1 AND f_role = 'sec_admin')
);
-- 审计管理员查看活跃访问日志
INSERT INTO t_log_scope_strategy(f_id, f_log_type, f_log_category, f_role, f_scope)
SELECT 111000222000333001, 10, 1, 'audit_admin', 'sys_admin,sec_admin'
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM t_log_scope_strategy
  WHERE f_id = 111000222000333001 OR (f_log_type = 10 AND f_log_category = 1 AND f_role = 'audit_admin')
);
-- 安全管理员查看活跃管理日志
INSERT INTO t_log_scope_strategy(f_id, f_log_type, f_log_category, f_role, f_scope)
SELECT 111000222000333002, 11, 1, 'sec_admin', 'audit_admin,normal_user'
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM t_log_scope_strategy
  WHERE f_id = 111000222000333002 OR (f_log_type = 11 AND f_log_category = 1 AND f_role = 'sec_admin')
);
-- 审计管理员查看活跃管理日志
INSERT INTO t_log_scope_strategy(f_id, f_log_type, f_log_category, f_role, f_scope)
SELECT 111000222000333003, 11, 1, 'audit_admin', 'sys_admin,sec_admin'
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM t_log_scope_strategy
  WHERE f_id = 111000222000333003 OR (f_log_type = 11 AND f_log_category = 1 AND f_role = 'audit_admin')
);
-- 安全管理员查看活跃操作日志
INSERT INTO t_log_scope_strategy(f_id, f_log_type, f_log_category, f_role, f_scope)
SELECT 111000222000333004, 12, 1, 'sec_admin', 'normal_user'
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM t_log_scope_strategy
  WHERE f_id = 111000222000333004 OR (f_log_type = 12 AND f_log_category = 1 AND f_role = 'audit_admin')
);
-- 安全管理员查看历史访问日志
INSERT INTO t_log_scope_strategy(f_id, f_log_type, f_log_category, f_role, f_scope)
SELECT 111000222000333005, 10, 2, 'sec_admin', ''
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM t_log_scope_strategy
  WHERE f_id = 111000222000333005 OR (f_log_type= 10 AND f_log_category = 2 AND f_role = 'sec_admin')
);
-- 安全管理员查看历史管理日志
INSERT INTO t_log_scope_strategy(f_id, f_log_type, f_log_category, f_role, f_scope)
SELECT 111000222000333006, 11, 2, 'sec_admin', ''
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM t_log_scope_strategy
  WHERE f_id = 111000222000333006 OR (f_log_type= 11 AND f_log_category = 2 AND f_role = 'sec_admin')
);
-- 审计管理员查看历史操作日志
INSERT INTO t_log_scope_strategy(f_id, f_log_type, f_log_category, f_role, f_scope)
SELECT 111000222000333007, 12, 2, 'sec_admin', ''
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1 FROM t_log_scope_strategy
  WHERE f_id = 111000222000333007 OR (f_log_type= 12 AND f_log_category = 2 AND f_role = 'sec_admin')
);

-- 暂时只用于redis分布式锁的value，保证value的唯一性
-- 【注意】这个和Personalization共用一张表，如果调整，两边都注意下是否一起调整相应地方
create table if not exists t_pers_rec_unique_id
(
    f_id        char(36) not null comment 'ulid生成的id',
    f_flag tinyint not null  comment '使用场景（1：数据库的主键，2：redis分布式锁value）',
    primary key (f_id, f_flag)
) ENGINE = InnoDB comment '个性化推荐 唯一id';

CREATE TABLE IF NOT EXISTS t_pers_rec_svc_config
(
    f_id         bigint        not null auto_increment,
    f_key        varchar(64)   not null comment '配置key',
    f_value      varchar(2048) not null comment '配置value',
    f_created_at bigint        not null comment '创建时间',
    f_updated_at bigint        not null default 0 comment '更新时间',
    primary key (f_id),
    unique key uk_key (f_key)
) ENGINE = InnoDB COMMENT '个性化推荐 服务配置（用于存储一些配置或标识等）';
