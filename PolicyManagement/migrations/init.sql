/*
MySQL: Database - policy_mgnt
*********************************************************************
*/
use policy_mgnt;

CREATE TABLE IF NOT EXISTS `t_policies` (
  `f_name` varchar(255) NOT NULL,
  `f_default` text NOT NULL,
  `f_value` text NOT NULL,
  `f_locked` BOOLEAN DEFAULT NULL,
  PRIMARY KEY (`f_name`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_network_restriction` (
  `f_id` varchar(36) NOT NULL,
  `f_name` varchar(128) DEFAULT NULL,
  `f_start_ip` varchar(40) NOT NULL,
  `f_end_ip` varchar(40) NOT NULL,
  `f_ip_address` varchar(40) NOT NULL,
  `f_ip_mask` varchar(15) NOT NULL,
  `f_segment_start` varchar(128) NOT NULL,
  `f_segment_end` varchar(128) NOT NULL,
  `f_type` varchar(15) NOT NULL,
  `f_ip_type` varchar(15) NOT NULL DEFAULT 'ipv4',
  `f_created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`f_id`),
  UNIQUE KEY `uix_t_network_restriction_f_name` (`f_name`),
  KEY `idx_t_network_restriction_f_start_ip` (`f_start_ip`),
  KEY `idx_t_network_restriction_f_end_ip` (`f_end_ip`),
  KEY `idx_t_network_restriction_f_ip_address` (`f_ip_address`),
  KEY `idx_t_network_restriction_f_type` (`f_type`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_network_accessor_relation` (
  `f_id` bigint(20) NOT NULL AUTO_INCREMENT,
  `f_network_id` varchar(36) NOT NULL,
  `f_accessor_id` varchar(36) NOT NULL,
  `f_accessor_type` varchar(10) NOT NULL,
  `f_created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`f_id`),
  UNIQUE KEY `idx_net_acc` (`f_network_id`,`f_accessor_id`),
  KEY `idx_t_network_accessor_relation_f_accessor_type` (`f_accessor_type`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_event_store` (
  `f_id` bigint(20) NOT NULL,
  `f_dispatched` BOOLEAN NOT NULL DEFAULT 0,
  `f_dispatched_at` datetime DEFAULT NULL,
  `f_payload` longblob NOT NULL,
  `f_options` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `f_headers` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  PRIMARY KEY (`f_id`)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `t_product_relation` (
    `f_primary_id` bigint(20) NOT NULL AUTO_INCREMENT COMMENT '自增主键',                     
    `f_account_id` varchar(40) NOT NULL COMMENT '授权对象id',
    `f_account_type` tinyint(4) NOT NULL COMMENT '授权类型，0：未知，1：普通用户',                              
    `f_product` varchar(255) NOT NULL COMMENT '产品名称，由license规定',                          
    PRIMARY KEY (`f_primary_id`) COMMENT '主键',
    KEY `idx_t_product_relation_f_account_id_f_product` (`f_account_id`, `f_product`),
    KEY `idx_t_product_relation_f_product_f_account_id` (`f_product`, `f_account_id`)
) ENGINE=InnoDB COMMENT='产品授权关系表';

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
