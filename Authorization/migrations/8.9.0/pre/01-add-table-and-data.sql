/*
MySQL: Database - anyshare
*********************************************************************
*/
use anyshare;

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