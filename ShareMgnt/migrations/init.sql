/*
MySQL: Database - sharemgnt
*********************************************************************
*/
use sharemgnt_db;

CREATE TABLE IF NOT EXISTS `t_user` (
    `f_user_id` char(40) NOT NULL,                                  -- 用户id
    `f_login_name` CHAR(150) NOT NULL,                              -- 登录名
    `f_display_name`  CHAR(150) NOT NULL,                           -- 显示名
    `f_remark` char(128),                                           -- 备注
    `f_idcard_number` char(32),                                     -- 身份证号
    `f_password` char(32) NOT NULL,                                 -- 密码的校验值
    `f_des_password` char(150) DEFAULT '',                          -- 管控密码的校验值
    `f_ntlm_password` char(32) DEFAULT '',                          -- ntlm密码的校验值
    `f_sha2_password` char(64) DEFAULT '',                          -- sha2密码的校验值
    `f_mail_address` char(150) NOT NULL,                            -- 邮箱地址
    `f_auth_type` tinyint(4) NOT NULL DEFAULT '0',                  -- 认证类型, 1为本地用户, 2为域用户, 3为第三方用户
    `f_status` tinyint(4) NOT NULL DEFAULT '0',                     -- 用户禁用状态, 0为正常使用, 1为禁用
    `f_freeze_status` tinyint(4) NOT NULL DEFAULT '0',              -- 用户冻结状态, 0为不冻结, 1为冻结
    `f_pwd_timestamp` datetime,                                     -- 密码修改时间
    `f_pwd_error_latest_timestamp` datetime,                        -- 上次密码输入错误的时间
    `f_pwd_error_cnt` tinyint(4) NOT NULL DEFAULT '0',              -- 密码错误次数
    `f_domain_object_guid`char(100) DEFAULT '',                     -- 域对象的guid
    `f_domain_path` char(255) DEFAULT '',                           -- 域路径
    `f_ldap_server_type` tinyint(4) NOT NULL DEFAULT '0',           -- ldap服务器类型
    `f_third_party_id` char(255),                                   -- 第三方系统中的id
    `f_third_party_depart_id` varchar(255),                         -- 第三方系统中的部门id
    `f_priority` smallint(6) NOT NULL DEFAULT '999',                -- 用户优先级
    `f_csf_level` tinyint(4) NOT NULL DEFAULT '5',                  -- 用户密级
    `f_pwd_control` tinyint(1) NOT NULL DEFAULT '0',                -- 本地用户的密码管控, 0为不使用密码管控, 1为使用
    `f_oss_id` char(40),                                            -- 用户归属对象存储
    `f_create_time` datetime DEFAULT now(),                         -- 用户创建时间
    `f_last_request_time` datetime DEFAULT now(),                   -- 用户最后一次请求的时间
    `f_last_client_request_time` datetime NOT NULL DEFAULT now(),   -- 用户在客户端最后一次请求的时间
    `f_auto_disable_status` tinyint(4) NOT NULL DEFAULT '0',        -- 用户自动禁用状态, 1为长时间不登录禁用, 2为用户过期禁用
    `f_agreed_to_terms_of_use` tinyint(4) NOT NULL DEFAULT '0',     -- 本地用户是否同意用户使用协议
    `f_real_name_auth_status` tinyint(4) NOT NULL DEFAULT '0',      -- 实名状态
    `f_tel_number` char(40) DEFAULT NULL,                           -- 电话号码
    `f_is_activate` tinyint(4) NOT NULL DEFAULT '0',                -- 是否激活
    `f_activate_status` tinyint(4) NOT NULL DEFAULT '0',            -- 用户是否登录过系统
    `f_third_party_attr` varchar(255) NOT NULL DEFAULT '',          -- 第三方应用属性
    `f_expire_time` int(11) NOT NULL DEFAULT '-1',                  -- 用户账号有效期, 单位为秒, 默认永久有效
    `f_user_document_read_status` bigint(20) DEFAULT '0',           -- 用户的文档已读状态，目前用于快速入门
    `f_manager_id` varchar(40) NOT NULL DEFAULT '' COMMENT '用户上级ID',
    `f_code` varchar(255) NOT NULL DEFAULT '' COMMENT '用户编码',
    `f_position` varchar(50) NOT NULL DEFAULT '' COMMENT '岗位',
    `f_csf_level2` tinyint(4) NOT NULL DEFAULT '51' COMMENT '用户密级2',
    PRIMARY KEY (`f_user_id`),
    KEY `f_mail_address_index` (`f_mail_address`),
    KEY `f_domain_object_guid` (`f_domain_object_guid`),
    UNIQUE KEY `f_login_name` (`f_login_name`),
    KEY `f_display_name_index` (`f_display_name`),
    KEY `f_remark_index` (`f_remark`),
    KEY `f_idcard_number_index` (`f_idcard_number`),
    KEY `f_third_party_id_index` (`f_third_party_id`),
    KEY `f_tel_number_index` (`f_tel_number`),
    KEY `f_priority_name_index` (`f_priority`,`f_display_name`),
    KEY `idx_t_user_code` (`f_code`),
    KEY `idx_t_user_position` (`f_position`),
    KEY `idx_t_user_manager_id` (`f_manager_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_third_party_auth` (
    `f_id` int(11) NOT NULL AUTO_INCREMENT,                         -- 自增主键
    `f_app_id` varchar(128) NOT NULL,                                -- 第三方App Id
    `f_app_name` varchar(128) NOT NULL DEFAULT '',                  -- 第三方App名
    `f_enable` tinyint(1) NOT NULL DEFAULT 0,                       -- 是否启用, 1为启用, 0为禁用
    `f_config` text,                                                -- 第三方配置, 外部可见
    `f_internal_config` text,                                       -- 第三方配置, 内部使用, 外部不可见
    `f_plugin_name` varchar(255) NOT NULL,                           -- 第三方插件名称
    `f_plugin_type` tinyint(4) NOT NULL,                            -- 第三方种类, 0: 认证, 1: 消息
    `f_object_id` char(110) NOT NULL,                               -- 文件ID, 用于确定在存储中的位置(兼容旧版本文件在对象存储的key，旧版本为三段结构(evfs前缀/cid/object_id)，新版本为一段(object_id))
    `f_oss_id` char(40) NOT NULL,                                   -- 第三方插件上传对象存储
    PRIMARY KEY (`f_id`),
    UNIQUE KEY `f_app_id` (`f_app_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_person_group` (
    `f_group_id` char(40) NOT NULL,                                 -- 联系人组id
    `f_user_id` char(40) NOT NULL,                                  -- 用户id
    `f_group_name` char(128) NOT NULL,                              -- 联系人组名
    `f_person_count` bigint(20) NOT NULL,                           -- 人员总数
    PRIMARY KEY (`f_group_id`),
    KEY `f_user_id` (`f_user_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_contact_person` (
    `f_id` int(11) NOT NULL AUTO_INCREMENT,                         -- 自增主键
    `f_group_id` char(40) NOT NULL,                                 -- 联系人组id
    `f_user_id` char(40) NOT NULL,                                  -- 用户id
    CONSTRAINT `t_contact_person_ibfk_1` FOREIGN KEY (`f_group_id`) REFERENCES `t_person_group` (`f_group_id`),
    PRIMARY KEY (`f_id`),
    KEY `f_group_id` (`f_group_id`,`f_user_id`),
    KEY `f_user_id` (`f_user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1;

CREATE TABLE IF NOT EXISTS `t_domain` (
    `f_domain_id` bigint(20) NOT NULL AUTO_INCREMENT,               -- 域标识id
    `f_domain_name` varchar(253) NOT NULL,                          -- 域名
    `f_ip_address` varchar(253) NOT NULL,                           -- IP
    `f_port` bigint(20) NOT NULL,                                   -- 端口
    `f_administrator` char(50) NOT NULL,                            -- 账户
    `f_password` char(255) NOT NULL,                                -- 密码
    `f_parent_domain_id` bigint(20) NOT NULL,                       -- 父域id
    `f_domain_type` tinyint(4) NOT NULL,                            -- 域类型
    `f_status` tinyint(4) NOT NULL,                                 -- 状态, 0为禁用, 1为启用
    `f_ldap_server_type` tinyint(4) NOT NULL DEFAULT '1',           -- LDAP服务器类型
    `f_sync` tinyint(4) NOT NULL DEFAULT 0,                         -- 同步状态, -1为关闭域同步, 0为开启正向同步, 1为开启反相同步
    `f_use_ssl` tinyint(4) NOT NULL DEFAULT 0,                      -- 是否使用SSL
    `f_config` text,                                                -- 配置
    `f_key_config` text,                                            -- 关键配置
    PRIMARY KEY (`f_domain_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_failover_domain` (
    `f_domain_id` bigint(20) NOT NULL AUTO_INCREMENT,               -- 域标识id
    `f_parent_domain_id` bigint(20) NOT NULL,                       -- 主域id
    `f_ip_address` char(50) NOT NULL,                               -- IP
    `f_port` bigint(20) NOT NULL,                                   -- 端口
    `f_administrator` char(50) NOT NULL,                            -- 账户
    `f_password` char(255) NOT NULL,                                -- 密码
    `f_use_ssl` tinyint(4) NOT NULL DEFAULT 0,                      -- 是否使用SSL
    PRIMARY KEY (`f_domain_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_department` (
    `f_department_id` char(40) NOT NULL,                            -- 部门id
    `f_auth_type` tinyint(4) NOT NULL,                              -- 认证类型, 1为本地创建的部门, 2为域控导入的部门, 3为第三方导入的部门
    `f_name` char(128) NOT NULL,                                    -- 部门名
    `f_domain_object_guid` char(100) DEFAULT '',                    -- 域对象的guid
    `f_domain_path` char(255) DEFAULT '',                           -- 域路径
    `f_is_enterprise` tinyint(4) NOT NULL,                          -- 是否为组织
    `f_third_party_id` char(255) DEFAULT '',                        -- 第三方系统中的id
    `f_priority` mediumint(9) NOT NULL DEFAULT '999999',            -- 部门优先级
    `f_oss_id` char(40),                                            -- 部门下用户的对象存储
    `f_mail_address` char(150) NOT NULL DEFAULT '',                 -- 邮箱地址
    `f_path` text NOT NULL,                                         -- 部门全路径
    `f_manager_id` varchar(40) NOT NULL DEFAULT '' COMMENT '负责人ID',
    `f_status` tinyint(4) NOT NULL DEFAULT 1 COMMENT '启用状态，1：启用 2：停用',
    `f_code` varchar(255) NOT NULL DEFAULT '' COMMENT '部门编码',
    `f_remark` varchar(128) NOT NULL DEFAULT '' COMMENT '部门备注',
    PRIMARY KEY (`f_department_id`),
    UNIQUE KEY `f_department_id_index` (`f_department_id`),
    KEY `f_name_index` (`f_name`),
    KEY `f_mail_address_index` (`f_mail_address`),
    KEY `f_is_enterprise_index` (`f_is_enterprise`),
    KEY `f_third_party_id_index` (`f_third_party_id`),
    KEY `f_path_index` (`f_path`(480)),                              -- 根据中铁30w部门层级最大深度13建立索引长度，480 = 37*13-1
    KEY `idx_t_department_manager_id` (`f_manager_id`),
    KEY `idx_t_department_status` (`f_status`),
    KEY `idx_t_department_code` (`f_code`),
    KEY `idx_t_department_remark` (`f_remark`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_department_relation` (
    `f_relation_id` bigint(20) NOT NULL AUTO_INCREMENT,             -- 自增主键
    `f_department_id` char(40) NOT NULL,                            -- 部门id
    `f_parent_department_id` char(40) NOT NULL,                     -- 父部门id
    PRIMARY KEY (`f_relation_id`),
    UNIQUE KEY `f_department_id_index` (`f_department_id`),
    KEY `f_parent_department_id_index` (`f_parent_department_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_user_department_relation` (
    `f_relation_id` bigint(20) NOT NULL AUTO_INCREMENT,             -- 自增主键
    `f_user_id` char(40) NOT NULL,                                  -- 用户id
    `f_department_id` char(40) NOT NULL,                            -- 用户所在部门id
    `f_path` text NOT NULL,                                         -- 用户所在部门全路径
    PRIMARY KEY (`f_relation_id`),
    KEY `f_user_id_index` (`f_user_id`),
    KEY `f_department_id_index` (`f_department_id`),
    KEY `f_path_index` (`f_path`(480))                              -- 根据中铁30w部门层级最大深度13建立索引长度，480 = 37*13-1
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_online_user_real_time` (
    `f_time` char(40) NOT NULL,                                     -- 时间字符串
    `f_count` bigint(20) NOT NULL,                                  -- 总数
    `f_uuid` char(40) NOT NULL,                                     -- 记录的唯一标识
    PRIMARY KEY (`f_uuid`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_max_online_user_day` (
`f_time` char(40) NOT NULL,                                     -- 时间字符串
`f_count` bigint(20) NOT NULL,                                  -- 总数
`f_uuid` char(40) NOT NULL,                                     -- 记录的唯一标识
PRIMARY KEY (`f_uuid`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_max_online_user_month` (
    `f_time` char(40) NOT NULL,                                     -- 时间字符串
    `f_count` bigint(20) NOT NULL,                                  -- 总数
    `f_uuid` char(40) NOT NULL,                                     -- 记录的唯一标识
    PRIMARY KEY (`f_uuid`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_ou_user` (
    `f_id` int(11) NOT NULL AUTO_INCREMENT,                         -- 自增主键
    `f_user_id` char(40) NOT NULL,                                  -- 用户id
    `f_ou_id` char(40) NOT NULL,                                    -- OU中标识id
    PRIMARY KEY (`f_id`),
    KEY `f_user_id_index` (`f_user_id`),
    KEY `f_ou_id_index` (`f_ou_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_ou_department` (
    `f_id` int(11) NOT NULL AUTO_INCREMENT,                         -- 自增主键
    `f_department_id` char(40) NOT NULL,                            -- 部门id
    `f_ou_id` char(40) NOT NULL,                                    -- OU中标识id
    PRIMARY KEY (`f_id`),
    KEY `f_department_id_index` (`f_department_id`),
    KEY `f_ou_id_index` (`f_ou_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_license` (
  `f_license_value` char(255) NOT NULL,                             -- 许可证值
  `f_active` tinyint(4) NOT NULL DEFAULT '0',                       -- 是否激活
  `f_type` tinyint(4) NOT NULL DEFAULT '0',                         -- 类型
  `f_version` tinyint(4) NOT NULL DEFAULT '0',                      -- 许可证版本, 5: 5.0许可证, 6: 6.0许可证
  PRIMARY KEY (`f_license_value`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_license_used` (
  `f_license_value` char(255) NOT NULL,                             -- 许可证值
  `f_active_time` bigint(20) NOT NULL DEFAULT '-1',                 -- 激活时间, 微秒的时间戳
  `f_type` tinyint(4) NOT NULL DEFAULT '0',                         -- 类型
  PRIMARY KEY (`f_license_value`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_oem_config` (
  `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,               -- 自增主键
  `f_section` char(32) NOT NULL,                                    -- 内容分类
  `f_option` char(32) NOT NULL,                                     -- 选项
  `f_value` mediumblob NOT NULL,                                    -- 值
  PRIMARY KEY (`f_primary_id`),
  UNIQUE KEY `f_index_section_option` (`f_section`,`f_option`) USING BTREE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_sharemgnt_config` (
  `f_key` char(32) NOT NULL,                                        -- 配置关键字
  `f_value` varchar(1024) NOT NULL,                                 -- 配置的值
  PRIMARY KEY (`f_key`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_perm_share_strategy` (
    `f_index` int(11) NOT NULL AUTO_INCREMENT,                      -- 自增主键
    `f_strategy_id` char(40) NOT NULL,                              -- 策略id
    `f_obj_id` char(40) NOT NULL,                                   -- 对象id
    `f_obj_type` tinyint(4) NOT NULL,                               -- 对象类型
    `f_parent_id` char(40),                                         -- 父对象id
    `f_sharer_or_scope` tinyint(4) NOT NULL,                        -- 共享者或共享范围
    `f_status` tinyint(4) NOT NULL DEFAULT '0',                     -- 策略状态
    PRIMARY KEY (`f_index`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_link_share_strategy` (
    `f_sharer_id` char(40) NOT NULL,                                -- 共享者id
    `f_sharer_type` tinyint(4) NOT NULL,                            -- 共享者类型
    PRIMARY KEY (`f_sharer_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_find_share_strategy` (
    `f_sharer_id` char(40) NOT NULL,                                -- 共享者id
    `f_sharer_type` tinyint(4) NOT NULL,                            -- 共享者类型
    PRIMARY KEY (`f_sharer_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_leak_proof_strategy` (
  `f_strategy_id` bigint(20) NOT NULL AUTO_INCREMENT,               -- 自增主键
  `f_accessor_id` char(40) NOT NULL,                                -- 访问者id
  `f_accessor_type` tinyint(4) NOT NULL,                            -- 访问者类型
  `f_perm_value` int(11) DEFAULT NULL,                              -- 权限值
  PRIMARY KEY (`f_strategy_id`),
  UNIQUE KEY `f_accessor_id_index` (`f_accessor_id`) USING HASH
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_cert` (
  `f_key` char(32) COLLATE utf8mb4_bin NOT NULL,                    -- 配置关键字
  `f_value` varchar(8192) COLLATE utf8mb4_bin NOT NULL,             -- 配置的值
  PRIMARY KEY (`f_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS `t_client_update_package` (
  `f_id` int  NOT NULL AUTO_INCREMENT,                              -- 自增主键
  `f_name` varchar(150) NOT NULL,                                   -- 客户端包名
  `f_os` int  NOT NULL ,                                            -- 系统类型
  `f_size` bigint(20) NOT NULL,                                     -- 安装包大小
  `f_version` varchar(50) NOT NULL,                                 -- 安装包版本
  `f_time` varchar(50) NOT NULL,                                    -- 安装包上传时间
  `f_mode` tinyint(1) NOT NULL,                                     -- 升级类型
  `f_pkg_location` tinyint(4) NOT NULL DEFAULT '1',                 -- 升级包位置，1表示本地上传到对象存储，2表示独立配置升级包下载地址
  `f_url` text NOT NULL,                                            -- 下载地址
  PRIMARY KEY (`f_id`),
  UNIQUE KEY `f_os_index` (`f_os`) USING BTREE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_site_info` (
  `f_site_id` varchar(36) NOT NULL,                                 -- 站点id
  `f_site_ip` varchar(64) DEFAULT NULL,                             -- 站点IP
  `f_site_name` varchar(128) NOT NULL,                              -- 站点名称
  `f_site_type` tinyint(3) NOT NULL,                                -- 站点类型, 0为普通站点, 1为总站点, 2为分站点
  `f_site_link_status` tinyint(1) DEFAULT NULL,                     -- 站点连接状态
  `f_site_status` tinyint(1) NOT NULL DEFAULT '1',                  -- 站点启用状态, 1为启用, 2为禁用
  `f_site_used_space` bigint(20) NOT NULL DEFAULT '0',              -- 站点已用存储空间
  `f_site_total_space` bigint(20) NOT NULL DEFAULT '0',             -- 站点总存储空间
  `f_site_key` varchar(10) NOT NULL,                                -- 站点标识key
  `f_site_master_ip` varchar(64) DEFAULT NULL,                      -- 主站点IP
  `f_site_is_sync` tinyint(1) NOT NULL DEFAULT '0',                 -- 站点信息是否同步
  `f_site_heart_rate` bigint(20) DEFAULT NULL,                      -- 心跳信息
  `f_uniq_index` int  NOT NULL AUTO_INCREMENT,                      -- 自增主键
  `f_site_master_db_ip` varchar(64) DEFAULT NULL,                   -- 主站点数据库ip
  `f_site_need_update_virusdb` tinyint(1) NOT NULL DEFAULT '0',     -- 站点病毒库的更新状态
  PRIMARY KEY (`f_site_id`),
  UNIQUE KEY `f_uniq_index_index` (`f_uniq_index`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS `t_third_party_db` (
    `f_third_db_id` char(50) NOT NULL,                              -- 第三方数据库标识id
    `f_name` char(50) DEFAULT "",                                   -- 第三方名称
    `f_ip` char(50) NOT NULL,                                       -- 数据库IP
    `f_port` bigint(20) NOT NULL,                                   -- 数据库端口
    `f_admin` char(50) NOT NULL,                                    -- 数据库用户名
    `f_password` char(50) NOT NULL,                                 -- 数据库密码
    `f_database` char(50) NOT NULL,                                 -- 数据库
    `f_db_type` tinyint(4) NOT NULL,                                -- 数据库类型
    `f_charset` char(50) NOT NULL DEFAULT '',                       -- 数据库字符集
    `f_status` tinyint(4) NOT NULL DEFAULT 0,                       -- 状态
    `f_parent_department_id` char(50) DEFAULT "",                   -- 父部门id
    `f_third_root_name` char(50) DEFAULT "",                        -- 第三方根对象名
    `f_third_root_id` char(50) DEFAULT "",                          -- 第三方根对象id
    `f_sync_interval` int(10) DEFAULT 3600,                         -- 同步周期
    `f_space_size` char(40) DEFAULT '5368709120',                   -- 用户空间配额
    `f_user_type` tinyint(4) DEFAULT 3,                             -- 用户类型
    PRIMARY KEY (`f_third_db_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_third_depart_table` (
    `f_table_id` char(50) NOT NULL,                                 -- 第三方数据表标识id
    `f_third_db_id` char(50) NOT NULL,                              -- 第三方数据库标识id
    `f_table_name` char(50) NOT NULL,                               -- 第三方数据表名
    `f_department_id` char(50) DEFAULT "",                          -- 部门id
    `f_department_name` char(50) DEFAULT "",                        -- 部门名
    `f_deparment_priority` char(50) DEFAULT "",                     -- 部门优先级
    `f_filter` text,                                                -- 过滤条件
    `f_sub_group` text,                                             -- 下级组
    PRIMARY KEY (`f_table_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_third_depart_relation_table` (
    `f_table_id` char(50) NOT NULL,                                 -- 第三方数据表标识id
    `f_third_db_id` char(50) NOT NULL,                              -- 第三方数据库标识id
    `f_table_name` char(50) NOT NULL,                               -- 第三方数据表名
    `f_department_id` char(50) DEFAULT "",                          -- 部门id
    `f_parent_department_id` char(50) DEFAULT "",                   -- 上级部门id
    `f_parent_group_table_id` char(50) DEFAULT "",                  -- 上级组表标识id
    `f_parent_group_name` char(50) DEFAULT "",                      -- 上级组名
    `f_filter` text,                                                -- 过滤条件
    PRIMARY KEY (`f_table_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_third_user_table` (
    `f_table_id` char(50) NOT NULL,                                 -- 第三方数据表标识id
    `f_third_db_id` char(50) NOT NULL,                              -- 第三方数据库标识id
    `f_table_name` char(50) NOT NULL,                               -- 第三方数据表名
    `f_user_id` char(50) DEFAULT "",                                -- 用户id
    `f_user_login_name` char(50) DEFAULT "",                        -- 用户登录名
    `f_user_display_name` char(50) DEFAULT "",                      -- 用户显示名
    `f_user_email` char(50) DEFAULT "",                             -- 用户邮箱地址
    `f_user_password` char(50) DEFAULT "",                          -- 用户密码
    `f_user_status` char(50) DEFAULT "",                            -- 用户状态
    `f_user_priority` char(50) DEFAULT "",                          -- 用户优先级
    `f_filter` text,                                                -- 过滤条件
    PRIMARY KEY (`f_table_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_third_user_relation_table` (
    `f_table_id` char(50) NOT NULL,                                 -- 第三方数据表标识id
    `f_third_db_id` char(50) NOT NULL,                              -- 第三方数据库标识id
    `f_table_name` char(50) NOT NULL,                               -- 第三方数据表名
    `f_user_id` char(50) DEFAULT "",                                -- 用户id
    `f_parent_department_id` char(50) DEFAULT "",                   -- 上级部门id
    `f_parent_group_table_id` char(50) DEFAULT "",                  -- 上级组表id
    `f_parent_group_name` char(50) DEFAULT "",                      -- 上级组名
    `f_filter` text,                                                -- 过滤条件
    PRIMARY KEY (`f_table_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_third_auth_info` (
    `f_app_id` varchar(50) NOT NULL,                                -- 第三方App Id
    `f_app_key` char(36) NOT NULL,                                  -- 第三方App Key
    `f_enabled` tinyint(1) NOT NULL DEFAULT 1,                      -- 是否启用, 1为启用, 0为禁用
    PRIMARY KEY (`f_app_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_third_party_tool_config` (
    `f_tool_id` char(128) NOT NULL,                                 -- 工具唯一标识id
    `f_enabled` tinyint(1) NOT NULL DEFAULT 0,                      -- 是否启用, 0为禁用, 1为启用
    `f_url` text,                                                   -- url访问地址
    `f_tool_name` char(128) NOT NULL,                               -- 第三方工具名称, 仅在工具标识为"CAD"时保存, 合法名称为"hc"或"mx"
    `f_app_id` char(50),                                            -- 鉴权唯一标识
    `f_app_key` char(150),                                          -- 鉴权密钥
    PRIMARY KEY (`f_tool_id`)
)ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_net_accessors_info` (
    `f_index` int(11) NOT NULL AUTO_INCREMENT,                      -- 自增主键
    `f_id` char(40) NOT NULL,                                       -- 记录id
    `f_ip` char(15) NOT NULL,                                       -- IP
    `f_sub_net_mask` char(15) NOT NULL,                             -- 子网掩码
    `f_accessor_id` char(40),                                       -- 访问者id
    `f_accessor_type` tinyint(4) NOT NULL,                          -- 访问者类型
    PRIMARY KEY (`f_index`)
)ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_nas_node` (
  `f_uuid` varchar(40) NOT NULL,                                    -- 节点标识, UUID
  PRIMARY KEY (`f_uuid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS `t_limit_rate` (
  `f_id` varchar(40) NOT NULL,                                      -- 限速规则id
  `f_obj_id` varchar(40) NOT NULL,                                  -- 对象id
  `f_obj_type` tinyint(4) NOT NULL,                                 -- 对象类型, 1为用户, 2为部门
  `f_limit_type` tinyint(4) NOT NULL,                               -- 限速类型, 0为用户, 1为部门
  `f_upload_rate` int NOT NULL,                                     -- 上传限速值
  `f_download_rate` int NOT NULL,                                   -- 下载限速值
  PRIMARY KEY (`f_id`, `f_obj_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS `t_nginx_user_rate` (
  `f_userid` varchar(40) NOT NULL,                                  -- 用户id
  `f_parent_deptids` text NOT NULL,                                 -- 上一层有规则的父部门id, 没有时置为-1, 用户级别限速时为空
  `f_download_req_cnt` int DEFAULT 0,                               -- 下载请求数量
  `f_upload_req_cnt` int DEFAULT 0,                                 -- 上传请求数量
  `f_upload_rate` int DEFAULT 0,                                    -- 上传限速
  `f_download_rate` int DEFAULT 0,                                  -- 下载限速
  PRIMARY KEY (`f_userid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS `t_department_responsible_person` (
  `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,               -- 自增主键
  `f_department_id` char(40) NOT NULL,                              -- 部门id
  `f_user_id` char(40) NOT NULL,                                    -- 管理者的用户id
  PRIMARY KEY (`f_primary_id`),
  UNIQUE KEY `responsible_person_depart_index` (`f_user_id`,`f_department_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_watermark_config` (
    `f_for_user_doc` tinyint(4) NOT NULL DEFAULT '0',               -- 是否用于个人文档
    `f_for_custom_doc` tinyint(4) NOT NULL DEFAULT '0',             -- 是否用于自定义文档库
    `f_for_archive_doc` tinyint(4) NOT NULL DEFAULT '0',            -- 是否用于归档库
    `f_config` mediumblob NOT NULL,                                 -- 配置内容
    `f_index` int(11) NOT NULL AUTO_INCREMENT,                      -- 自增主键
    PRIMARY KEY (`f_index`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_watermark_doc` (
    `f_obj_id` char(255) NOT NULL,                                  -- 文档库对象id
    `f_watermark_type` tinyint(4) NOT NULL,                         -- 水印类型
    `f_time` bigint(20) NOT NULL,                                   -- 记录的时间
    PRIMARY KEY (`f_obj_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_link_template` (
  `f_index` int(11) NOT NULL AUTO_INCREMENT,                        -- 自增主键
  `f_template_id` char(40) NOT NULL,                                -- 模板id
  `f_template_type` tinyint(4) NOT NULL,                            -- 模板类型
  `f_sharer_id` char(40) NOT NULL,                                  -- 共享者id
  `f_sharer_type` tinyint(1) NOT NULL,                              -- 共享者类型
  `f_create_time` bigint(20) NOT NULL,                              -- 记录创建时间, 微秒的时间戳
  `f_config` text NOT NULL,                                         -- 配置信息
  PRIMARY KEY (`f_index`),
  KEY `f_template_id_index` (`f_template_id`),
  KEY `f_sharer_id_index` (`f_sharer_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_net_docs_limit_info` (
    `f_index` int(11) NOT NULL AUTO_INCREMENT,                      -- 自增主键
    `f_id` char(40) NOT NULL,                                       -- 记录id
    `f_ip` char(15) NOT NULL,                                       -- IP
    `f_sub_net_mask` char(15) NOT NULL,                             -- 子网掩码
    `f_doc_id` char(40),                                            -- 文档入口id
    PRIMARY KEY (`f_index`),
    KEY `f_doc_id_index` (`f_doc_id`)
)ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_user_verification_code` (
  `f_user_id` char(40) NOT NULL,                                    -- 用户id
  `f_vcode` varchar(40) NOT NULL,                                   -- 验证码值
  `f_create_time` bigint(20) NOT NULL,                              -- 验证码创建时间, 微秒的时间戳
  PRIMARY KEY (`f_user_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_antivirus_admin` (
    `f_user_id` char(40) NOT NULL,                                  -- 用户id
    PRIMARY KEY (`f_user_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_hide_ou` (
    `f_department_id` char(40) NOT NULL,                            -- 部门id
    PRIMARY KEY (`f_department_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_vcode` (
    `f_uuid` char(40) NOT NULL,                                     -- 记录标识, UUID
    `f_vcode` char(40) NOT NULL,                                    -- 校验码的值
    `f_vcode_type` int(4) default 1,                                -- 校验码类型, 1: 其他情况, 2: 忘记密码时创建的验证码
    `f_vcode_error_cnt` int(4) default 0,                           -- 验证码输入错误次数
    `f_createtime` datetime DEFAULT now(),                          -- 创建时间
    PRIMARY KEY (`f_uuid`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_recycle` (
    `f_cid` char(40) NOT NULL,                                      -- 文档入口CID
    `f_gns` char(80) NOT NULL,                                      -- 文档路径标识
    `f_setter` char(40) NOT NULL,                                   -- 配置者id
    `f_retention_days` int DEFAULT -1,                              -- 保留天数, -1为不自动清理
    PRIMARY KEY (`f_cid`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_sms_code` (
    `f_tel_number` char(40) NOT NULL,                               -- 电话号码
    `f_verify_code` char(6) NOT NULL,                               -- 确认码
    `f_create_time` datetime DEFAULT now(),                         -- 创建时间
    PRIMARY KEY (`f_tel_number`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_copy_limit_rate` (
  `f_obj_id` varchar(40) NOT NULL,                                  -- 限速规则id
  `f_parent_id` varchar(40) NOT NULL,                               -- 上一级有规则的父部门id
  `f_obj_type` tinyint(4) NOT NULL,                                 -- 限速对象类型, 1为用户, 2为部门
  `f_upload_rate` int NOT NULL,                                     -- 上传限速值
  `f_download_rate` int NOT NULL,                                   -- 下载限速值
  PRIMARY KEY (`f_obj_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

CREATE TABLE IF NOT EXISTS `t_active_user_day` (
    `f_id` bigint(20) NOT NULL AUTO_INCREMENT,                      -- 自增主键
    `f_time` char(40) NOT NULL,                                     -- 时间字符串
    `f_active_count` bigint(20) NOT NULL DEFAULT '0',               -- 活跃用户数
    `f_activate_count` bigint(20) NOT NULL DEFAULT '0',             -- 激活用户数
    PRIMARY KEY (`f_id`),
    UNIQUE KEY `f_time_index` (`f_time`) USING BTREE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_active_user_month` (
    `f_id` bigint(20) NOT NULL AUTO_INCREMENT,                      -- 自增主键
    `f_time` char(40) NOT NULL,                                     -- 时间字符串
    `f_active_count` bigint(20) NOT NULL DEFAULT '0',               -- 活跃用户数
    `f_total_count` bigint(20) NOT NULL DEFAULT '0',                -- 当月用户总数
    `f_activate_count` bigint(20) NOT NULL DEFAULT '0',             -- 激活用户数
    PRIMARY KEY (`f_id`),
    UNIQUE KEY `f_time_index` (`f_time`) USING BTREE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_active_user_year` (
    `f_id` bigint(20) NOT NULL AUTO_INCREMENT,                      -- 自增主键
    `f_time` char(40) NOT NULL,                                     -- 时间字符串
    `f_total_count` bigint(20) NOT NULL DEFAULT '0',                -- 当年用户总数
    `f_activate_count` bigint(20) NOT NULL DEFAULT '0',             -- 激活用户数
    PRIMARY KEY (`f_id`),
    UNIQUE KEY `f_time_index` (`f_time`) USING BTREE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_operation_problem` (
    `f_id` bigint(20) NOT NULL AUTO_INCREMENT,                      -- 自增主键
    `f_ip` char(40) NOT NULL,                                       -- IP
    `f_time` bigint(20) NOT NULL,                                   -- 异常开始至结束的时间中间值
    `f_time_from` bigint(20) NOT NULL DEFAULT '0',                  -- 异常开始时间
    `f_time_util` bigint(20) NOT NULL DEFAULT '0',                  -- 异常结束时间
    `f_obj_id` char(40) NOT NULL,                                   -- 触发异常的triggerid或历史数据异常的itemid
    `f_type` tinyint(4) NOT NULL,                                   -- 异常类型
    `f_description` text NOT NULL,                                  -- trigger或item的名称
    `f_monitoring_range` char(40) DEFAULT NULL,                     -- 异常监控值范围
    PRIMARY KEY (`f_id`),
    KEY `f_time_index` (`f_time`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_role`(
    `f_role_id` char(40) NOT NULL,                                  -- 角色id
    `f_name` char(150) NOT NULL DEFAULT '',                         -- 角色名称
    `f_description` text NOT NULL,                                  -- 角色职能描述
    `f_creator_id` char(40) NOT NULL DEFAULT '',                    -- 角色创建者id
    `f_priority` smallint(6) NOT NULL DEFAULT '999',                -- 角色权重值
    PRIMARY KEY (`f_role_id`),
    KEY `f_creator_id_index` (`f_creator_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_user_role_relation` (
    `f_user_id` char(40) NOT NULL,                                  -- 用户id
    `f_role_id` char(40) NOT NULL,                                  -- 角色id
    PRIMARY KEY (`f_user_id`, `f_role_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_department_audit_person` (
    `f_user_id` char(40) NOT NULL,                                  -- 用户id
    `f_department_id` char(40) NOT NULL,                            -- 部门id
    PRIMARY KEY (`f_user_id`, `f_department_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_user_role_attribute` (
    `f_user_id` char(40) NOT NULL,                                  -- 用户id
    `f_mail_address` varchar(1024) NOT NULL,                        -- 用户角色邮箱列表
    PRIMARY KEY (`f_user_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_file_crawl_strategy`(
    `f_strategy_id` int(8) NOT NULL AUTO_INCREMENT,                 -- 自增主键
    `f_user_id` char(40) NOT NULL,                                  -- 用户id
    `f_doc_id` char(40) NOT NULL,                                   -- 文档库路径
    `f_file_crawl_type` text NOT NULL,                              -- 抓取类型, 后缀名+空格组成
    PRIMARY KEY (`f_strategy_id`),
UNIQUE KEY `f_user_id_index` (`f_user_id`) USING BTREE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_doc_auto_archive_strategy` (
  `f_index` bigint(20) NOT NULL AUTO_INCREMENT,                     -- 唯一自增id
  `f_strategy_id` char(40) NOT NULL,                                -- 策略id，不同用户、部门的策略id可能相同，返回给前端时会合并
  `f_obj_id` char(40) NOT NULL,                                     -- 对象id，可能是用户、部门id
  `f_obj_type` tinyint(4) NOT NULL,                                 -- 对象类型，1：用户 2：部门
  `f_archive_dest_doc_id` char(40) NOT NULL,                        -- 目的归档库gns
  `f_archive_cycle` bigint(20) NOT NULL,                            -- 归档周期，天数
  `f_archive_cycle_modify_time` bigint(20) NOT NULL,                -- 归档周期的变更时间
  `f_create_time` bigint(20) NOT NULL,                              -- 记录创建时间
  PRIMARY KEY (`f_index`),
  UNIQUE KEY `f_obj_id` (`f_obj_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_doc_auto_clean_strategy` (
`f_index` bigint(20) NOT NULL AUTO_INCREMENT,       -- 唯一自增id
`f_strategy_id` char(40) NOT NULL,                  -- 使用类似于生成t_user表中f_user_id的方法生成
`f_obj_id` char(40) NOT NULL,                       -- 使用该策略的用户/部门/角色(6.0)id
`f_obj_type` tinyint(4) NOT NULL,                   -- 使用该策略的id的类型，用户1/部门2/角色4(6.0)
`f_enable_remain_hours` tinyint(4) NOT NULL,        -- 启用数据保留时间
`f_remain_hours` bigint(20) NOT NULL,               -- 数据在正常位置的保留时间
`f_clean_cycle_days` bigint(20) NOT NULL,           -- 清理周期
`f_clean_cycle_modify_time` bigint(20) NOT NULL,    -- 清理周期的变更时间
`f_create_time` bigint(20) NOT NULL,                -- 策略的创建时间
`f_status` tinyint(4) NOT NULL,                     -- 策略的启用/禁用标志位
PRIMARY KEY (`f_index`),
UNIQUE KEY `f_obj_id` (`f_obj_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_local_sync_strategy` (
    `f_index` int(11) NOT NULL AUTO_INCREMENT,                      -- 唯一自增id
    `f_strategy_id` char(40) NOT NULL,                              -- 策略id
    `f_obj_id` char(40) NOT NULL,                                   -- 对象id
    `f_obj_type` tinyint(4) NOT NULL,                               -- 对象类型, 1: 用户, 2: 部门
    `f_open_status` tinyint(4) NOT NULL,                            -- 本地同步策略开启状态
    `f_delete_status` tinyint(4) NOT NULL,                          -- 是否允许删除配置的同步任务
    `f_create_time` bigint(20) NOT NULL,                            -- 策略创建时间, 微秒的时间戳
    PRIMARY KEY (`f_index`),
    UNIQUE KEY `f_obj_id` (`f_obj_id`),
    KEY `f_strategy_id_index` (`f_strategy_id`),
    KEY `f_create_time_index` (`f_create_time`)
) ENGINE=InnoDB AUTO_INCREMENT=41;

CREATE TABLE IF NOT EXISTS `t_user_custom_attr` (
    `f_id` char(26) NOT NULL COMMENT '自定义属性id(主键)',
    `f_user_id` char(36) NOT NULL COMMENT '用户id(唯一索引)',
    `f_custom_attr` longtext NOT NULL COMMENT '自定义属性(json)',
    PRIMARY KEY (`f_id`),
    UNIQUE KEY `uk_user_id_index` (`f_user_id`) USING BTREE
) ENGINE=InnoDB COMMENT '用户自定义属性表';

INSERT INTO `t_sharemgnt_config`(`f_key`, `f_value`) SELECT 'reserved_name_lock', 'locked' FROM DUAL WHERE NOT EXISTS (SELECT `f_key` FROM `t_sharemgnt_config` WHERE `f_key` = 'reserved_name_lock');
