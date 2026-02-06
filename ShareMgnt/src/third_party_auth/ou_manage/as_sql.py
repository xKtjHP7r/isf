#!/usr/bin/python3
# -*- coding:utf-8 -*-

from src.common.db.db_manager import get_db_name

check_user_exist_by_id_sql = """
SELECT COUNT(*) AS cnt
FROM `t_user`
WHERE `f_third_party_id` = %s
"""

check_user_exist_by_name_sql = """
SELECT COUNT(*) AS cnt
FROM `t_user`
WHERE `f_login_name` = %s
"""

insert_user_sql = """
INSERT INTO `t_user`
(`f_user_id`, `f_login_name`, `f_display_name`,
`f_password`, `f_mail_address`, `f_auth_type`, `f_status`,
`f_third_party_id`, `f_domain_path`, `f_ldap_server_type`,
`f_pwd_timestamp`, `f_pwd_error_latest_timestamp`, `f_priority`,
`f_oss_id`, `f_third_party_attr`, `f_expire_time`, `f_idcard_number`,`f_tel_number`, `f_csf_level`,
`f_position`, `f_code`, `f_csf_level2`)
VALUES(%s, %s, %s, %s, %s, %s, %s,
%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
"""
insert_domain_user_sql = """
INSERT INTO `t_user`
(`f_user_id`, `f_login_name`, `f_display_name`,
`f_password`, `f_mail_address`, `f_auth_type`, `f_status`,
`f_third_party_id`, `f_domain_path`, `f_ldap_server_type`,
`f_pwd_timestamp`, `f_pwd_error_latest_timestamp`, `f_priority`,
`f_oss_id`, `f_third_party_attr`, `f_expire_time`, `f_idcard_number`,`f_tel_number`,`f_csf_level`,
`f_position`, `f_code`, `f_csf_level2`)
VALUES(%s, %s, %s, %s, %s, %s, %s,
%s, %s, %s, now(), now(), %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
"""

insert_user_depart_sql = """
INSERT INTO `t_user_department_relation`
(`f_user_id`, `f_department_id`, `f_path`)
VALUES(%s, %s, %s)
"""

delete_user_depart_sql = """
DELETE FROM `t_user_department_relation`
WHERE `f_user_id` = %s AND `f_path` = %s
"""

check_loginname_excluede_third_id_sql = """
SELECT `f_user_id`, `f_display_name` FROM `t_user`
WHERE (`f_login_name` = %s AND `f_third_party_id` != %s)
LIMIT 1
"""

update_user_by_third_id_sql = """
UPDATE `t_user` SET `f_login_name` = %s,
`f_display_name` = %s,
`f_password` = %s,
`f_mail_address` = %s,
`f_domain_path` = %s,
`f_ldap_server_type` = %s,
`f_auth_type` = %s,
`f_priority` = %s,
`f_third_party_attr` = %s,
`f_idcard_number` = %s,
`f_tel_number` = %s,
`f_position` = %s,
`f_code` = %s,
`f_csf_level2` = %s
WHERE `f_third_party_id` = %s
"""

update_user_by_third_id_contain_csf_sql = """
UPDATE `t_user` SET `f_login_name` = %s,
`f_display_name` = %s,
`f_password` = %s,
`f_mail_address` = %s,
`f_domain_path` = %s,
`f_ldap_server_type` = %s,
`f_auth_type` = %s,
`f_priority` = %s,
`f_third_party_attr` = %s,
`f_idcard_number` = %s,
`f_tel_number` = %s,
`f_csf_level` = %s,
`f_position` = %s,
`f_code` = %s,
`f_csf_level2` = %s
WHERE `f_third_party_id` = %s
"""

update_user_by_third_id_without_pwd_sql = """
UPDATE `t_user` SET `f_login_name` = %s,
`f_display_name` = %s,
`f_mail_address` = %s,
`f_domain_path` = %s,
`f_ldap_server_type` = %s,
`f_auth_type` = %s,
`f_priority` = %s,
`f_third_party_attr` = %s,
`f_idcard_number` = %s,
`f_tel_number` = %s,
`f_csf_level` = %s,
`f_position` = %s,
`f_code` = %s,
`f_csf_level2` = %s
WHERE `f_third_party_id` = %s
"""

update_user_by_third_id_without_pwd_csf_sql = """
UPDATE `t_user` SET `f_login_name` = %s,
`f_display_name` = %s,
`f_mail_address` = %s,
`f_domain_path` = %s,
`f_ldap_server_type` = %s,
`f_auth_type` = %s,
`f_priority` = %s,
`f_third_party_attr` = %s,
`f_idcard_number` = %s,
`f_tel_number` = %s,
`f_position` = %s,
`f_code` = %s,
`f_csf_level2` = %s
WHERE `f_third_party_id` = %s
"""

update_user_by_user_id_sql = """
UPDATE `t_user` SET `f_login_name` = %s,
`f_display_name` = %s,
`f_password` = %s,
`f_mail_address` = %s,
`f_third_party_id` = %s,
`f_auth_type` = %s,
`f_domain_path` =%s,
`f_ldap_server_type` = %s,
`f_priority` = %s,
`f_idcard_number` = %s,
`f_tel_number` = %s,
`f_position` = %s,
`f_code` = %s,
`f_csf_level2` = %s
WHERE `f_user_id` = %s
"""

update_user_by_user_id_contain_csf_sql = """
UPDATE `t_user` SET `f_login_name` = %s,
`f_display_name` = %s,
`f_password` = %s,
`f_mail_address` = %s,
`f_third_party_id` = %s,
`f_auth_type` = %s,
`f_domain_path` =%s,
`f_ldap_server_type` = %s,
`f_priority` = %s,
`f_idcard_number` = %s,
`f_tel_number` = %s,
`f_csf_level` = %s,
`f_position` = %s,
`f_code` = %s,
`f_csf_level2` = %s
WHERE `f_user_id` = %s
"""

update_user_by_user_id_without_pwd_sql = """
UPDATE `t_user` SET `f_login_name` = %s,
`f_display_name` = %s,
 `f_mail_address` = %s,
 `f_third_party_id` = %s,
`f_auth_type` = %s,
`f_domain_path` =%s,
`f_ldap_server_type` = %s,
`f_priority` = %s,
`f_idcard_number` = %s,
`f_tel_number` = %s,
`f_csf_level` = %s,
`f_position` = %s,
`f_code` = %s,
`f_csf_level2` = %s
WHERE `f_user_id` = %s
"""

update_user_by_user_id_without_pwd_csf_sql = """
UPDATE `t_user` SET `f_login_name` = %s,
`f_display_name` = %s,
 `f_mail_address` = %s,
 `f_third_party_id` = %s,
`f_auth_type` = %s,
`f_domain_path` =%s,
`f_ldap_server_type` = %s,
`f_priority` = %s,
`f_idcard_number` = %s,
`f_tel_number` = %s,
`f_position` = %s,
`f_code` = %s,
`f_csf_level2` = %s
WHERE `f_user_id` = %s
"""

set_user_status_sql = """
UPDATE `t_user`
SET `f_status` = %s
WHERE `f_user_id` = %s
"""

insert_user_ou_sql = """
INSERT INTO `t_ou_user`
(`f_user_id`, `f_ou_id`)
VALUES(%s,%s)
"""

check_user_ou_sql = """
SELECT * FROM  `t_ou_user`
WHERE `f_user_id` = %s AND `f_ou_id` = %s
"""

check_user_depart_sql = """
SELECT * FROM `t_user_department_relation`
WHERE `f_user_id` = %s AND `f_path` = %s
"""

insert_group_sql = """
INSERT INTO `t_person_group`
(`f_group_id`, `f_user_id`, `f_group_name`, `f_person_count`)
VALUES(%s,%s,%s,0)
"""

insert_depart_sql = """
INSERT INTO `t_department`
(`f_department_id`, `f_auth_type`, `f_name`, `f_is_enterprise`,
`f_third_party_id`, `f_domain_path`, `f_priority`, `f_oss_id`, `f_path`,
`f_remark`, `f_status`, `f_code`)
VALUES(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
"""

select_third_id_sql = """
SELECT `f_department_id` from `t_department`
WHERE `f_third_party_id` = %s
"""

select_dept_oss_id_sql = """
SELECT `f_oss_id` from `t_department`
WHERE `f_department_id` = %s
"""

select_depart_by_depart_id_sql = """
SELECT * from `t_department`
WHERE `f_department_id` = %s
"""

select_depart_ou_id_sql = """
SELECT `f_ou_id` FROM `t_ou_department`
WHERE `f_department_id` = %s
"""

check_depart_belong_ou_sql = """
SELECT COUNT(*) AS cnt FROM `t_ou_department`
WEHRE `f_department_id` = %s AND `f_ou_id` = %s
"""

select_sub_user_by_dept_third_id_sql = """
SELECT  *
FROM `t_user`
WHERE `f_third_party_id` = %s
"""

select_depart_by_third_id_sql = """
SELECT `f_name`, `f_department_id`, `f_priority`, `f_status`, `f_manager_id`, `f_code`, `f_remark` FROM `t_department`
WHERE `f_third_party_id` = %s
"""

check_dept_exist_by_id_sql = """
SELECT COUNT(*) AS cnt
FROM `t_department`
WHERE `f_third_party_id` = %s
"""

insert_depart_relation_sql = """
INSERT INTO `t_department_relation`
(`f_department_id`, `f_parent_department_id`)
 VALUES(%s, %s)
 """

insert_depart_ou_sql = """
INSERT INTO `t_ou_department`
(`f_department_id`, `f_ou_id`)
VALUES(%s, %s)
"""

check_depart_exists_sql = """
SELECT COUNT(*)  as cnt
FROM `t_department`
WHERE `f_department_id` IN
(SELECT `f_department_id` FROM `t_department_relation`
WHERE `f_parent_department_id` = %s)
"""

update_depart_sql = """
UPDATE `t_department` SET `f_name` = %s,
`f_domain_path` = %s,
`f_priority` = %s,
`f_remark` = %s,
`f_status` = %s,
`f_code` = %s
WHERE `f_third_party_id` = %s
"""

# 删除用户部门关系
del_relation_sql = """
DELETE FROM `t_user_department_relation`
WHERE `f_user_id` = %s AND `f_path` = %s
"""

# 删除用户组织索引
del_ou_sql = """
DELETE FROM `t_ou_user`
WHERE `f_user_id` = %s
"""

# 插入用户部门关系
insert_relation_sql = """
INSERT INTO `t_user_department_relation`
(`f_user_id`, `f_department_id`, `f_path`)
VALUES(%s, %s, %s)
"""

# 检查用户是否属于其他部门
check_user_belong_other_depart_sql = """
SELECT COUNT(*) AS cnt FROM `t_user_department_relation`
WHERE `f_user_id` = %s
AND `f_path` != %s
"""

# 检查用户所属本组织下的部门
check_user_belong_other_dept_same_ou_sql = """
SELECT COUNT(*) AS cnt
FROM `t_user_department_relation`
WHERE `f_user_id` = %s
    AND `f_path` like %s
    AND `f_path` != %s
"""

# 检查用户是否属于某个部门
check_user_belong_depart = """
SELECT COUNT(*) AS cnt FROM `t_user_department_relation`
WHERE `f_user_id` = %s AND `f_department_id` = %s
"""

select_sub_depart_sql = """
SELECT `t_department`.`f_department_id`,
`t_department`.`f_name`,
`t_department`.`f_third_party_id`,
`t_department`.`f_priority`,
`t_department`.`f_remark`,
`t_department`.`f_status`,
`t_department`.`f_code`,
`t_department`.`f_manager_id`
FROM `t_department`
WHERE `f_path` like %s
"""

select_sub_user_sql = """
SELECT `user`.`f_user_id`,`user`.`f_login_name`,
`user`.`f_password`, `user`.`f_display_name`,
`user`.`f_mail_address`, `user`.`f_idcard_number`, `user`.`f_tel_number`,`user`.`f_third_party_id`,
`user`.`f_status`, `user`.`f_domain_path`,
`user`.`f_auth_type`, `user`.`f_priority`, `user`.`f_third_party_attr`, `user`.`f_csf_level`, `user`.`f_code`, `user`.`f_position`,
`user`.`f_csf_level2`
FROM `t_user` AS `user`
JOIN `t_user_department_relation` AS `relation`
ON `relation`.`f_user_id` = `user`.`f_user_id`
WHERE `relation`.`f_path` = %s
    AND `relation`.`f_user_id` != %s
    AND `relation`.`f_user_id` != %s
    AND `relation`.`f_user_id` != %s
    AND `user`.`f_third_party_id` != ''
"""

# 获取用户所属的部门
select_user_belong_depart_id = """
SELECT `f_department_id` FROM `t_user_department_relation`
WHERE `f_user_id` = %s
"""

# 获取某一组织下用户所属的部门
select_belong_depart_in_ou_sql = """
SELECT `f_path` FROM `t_user_department_relation`
WHERE `f_user_id` = %s AND `f_path` like %s
"""

# 获取所有的组织管理员id
select_responsible_person_id = """
SELECT DISTINCT f_user_id FROM t_department_responsible_person
"""

# 清除部门文档及数据库信息
del_depart_sql_list = [
    # 删除用户组成员
    f"""
    DELETE FROM {get_db_name('user_management')}.t_group_member
    WHERE `f_member_id` = %s
    """,
    # 删除该部门关联记录
    """
    DELETE FROM `t_department_relation`
    WHERE `f_department_id` = %s
    """,
    # 删除该部门记录
    """
    DELETE FROM `t_department`
    WHERE `f_department_id` = %s
    """,
    # 删除部门索引记录
    """
    DELETE FROM `t_ou_department`
    WHERE `f_department_id` = %s
    """,
    # 删除权限共享范围策略信息
    """
    DELETE FROM `t_perm_share_strategy` WHERE `f_obj_id` = %s
    """,
    # 删除外链共享策略信息
    """
    DELETE FROM `t_link_share_strategy` WHERE `f_sharer_id` = %s
    """,
    # 删除发现共享策略信息
    """
    DELETE FROM `t_find_share_strategy` WHERE `f_sharer_id` = %s
    """,
    # 删除防泄密策略信息
    """
    DELETE FROM `t_leak_proof_strategy` WHERE `f_accessor_id` = %s
    """,
    # 删除组织管理员关系
    """
    DELETE FROM `t_department_responsible_person` WHERE `f_department_id` = %s
    """,
    # 删除内外链共享模板
    """
    DELETE FROM `t_link_template` WHERE `f_sharer_id` = %s
    """,
    # 删除自动归档策略
    """
    DELETE FROM `t_doc_auto_archive_strategy` WHERE `f_obj_id` = %s
    """,
    # 删除组织审计员关系
    """
    DELETE FROM `t_department_audit_person` WHERE `f_department_id` = %s
    """,
    # 删除自动清理策略
    """
    DELETE FROM `t_doc_auto_clean_strategy` WHERE `f_obj_id` = %s
    """,
    # 删除本地同步策略
    """
    DELETE FROM `t_local_sync_strategy` WHERE `f_obj_id` = %s
    """,
]
