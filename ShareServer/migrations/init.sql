/*
MySQL: Database - anyshare-eacp
*********************************************************************
*/
use anyshare;

CREATE TABLE IF NOT EXISTS `t_acs_custom_perm` (                                              -- 此表记录文档权限
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,                                          -- 自增主键
    `f_doc_id` text NOT NULL,                                                                   -- 文档路径
    `f_accessor_id` char(40) NOT NULL,                                                          -- 被配置权限的对象id
    `f_accessor_name` varchar(150) NOT NULL,                                                    -- 被配置权限的显示名
    `f_accessor_type` tinyint(4) NOT NULL,                                                      -- 权限所有者类型, 1: 用户, 2: 组织/部门, 3: 联系人组, 4: 匿名用户
    `f_type` tinyint(4) NOT NULL,                                                               -- 权限类型, 1: 拒绝, 2: 允许 3: 禁用继承
    `f_perm_value` int(11) NOT NULL DEFAULT '1',                                                -- 权限值
    `f_source` tinyint(4) NOT NULL DEFAULT '1',                                                 -- 权限来源  1: 用户配置, 2: 系统内部配置
    `f_end_time` bigint(20) DEFAULT '-1',                                                       -- 权限结束时间, 微秒的时间戳, -1标识永久有效
    `f_modify_time` bigint(20) NOT NULL DEFAULT '0',                                            -- 记录修改时间, 微秒的时间戳
    `f_create_time` bigint(20) NOT NULL,                                                        -- 记录创建时间, 微秒的时间戳
    PRIMARY KEY (`f_primary_id`),
    KEY `t_perm_f_doc_id_index` (`f_doc_id`(120)),
    KEY `t_perm_f_accessor_id_index` (`f_accessor_id`),
    KEY `t_perm_f_accessor_type_index` (`f_accessor_type`),
    KEY `t_perm_f_type_index` (`f_type`),
    KEY `t_perm_f_end_time_index` (`f_end_time`) USING BTREE
  ) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_acs_doc` (                        -- 此表保存文档入口视图
    `f_doc_id` char(40) NOT NULL,                                 -- 文档入口id
    `f_doc_type` tinyint(4) NOT NULL,                             -- 文档入口类型, 1: 个人文档, 3: 文档库
    `f_type_name` char(128) NOT NULL,                             -- 文档入口类型名
    `f_status` int(11) DEFAULT '1',                               -- 文档入口状态, 1: 启用, 其他: 禁用
    `f_create_time` bigint(20) NOT NULL DEFAULT '0',              -- 文档入口创建时间, 微秒的时间戳
    `f_delete_time` bigint(20) DEFAULT '0',                       -- 文档入口被删除的时间
    `f_deleter_id` char(40) NOT NULL DEFAULT '',                  -- 文档入口删除者id
    `f_obj_id` char(40) NOT NULL,                                 -- 文档入口标识id
    `f_name` char(128) NOT NULL,                                  -- 文档入口名
    `f_creater_id` char(40) NOT NULL,                             -- 文档入口的创建者id
    `f_creater_name` varchar(150) NOT NULL,                       -- 文档入口的创建者名称
    `f_creater_type` tinyint(4) NOT NULL,                         -- 文档入口的创建者类型
    `f_oss_id` char(150) NOT NULL DEFAULT '',                     -- 文档入口所属对象存储id
    `f_relate_depart_id` char(40) NOT NULL DEFAULT '',            -- 关联部门id
    `f_subtype_id` char(40) NOT NULL DEFAULT '',                  -- 所属文档库分类id
    `f_display_order` MEDIUMINT DEFAULT -1,                       -- 自定义文档库显示顺序
    `f_owners_id` text NOT NULL,                                  -- 文档库所有者id
    `f_owners_name` text NOT NULL,                                -- 文档库所有者名称
    `f_depart_manager_as_owner` tinyint(4) NOT NULL DEFAULT 0,    -- 部门负责人为所有者  0：否   1：是
    PRIMARY KEY (`f_doc_id`),
    KEY `t_doc_f_doc_type_index` (`f_doc_type`) USING BTREE,
    KEY `t_doc_f_obj_id_index` (`f_obj_id`) USING BTREE,
    KEY `t_doc_f_name_index` (`f_name`) USING BTREE,
    KEY `t_doc_f_type_name_index` (`f_type_name`) USING BTREE,
    KEY `t_doc_f_relate_depart_id_index` (`f_relate_depart_id`) USING BTREE,
    KEY `t_display_order_index` (`f_display_order`) USING BTREE,
    KEY `t_doc_f_creater_id_index` (`f_creater_id`) USING BTREE
  ) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_acs_doc_quit` (                   -- 此表记录屏蔽共享信息
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,            -- 自增主键
    `f_user_id` char(40) NOT NULL,                                -- 用户id
    `f_doc_id` char(40) NOT NULL,                                 -- 入口文档id
    PRIMARY KEY (`f_primary_id`),
    UNIQUE KEY `t_acs_doc_unique_index` (`f_user_id`,`f_doc_id`) USING HASH,
    KEY `t_acs_doc_quit_f_doc_id_index` (`f_doc_id`) USING BTREE
  ) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_acs_owner` (                                                      -- 此表保存文档入口所有者信息
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,                                            -- 自增主键
    `f_gns_path` text NOT NULL,                                                                   -- 文档路径
    `f_owner_id` char(40) NOT NULL,                                                               -- 所有者id
    `f_owner_name` varchar(150) NOT NULL,                                                         -- 所有者显示名
    `f_type` tinyint(4) NOT NULL,                                                                 -- 用户类型, 1: 用户, 2: 组织/部门, 3: 联系人组, 4: 匿名用户, 5: 用户组， 6: 应用账户
    `f_modify_time` bigint(20) NOT NULL DEFAULT '0',                                              -- 记录修改时间, 微秒的时间戳
    `f_deletable` BOOLEAN NOT NULL,                                                               -- 允许删除标记, 1: 允许删除, 0: 禁止删除
    PRIMARY KEY (`f_primary_id`),
    KEY `t_owner_f_gns_path_index` (`f_gns_path`(120)),
    KEY `t_owner_f_owner_id_index` (`f_owner_id`)
  ) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_device` (                         -- 此表记录设备信息
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,            -- 自增主键
    `f_user_id` char(40) NOT NULL,                                -- 用户id
    `f_udid` char(40) NOT NULL,                                   -- 用户设备标识
    `f_name` char(128) NOT NULL,                                  -- 设备名
    `f_os_type` tinyint(4) NOT NULL,                              -- 系统类型
    `f_device_type` char(128) NOT NULL,                           -- 设备类型
    `f_last_login_ip` char(40) NOT NULL,                          -- 最后登录IP
    `f_last_login_time` bigint(20) NOT NULL,                      -- 最后登录时间, 微秒的时间戳
    `f_erase_flag` tinyint(4) NOT NULL DEFAULT '0',               -- 设备擦除标记
    `f_last_erase_time` bigint(20) NOT NULL DEFAULT '0',          -- 最后擦除时间, 微秒的时间戳
    `f_disable_flag` tinyint(4) NOT NULL DEFAULT '0',             -- 设备禁用标记
    `f_bind_flag` tinyint(4) NOT NULL DEFAULT '0',                -- 设备绑定标记
    PRIMARY KEY (`f_primary_id`),
    UNIQUE KEY `unique_index` (`f_user_id`,`f_udid`) USING BTREE,
    KEY `f_udid_index` (`f_udid`) USING BTREE
  ) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_lock` (                           -- 此表记录锁定的文件
    `f_primary_id` char(40) NOT NULL,                             -- 自增主键
    `f_doc_id` text NOT NULL,                                     -- 文件路径标识
    `f_user_id` char(40) NOT NULL,                                -- 用户id
    `f_user_name` char(150) NOT NULL,                             -- 用户显示名
    `f_user_type` tinyint(4) NOT NULL,                            -- 用户类型  1: 用户, 6: 应用账户
    `f_source` tinyint(4) NOT NULL DEFAULT '1',                   -- 锁记录来源  1: 用户配置, 2: 系统内部配置
    `f_create_date` bigint(20) NOT NULL DEFAULT '-1',             -- 锁创建时间
    `f_refresh_date` bigint(20) NOT NULL DEFAULT '-1',            -- 锁刷新时间
    `f_expire_time` bigint(20) NOT NULL DEFAULT '-2',             -- 锁过期时间, -1: 永久有效, -2: 服务器配置的超期间隔(单位: 秒)
    PRIMARY KEY (`f_primary_id`),
    KEY `t_finder_f_doc_id_index` (`f_doc_id`(120)) USING BTREE,
    KEY `t_lock_f_refresh_date_index` (`f_refresh_date`) USING BTREE,
    KEY `t_lock_f_expire_time_index` (`f_expire_time`) USING BTREE
  ) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_audit` (
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT  COMMENT '自增主键',
    `f_apply_id` char(40) NOT NULL COMMENT '申请唯一标识',
    `f_apply_type` tinyint(4) NOT NULL COMMENT '申请的类型',
    `f_doc_id` text NOT NULL COMMENT '文档路径标识',
    `f_sharer_id` char(40) NOT NULL COMMENT '申请者id',
    `f_create_date` bigint(20) NOT NULL COMMENT '申请创建时间, 微秒的时间戳',
    `f_accessor_id` char(40) NOT NULL COMMENT '被共享者id',
    `f_accessor_name` char(150) NOT NULL COMMENT '被共享者名字',
    `f_accessor_type` tinyint(4) NOT NULL COMMENT '被共享者类型',
    `f_detail` text NOT NULL COMMENT '申请详情',
    PRIMARY KEY (`f_primary_id`),
    UNIQUE KEY `uk_apply_id` (`f_apply_id`),
    KEY `idx_doc_id` (`f_doc_id`(137)) COMMENT '大小依据 4层目录+分隔符+gns前缀',
    KEY `idx_sharer_id` (`f_sharer_id`),
    KEY `idx_accessor_id` (`f_accessor_id`)
  ) ENGINE=InnoDB COMMENT='审核申请信息表';

CREATE TABLE IF NOT EXISTS `t_active_user_info` (               -- 此表记录活跃用户信息
    `f_id` bigint(20) NOT NULL AUTO_INCREMENT,                    -- 自增主键
    `f_time` char(40) NOT NULL,                                   -- 统计时间, 如: 2018-08-08
    `f_user_id` char(40) NOT NULL,                                -- 用户id
    PRIMARY KEY (`f_id`),
    KEY `idx_userid` (`f_user_id`),
    KEY `idx_time` (`f_time`)
  ) ENGINE=InnoDB;

  CREATE TABLE IF NOT EXISTS `t_conf` (                           -- 此表记录基本配置信息
    `f_key` char(32) NOT NULL,                                    -- 配置关键字
    `f_value` char(255) NOT NULL,                                 -- 配置的值
    PRIMARY KEY (`f_key`)
  ) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_eacp_outbox` (
    `f_id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '自增主键',
    `f_message` longtext NOT NULL COMMENT '消息内容，json格式字符串',
    `f_create_time` bigint(20) NOT NULL COMMENT '消息创建时间',
    PRIMARY KEY (`f_id`),
    KEY `idx_create_time` (`f_create_time`)
  ) ENGINE=InnoDB COMMENT='outbox信息表';

INSERT INTO t_conf(f_key,f_value) SELECT 'auto_lock','true' FROM DUAL WHERE NOT EXISTS(SELECT f_value FROM t_conf WHERE f_key = 'auto_lock');
INSERT INTO t_conf(f_key,f_value) SELECT 'oem_allow_auth_low_csf_user','true' FROM DUAL WHERE NOT EXISTS(SELECT f_value FROM t_conf WHERE f_key ='oem_allow_auth_low_csf_user');
INSERT INTO t_conf(f_key,f_value) SELECT 'oem_allow_owner','true' FROM DUAL WHERE NOT EXISTS(SELECT f_value FROM t_conf WHERE f_key = 'oem_allow_owner');
INSERT INTO t_conf(f_key,f_value) SELECT 'oem_client_logout_time','-1' FROM DUAL WHERE NOT EXISTS(SELECT f_value FROM t_conf WHERE f_key = 'oem_client_logout_time');
INSERT INTO t_conf(f_key,f_value) SELECT 'oem_indefinite_perm','true' FROM DUAL WHERE NOT EXISTS(SELECT f_value FROM t_conf WHERE f_key = 'oem_indefinite_perm');
INSERT INTO t_conf(f_key,f_value) SELECT 'oem_max_pass_expired_days','-1' FROM DUAL WHERE NOT EXISTS(SELECT f_value FROM t_conf WHERE f_key = 'oem_max_pass_expired_days');
INSERT INTO t_conf(f_key,f_value) SELECT 'oem_remember_pass','true' FROM DUAL WHERE NOT EXISTS(SELECT f_value FROM t_conf WHERE f_key = 'oem_remember_pass');
INSERT INTO t_conf(f_key,f_value) SELECT 'web_client_host','' FROM DUAL WHERE NOT EXISTS(SELECT f_value FROM t_conf WHERE f_key = 'web_client_host');
INSERT INTO t_conf(f_key,f_value) SELECT 'web_client_port','443' FROM DUAL WHERE NOT EXISTS(SELECT f_value FROM t_conf WHERE f_key = 'web_client_port');
INSERT INTO t_conf(f_key,f_value) SELECT 'web_client_http_port','80' FROM DUAL WHERE NOT EXISTS(SELECT f_value FROM t_conf WHERE f_key ='web_client_http_port');
INSERT INTO t_conf(f_key,f_value) SELECT 'eacp_https_port','9999' FROM DUAL WHERE NOT EXISTS(SELECT f_value FROM t_conf WHERE f_key = 'eacp_https_port');
INSERT INTO t_conf(f_key,f_value) SELECT 'efast_https_port','9124' FROM DUAL WHERE NOT EXISTS(SELECT f_value FROM t_conf WHERE f_key = 'efast_https_port');
INSERT INTO t_conf(f_key,f_value) SELECT 'oem_default_perm_expired_days','-1' FROM DUAL WHERE NOT EXISTS(SELECT f_value FROM t_conf WHERE f_key = 'oem_default_perm_expired_days');
INSERT INTO t_conf(f_key,f_value) SELECT 'auto_lock_expired_interval','180' FROM DUAL WHERE NOT EXISTS(SELECT f_value FROM t_conf WHERE f_key = 'auto_lock_expired_interval');
