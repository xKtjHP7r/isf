/*
MySQL: Database - anyshare
*********************************************************************
*/
use anyshare;

CREATE TABLE IF NOT EXISTS `t_resource_type`
(
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_id`  char(40) NOT NULL COMMENT '资源类型唯一标识',
    `f_name`  char(255) NOT NULL COMMENT '资源类型名称',
    `f_description`  text NOT NULL COMMENT '资源类型描述',
    `f_instance_url`  text NOT NULL COMMENT '资源类型实例URL',
    `f_data_struct`  char(40) NOT NULL COMMENT '数据结构, 支持tree、array、string',
    `f_operation`   longtext NOT NULL COMMENT '操作, 内容是json数组',
    `f_hidden` tinyint(4) NOT NULL COMMENT '是否隐藏, 0: 不隐藏, 1: 隐藏',
    `f_create_time` bigint(20) NOT NULL COMMENT '创建时间',
    `f_modify_time` bigint(20) NOT NULL COMMENT '修改时间',
    UNIQUE KEY `uk_id` (`f_id`),
    PRIMARY KEY (`f_primary_id`)
) ENGINE = InnoDB COMMENT='资源类型表';


CREATE TABLE IF NOT EXISTS `t_policy`
(
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_id` char(40) NOT NULL COMMENT '策略ID',
    `f_resource_id`  char(40) NOT NULL COMMENT '资源实例ID',
    `f_resource_type` char(40) NOT NULL COMMENT '资源类型',
    `f_resource_name`  char(255) NOT NULL COMMENT '资源名称',
    `f_accessor_id`  char(40) NOT NULL COMMENT '访问者',
    `f_accessor_type`  tinyint(4) NOT NULL COMMENT '访问者类型 1: 用户, 2: 组织/部门,  5: 用户组， 6: 应用账户 7: 角色' ,
    `f_accessor_name`  varchar(150) NOT NULL COMMENT '访问者名称',
    `f_operation`     longtext   NOT NULL COMMENT '操作',
    `f_condition`     longtext   NOT NULL COMMENT '条件',
    `f_ancestors`     longtext   NOT NULL COMMENT '祖先信息',
    `f_end_time` bigint(20) NOT NULL COMMENT '过期时间',
    `f_create_time` bigint(20) NOT NULL COMMENT '创建时间',
    `f_modify_time` bigint(20) NOT NULL COMMENT '修改时间',
    KEY `idx_f_id` (`f_id`),
    KEY `idx_accessor_type` (`f_accessor_id`, `f_accessor_type`),
    KEY `idx_resource_type_accessor` (`f_resource_type`, `f_accessor_id`),
    KEY `idx_resource_id_type_accessor` (`f_resource_id`, `f_resource_type`, `f_accessor_id`),
    PRIMARY KEY (`f_primary_id`)
) ENGINE = InnoDB COMMENT ='策略配置表';


CREATE TABLE IF NOT EXISTS `t_role` (
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_id` char(40) NOT NULL COMMENT '角色唯一标识',
    `f_name` varchar(512) NOT NULL COMMENT '角色名称',
    `f_description` text NOT NULL COMMENT '描述',
    `f_source` tinyint(4) NOT NULL DEFAULT 3 COMMENT '角色来源, 1: 系统, 2: 业务内置, 3: 用户自定义',
    `f_visibility` tinyint(4) NOT NULL COMMENT '是否可见, 0: 不可见, 1: 可见',
    `f_resource_scope` longtext   NOT NULL COMMENT '资源范围',
    `f_created_time` bigint(40) NOT NULL COMMENT '创建时间',
    `f_modify_time`  bigint(20) NOT NULL COMMENT '修改时间',
    KEY `idx_t_role_f_id` (`f_id`),
    PRIMARY KEY (`f_primary_id`)
) ENGINE=InnoDB COMMENT='角色表';


CREATE TABLE IF NOT EXISTS `t_role_member` (
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_role_id` char(40) NOT NULL COMMENT '角色唯一标识',
    `f_member_id` char(40) NOT NULL COMMENT '成员唯一标识',
    `f_member_type` tinyint(4) NOT NULL COMMENT '成员类型,1: 用户, 2: 组织/部门, 5: 用户组 6: 应用账户',
    `f_member_name` varchar(150) NOT NULL COMMENT '成员名称',
    `f_created_time` bigint(40) NOT NULL COMMENT '创建时间',
    `f_modify_time`  bigint(20) NOT NULL COMMENT '修改时间',
    KEY `idx_f_role_id` (`f_role_id`),
    KEY `idx_f_member_id` (`f_member_id`),
    PRIMARY KEY (`f_primary_id`)
) ENGINE=InnoDB COMMENT='角色成员表';

CREATE TABLE IF NOT EXISTS `t_obligation_type` (
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_id`  char(255) NOT NULL COMMENT '义务类型唯一标识',
    `f_name`  char(255) NOT NULL COMMENT '义务类型名称',
    `f_description`  text NOT NULL COMMENT '义务类型描述',
    `f_applicable_resource_types`   longtext NOT NULL COMMENT '资源类型范围, 格式是json',
    `f_schema`   longtext NOT NULL COMMENT '参数配置，格式为JSON Schema',
    `f_ui_schema`   longtext NOT NULL COMMENT 'uiSchema, 格式是json',
    `f_default_value`   longtext NOT NULL COMMENT '义务类型默认值, 格式是json',
    `f_created_at` bigint(40) NOT NULL COMMENT '创建时间',
    `f_modified_at`  bigint(20) NOT NULL COMMENT '修改时间',
    KEY `idx_f_id` (`f_id`),
    PRIMARY KEY (`f_primary_id`)
) ENGINE = InnoDB COMMENT='义务类型表';

CREATE TABLE IF NOT EXISTS `t_obligation` (
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_id`  char(40) NOT NULL COMMENT '义务唯一标识',
    `f_type_id`  char(255) NOT NULL COMMENT '义务类型',
    `f_name`  varchar(255) NOT NULL COMMENT '义务名称',
    `f_description`  text NOT NULL COMMENT '义务描述',
    `f_value`   longtext NOT NULL COMMENT '义务配置, 格式是json',
    `f_created_at` bigint(40) NOT NULL COMMENT '创建时间',
    `f_modified_at`  bigint(20) NOT NULL COMMENT '修改时间',
    KEY `idx_f_id` (`f_id`),
    KEY `idx_f_type_id` (`f_type_id`),
    PRIMARY KEY (`f_primary_id`)
) ENGINE = InnoDB COMMENT='义务表';


CREATE TABLE IF NOT EXISTS `t_resource_type_hierarchy`
(
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT,
    `f_resource_type_id`  char(40) NOT NULL COMMENT '根节点的资源类型唯一标识',
    `f_children`          longtext NOT NULL COMMENT '下级节点信息',
    `f_created_at` bigint(20) NOT NULL COMMENT '创建时间',
    `f_modified_at` bigint(20) NOT NULL COMMENT '修改时间',
    UNIQUE KEY `uk_resource_type_id` (`f_resource_type_id`),
    PRIMARY KEY (`f_primary_id`)
) ENGINE = InnoDB COMMENT='资源类型层级关系表';
