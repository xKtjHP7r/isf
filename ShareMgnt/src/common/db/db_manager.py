import MySQLdb
import io
import os
import json
import uuid
import time
import sys
import warnings
import rdsdriver
from eisoo import logger, clusterconf
from src.common import global_info
from src.common.lib import (raise_exception,
                            generate_random_key)
from src.common.business_date import BusinessDate
from EThriftException.ttypes import ncTException
from ShareMgnt.ttypes import (ncTUsrmUserType,
                              ncTUsrmDepartType,
                              ncTTemplateType)
from ShareMgnt.constants import (NCT_USER_ADMIN,
                                 NCT_USER_AUDIT,
                                 NCT_USER_SYSTEM,
                                 NCT_USER_SECURIT,
                                 NCT_DEFAULT_ORGANIZATION,
                                 NCT_ALL_USER_GROUP,
                                 NCT_DIRECT_DEPARTMENT,
                                 NCT_DIRECT_ORGANIZATION,
                                 NCT_SYSTEM_ROLE_SUPPER,
                                 NCT_SYSTEM_ROLE_ADMIN,
                                 NCT_SYSTEM_ROLE_SECURIT,
                                 NCT_SYSTEM_ROLE_AUDIT,
                                 NCT_SYSTEM_ROLE_ORG_MANAGER,
                                 NCT_SYSTEM_ROLE_ORG_AUDIT)
from src.common.http import pub_nsq_msg
from src.common.sharemgnt_logger import ShareMgnt_Log

TOPIC_DEPT_CREATED = "core.user_management.dept.created"


def get_db_name(db_name: str) -> str:
    return f"{global_info.SYSTEM_ID}{db_name}"


class SharemgntDBManager(object):
    """
    ShareMgnt数据库管理类
    """
    def __init__(self, b_test=False):
        """
        初始化函数,获取MQ_IP,获取失败则集群没有激活
        """
        self.table_list = ['t_user', 't_person_group', 't_contact_person',
              't_domain', 't_department', 't_department_relation',
              't_user_department_relation', 't_online_user_real_time',
              't_max_online_user_day', 't_max_online_user_month',
              't_ou_user', 't_ou_department', 't_license', 't_license_used',
              't_third_party_auth', 't_oem_config', 't_sharemgnt_config',
              't_perm_share_strategy', 't_link_share_strategy', 't_find_share_strategy',
              't_leak_proof_strategy', 't_cert', 't_client_update_package',
              't_site_info', 't_third_party_db',
              't_third_depart_table', 't_third_depart_relation_table',
              't_third_user_table', 't_third_user_relation_table', 't_third_auth_info',
              't_third_party_tool_config', 't_net_accessors_info',
              't_nas_node', 't_limit_rate',
              't_nginx_user_rate', 't_department_responsible_person',
              't_watermark_config', 't_watermark_doc',
              't_link_template', 't_net_docs_limit_info',
              't_user_verification_code', 't_antivirus_admin',
              't_hide_ou', 't_vcode', 't_sms_code', 't_copy_limit_rate',
              't_active_user_day', 't_active_user_month', 't_active_user_year', 't_operation_problem',
              't_role', 't_user_role_relation', 't_department_audit_person', 't_user_role_attribute',
              't_file_crawl_strategy', 't_doc_auto_archive_strategy', 't_doc_auto_clean_strategy',
              't_local_sync_strategy']

    def init_conn(self):
        """
        初始化数据库连接
        """
        self.conn = rdsdriver.connect(host=global_info.DB_WRITE_IP,
                                      port=global_info.DB_PORT,
                                      user=global_info.DB_USER,
                                      password=global_info.DB_PWD,
                                      database=global_info.DB_NAME)

    def close_conn(self):
        """
        关闭相关的数据库连接
        """
        if self.conn:
            self.conn.close()

    def check_db_ip_port(self):
        """
        检查数据库读写IP以及端口
        参数：无
        返回值：
              True：数据库IP和端口已设置
              False:数据库IP和端口未设置
        """
        if (global_info.DB_READ_IP and global_info.DB_WRITE_IP and global_info.DB_PORT):
            return True
        return False

    def get_db_ip_port(self):
        """
        获取数据库读写IP及端口
        """
        # 获取集群自带数据库的IP和端口
        try:
            # 判断是否使用第三方数据库
            if not clusterconf.ClusterConfig.if_use_external_db():
                db_ip = clusterconf.ClusterConfig.get_db_host()
                db_port = clusterconf.ClusterConfig.get_db_port()

            else:
                db_info = clusterconf.ClusterConfig.get_external_db_info()
                db_ip = db_info['db_host']
                db_port = db_info['db_port']
                global_info.DB_USER = db_info['db_user']
                global_info.DB_PWD = db_info['db_password']

            global_info.DB_WRITE_IP = db_ip
            global_info.DB_READ_IP = db_ip
            global_info.DB_PORT = db_port

            global_info.DB_TYPE = os.getenv('DB_TYPE', 'mysql').lower()
            return True
        except ncTException:
            return False

    def check_db_service(self):
        """
        检查数据库服务，步骤如下：
            1.检查数据库IP和端口，已设置则说明数据库服务正常
            2.数据库IP和端口未设置，则从EDBC获取IP和端口
            3.从EDBC获取不到IP和端口，则说明数据库实例没有创建，则创建数据库实例.
            4.初始化数据信息
        参数：无
        返回值：无
        """
        # 检查数据库IP和端口
        if self.check_db_ip_port():
            return

        # 获取数据库IP和端口
        is_exists = self.get_db_ip_port()

        # 如果没有创建数据库，则抛错
        if not is_exists:
            raise_exception('sharemgnt 数据库尚未创建')

    def create_test_sharemgnt_db(self):
        """
        检查单元测试数据库
        """
        conn = MySQLdb.connect(host=global_info.DB_WRITE_IP,
                               port=global_info.DB_PORT,
                               user=global_info.DB_USER,
                               passwd=global_info.DB_PWD)
        cursor = conn.cursor()
        cursor.execute('SET NAMES utf8;')

        create_str = 'CREATE DATABASE IF NOT EXISTS %s' % global_info.DB_NAME
        cursor.execute(create_str)
        conn.commit()

    def create_test_ets_db(self):
        """
        检查单元测试数据库
        """
        conn = MySQLdb.connect(host=global_info.DB_WRITE_IP,
                               port=global_info.DB_PORT,
                               user=global_info.DB_USER,
                               passwd=global_info.DB_PWD,
                               charset='utf8mb4')
        cursor = conn.cursor()
        cursor.execute('SET NAMES utf8;')

        create_str = 'CREATE DATABASE IF NOT EXISTS ets'
        cursor.execute(create_str)
        conn.commit()

        SPACE_QUOTA = """
        CREATE TABLE IF NOT EXISTS `space_quota` (                        -- 此表保存配额空间信息
        `cid` char(40) NOT NULL,                                        -- 入口文档id
        `quota` bigint(20) DEFAULT '0',                                 -- 总配额
        `usedsize` bigint(20) DEFAULT '0',                              -- 已用大小
        PRIMARY KEY (`cid`)
        ) ENGINE=InnoDB;
        """

        cursor.execute('use ets;')
        cursor.execute(SPACE_QUOTA)
        conn.commit()

    def create_test_anyshare_db(self):
        """
        检查单元测试数据库
        """
        conn = MySQLdb.connect(host=global_info.DB_WRITE_IP,
                               port=global_info.DB_PORT,
                               user=global_info.DB_USER,
                               passwd=global_info.DB_PWD,
                               charset='utf8mb4')
        cursor = conn.cursor()
        cursor.execute('SET NAMES utf8;')

        create_str = 'CREATE DATABASE IF NOT EXISTS ets'
        cursor.execute(create_str)
        conn.commit()

        create_str = 'CREATE DATABASE IF NOT EXISTS anyshare'
        cursor.execute(create_str)
        conn.commit()

        t_acs_doc = """
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
        `f_oss_id` char(150) NOT NULL DEFAULT '',                     -- 文档入口所属对象存储id
        `f_relate_depart_id` char(40) NOT NULL DEFAULT '',            -- 关联部门id
        `f_display_order` MEDIUMINT DEFAULT -1,                       -- 自定义文档库显示顺序
        PRIMARY KEY (`f_doc_id`),
        KEY `t_doc_f_doc_type_index` (`f_doc_type`) USING BTREE,
        KEY `t_doc_f_obj_id_index` (`f_obj_id`) USING BTREE,
        KEY `t_doc_f_name_index` (`f_name`) USING BTREE,
        KEY `t_doc_f_type_name_index` (`f_type_name`) USING BTREE,
        KEY `t_doc_f_relate_depart_id_index` (`f_relate_depart_id`) USING BTREE,
        KEY `t_display_order_index` (`f_display_order`) USING BTREE,
        KEY `t_doc_f_creater_id_index` (`f_creater_id`) USING BTREE
        ) ENGINE=InnoDB;
        """

        cursor.execute('use anyshare;')
        cursor.execute(t_acs_doc)
        conn.commit()

    def init_db(self):
        """
        初始化数据库
        """
        self.init_conn()

        # 初始化数据库数据
        self.init_datas()

        self.close_conn()

    def delete_tables(self):
        """
        删除所有数据库表 (单元测试使用)
        """
        self.init_conn()

        cursor = self.conn.cursor()
        sql = "DROP TABLE IF EXISTS `{0}`"
        for table in self.table_list[::-1]:
            drop_sql = sql.format(table)
            cursor.execute(drop_sql)
        self.conn.commit()

        # 删除anyshare数据库表
        sql = "DROP TABLE IF EXISTS anyshare.t_acs_doc"
        cursor.execute(sql)
        self.conn.commit()
        cursor.close()

    def clear_tables(self):
        """
        清空所有的表数据(单元测试使用)
        """
        self.init_conn()

        cursor = self.conn.cursor()
        sql = "DELETE FROM `{0}`"
        for table in self.table_list[::-1]:
            drop_sql = sql.format(table)
            cursor.execute(drop_sql)

        self.conn.commit()
        cursor.close()

    def init_datas(self):
        """
        初始化数据库数据
        """
        # 初始化组织结构
        self.__init_organization()

        # 初始化管理员账号
        self.__init_admin()

        # 初始化sharemgnt配置信息
        self.__init_sharemgnt_cofing()

        # 初始化权限共享范围信息
        self.__init_perm_share_strategy()

        # 初始化OEM信息
        self.__init_oem()

        # 初始化t_link_template
        self.__init_link_template()

        # 初始化水印配置
        self.__init_watermark_config()

        # 初始化角色配置
        self.__init_role_config()

        # 初始化自动清理策略
        self.__init_auto_clean_config()

        # 初始化本地同步配置
        self.__init_local_sync_config()

    def check_ou_init(self):
        """
        检查组织是否初始化
        """
        query_sql = """
        SELECT *
        FROM t_sharemgnt_config
        WHERE f_key = 'defaule_ou_init'
        """
        cursor = self.conn.cursor()
        cursor.execute(query_sql)
        result = cursor.fetchall()
        if not result:
            return False
        return True

    def __init_organization(self):
        """
        检查组织结构
        """
        if self.check_ou_init():
            return
        query_sql = """SELECT * from `t_department` WHERE `f_department_id` = %s"""

        insert_sql = """
        INSERT INTO t_ou_department (f_department_id, f_ou_id)
        VALUES('{0}', '{0}')
        """

        I_ORGANIZATION = """
        INSERT INTO t_department (f_department_id, f_auth_type,
        f_name, f_is_enterprise, f_oss_id, f_mail_address, f_path)
        VALUES ( '{0}', {1}, '{2}', 1, '{3}', '', '{4}')
        """

        cursor = self.conn.cursor()
        cursor.execute(
            query_sql, (NCT_DEFAULT_ORGANIZATION,))
        result = cursor.fetchall()
        if not result:
            cursor.execute(I_ORGANIZATION.format(
                           NCT_DEFAULT_ORGANIZATION,
                           ncTUsrmDepartType.NCT_DEPART_TYPE_LOCAL,
                           _("IDS_DEFAULT_ORGANIZATION"), "", NCT_DEFAULT_ORGANIZATION))

            cursor.execute(insert_sql.format(NCT_DEFAULT_ORGANIZATION))
            pub_nsq_msg(TOPIC_DEPT_CREATED,{"id":NCT_DEFAULT_ORGANIZATION,"name":_("IDS_DEFAULT_ORGANIZATION")})
            self.__replace_config(cursor, 't_sharemgnt_config', 'defaule_ou_init', 'true')
            self.conn.commit()
            cursor.close()
        else:
            self.__replace_config(cursor, 't_sharemgnt_config', 'defaule_ou_init', 'true')
            self.conn.commit()
            cursor.close()

    def __init_admin(self):
        """
        检查管理员账号
        """
        # 检查管理员账号是否初始化
        admin_list = [NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT]
        query_sql = """
        SELECT * from `t_user`
        WHERE `f_user_id` = %s
        """

        I_USER = """
        INSERT INTO t_user (f_user_id, f_login_name, f_display_name,
        f_password, f_pwd_timestamp, f_pwd_error_latest_timestamp,
        f_mail_address, f_auth_type, f_oss_id, f_sha2_password)
        VALUES ( '{0}', '{1}', '{2}', '{3}', now(), now(), '', {4} ,'{5}', '{6}')
        """

        # 初始化admin
        cursor = self.conn.cursor()
        try:
            for admin in admin_list:
                cursor.execute(query_sql, (admin,))
                result = cursor.fetchall()
                if not result:
                    if admin == NCT_USER_ADMIN:
                        cursor.execute(
                            I_USER.format(
                                NCT_USER_ADMIN,
                                global_info.ADMIN_NAME, global_info.ADMIN_NAME,
                                '',
                                ncTUsrmUserType.NCT_USER_TYPE_LOCAL,
                                "",
                                '07802cf5b92eee786df1b8691318e6b3b8b5860dcf07866ada4014f5a6a5cc55'
                            )
                        )
                    if admin == NCT_USER_AUDIT:
                        cursor.execute(
                            I_USER.format(
                                NCT_USER_AUDIT,
                                global_info.AUDIT_NAME, global_info.AUDIT_NAME,
                                '',
                                ncTUsrmUserType.NCT_USER_TYPE_LOCAL,
                                "",
                                '07802cf5b92eee786df1b8691318e6b3b8b5860dcf07866ada4014f5a6a5cc55'
                            )
                        )
                    if admin == NCT_USER_SYSTEM:
                        cursor.execute(
                            I_USER.format(
                                NCT_USER_SYSTEM,
                                global_info.SYSTEM_NAME, global_info.SYSTEM_NAME,
                                '',
                                ncTUsrmUserType.NCT_USER_TYPE_LOCAL,
                                "",
                                '07802cf5b92eee786df1b8691318e6b3b8b5860dcf07866ada4014f5a6a5cc55'
                            )
                        )
                    if admin == NCT_USER_SECURIT:
                        cursor.execute(
                            I_USER.format(
                                NCT_USER_SECURIT,
                                global_info.SECURIT_NAME, global_info.SECURIT_NAME,
                                '',
                                ncTUsrmUserType.NCT_USER_TYPE_LOCAL,
                                "",
                                '07802cf5b92eee786df1b8691318e6b3b8b5860dcf07866ada4014f5a6a5cc55'
                            )
                        )
        finally:
            self.conn.commit()
        cursor.close()

    def __init_perm_share_strategy(self):
        """
        初始化权限共享范围信息
        """
        query_sql = """
        SELECT * FROM `t_perm_share_strategy`
        WHERE `f_strategy_id` = %s
        """

        cursor = self.conn.cursor()

        # 判断是否初始化配置所有用户直属部门共享设置
        cursor.execute(query_sql, ('-1',))
        result = cursor.fetchall()
        if not result:
            insert_sql = """
            INSERT INTO `t_perm_share_strategy`
            (`f_strategy_id`, `f_obj_id`, `f_obj_type`,
             `f_sharer_or_scope`, `f_status`)
            VALUES(%s, %s, %s, %s, %s)
            """
            cursor.execute(insert_sql, ('-1', NCT_ALL_USER_GROUP, 1, 1, 1))
            cursor.execute(insert_sql, ('-1', NCT_DIRECT_DEPARTMENT, 2, 2, 1))

        # 判断是否初始化配置所有用户直属组织共享设置
        cursor.execute(query_sql, ('-2',))
        result = cursor.fetchall()
        if not result:
            insert_sql = """
            INSERT INTO `t_perm_share_strategy`
            (`f_strategy_id`, `f_obj_id`, `f_obj_type`,
             `f_sharer_or_scope`, `f_status`)
            VALUES(%s, %s, %s, %s, %s)
            """
            cursor.execute(insert_sql, ('-2', NCT_ALL_USER_GROUP, 1, 1, 1))
            cursor.execute(insert_sql, ('-2', NCT_DIRECT_ORGANIZATION, 2, 2, 1))

        self.conn.commit()
        cursor.close()

    def __init_link_template(self):
        """
        初始化内外链模板信息表
        """
        query_sql = """
        SELECT * FROM `t_link_template`
        WHERE `f_sharer_id` = %s and `f_template_type` = %s
        """

        insert_sql = """
            INSERT INTO `t_link_template`
            (`f_template_id`, `f_template_type`, `f_sharer_id`, `f_sharer_type`,
             `f_create_time`, `f_config`)
            VALUES(%s, %s, %s, %s, %s, %s)
        """

        cursor = self.conn.cursor()

        # 判断是否初始化配置内链共享模板默认设置
        cursor.execute(query_sql, (NCT_ALL_USER_GROUP,
                       ncTTemplateType.INTERNAL_LINK))
        result = cursor.fetchall()
        if not result:
            template_id = str(uuid.uuid1())

            secret_mode = False

            try:
                select_config_sql = """
                SELECT `f_value` FROM `t_sharemgnt_config`
                WHERE `f_key` = %s
                """
                cursor.execute(select_config_sql, ("enable_secret_mode",))
                secret_mode = bool(int(cursor.fetchone()[0]))
            except Exception:
                secret_mode = False

            if secret_mode:
                config = '{"allowPerm": 31, "defaultPerm": 7, "allowOwner": false, "defaultOwner": false, "limitExpireDays": false, "allowExpireDays": 30}'
            else:
                config = '{"allowPerm": 63, "defaultPerm": 7, "allowOwner": true, "defaultOwner": false, "limitExpireDays": false, "allowExpireDays": -1}'

            cursor.execute(insert_sql, (template_id, ncTTemplateType.INTERNAL_LINK,
                           NCT_ALL_USER_GROUP, 2, 0, config))

        # 判断是否初始化配置外链共享模板默认设置
        cursor.execute(query_sql, (NCT_ALL_USER_GROUP,
                       ncTTemplateType.EXTERNAL_LINK))
        result = cursor.fetchall()
        if not result:
            template_id = str(uuid.uuid1())

            config = '{"limitExpireDays": false, "allowExpireDays": -1, "allowPerm": 31, "defaultPerm": 7, "limitAccessTimes": false, "allowAccessTimes": 10, "accessPassword": false}'

            cursor.execute(insert_sql, (template_id, ncTTemplateType.EXTERNAL_LINK,
                           NCT_ALL_USER_GROUP, 2, 0, config))
        self.conn.commit()
        cursor.close()

    def __init_sharemgnt_cofing(self):
        """
        检查sharemgnt配置信息
        """
        config_dict = {
            'enable_des_password': 0,
            'sync_interval': 1440,
            'windows_ad_sso': 0,
            'strong_pwd_status': 0,
            'pwd_expire_time': -1,
            'enable_pwd_lock': 0,
            'pwd_lock_time': 60,
            'default_space_size': 5368709120,
            'enable_user_doc': 0,
            'perm_share_status': 1,
            'link_share_status': 0,
            'find_share_status': 0,
            'leak_proof_status': 0,
            'clear_cache_interval': -1,
            'clear_cache_size': -1,
            'login_strategy_status': 0,
            'force_clear_client_cache': 0,
            'client_detect_interval': 30,
            'hide_client_cache_setting': 0,
            'multi_tenant': 0,
            'client_https': 1,
            'console_https': 1,
            'system_init_status': 0,
            'pwd_err_cnt': 5,
            'csf_level_enum': '',
            'csf_level2_enum': '',
            'doc_auto_clean_status': 0,
            'global_recycle_retention_config': '{"isEnable": false, "days": 30}',
            'enable_uninstall_pwd': 0,
            'uninstall_pwd': '123456',
            'invitation_share_status': 0,
            'limit_rate_config': '{"isEnabled": false, "limitType": 0}',
            'third_csfsys_config': '{"isEnabled":false,"id":"b937b8e3-169c-4bee-85c5-865b03d8c29a","only_upload_classified":false,"only_share_classified":false,"auto_match_doc_classfication":false}',
            'enable_net_docs_limit': 0,
            'enable_ddl_email_notify': 1,
            'enable_third_pwd_lock': 0,
            'enable_user_doc_inner_link': 1,
            'enable_user_doc_out_link': 1,
            'default_strategy_superim_status': 1,
            'auto_disable_config': '{"isEnabled":0, "days":90}',
            'switch_network_auto_logout': 0,
            'third_pwd_modify_url': '',
            'enable_secret_mode': 0,
            'enable_generate_ca_cert': 1,
            'hide_user_info': 0,
            'hide_ou_info': 0,
            'cmp_host': '',
            'cmp_port': '5672',
            'cmp_user': 'abcloud',
            'cmp_password': 'P@sswd4Eis00',
            'cmp_report_interval': 86400,
            'cmp_tenant_name': '',
            'cmp_tenant_pwd': '',
            'cmp_heartbeat_interval': 1800,
            'cmp_aes_key': '',
            'retain_out_link_status': 0,
            'enable_freeze': 0,
            'vcode_login_config': '{"isEnable":false, "passwdErrCnt":0}',
            'export_log_with_pwd': 0,
            'enable_real_name_auth': 0,
            'search_user_config': '{"exactSearch":false, "searchRange":3, "searchResults":2}',
            'show_priority_access_tab': 0,
            'priority_access_config': '{"isEnable":false, "limitCPU":90, "limitMemory":90, "limitPriority":999}',
            'reached_server_threshold': 0,
            'thread_wait_time': 30,
            'sms_activate': 0,
            'sms_config': '{"server_id": "", "server_name": "", "app_id": "", \
                     "secret_id":"", "secret_key":"", "international":0, \
                     "template_id": "", "expire_time": "30"}',
            'enable_active_report_notify': 0,
            'eisoo_recipient_config': '["anyshare_ope@eisoo.com"]',
            'enable_tri_system_status': 0,
            'enable_exit_pwd': 0,
            'exit_pwd': '123456',
            'id_card_login_status': 0,
            'doc_auto_archive_status': 0,
            'strong_pwd_length': 8,
            'enable_recycle_delay_delete': 1,
            'recycle_delete_delay_time': 30,
            'enable_antivirus': 0,
            'only_share_to_user': 0,
            'file_crawl_status': 0,
            'file_crawl_show_status': 0,
            'antivirus_config': '{}',
            'enable_pwd_control': 1,
            'enable_set_delete_perm': 1,
            'enable_set_folder_security_level': 1,
            'vcode_server_status':'{"send_vcode_by_sms" : false, "send_vcode_by_email" : false}',
            'enable_outlink_watermark': 0,
            'dualfactor_auth_server_status': '{"auth_by_sms" : false, "auth_by_email" : false, "auth_by_OTP":false, "auth_by_Ukey":false}',
            'enable_update_virus_db': 0,
            'recycle_delete_delay_time_unit': 'month',
            'get_available_domains_interval': 3600,
            'catelogue_template_count': 10,
            'catelogue_count': 5,
            'enable_get_subobj_csf_level': 1,
            'tag_max_num':30,
            "anyrobot_config": '{"uri": "/app/kibana#/appLogin", "appId": "AnyShare", "appSecret": "XJ2S93CF"}',
        }

        query_sql = """
        SELECT f_value from `t_sharemgnt_config`
        WHERE `f_key` = %s
        """
        insert_sql = """
        INSERT INTO `t_sharemgnt_config` (`f_key`, `f_value`)
        VALUES( %s, %s);
        """

        cursor = self.conn.cursor()
        for key, value in list(config_dict.items()):
            cursor.execute(query_sql, (key,))
            result = cursor.fetchall()
            if not result:
                cursor.execute(insert_sql, (key, value))

        self.conn.commit()
        cursor.close()

    def __init_oem(self):
        """
        初始化OEM信息
        """
        config_files = [
            os.path.realpath(os.path.join(os.path.dirname(sys.argv[0]), "../conf/anyshare_en-us.json")),
            os.path.realpath(os.path.join(os.path.dirname(sys.argv[0]), "../conf/anyshare_zh-cn.json")),
            os.path.realpath(os.path.join(os.path.dirname(sys.argv[0]), "../conf/anyshare_zh-tw.json")),
            os.path.realpath(os.path.join(os.path.dirname(sys.argv[0]), "../conf/anyshare.json"))
        ]

        cursor = self.conn.cursor()
        try:
            # 读取oem信息，并写入到数据库
            need_init_edms_conf = False # 7.0暂不支持涉密包
            for config_file in config_files:
                with io.open(config_file, 'r', encoding='utf8') as json_file:
                    json_obj = json.load(json_file)
                    for obj in json_obj["options"]:
                        try:
                            str_sql = """
                            select f_value from t_oem_config
                            where f_section = %s and f_option = %s
                            """
                            cursor.execute(
                                str_sql, (obj["section"], obj["option"]))
                            result = cursor.fetchall()
                            if result:
                                continue

                            # 初始化涉密包配置
                            if need_init_edms_conf:
                                self.__init_edms_config()
                                need_init_edms_conf = False

                            str_sql = """
                            insert into t_oem_config(f_section, f_option, f_value)
                            values(%s, %s, %s)
                            """
                            cursor.execute(str_sql, (obj["section"], obj["option"], obj["value"]))
                        except Exception as ex:
                            logger.log_debug("ShareMgnt", "数据库检查--OEM配置异常：%s" % str(ex))
        except Exception as ex:
            logger.log_debug("ShareMgnt", "数据库OEM配置初始化异常：%s" % str(ex))

        self.conn.commit()
        cursor.close()

    def __init_watermark_config(self):
        """
        初始化t_watermark_config
        """
        query_sql = """
        SELECT * FROM t_watermark_config
        """
        cursor = self.conn.cursor()
        cursor.execute(query_sql)
        result = cursor.fetchall()
        if not result:
            initConfig = """
            {
                "text": {
                    "layout": 1,
                    "color": "#999999",
                    "enabled": false,
                    "content": "",
                    "fontSize": 36,
                    "transparency": 30
                },
                "image": {
                    "src": "",
                    "scale": 100,
                    "enabled": false,
                    "transparency": 30,
                    "layout": 1
                },
                "user": {
                    "color": "#999999",
                    "fontSize": 18,
                    "enabled": false,
                    "transparency": 30,
                    "layout": 1
                },
                "date": {
                    "layout": 1,
                    "color": "#999999",
                    "enabled": false,
                    "fontSize": 18,
                    "transparency": 30
                }
            }
            """
            insert_sql = """
            INSERT INTO `t_watermark_config` (f_for_user_doc, f_for_custom_doc, f_for_archive_doc, f_config)
            VALUES (0, 0, 0, %s);
            """
            cursor.execute(insert_sql, (json.dumps(json.loads(initConfig)),))

        self.conn.commit()
        cursor.close()

    def __init_role_config(self):
        """
        初始化系统角色配置
        """
        cursor = self.conn.cursor()
        try:
            # 权重值越小，优先级越高，内置管理员的优先级为
            # 超级管理员(权重值为1) > 系统管理员(2) > 安全管理员(3) > 审计管理员(4) >
            # 组织管理员(5) > 组织审计员(6) > 共享审核员(7) > 文档审核员(8) > 定密审核员(9)

            query_sql = """
            SELECT f_value
            FROM t_sharemgnt_config
            WHERE f_key = 'enable_tri_system_status'
            """
            cursor.execute(query_sql)
            result = cursor.fetchone()
            if int(result[0]) == 0:
                self.__replace_role(cursor, NCT_SYSTEM_ROLE_SUPPER, 1, "supper")
                self.__add_role_relation(cursor, NCT_USER_ADMIN, NCT_SYSTEM_ROLE_SUPPER)
            else:
                self.__replace_role(cursor, NCT_SYSTEM_ROLE_ADMIN, 2, "admin")
                self.__add_role_relation(cursor, NCT_USER_ADMIN, NCT_SYSTEM_ROLE_ADMIN)
                self.__replace_role(cursor, NCT_SYSTEM_ROLE_SECURIT, 3, "security")
                self.__add_role_relation(cursor, NCT_USER_SECURIT, NCT_SYSTEM_ROLE_SECURIT)
                self.__replace_role(cursor, NCT_SYSTEM_ROLE_AUDIT, 4, "audit")
                self.__add_role_relation(cursor, NCT_USER_AUDIT, NCT_SYSTEM_ROLE_AUDIT)
            self.__replace_role(cursor, NCT_SYSTEM_ROLE_ORG_MANAGER, 5, "organization manager")
            self.__replace_role(cursor, NCT_SYSTEM_ROLE_ORG_AUDIT, 6, "organization audit")

        finally:
            self.conn.commit()
        cursor.close()

    def __init_auto_clean_config(self):
        """
        初始化自动清理策略
        """
        query_sql = """
        SELECT * FROM t_doc_auto_clean_strategy
        WHERE f_obj_id = '-2'
        """
        cursor = self.conn.cursor()
        cursor.execute(query_sql)
        result = cursor.fetchall()
        if not result:
            strategyId = str(uuid.uuid1())
            date = int(BusinessDate.time() * 1000000)
            insert_sql = """
            INSERT INTO `t_doc_auto_clean_strategy`
            (f_strategy_id, f_obj_id, f_obj_type, f_enable_remain_hours, f_remain_hours, f_clean_cycle_days, f_clean_cycle_modify_time, f_create_time, f_status)
            VALUES (%s, '-2', 1, 0, 24, 30, %s, %s, %s);
            """
            cursor.execute(insert_sql, (strategyId, date, sys.maxsize, 0))

        self.conn.commit()
        cursor.close()

    def __init_local_sync_config(self):
        """
        初始化本地同步策略
        """
        query_sql = """
        SELECT * FROM t_local_sync_strategy
        WHERE f_obj_id = '-2'
        """
        cursor = self.conn.cursor()
        cursor.execute(query_sql)
        result = cursor.fetchall()
        if not result:
            strategyId = str(uuid.uuid1())
            insert_sql = """
            INSERT INTO `t_local_sync_strategy`
            (f_strategy_id, f_obj_id, f_obj_type, f_open_status, f_delete_status, f_create_time)
            VALUES (%s, '-2', 1, 1, 1, %s);
            """
            cursor.execute(insert_sql, (strategyId, sys.maxsize))

        self.conn.commit()
        cursor.close()

    def __init_edms_config(self):
        """
        初始化涉密包配置
        """

        cursor = self.conn.cursor()

        # 默认开启修改密码签名认证
        self.__replace_config(
            cursor,
            f'`{get_db_name("anyshare")}`.`t_conf`',
            "enable_eacp_check_sign",
            "true",
        )

        # 只允许共享给用户
        self.__replace_config(
            cursor,
            f'`{get_db_name("sharemgnt_db")}`.`t_sharemgnt_config`',
            "only_share_to_user",
            "1",
        )

        # 开启上传定密开关
        cursor.execute("select f_value from t_sharemgnt_config where f_key='third_csfsys_config'")
        res = cursor.fetchone()
        if res and res[0]:
            j_value = json.loads(res[0])
            j_value["isEnabled"] = True
            j_value["only_upload_classified"] = True
            self.__replace_config(
                cursor,
                f'`{get_db_name("sharemgnt_db")}`.`t_sharemgnt_config`',
                "third_csfsys_config",
                json.dumps(j_value, ensure_ascii=False),
            )

        self.conn.commit()
        cursor.close()

    def __replace_config(self, cursor, table_name, key, value):
        """
        替换配置信息
        """

        check_config_sql = """
        select f_value from {0} where f_key = %s
        """.format(table_name)

        insert_config_sql = """
        insert into {0} (f_key, f_value) values (%s, %s)
        """.format(table_name)

        update_config_sql = """
        update {0} set f_value = %s where f_key = %s
        """.format(table_name)

        cursor.execute(check_config_sql, (key,))
        result = cursor.fetchall()

        if result:
            cursor.execute(update_config_sql, (value, key))
        else:
            cursor.execute(insert_config_sql, (key, value))

    def __replace_role(self, cursor, role_id, priority, description):
        """
        替换角色信息
        """

        check_role_sql = """
        select f_priority from t_role where f_role_id = %s
        """

        insert_role_sql = """
        insert into t_role (f_role_id, f_priority, f_description) values (%s, %s, %s);
        """

        update_role_sql = """
        update t_role set f_priority = %s, f_description = %s where f_role_id = %s
        """

        cursor.execute(check_role_sql, (role_id,))
        result = cursor.fetchall()

        if result:
            cursor.execute(update_role_sql, (priority, description, role_id))
        else:
            cursor.execute(insert_role_sql, (role_id, priority, description))

    def __add_role_relation(self, cursor, user_id, role_id):
        """
        替换角色信息
        """

        check_role_relation_sql = """
        select f_role_id from t_user_role_relation where f_user_id = %s and f_role_id = %s
        """

        insert_role_relation_sql = """
        insert into t_user_role_relation (f_user_id, f_role_id) values (%s, %s);
        """
        cursor.execute(check_role_relation_sql, (user_id, role_id))
        result = cursor.fetchall()

        if not result:
            cursor.execute(insert_role_relation_sql, (user_id, role_id))
