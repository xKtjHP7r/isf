#!/usr/bin/python3
# -*- coding:utf-8 -*-
"""This is spcae manage class"""
import re
import json
import ldap
import threading
import time
import copy
from src.common import global_info
from src.common.db.connector import DBConnector, ConnectorManager
from src.modules.department_manage import DepartmentManage
from src.modules.user_manage import UserManage
from src.modules.ldap_manage import (LdapManage, name2dn, dn2name)
from src.modules.config_manage import ConfigManage
from src.common.encrypt.simple import des_decrypt_with_padzero
from src.common.lib import (raise_exception,
                            check_email,
                            check_name,)
from src.modules.ossgateway import get_oss_info
from ShareMgnt.ttypes import *
from EThriftException.ttypes import ncTException
from ShareMgnt.ttypes import (ncTShareMgntError,
                              ncTUsrmDomainSyncMode,
                              ncTSyncType)
from ShareMgnt.constants import (NCT_UNDISTRIBUTE_USER_GROUP,
                                 NCT_ALL_USER_GROUP)
from src.common import global_info
from src.common.sharemgnt_logger import ShareMgnt_Log
from src.common.http import pub_nsq_msg
from src.common.business_date import BusinessDate

UPN_KEY = 'userPrincipalName'
AVAILABLE_DOMAIN_POOL = {}
TOPIC_ORG_NAME_MODIFY = "core.org.name.modify"
TOPIC_USER_MODIFIED = "user_management.user.modified"

class DomainManage(DBConnector):
    """
    domain manage
    """
    def __init__(self):
        super(DomainManage, self).__init__()
        self.depart_manage = DepartmentManage()
        self.user_manage = UserManage()
        self.config_manage = ConfigManage()
        self.domain_import_mutex = threading.Lock()
        self.available_domain_pool_mutex = threading.Lock()

    def convert_ldap_users(self, ldap_user_infos):
        """
        Ldap用户数据DomainUserInfo转换为ncTUsrmDomainUser
        """
        domain_users = []
        for user_info in ldap_user_infos:
            domain_user = ncTUsrmDomainUser()
            domain_user.status = user_info.status
            domain_user.loginName = user_info.login_name
            domain_user.displayName = self.replace_invalid_characters(user_info.display_name)
            domain_user.email = user_info.email
            domain_user.idcardNumber = user_info.idcard_number
            domain_user.telNumber = user_info.tel_number
            domain_user.dnPath = user_info.dn
            domain_user.ouPath = user_info.ou_dn
            if isinstance(user_info.third_id, str):
                domain_user.objectGUID = user_info.third_id
            else:
                domain_user.objectGUID = bytes.decode(user_info.third_id)
            domain_user.server_type = user_info.server_type
            domain_users.append(domain_user)
        return domain_users

    def replace_invalid_characters(self, display_name):
        """
        替换域用户显示名中的非法字符为'_'
        """
        if display_name is None:
            raise_exception(exp_msg=_("IDS_INVALID_DISPLAY_NAME"),
                            exp_num=ncTShareMgntError.NCT_INVALID_DISPLAY_NAME)

        # 除去前面的空格，末尾的空格和点
        striped_name = display_name.lstrip()
        striped_name = striped_name.rstrip(". ")

        # 正则匹配，替换 '\''/'字符为'_'，且长度为[1, 128]
        return re.sub(r'[\\\/]{1,128}', '_', striped_name)

    def convert_ldap_ous(self, ldap_ou_infos):
        """
        Ldap组织数据DomainOuInfo转换为ncTUsrmDomainOU
        """
        domain_ous = []
        for ou_info in ldap_ou_infos:
            domain_ou = ncTUsrmDomainOU()
            domain_ou.name = self.replace_invalid_characters(ou_info.ou_name)
            if isinstance(ou_info.third_id, str):
                domain_ou.objectGUID = ou_info.third_id
            else:
                domain_ou.objectGUID = bytes.decode(ou_info.third_id)
            domain_ou.pathName = self.format_dn(ou_info.dn)
            domain_ou.parentOUPath = ou_info.dn[ou_info.dn.index(',') + 1:]
            domain_ous.append(domain_ou)
        return domain_ous

    def format_dn(self, dn):
        results = dn.split(",")
        retstr = ""
        for i in range(len(results)):
            retstr += results[i].strip()
            if i != (len(results) - 1):
                retstr += ","
        return retstr

    def check_ous(self, depart_result, option):
        """
        检查部门是否已经导入过
        如果没有导入过或者已经导入过，并且导入目的一致，则通过验证
        如果导入目的不一致，则异常
        """
        self.depart_manage.check_depart_exists(option.departmentId, True)
        # 只导入域用户，部门会是空的
        if not depart_result:
            return

        if option.departmentId == NCT_UNDISTRIBUTE_USER_GROUP:
            raise_exception(exp_msg=_("could not import to undistribute"),
                            exp_num=ncTShareMgntError.
                            NCT_CANNOT_IMPORT_DOMAIN_USER_TO_UNDISTRIBUTE)

        if option.departmentId == NCT_ALL_USER_GROUP:
            raise_exception(exp_msg=_("could not import to all group"),
                            exp_num=ncTShareMgntError.
                            NCT_CANNOT_IMPORT_DOMAIN_USER_TO_ALL)
        #
        # 获取勾选的组织、部门中的第一级部门进行验证
        #

        # 按照路径长度排序
        def sort_depart(valuea):
            """
            根据路径长度升序排序
            """
            count1 = valuea.pathName.count(',') + 1
            return count1

        depart_result.sort(key=sort_depart)
        min_ou_len = depart_result[0].pathName.count(',') + 1

        # 取路径长度最小值
        for depart in depart_result:
            # 只判断路径长度值等于最小值的部门进行验证
            if depart.pathName.count(',') + 1 != min_ou_len:
                continue

            sql = """
            SELECT `f_path`
            FROM `t_department`
            WHERE `f_third_party_id` = %s
            LIMIT 1
            """
            result = self.r_db.one(sql, depart.objectGUID)
            if result:
                result = result['f_path'].split('/')
                if len(result) < 2:
                    result = []
                else:
                    parent_id = result[-2]

            # 之前导入到了别的部门
            if result and parent_id != option.departmentId:
                raise_exception(exp_msg=_("could not import again"),
                                exp_num=ncTShareMgntError.
                                NCT_CANNOT_IMPORT_AGAIN)

    def add_domain_ou(self, ou_info, parent_id, oss_id):
        """
        添加域组织单元
        """
        # 设置默认部门排序
        ou_info.priority = global_info.DEFAULT_DEPART_PRIORITY
        # 检查父部门下的子部门是否存在同名部门
        parent_path = self.depart_manage.get_department_path_by_dep_id(
            parent_id)
        if parent_path == '':
            ShareMgnt_Log('departmentinfo illegal depart_path is None, depart_id:%s,%s,%s,%s',
                          parent_id, ou_info.pathName, ou_info.objectGUID, ou_info.name)
            raise_exception(exp_msg=_("departmentinfo illegal"),
                            exp_num=ncTShareMgntError.
                            NCT_ORG_OR_DEPART_NOT_EXIST)

        depart_str = "/____________________________________"
        sql = """
        SELECT `f_department_id`
        FROM `t_department`
        WHERE `f_path` like %s
            AND `f_name` = %s
        """
        result = self.r_db.one(sql, parent_path + depart_str, ou_info.name)

        # 存在同名部门，覆盖
        if result:
            sql = """
            UPDATE `t_department`
            SET `f_auth_type` = %s, `f_third_party_id` = %s,
            `f_domain_path` = %s, `f_priority` = %s
            WHERE `f_department_id` = %s
            """
            self.w_db.query(sql, ncTUsrmDepartType.NCT_DEPART_TYPE_DOMAIN,
                            ou_info.objectGUID, ou_info.pathName, ou_info.priority, result['f_department_id'])

            ShareMgnt_Log('edit ou success(name exists), %s,%s,%s',
                          ou_info.pathName, ou_info.objectGUID, ou_info.name)
        else:
            sql = """
            SELECT `f_department_id`, `f_name` FROM `t_department`
            WHERE `f_third_party_id` = %s
            """
            # 已导入过部门，则更新
            db_object = self.r_db.one(sql, ou_info.objectGUID)

            if db_object:
                sql = """
                UPDATE `t_department`
                SET `f_name` = %s, `f_domain_path` = %s, `f_priority` = %s
                WHERE `f_third_party_id` = %s
                """
                self.w_db.query(sql, ou_info.name, ou_info.pathName, ou_info.priority, ou_info.objectGUID)

                if db_object['f_name'] != ou_info.name:
                    # 发送部门显示名更新nsq消息  只有一个对象
                    pub_nsq_msg(TOPIC_ORG_NAME_MODIFY, {"id": db_object['f_department_id'], "new_name": ou_info.name, "type": "department"})

                ShareMgnt_Log('edit ou success(guid exists), %s,%s,%s',
                              ou_info.pathName, ou_info.objectGUID, ou_info.name)
            else:
                self.depart_manage.add_depart_to_db('', oss_id, parent_id, ou_info)
                ShareMgnt_Log('add ou success, %s,%s,%s',
                              ou_info.pathName, ou_info.objectGUID, ou_info.name)

    def add_domain_ous(self, domain, depart_result, option):
        """
        添加域组织
        """
        domain_name = domain.name
        base_dn = name2dn(domain_name)
        for depart in depart_result:
            depart_id = NCT_UNDISTRIBUTE_USER_GROUP
            # 如果是第一级部门
            if depart.parentOUPath == base_dn:
                depart_id = option.departmentId
            else:
                sql = """
                SELECT `f_department_id` FROM `t_department`
                WHERE `f_domain_path` = %s
                """
                db_depart = self.r_db.one(sql, depart.parentOUPath)
                if not db_depart:
                    depart_id = option.departmentId
                else:
                    depart_id = db_depart['f_department_id']

            # 检查部门名是否合法
            if not check_name(depart.name):
                msg = _("IDS_IMPORT_DEAPRT_ERROR") % (depart.name, _("IDS_INVALID_DEPART_NAME"))

                global_info.IMPORT_FAIL_NUM += 1
                global_info.IMPORT_FAIL_INFO.append(msg)
                ShareMgnt_Log(msg)

                continue

            try:
                self.add_domain_ou(depart, depart_id, option.oss_id)
            except ncTException as ex:
                msg = "部门导入出现异常。%s"
                msg = msg % "异常内容 %s 。%%s" % ex.expMsg
                msg = msg % "部门名：%s" % depart.name

                global_info.IMPORT_FAIL_NUM += 1
                global_info.IMPORT_FAIL_INFO.append(msg)

                ShareMgnt_Log(msg)
                return

            global_info.IMPORT_SUCCESS_NUM += 1

    def update_relation(self, user_id, depart_id):
        """
        更新已存在域用户的用户-部门关系
        Args:
            user_id: string 要更新的用户
            depart_id: string 要将用户分配到的部门
        """
        self.depart_manage.add_user_to_department([user_id], depart_id)

    def domain_name_exists(self, name, domain_id=None):
        """
        检测域名是否存在
        Args:
            name: string 域名
            domain_id: int32 域控id，用于编辑时验证
        Return:
            bool 检测结果
        """
        where = ''
        if domain_id:
            where = "AND `f_domain_id` <> {0}".format(domain_id)

        sql = """
        SELECT COUNT(*) AS cnt FROM `t_domain`
        WHERE `f_domain_name` = %s {0}
        """.format(where)
        count = self.r_db.one(sql, name)['cnt']
        return count != 0

    def domain_exists(self, domain_id):
        """
        检测域控是否存在
        Args:
            domain_id: i32 数据库中的域控id
        Return:
            bool 检测结果
        """
        sql = """
        SELECT COUNT(*) AS cnt FROM `t_domain`
        WHERE `f_domain_id` = %s
        """
        count = self.r_db.one(sql, domain_id)['cnt']
        return count != 0

    def check_amdin(self, admin_name, domain_name):
        """
        检查域管理员的域控信息是否正确：
        Args:
            admin_name: 域管理员
            domain_name:域名
        Return:
            bool 检测结果
        """
        if admin_name.find("dc") != -1:
            admin_name = admin_name.replace(' ', '')
            s_index = admin_name.find('dc')
            if s_index != -1:
                if admin_name[s_index:].replace('dc=', '').replace(',', '.') == domain_name:
                    return True
        else:
            if admin_name[-len(domain_name):] == domain_name:
                return True

        return False

    def fecth_domain(self, db_domain):
        """
        """
        domain_info = ncTUsrmDomainInfo()
        domain_info.id = db_domain['f_domain_id']
        domain_info.name = db_domain['f_domain_name']
        domain_info.ipAddress = db_domain['f_ip_address']
        domain_info.port = db_domain['f_port']
        domain_info.adminName = db_domain['f_administrator']
        domain_info.password = db_domain['f_password']
        domain_info.parentId = db_domain['f_parent_domain_id']
        domain_info.type = db_domain['f_domain_type']
        domain_info.status = db_domain['f_status']
        domain_info.syncStatus = db_domain['f_sync']
        domain_info.useSSL = db_domain['f_use_ssl']

        return domain_info

    def add_domain(self, domain):
        """
        添加域控
        """
        # 是否为第一级域
        is_root = (domain.parentId == -1)
        # 是否为主域
        is_primary = (domain.type == ncTUsrmDomainType.NCT_DOMAIN_TYPE_PRIMARY)
        # 是一级域，不是主域
        # 不是一级域，是主域
        # 认为参数错误
        if is_root ^ is_primary:
            raise_exception(exp_msg=_("domain type not match parent id"),
                            exp_num=ncTShareMgntError.
                            NCT_DOMAIN_TYPE_NOT_MATCH)

        # 检查admin的域名是否正确
        # if self.check_amdin(domain.adminName, domain.name) is False:
        #     raise_exception(exp_msg=_("admin parameter fail"),
        #                     exp_num=ncTShareMgntError.
        #                     NCT_INVALID_DOMAIN_PARAMETER)

        if is_primary == False:
            parent_domain = self.get_domain_by_id(domain.parentId)
            # 主域必须存在
            if not parent_domain:
                raise_exception(exp_msg=_("IDS_PARENT_DOMAIN_NOT_EXIST"),
                                exp_num=ncTShareMgntError.NCT_PARENT_DOMAIN_NOT_EXIST)
        # 不能重复添加
        if self.domain_name_exists(domain.name):
            raise_exception(exp_msg=_("domain {0} exists").format(domain.name),
                            exp_num=ncTShareMgntError.
                            NCT_DOMAIN_HAS_EXIST)

        # 验证是否能登录到域
        ldap_manage = self.__get_ldap_manage(domain)

        # 验证域控名称是否正确
        try:
            base_dn = name2dn(domain.name)
            ldap_manage.check_base_ou(base_dn)
        except Exception:
            raise_exception(exp_msg=_("domain name {0} config error").format(domain.name),
                            exp_num=ncTShareMgntError.NCT_DOMAIN_NAME_CONFIF_ERROR)

        ldap_server_type = ldap_manage.domain_type

        data = {
            "f_domain_name": domain.name,
            "f_ip_address": domain.ipAddress,
            "f_port": domain.port,
            "f_administrator": domain.adminName,
            "f_password": domain.password,
            "f_parent_domain_id": domain.parentId,
            "f_domain_type": domain.type,
            "f_status": domain.status,
            "f_ldap_server_type": ldap_server_type,
            "f_sync": -1,
            "f_use_ssl": domain.useSSL if domain.useSSL is not None else False
        }
        self.w_db.insert('t_domain', data)
        sql = """
        SELECT f_domain_id FROM `t_domain`
        WHERE `f_domain_name` = %s
        """
        domain_id = self.r_db.one(sql, domain.name)['f_domain_id']

        # 增加域同步配置
        if domain.type == ncTUsrmDomainType.NCT_DOMAIN_TYPE_PRIMARY:
            config = ncTUsrmDomainConfig()
            config.destDepartId = ""
            config.ouPath = []
            config.syncInterval = 5
            config.syncMode = ncTUsrmDomainSyncMode.NCT_SYNC_UPPER_OU
            config.userEnableStatus = True
            config.validPeriod = -1
            self.set_domain_sync_config(domain_id, config)

        # 增加域搜索配置
        key_config = ncTUsrmDomainKeyConfig()
        key_config.departNameKeys = ['ou', 'cn']
        key_config.departThirdIdKeys = ['objectGUID', 'entryUUID']
        key_config.loginNameKeys = ['userPrincipalName', 'sAMAccountName', 'uid']
        key_config.displayNameKeys = ['displayName', 'name', 'cn']
        key_config.emailKeys = ['mail']
        key_config.idcardNumberKeys = ['']
        key_config.telNumberKeys = ['']
        key_config.userThirdIdKeys = ['objectGUID', 'entryUUID', 'uid']
        key_config.groupKeys = ['group']
        key_config.statusKeys = ['userAccountControl']
        key_config.subOuFilter = "(|(objectClass=posixGroup)(objectClass=organizationalUnit))"
        key_config.subUserFilter = 'objectClass=organizationalPerson'
        key_config.baseFilter = 'objectClass=*'

        self.set_domain_key_config(domain_id, key_config)

        self.init_available_domain_pool(domain_id)
        return domain_id

    def delete_domain(self, domain_id):
        """
        删除域控
        """
        # 检查域是否存在
        self.check_domain_exists(domain_id)

        # 清理可用域池
        if domain_id in AVAILABLE_DOMAIN_POOL:
            with self.available_domain_pool_mutex:
                AVAILABLE_DOMAIN_POOL.pop(domain_id)

        # 使用事务
        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()

        try:
            # 删除备用域
            checksql = """
            SELECT COUNT(*) AS cnt FROM `t_failover_domain` WHERE `f_parent_domain_id` = %s
            """
            cursor.execute(checksql, (domain_id,))
            result = cursor.fetchone()['cnt']
            if result:
                sql = """
                DELETE FROM `t_failover_domain` WHERE `f_parent_domain_id` = %s
                """
                cursor.execute(sql, (domain_id,))

            # 删除子域及当前域
            sql = """
            DELETE FROM `t_domain`
            WHERE `f_domain_id` = %s OR `f_parent_domain_id` = %s
            """
            cursor.execute(sql, (domain_id, domain_id))

            conn.commit()

            # 关闭指定域同步线程
            if global_info.DOMAIN_SYNC_THREAD.get(domain_id):
                global_info.DOMAIN_SYNC_THREAD[domain_id].close()
                del global_info.DOMAIN_SYNC_THREAD[domain_id]

        except Exception as e:
            conn.rollback()
            raise_exception(exp_msg=str(e),
                            exp_num=ncTShareMgntError.NCT_DB_OPERATE_FAILED)
        finally:
            cursor.close()
            conn.close()

    def edit_domain(self, domain):
        """
        编辑域控
        """
        # 检查域是否存在
        self.check_domain_exists(domain.id)

        if self.domain_name_exists(domain.name, domain.id):
            raise_exception(exp_msg=_("domain {0} exists").format(domain.name),
                            exp_num=ncTShareMgntError.
                            NCT_DOMAIN_HAS_EXIST)

        # 检查首选域ip是否与备用域ip重复
        self.__check_conflict_with_failover_domain(domain)

        # 验证是否能登录到域
        ldap_manage = self.__get_ldap_manage(domain)
        ldap_server_type = ldap_manage.domain_type

        try:
            base_dn = name2dn(domain.name)
            ldap_manage.check_base_ou(base_dn)
        except Exception:
            raise_exception(exp_msg=_("domain name {0} config error").format(domain.name),
                            exp_num=ncTShareMgntError.NCT_DOMAIN_NAME_CONFIF_ERROR)
        # 更新域信息
        sql = """
        UPDATE `t_domain`
        SET `f_domain_name` = %s, `f_ip_address` = %s, `f_port` = %s, `f_use_ssl` = %s,
        `f_administrator` = %s, `f_password` = %s, `f_ldap_server_type` = %s
        WHERE `f_domain_id` = %s
        """
        self.w_db.query(sql, domain.name, domain.ipAddress, domain.port, domain.useSSL,
                        domain.adminName, domain.password, ldap_server_type, domain.id)

        self.init_available_domain_pool(domain.id)

    def __check_conflict_with_failover_domain(self, domain):
        """
        检查首选域ip是否与备用域ip重复
        """
        sql = """
        SELECT * FROM t_failover_domain
        """
        result = self.r_db.all(sql)
        for row in result:
            if row.get('f_ip_address').lower() == domain.ipAddress.lower():
                raise_exception(exp_msg=_("IDS_FAILOVER_DOMAIN_ADDRESS_SAME_WITH_PARENT"),
                                exp_num=ncTShareMgntError.NCT_FAILOVER_DOMAIN_ADDRESS_SAME_WITH_PARENT)

    def get_all_domains(self):
        """
        获取所有域控
        返回数据形式
        [ncTUsrmDomainInfo, ncTUsrmDomainInfo...]
        """
        domains = []
        db_domains = self.r_db.all("SELECT * FROM t_domain")

        for db_domain in db_domains:
            domain_info = self.fecth_domain(db_domain)
            domain_info.config = self.get_domain_sync_config(domain_info.id)
            domain_info.key_config = self.get_domain_key_config(domain_info.id)
            domains.append(domain_info)

        return domains

    def get_domain_by_id(self, domain_id):
        """
        根据域id获取域信息
        """
        # 修改完第三方配置，自动同步时，传入可能是第三方appid，不是整数
        try:
            domain_id = int(domain_id)
        except ValueError:
            return

        sql = """
        SELECT * FROM `t_domain`
        WHERE `f_domain_id` = %s
        """

        db_domain = self.r_db.one(sql, domain_id)
        if db_domain:
            domain_info = self.fecth_domain(db_domain)
            domain_info.config = self.get_domain_sync_config(domain_id)
            domain_info.key_config = self.get_domain_key_config(domain_info.id)
            return domain_info
        else:
            raise_exception(exp_msg=_("domain not exists"),
                            exp_num=ncTShareMgntError.
                            NCT_DOMAIN_NOT_EXIST)

    def get_domain_by_name(self, domain_name):
        """
        根据域名获取域id
        """
        sql = """
        SELECT * FROM `t_domain`
        WHERE `f_domain_name` = %s
        """

        db_domain = self.r_db.one(sql, domain_name)
        if db_domain:
            domain_info = self.fecth_domain(db_domain)
            domain_info.config = self.get_domain_sync_config(domain_info.id)
            domain_info.key_config = self.get_domain_key_config(domain_info.id)
            return domain_info

    def set_domain_status(self, domain_id, status):
        """
        启用/禁用 域控
        """
        # 检查域是否存在
        self.check_domain_exists(domain_id)

        # 更新域信息
        sql = """
        UPDATE `t_domain`
        SET `f_status` = %s
        WHERE `f_domain_id` = %s
        """
        self.w_db.query(sql, bool(status), domain_id)

        # 首选域域配置变化时，可用域池配置也要更新
        self.init_available_domain_pool(domain_id)

    def check_depart(self, depart_id):
        """
        检测部门是否存在、是否是未分配组
        """
        if depart_id == NCT_UNDISTRIBUTE_USER_GROUP:
            raise_exception(exp_msg=_("could not import to undistribute"),
                            exp_num=ncTShareMgntError.
                            NCT_CANNOT_IMPORT_DOMAIN_USER_TO_UNDISTRIBUTE)

        if depart_id == NCT_ALL_USER_GROUP:
            raise_exception(exp_msg=_("could not import to all group"),
                            exp_num=ncTShareMgntError.
                            NCT_CANNOT_IMPORT_DOMAIN_USER_TO_ALL)

        # 判断部门是否存在
        sql = """
        SELECT COUNT(*) AS cnt FROM `t_department`
        WHERE `f_department_id` = %s
        """
        count = self.r_db.one(sql, depart_id)['cnt']

        if count == 0:
            raise_exception(exp_msg=_("depart not exists"),
                            exp_num=ncTShareMgntError.
                            NCT_DEPARTMENT_NOT_EXIST)

    def get_domain_users(self, user_list, ldap_manage, key_config):
        """
        检查并获取勾选的域用户信息
        """
        if not isinstance(user_list, list):
            return []

        ldap_users = []
        domain_users = []
        for user in user_list:
            ldap_user = ldap_manage.get_domain_user(user.dnPath, key_config)
            if ldap_user:
                ldap_users.append(ldap_user)

        domain_users = self.convert_ldap_users(ldap_users)
        return domain_users

    def get_domain_sub_users(self, ou_list, ldap_manage, key_config):
        """
        获取勾选的组织下的用户
        """
        if not isinstance(ou_list, list):
            return []

        all_users = []
        for ouopt in ou_list:
            if ouopt.importAll:
                ldap_users = ldap_manage.get_all_sub_users(ouopt.pathName, key_config)
                if ldap_users:
                    domain_users = self.convert_ldap_users(ldap_users)
                    all_users.extend(domain_users)

        return all_users

    def get_all_ous(self, ou_list, ldap_manage, key_config):
        """
        获取勾选的组织及子组织
        """
        if not isinstance(ou_list, list):
            return []

        organ_list = []
        pathNamePool = set()
        for ouopt in ou_list:
            if ouopt.importAll:
                ldap_ous = ldap_manage.get_all_sub_ous(ouopt.pathName, key_config)
                domain_ous = self.convert_ldap_ous(ldap_ous)
                if domain_ous:
                    # 对域名进行重复性检查
                    for domain_ou in domain_ous:
                        if domain_ou.pathName not in pathNamePool:
                            pathNamePool.add(domain_ou.pathName)
                            organ_list.append(domain_ou)

            # 没有这句不能导入组织结构
            if ouopt.pathName not in pathNamePool:
                pathNamePool.add(ouopt.pathName)
                organ_list.append(ouopt)

        return organ_list

    def add_user_to_db(self, domain, user_list, option, auto_select=False):
        """
        添加用户到数据库
        Args:
            base_dn: string 域控根节点
            user_list: list<ncTUsrmDomainUser> 要添加的用户列表
            option: ncTUsrmImportOption 导入选项
            auto_select: 是否自动选择导入部门
        """
        def select_depart(ou_path):
            """
            为用户选择一个部门进行导入
            """
            sql = "select  f_department_id from t_department where f_domain_path = %s limit 1"
            if isinstance(ou_path, bytes):
                new_ou_path = bytes.decode(ou_path)
            else:
                new_ou_path = ou_path
            db_department = self.r_db.one(sql, new_ou_path)
            # 所属部门被导入，则导入到这个部门
            if db_department:
                depart_id = db_department['f_department_id']
            # 否则使用导入选项中的部门
            else:
                # 当将要导入的用户是多个安全组的成员，而手动导入时只选择某个安全组下的该用户导入，则会在此函数中判断其他未导入安全组不存在
                # 进而导致该用户导入后额外隶属于根组织（而这样是不对的）
                # 所以此处增加判断，如果是安全组未导入(安全组的ou_path字段包含CN，正常的部门不会含有CN)，则返回空，不返回根组织ID
                if b"CN" in ou_path:
                    return
                depart_id = option.departmentId

            return depart_id


        domain_name = domain.name
        base_dn = name2dn(domain_name)
        for user in user_list:
            # 是否导入邮箱
            if option.userEmail is False:
                user.email = ''

            # 如果邮箱非法，保留之前合法数据
            if not user.email:
                if self.user_manage.get_olduserinfo_by_loginName(user.loginName):
                    user.email = self.user_manage.get_olduserinfo_by_loginName(user.loginName)['f_mail_address']
                else:
                    user.email = ""
            else:
                user.email = self.user_manage.is_email_valid(user.loginName,user.email)

            # 是否导入身份证
            if option.userIdcardNumber is False:
                user.idcardNumber = ''

             # 如果身份证号非法，保留之前合法数据
            if not user.idcardNumber:
                if self.user_manage.get_olduserinfo_by_loginName(user.loginName):
                    user.idcardNumber = self.user_manage.get_olduserinfo_by_loginName(user.loginName)['f_idcard_number']
                else:
                    user.idcardNumber = ""
            else:
                user.idcardNumber = self.user_manage.is_idcardNumber_valid(user.idcardNumber,user.loginName)
                if self.user_manage.check_user_exists_by_idcardNumber_loginName(user.idcardNumber, user.loginName) == False:
                    if self.user_manage.get_olduserinfo_by_loginName(user.loginName):
                        user.idcardNumber = self.user_manage.get_olduserinfo_by_loginName(user.loginName)['f_idcard_number']
                    else:
                        user.idcardNumber = ""

            # 是否导入手机号码
            if option.userTelNumber is False:
                user.telNumber = ''

            # 如果手机号码非法，保留之前合法数据
            if not user.telNumber:
                if self.user_manage.get_olduserinfo_by_loginName(user.loginName):
                    user.telNumber =  self.user_manage.get_olduserinfo_by_loginName(user.loginName)['f_tel_number']
                else:
                    user.telNumber = ""
            else:
                user.telNumber = self.user_manage.is_teleNumber_valid(user.telNumber,user.loginName)

            # 自动选择导入部门
            depart_id = []
            ldap_manage = self.__get_ldap_manage(domain)
            user_info = ldap_manage.base_ou(user.dnPath, "objectClass=organizationalPerson")
            if not user_info:
                ShareMgnt_Log("Search ou error: %s not found." % (user.dnPath))
            else:
                if "memberOf" in list(user_info[0][1].keys()):
                    ouPath_list = user_info[0][1]["memberOf"]
                    depart_id = list(set([select_depart(ouPath) for ouPath in ouPath_list if select_depart(ouPath)]))

            # 域用户所属部门路径不是根路径
            if auto_select and user.ouPath != base_dn:
                depart_id.append(select_depart(bytes(user.ouPath, encoding = "utf8")))
            # 否则使用导入选项中的部门
            else:
                depart_id.append(option.departmentId)

            # 是否导入显示名
            if option.userDisplayName is False:
                user.displayName = user.loginName

            # 显示名为空，则默认为登录名
            if not user.displayName:
                user.displayName = user.loginName

            if user.server_type == 1:
                user.dnPath = domain_name

            # 检查是否存在同名用户，
            # 存在同名，覆盖，更新数据库；跳过，什么都不处理
            # 不存在同名用户，创建权限，并且保存到数据库
            sql = """
            SELECT `f_user_id` , `f_display_name`, `f_tel_number`, `f_mail_address` FROM `t_user`
            WHERE `f_login_name` = %s
            """
            db_user = self.r_db.one(sql, user.loginName)

            if db_user:
                if option.userCover is True:
                    if user.loginName.lower() in self.admin_list:
                        msg = 'edit user failed(account in admin list), %s,%s,%s,%s' %  \
                            (user.dnPath, user.objectGUID, user.loginName, user.displayName)
                        ShareMgnt_Log(msg)
                        global_info.IMPORT_FAIL_NUM += 1
                        global_info.IMPORT_FAIL_INFO.append(msg)
                        continue

                    # 检查是否存在同显示名用户
                    user.idcardNumber = bytes.decode(des_decrypt_with_padzero(global_info.des_key,
                                                        user.idcardNumber,
                                                        global_info.des_key)[:18])
                    user.displayName = self.user_manage.get_unique_displayname(user.displayName,
                                                                               db_user['f_user_id'])
                    user.userType = ncTUsrmUserType.NCT_USER_TYPE_DOMAIN
                    ncUser = ncTUsrmUserInfo()
                    ncUser.loginName = user.loginName
                    ncUser.displayName = user.displayName
                    ncUser.email = user.email
                    ncUser.idcardNumber = user.idcardNumber
                    ncUser.telNumber = user.telNumber
                    self.user_manage.check_user(ncUser)
                    sql = """
                    UPDATE `t_user` SET `f_display_name` = %s,
                        `f_password` = '',
                        `f_mail_address` = %s,
                        `f_auth_type` = %s,
                        `f_third_party_id` = %s,
                        `f_domain_path` = %s,
                        `f_ldap_server_type` = %s,
                        `f_idcard_number` = %s,
                        `f_tel_number` = %s
                    WHERE `f_login_name` = %s
                    """
                    self.w_db.query(sql, user.displayName, user.email,
                                    user.userType,
                                    user.objectGUID,
                                    user.dnPath,
                                    user.server_type,
                                    ncUser.idcardNumber,
                                    user.telNumber,
                                    user.loginName)

                    # 更新部门关系
                    for a_depart_id in depart_id:
                        self.update_relation(db_user['f_user_id'], a_depart_id)

                    if db_user['f_display_name'] != user.displayName:
                        # 发送用户显示名更新nsq消息
                        pub_nsq_msg(TOPIC_ORG_NAME_MODIFY, {"id": db_user['f_user_id'], "new_name": user.displayName, "type": "user"})

                    user_modify_info = {}
                    if db_user['f_tel_number'] != user.telNumber:
                        user_modify_info["new_telephone"] = user.telNumber
                    if db_user['f_mail_address'] != user.email:
                        user_modify_info["new_email"] = user.email
                    if len(user_modify_info) > 0:
                        user_modify_info["user_id"] = db_user['f_user_id']
                        pub_nsq_msg(TOPIC_USER_MODIFIED, user_modify_info)

                    ShareMgnt_Log('edit user success(account exist), %s,%s,%s,%s',
                                  user.dnPath, user.objectGUID,
                                  user.loginName, user.displayName)
            else:
                # 检查是否存在相同显示名的用户
                user.displayName = self.user_manage.get_unique_displayname(user.displayName)

                sql = """
                SELECT `f_user_id`, `f_display_name`, `f_tel_number`, `f_mail_address` FROM `t_user`
                WHERE `f_third_party_id` = %s
                """
                db_object = self.r_db.one(sql, user.objectGUID)
                # 用户已存在，但是登录名变了
                if db_object:
                    sql = """
                    UPDATE `t_user` SET `f_login_name` = %s,
                        `f_display_name` = %s,
                        `f_password` = '',
                        `f_mail_address` = %s,
                        `f_auth_type` = %s,
                        `f_domain_path` = %s,
                        `f_ldap_server_type` = %s,
                        `f_idcard_number` = %s,
                        `f_tel_number` = %s
                    WHERE `f_third_party_id` = %s
                    """
                    user.idcardNumber = self.user_manage.is_idcardNumber_valid(user.idcardNumber,user.loginName)
                    user.telNumber = self.user_manage.is_teleNumber_valid(user.telNumber,user.loginName)
                    if self.user_manage.check_user_exists_by_idcardNumber_loginName(user.idcardNumber, user.loginName) == False:
                        if self.user_manage.get_olduserinfo_by_loginName(user.loginName):
                            user.idcardNumber = self.user_manage.get_olduserinfo_by_loginName(user.loginName)['f_idcard_number']
                        else:
                            user.idcardNumber = ""
                    self.w_db.query(sql, user.loginName, user.displayName,
                                    user.email,
                                    ncTUsrmUserType.NCT_USER_TYPE_DOMAIN,
                                    user.dnPath,
                                    user.server_type,
                                    user.idcardNumber,
                                    user.telNumber,
                                    user.objectGUID)
                    global_info.IMPORT_SUCCESS_NUM += 1

                    if db_object['f_display_name'] != user.displayName:
                        # 发送用户显示名更新nsq消息
                        pub_nsq_msg(TOPIC_ORG_NAME_MODIFY, {"id": db_object['f_user_id'], "new_name": user.displayName, "type": "user"})

                    user_modify_info = {}
                    if db_object['f_tel_number'] != user.telNumber:
                        user_modify_info["new_telephone"] = user.telNumber
                    if db_object['f_mail_address'] != user.email:
                        user_modify_info["new_email"] = user.email

                    if len(user_modify_info) > 0:
                        # 发送用户信息更新nsq消息
                        user_modify_info["user_id"] = db_object['f_user_id']
                        pub_nsq_msg(TOPIC_USER_MODIFIED, user_modify_info)

                    ShareMgnt_Log('edit user success(guid exist), %s,%s,%s,%s',
                                  user.dnPath, user.objectGUID,
                                  user.loginName, user.displayName)
                    continue

                # 重新添加用户
                # 转换 ncTUsrmDomainUser 为 ncTUsrmAddUserInfo
                user.idcardNumber = bytes.decode(des_decrypt_with_padzero(global_info.des_key,
                                                        user.idcardNumber,
                                                        global_info.des_key)[:18])
                add_user = ncTUsrmAddUserInfo()
                add_user.password = ''
                add_user.user = ncTUsrmUserInfo()
                add_user.user.loginName = user.loginName
                add_user.user.displayName = user.displayName
                add_user.user.email = user.email
                add_user.user.idcardNumber = user.idcardNumber
                add_user.user.telNumber = user.telNumber
                add_user.user.userType = ncTUsrmUserType.NCT_USER_TYPE_DOMAIN
                add_user.user.priority = 999
                add_user.user.csfLevel = option.csfLevel
                add_user.user.csfLevel2 = self.config_manage.get_min_csf_level2()
                add_user.user.ossInfo = get_oss_info(option.oss_id) or ncTUsrmOSSInfo()
                add_user.user.expireTime = option.expireTime

                # 先取用户单独的启用禁用状态，如没有再取默认创建状态
                if user.status is not None:
                    add_user.user.status = ncTUsrmUserStatus.NCT_STATUS_ENABLE if user.status else ncTUsrmUserStatus.NCT_STATUS_DISABLE
                else:
                    add_user.user.status = option.userStatus

                # Python支持动态添加属性
                add_user.user.objectGUID = user.objectGUID
                add_user.user.dnPath = user.dnPath
                add_user.user.server_type = user.server_type
                add_user.user.departmentIds = depart_id
                add_user.user.pwdControl = 0

                try:
                    self.user_manage.check_user(add_user.user)
                    self.user_manage.add_user_to_db(add_user)

                    ShareMgnt_Log('add user success, %s,%s,%s,%s',
                                  user.dnPath, user.objectGUID,
                                  user.loginName, user.displayName)
                except ncTException as ex:
                    msg = "add user failed: %s,%s,%s,%s, error: %s" % \
                        (user.dnPath, user.objectGUID,
                         user.loginName, user.displayName, ex.expMsg)

                    ShareMgnt_Log(msg)

                    global_info.IMPORT_FAIL_NUM += 1
                    global_info.IMPORT_FAIL_INFO.append(msg)
                    if ex.expMsg == _("user num overflow"):
                        global_info.IMPORT_IS_STOP = True
                    return

            global_info.IMPORT_SUCCESS_NUM += 1

    def search_info_by_name(self, domain_id, name, start, limit):
        """
        通过组织名或用户名获取对应组织和用户
        """
        # 检查start和limit参数是否合法
        if start < 0:
            raise_exception(exp_msg=_("IDS_START_LESS_THAN_ZERO"),
                            exp_num=ncTShareMgntError.NCT_START_LESS_THAN_ZERO)
        if limit < -1:
            raise_exception(exp_msg=_("IDS_LIMIT_LESS_THAN_MINUS_ONE"),
                            exp_num=ncTShareMgntError.NCT_LIMIT_LESS_THAN_MINUS_ONE)

        ous = []
        users = []
        domain_node = ncTUsrmDomainNode()
        domain_node.ous = []
        domain_node.users = []
        striped_name = name.strip()
        if striped_name == "":
            return domain_node
        if "(" in striped_name or ")" in striped_name:
            striped_name = striped_name.replace(
                "(", "\\28").replace(")", "\\29")
        # 当domain_id为-1时返回所有域中符合搜索名的信息
        if domain_id == -1:
            sql = """
            SELECT f_domain_id FROM  t_domain
            """
            domain_ids = self.r_db.all(sql)
            if domain_ids is None:
                return domain_node
            for domain in domain_ids:
                # 通过名字返回获取域信息
                ous,users = self.get_domainnodeinfo_by_name(striped_name, domain["f_domain_id"], ous, users, start, limit)
        else:
            ous,users = self.get_domainnodeinfo_by_name(striped_name, domain_id, ous, users, start, limit)
        ouscount = len(ous)
        userscount = len(users)
        # 查询返回根据start和limit返回信息
        if limit == 0:
            return domain_node
        elif limit == -1:
            domain_node.ous = ous
            domain_node.users = users
            return domain_node
        if start < ouscount:
            if limit < ouscount-start:
                domain_node.ous = ous[start:start+limit]
            elif limit >= ouscount-start:
                if limit-(ouscount-start) <= userscount:
                    domain_node.ous = ous[start:]
                    domain_node.users = users[:limit-(ouscount-start)]
                elif limit-(ouscount-start) > userscount:
                    domain_node.ous = ous[start:]
                    domain_node.users = users
        elif start >= ouscount and start <= ouscount+userscount:
            if start+limit >= ouscount+userscount:
                domain_node.users = users[start-ouscount:]
            elif start+limit < ouscount+userscount:
                domain_node.users = users[start-ouscount:start-ouscount+limit]
        return domain_node

    def get_domainnodeinfo_by_name(self, name, domain_id, ous, users, start, limit):
        """
        根据域id和搜索名获取域节点信息
        """
        ldap_manage, domain = self.connect_domain(domain_id)
        # ShareMgnt_Log("domain:%s",domain)
        if domain is not None:
            base_dn = name2dn(domain.name)
            ou_config = re.findall('[A-Za-z0-9=]+',domain.key_config.subOuFilter)
            ous_filter=""
            # 根据ou的subOuFilter和名字合并过滤信息
            for ou in ou_config:
                ous_filter= ous_filter + "(" + ou + ")"
            ous_filter = "(&(ou=*"+ name + "*)(|"+ ous_filter +"))"
            try:
                # 根据过滤信息获取域中信息
                results = ldap_manage.search_subtree(base_dn, ous_filter)
                # 当部门为安全组时获取过滤信息
                if 'objectClass=group' in ou_config:
                    ous_filter = "(&(cn=*"+ name + "*)(objectClass=group))"
                    groupresult = ldap_manage.search_subtree(base_dn, ous_filter)
                    results.extend(groupresult)
                # 对获取的信息进行过滤
                ou_results = self.search_results_filter(base_dn, results, ldap_manage, domain.key_config.subOuFilter, start, limit)
                ldap_ous = ldap_manage.get_search_info_ous(ou_results,domain.key_config)
                ou_list = self.convert_ldap_ous(ldap_ous)
                for ou in ou_list:
                    ous.append(ou)
                user_config = re.findall('[A-Za-z0-9=]+',domain.key_config.subUserFilter)
                users_filter=""
                # 根据user的subUserFilter和名字合并过滤信息
                for user in user_config:
                    users_filter= users_filter + "(" + user + ")"
                users_filter = "(&(|(name=*"+ name + "*)(displayName=*"+ name + "*)(cn=*"+ name + "*))(|"+ users_filter +"))"
                ShareMgnt_Log('users_filter: %s, user_info: %s',
                              users_filter, results[0][0])
                # 根据过滤信息获取域中信息
                results = ldap_manage.search_subtree(base_dn, users_filter)
                # 对获取的信息进行过滤
                results = self.search_results_filter(base_dn, results, ldap_manage, domain.key_config.subOuFilter, start, limit)
                ldap_users = ldap_manage.get_search_info_users(results,domain.key_config)
                user_list = self.convert_ldap_users(ldap_users)
                for user in user_list:
                    users.append(user)
            except ncTException as ex:
                if ex.errDetail == 'Size limit exceeded':
                    if users is None and ous is None:
                        raise_exception(exp_msg=_("data size limit exceeded"),
                                    exp_num=ncTShareMgntError.
                                    NCT_SIZE_LIMIT_EXCEEDED)
                elif ex.errDetail == 'Bad search filter':
                    return [], []
                else:
                    raise ex
            return ous,users

    def search_results_filter(self, base_dn, results, ldap_manage, subOuFilter, start, limit):
        """
        过滤搜索结果
        """
        removecount = 0
        count = 0
        remove_list=[]
        remove_base_ou = ["CN=Computers",
                          "CN=Builtin", "OU=Domain Controllers"]
        for result in results:
            count += 1
            if result[0] is not None:
                base_ous = result[0].split(",")
                base_ou = ""
                index = 0
                for i, ou in enumerate(base_ous):
                    if "(" in ou or ")" in ou:
                        ou = ou.replace(
                            "(", "\\28").replace(")", "\\29")
                        base_ous[i] = ou
                    if ou[:3] == "DC=":
                        base_ou =base_ou +','+ ou
                        index +=1
                base_ou = base_ous[len(base_ous)-index-1] + base_ou
                ous_filter = "(&(distinguishedName="+ base_ou +")" + subOuFilter + ")"
                base_ou_info = ldap_manage.search_subtree(base_dn, ous_filter)
                ShareMgnt_Log('ous_filter: %s, base_ou_info: %s',
                              ous_filter, base_ou_info[0][0])
                if base_ou_info[0][0] is None or base_ous[len(base_ous)-index-1] in remove_base_ou:
                    remove_list.append(result)
                    removecount += 1
            if limit != -1 and count - removecount == start + limit:
                break
        for result in remove_list:
            results.remove(result)
        return results

    def expand_domain_node(self, domain, node_path):
        """
        展开域控节点
        """
        # 根据domain id重新获取一次域信息
        if domain.id:
            domain = self.get_domain_by_id(domain.id)

        ldap_manage, domain = self.connect_domain(domain.id)

        key_config = self.get_domain_key_config(domain.id)

        # 获取扩展的域组织和域用户
        ldap_users = ldap_manage.get_onelevel_sub_users(node_path, key_config)
        ldap_ous = ldap_manage.get_onelevel_sub_ous(node_path, key_config)

        domain_node = ncTUsrmDomainNode()
        domain_node.users = self.convert_ldap_users(ldap_users)
        domain_node.ous = self.convert_ldap_ous(ldap_ous)

        return domain_node

    def import_domain_ous(self, content, option, responsible_person_id):
        """
        导入域用户以及组织结构
        """
        self.admin_list = list(self.user_manage.get_all_admin_account().values())
        if self.domain_import_mutex.locked():
            raise_exception(exp_msg=_("DOMAIN_IMPORTING"),
                            exp_num=ncTShareMgntError.NCT_DOMAIN_IMPORTING)
        # 导入域用户组织方法加锁
        with self.domain_import_mutex:
            ShareMgnt_Log('import_domain_ous, content: %s, option: %s',
                          str(content), str(option))

            global_info.init_import_variable()

            self.check_import_domain_option(option)

            # 根据domain id重新获取一次域信息
            if content.domain.id:
                content.domain = self.get_domain_by_id(content.domain.id)
            if content.domainName and content.domain.name != content.domainName:
                raise_exception(exp_msg=_("domain import parameter fail"),
                                exp_num=ncTShareMgntError.
                                NCT_INVALID_DOMAIN_PARAMETER)

            # 导入整个域控
            if content.domainName is not None:
                self.import_domain_all(content.domain, option, responsible_person_id)
                return

            # 获取域控信息
            ldap_manage, content.domain = self.connect_domain(content.domain.id)

            key_config = self.get_domain_key_config(content.domain.id)

            domain_ous = []
            domain_users = []

            # 获取勾选的组织及子组织信息
            domain_ous = self.get_all_ous(content.ous, ldap_manage, key_config)
            self.check_ous(domain_ous, option)

            # 获取勾选的域用户信息
            domain_users += self.get_domain_users(content.users, ldap_manage, key_config)

            # 获取勾选的组织下的用户
            domain_users += self.get_domain_sub_users(content.ous, ldap_manage, key_config)

            global_info.IMPORT_TOTAL_NUM = len(domain_users) + len(domain_ous)

            # 获取oss_id
            option.oss_id = self.depart_manage.get_oss_id_by_dept_id(option.departmentId)
            if not option.oss_id:
                option.oss_id = ""

            # 创建域组织
            self.add_domain_ous(content.domain, domain_ous, option)

            # 创建用户
            self.add_user_to_db(content.domain, domain_users, option, True)

    def import_domain_all(self, domain, option, responsible_person_id):
        """
        导入整个域控
        """
        ShareMgnt_Log('import_domain_all, domain: %s, option: %s',
                      str(domain), str(option))

        global_info.init_import_variable()
        self.admin_list = list(self.user_manage.get_all_admin_account().values())

        self.check_import_domain_option(option)

        # 获取域控信息
        ldap_manage, domain = self.connect_domain(domain.id)

        base_dn = name2dn(domain.name)
        key_config = self.get_domain_key_config(domain.id)

        # 获取所有组织
        ldap_ous = ldap_manage.get_all_sub_ous(base_dn, key_config)
        domain_ous = self.convert_ldap_ous(ldap_ous)

        # 检查部门
        self.check_ous(domain_ous, option)

        # 获取所有用户
        all_domain_users = []
        for depart in domain_ous:
            ldap_users = ldap_manage.get_onelevel_sub_users(depart.pathName, key_config)
            all_domain_users += self.convert_ldap_users(ldap_users)

        # 域控根的用户
        ldap_users = ldap_manage.get_onelevel_sub_users(base_dn, key_config)
        all_domain_users += self.convert_ldap_users(ldap_users)

        # 导入的总数
        global_info.IMPORT_TOTAL_NUM = len(domain_ous) + len(all_domain_users)

        # 使用默认
        option.oss_id = ""

        # 创建域组织
        self.add_domain_ous(domain, domain_ous, option)

        # 创建用户
        self.add_user_to_db(domain, all_domain_users, option, True)

    def import_domain_users(self, content, option, responsible_person_id):
        """
        导入域用户
        """
        self.admin_list = list(self.user_manage.get_all_admin_account().values())
        if self.domain_import_mutex.locked():
            raise_exception(exp_msg=_("DOMAIN_IMPORTING"),
                            exp_num=ncTShareMgntError.NCT_DOMAIN_IMPORTING)
        # 导入域用户组织方法加锁
        with self.domain_import_mutex:
            ShareMgnt_Log('import_domain_users, content: %s, option: %s',
                          str(content), str(option))
            global_info.init_import_variable()

            self.check_import_domain_option(option)

            # 检查目的部门id
            self.check_depart(option.departmentId)

            # 获取oss_id
            option.oss_id = self.depart_manage.get_oss_id_by_dept_id(option.departmentId)
            if not option.oss_id:
                option.oss_id = ""

            # 根据domain id重新获取一次域信息
            if content.domain.id:
                content.domain = self.get_domain_by_id(content.domain.id)
            if(content.domainName and content.domain.name != content.domainName):
                raise_exception(exp_msg=_("domain import parameter fail"),
                                exp_num=ncTShareMgntError.
                                NCT_INVALID_DOMAIN_PARAMETER)

            # 转换域控登录名
            ldap_manage = self.__get_ldap_manage(content.domain)

            base_dn = name2dn(content.domain.name)
            key_config = self.get_domain_key_config(content.domain.id)

            # 获取勾选的域控的所有用户
            all_domain_users = []
            if content.domainName:
                # 获取域根组织下的用户
                ldap_users = ldap_manage.get_onelevel_sub_users(base_dn, key_config)
                all_domain_users += self.convert_ldap_users(ldap_users)

                # 获取子组织下的用户
                ldap_users = ldap_manage.get_all_sub_users(base_dn, key_config)
                all_domain_users += self.convert_ldap_users(ldap_users)

            # 获取勾选的域用户信息
            all_domain_users += self.get_domain_users(content.users, ldap_manage, key_config)

            # 获取勾选的组织下的用户
            all_domain_users += self.get_domain_sub_users(content.ous, ldap_manage, key_config)

            global_info.IMPORT_TOTAL_NUM = len(all_domain_users)

            # 添加用户到数据库
            self.add_user_to_db(content.domain, all_domain_users, option, False)

    def set_domain_sync_status(self, domain_id, status):
        """
        设置域同步的状态
        Arg：
            domain_id：域控id
            statu: -1: 关闭域同步
                    0：开启正向同步
        """
        if not domain_id or (status not in [-1, 0]):
            raise_exception(exp_msg=_("domain config parameter error"),
                            exp_num=ncTShareMgntError.
                            NCT_DOMAIN_CONFIF_PARAMETER)

        # 检查域是否存在
        self.check_domain_exists(domain_id)

        try:
            # 更新域同步状态字段
            sql = """
            UPDATE `t_domain`
            SET `f_sync` = %s
            WHERE `f_domain_id` = %s
            """
            self.w_db.query(sql, status, domain_id)

        except Exception as ex:
            raise_exception(exp_msg=str(ex),
                            exp_num=ncTShareMgntError.
                            NCT_DB_OPERATE_FAILED)

    def get_domain_sync_status(self, domain_id):
        """
        获取某个域同步的状态,
        """
        self.check_domain_exists(domain_id)

        sql = """
        SELECT `f_sync` FROM `t_domain`
        WHERE `f_domain_id` = %s
        """
        result = self.r_db.one(sql, domain_id)
        return result['f_sync']

    def check_domain_exists(self, domain_id):
        """
        检查域是否存在
        """
        sql = """
        SELECT `f_domain_name` FROM  `t_domain`
        WHERE `f_domain_id` = %s
        """
        result = self.r_db.one(sql, domain_id)
        if not result:
            raise_exception(exp_msg=_("domain not exists"),
                            exp_num=ncTShareMgntError.
                            NCT_DOMAIN_NOT_EXIST)

    def set_domain_sync_config(self, domain_id, config):
        """
        设置域的配置信息
        """
        # 检查域是否存在
        self.check_domain_exists(domain_id)

        config_json = {}
        if config.destDepartId:
            is_exists = self.depart_manage.check_depart_exists(config.destDepartId, True, False)
            if not is_exists:
                raise_exception(exp_msg=_("dest department not exist"),
                                exp_num=ncTShareMgntError.
                                NCT_DOMAIN_CONFIF_PARAMETER)
            else:
                config_json['parent_id'] = config.destDepartId

        config_json['ous'] = config.ouPath if config.ouPath else []
        config_json['sync_interval'] = config.syncInterval if config.syncInterval else 5

        if config.syncMode is None:
            config.syncMode = ncTUsrmDomainSyncMode.NCT_SYNC_UPPER_OU

        if config.syncMode < ncTUsrmDomainSyncMode.NCT_SYNC_UPPER_OU or \
                config.syncMode > ncTUsrmDomainSyncMode.NCT_SYNC_USERS_ONLY:
            raise_exception(exp_msg=_("IDS_INVALID_DOMAIN_SYNC_MODE"),
                            exp_num=ncTShareMgntError.NCT_DOMAIN_CONFIF_PARAMETER)

        config_json['sync_mode'] = config.syncMode

        if config.validPeriod is None:
            config.validPeriod = -1
        else:
            if (config.validPeriod != -1) and (config.validPeriod < 0):
                raise_exception(exp_msg=_("IDS_INVALID_DATE"),
                                exp_num=ncTShareMgntError.NCT_INVALID_DATETIME)

        config_json['valid_period'] = config.validPeriod

        # 用户默认创建状态
        if config.userEnableStatus is not None:
            config_json['user_enable_status'] = config.userEnableStatus

        # 用户密级
        if config.csfLevel:
            self.user_manage.check_user_csflevel(config.csfLevel)
            config_json['csf_level'] = config.csfLevel
        else:
            # 未传密级，默认为用户密级最小值
            config_json['csf_level'] = self.config_manage.get_min_csf_level()

        if config_json:
            config_json = json.dumps(config_json)

            sql = """
            UPDATE `t_domain`
            SET `f_config` = %s
            WHERE `f_domain_id` = %s
            """
            self.w_db.query(sql, config_json, str(domain_id))

        # 首选域域配置变化时，可用域池配置也要更新
        self.init_available_domain_pool(domain_id)

    def get_domain_sync_config(self, domain_id):
        """
        获取域的配置信息
        """
        # 检查域是否存在
        self.check_domain_exists(domain_id)

        sql = """
        SELECT `f_config` FROM `t_domain`
        WHERE `f_domain_id` = %s
        """
        result = self.r_db.one(sql, domain_id)

        domain_config = ncTUsrmDomainConfig()
        if result:
            config = result['f_config']
            if config:
                config = json.loads(config)

                if 'parent_id' in config:
                    domain_config.destDepartId = config['parent_id']

                    b_exist = self.depart_manage.check_depart_exists(domain_config.destDepartId, True, False)
                    if domain_config.destDepartId in [NCT_UNDISTRIBUTE_USER_GROUP, NCT_ALL_USER_GROUP]:
                        b_exist = False

                    if b_exist:
                        dept_info = self.depart_manage.get_department_info(domain_config.destDepartId, True)
                        if dept_info:
                            domain_config.desetDepartName = dept_info.departmentName

                if 'ous' in config:
                    domain_config.ouPath = []
                    for ou in config['ous']:
                        domain_config.ouPath.append(ou)

                if 'sync_interval' in config:
                    domain_config.syncInterval = config['sync_interval']

                if 'sync_mode' in config:
                    domain_config.syncMode = int(config['sync_mode'])
                else:
                    domain_config.syncMode = ncTUsrmDomainSyncMode.NCT_SYNC_UPPER_OU

                if 'user_enable_status' in config:
                    domain_config.userEnableStatus = config['user_enable_status']

                if 'forced_sync' in config:
                    domain_config.forcedSync = config['forced_sync']
                else:
                    domain_config.forcedSync = True

                if 'valid_period' in config:
                    domain_config.validPeriod = config['valid_period']
                else:
                    domain_config.validPeriod = -1

                min_csf_level = self.config_manage.get_min_csf_level()
                domain_config.csfLevel = config.get("csf_level", min_csf_level)

        return domain_config

    def set_domain_key_config(self, domain_id, key_config):
        """
        设置域控关键字属性配置
        """
        # 检查域是否存在
        self.check_domain_exists(domain_id)

        config_keys = ["departNameKeys", "departThirdIdKeys", "loginNameKeys", "idcardNumberKeys",
                       "telNumberKeys","displayNameKeys", "emailKeys", "userThirdIdKeys", "groupKeys",
                       "subOuFilter", "subUserFilter", "baseFilter", "statusKeys"]

        cur_config = self.get_domain_key_config(domain_id)
        config_json = {}
        for keys in config_keys:
            if key_config.__dict__[keys]:
                config_json[keys] = key_config.__dict__[keys]
            else:
                config_json[keys] = cur_config.__dict__[keys]

        config_json = json.dumps(config_json)

        sql = """
        UPDATE `t_domain`
        SET `f_key_config` = %s
        WHERE `f_domain_id` = %s
        """
        self.w_db.query(sql, config_json, str(domain_id))

        # 首选域域配置变化时，可用域池配置也要更新
        self.init_available_domain_pool(domain_id)

    def get_domain_key_config(self, domain_id):
        """
        获取域控关键字属性配置
        """
        # 检查域是否存在
        self.check_domain_exists(domain_id)

        sql = """
        SELECT `f_key_config` FROM `t_domain`
        WHERE `f_domain_id` = %s
        """
        result = self.r_db.one(sql, domain_id)

        config_keys = ["departNameKeys", "departThirdIdKeys", "loginNameKeys", "idcardNumberKeys",
                       "telNumberKeys","displayNameKeys", "emailKeys", "userThirdIdKeys", "groupKeys",
                       "subOuFilter", "subUserFilter", "baseFilter", "statusKeys"]

        domain_config = ncTUsrmDomainKeyConfig()
        if result:
            config = result['f_key_config']
            if config:
                config = json.loads(config)
                for keys in config_keys:
                    if keys in config:
                        value = config[keys]
                        if value:
                            domain_config.__dict__[keys] = value

        if domain_config.statusKeys is None:
            domain_config.statusKeys = ["userAccountControl"]

        return domain_config

    def __convert_domain_config(self, conf_str):
        """
        将json字符串配置转换为ncTUsrmDomainKeyConfig对象
        遍历对象中拥有的配置项，然后从json配置中获取指定值
        将
        """
        domain_config = ncTUsrmDomainKeyConfig()
        json_config = json.loads(conf_str)
        for key in domain_config.__dict__:
            values = json_config.get(key)
            if not values:
                continue
            # 对配置内容进行转码
            if isinstance(values, list):
                for idx, val in enumerate(values):
                    values[idx] = val
            domain_config.__dict__[key] = values

        if domain_config.statusKeys is None:
            domain_config.statusKeys = ["userAccountControl"]

        return domain_config

    def get_sync_interval(self):
        """
        获取域同步周期
        """
        sql = """
        SELECT `f_value`
        FROM `t_sharemgnt_config`
        WHERE `f_key` = %s
        """
        result = self.r_db.one(sql, 'sync_interval')
        return int(result['f_value'])

    def set_ad_sso_status(self, status):
        """
        设置windows ad会话凭证是否可以登录anyshare
        """
        if status:
            value = 1
        else:
            value = 0

        sql = """
        UPDATE `t_sharemgnt_config`
        SET `f_value` = %s
        WHERE `f_key` = 'windows_ad_sso'
        """
        self.w_db.query(sql, value)

    def get_ad_sso_status(self):
        """
        获取windows ad会话凭证是否可以登录anyshare
        """
        sql = """
        SELECT `f_value`
        FROM `t_sharemgnt_config`
        WHERE `f_key` = %s
        """
        result = self.r_db.one(sql, 'windows_ad_sso')
        if (result['f_value'] == '1'):
            return True
        else:
            return False

    def check_import_domain_option(self, option):
        """
        检测导入用户参数合法性
        """

        # 检查用户账号有效期
        if option.expireTime is None:
            option.expireTime = -1
        else:
            if option.expireTime != -1 and option.expireTime < int(BusinessDate.time()):
                raise_exception(exp_msg=_("IDS_INVALID_DATE"),
                                exp_num=ncTShareMgntError.NCT_INVALID_DATETIME)

        # 检查用户设置密级
        self.user_manage.check_user_csflevel(option.csfLevel)

    def user_login(self, loginName, ldapServerType, domainPath, password):
        """
        域账号登录
        """
        # 分离账号以及域名
        user_name, domain_name = loginName.rsplit("@", 1)

        # 如果是windows AD域，用户的登录名后缀可能是UPN, 所以需要重新设置下域名
        # AD域用户的f_domain_path存储的是域名
        ldap_type = ldapServerType

        if ldap_type == 1 and domainPath:
            domain_name = domainPath

        # 根据域名检查域是否存在
        db_domain = self.__get_domain_by_name(domain_name)

        # 域不存在
        if not db_domain:
            raise_exception(exp_msg=_("user domain not exists"),
                            exp_num=ncTShareMgntError.NCT_DOMAIN_NOT_EXIST)

        # 域控被禁用
        if not db_domain['f_status']:
            raise_exception(exp_msg=_("user domain disabled"),
                            exp_num=ncTShareMgntError.NCT_DOMAIN_DISABLE)

        user_name = self.__get_login_name(user_name, loginName, domainPath, db_domain)
        if not user_name:
            raise_exception(exp_msg=_("user not exist in domain"),
                            exp_num=ncTShareMgntError.NCT_DOMAIN_USER_NOT_EXIST)

        # 验证能否登录
        try:
            domain = self.fecth_domain(db_domain)
            ldap_manage, tmp = self.connect_domain(domain.id, user_name, password)
        except ncTException as ex:
            if ex.errID == ncTShareMgntError.NCT_INVALID_ACCOUNT_OR_PASSWORD:
                return False
            else:
                raise ex
        return True

    def __get_domain_by_name(self, domain_name):
        """
        根据域名获取域信息
        """
        sql = "SELECT * FROM t_domain"
        result = self.r_db.all(sql)
        for row in result:
            db_name = dn2name(row['f_domain_name'])
            if domain_name == db_name:
                return row

    def __get_login_name(self, user_name, loginName, domainPath, db_domain):
        """
        获取域登录账号
        Args:
            user_name string 去除过与后缀的账号
            db_user dict 用户的数据库信息
            db_domain dict 域的数据库信息
        逻辑说明：
        1. f_domain_path中，保存了用户dn，直接使用第一个key作为账号搜索关键字，比如：uid=xxx,OU=People,DC=test,DC=com
        2. 判断域同步关键字，如果loginNameKey第一个值为：userPrincipleName，则直接进行登录
        3. 如果是其他同步关键字，则以该关键字作为账号搜索关键字
        """
        domain_path = domainPath
        if domain_path and domain_path.find(',') != -1:
            # f_domain_path中，保存了用户dn，直接使用第一个key作为账号搜索关键字，比如：uid=xxx,OU=People,DC=test,DC=com
            search_key = domain_path.split(',', 1)[0]
        else:
            # 查询域同步关键字配置
            key_config = self.__convert_domain_config(db_domain['f_key_config'])
            if key_config.loginNameKeys[0] == UPN_KEY:
                return loginName
            else:
                search_key = "%s=%s" % (key_config.loginNameKeys[0], user_name)

        host = db_domain['f_ip_address']
        if (":" in host) and ("[" not in host):
            host = f"[{host}]"
        ldap_manage = LdapManage(host,
                                 db_domain['f_administrator'],
                                 db_domain['f_password'],
                                 db_domain['f_port'],
                                 use_ssl=db_domain['f_use_ssl'])
        base_dn = name2dn(db_domain['f_domain_name'])
        result = ldap_manage.search_subtree(base_dn, search_key)
        result = list(result)
        # 空结果，返回None
        if not result:
            return
        return result[0][0]

    def __check_failover_domain_config(self, failover_domain):
        """
        检查备用域信息是否正确
        """
        parent_domain = self.get_domain_by_id(failover_domain.parentId)
        # 主域必须存在
        if not parent_domain:
            raise_exception(exp_msg=_("IDS_PARENT_DOMAIN_NOT_EXIST"),
                            exp_num=ncTShareMgntError.NCT_PARENT_DOMAIN_NOT_EXIST)

        # 只有主域才能添加备用域
        if parent_domain.type != ncTUsrmDomainType.NCT_DOMAIN_TYPE_PRIMARY:
            raise_exception(exp_msg=_("domain type not match parent id"),
                            exp_num=ncTShareMgntError.NCT_DOMAIN_TYPE_NOT_MATCH)

        # 从域地址不能与主域相同
        if parent_domain.ipAddress.lower() == failover_domain.address.lower():
            raise_exception(exp_msg=_("IDS_FAILOVER_DOMAIN_ADDRESS_SAME_WITH_PARENT"),
                            exp_num=ncTShareMgntError.NCT_FAILOVER_DOMAIN_ADDRESS_SAME_WITH_PARENT)

        # 验证是否能登录到域
        ldap_manage = self.__get_ldap_manage(failover_domain)

        # 验证主从域的域名是否匹配
        if parent_domain.name.lower() != ldap_manage.domain_name.lower():
            raise_exception(exp_msg=_("IDS_FAILOVER_NOT_MATCH_PARENT"),
                            exp_num=ncTShareMgntError.NCT_FAILOVER_NOT_MATCH_PARENT)

    def __edit_failover_domain_db(self, failover_domains, parent_domain_id):
        """
        添加备用域到数据库
        """
        # 使用事务
        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()

        try:
            delete_sql = """
            DELETE FROM `t_failover_domain` WHERE `f_parent_domain_id` = %s
            """
            cursor.execute(delete_sql, (parent_domain_id,))

            for each in failover_domains:
                sql = """
                INSERT INTO `t_failover_domain`(`f_parent_domain_id`,
                                                `f_ip_address`,
                                                `f_port`,
                                                `f_administrator`,
                                                `f_password`,
                                                `f_use_ssl`)
                VALUES(%s, %s, %s, %s, %s, %s)
                """

                cursor.execute(sql, (each.parentId, each.address, each.port, each.adminName, each.password,
                                    each.useSSL if each.useSSL is not None else False))

            conn.commit()

        except Exception as e:
            conn.rollback()
            raise_exception(exp_msg=str(e),
                            exp_num=ncTShareMgntError.NCT_DB_OPERATE_FAILED)
        finally:
            cursor.close()
            conn.close()

    def __get_all_primary_domain_id(self):
        result = self.r_db.all("SELECT f_domain_id from t_domain WHERE f_domain_type = 1")
        return [row['f_domain_id'] for row in result if row]

    def __get_ldap_manage(self, domain, user_name = None, password = None):
        """
        通用连接方式
        不传用户名和密码，使用管理员账号登录；否则，使用传入的用户账号登录
        """
        # 如果域同步，使用配置的账号密码
        if user_name is None and password is None:
            admin = domain.adminName
            pwd = domain.password
            pwd_enc = True
        # 如果user_name、password存在，使用用户账号
        elif user_name and password:
            admin = user_name
            pwd = password
            pwd_enc = False
        # 其他情况报错（比如user_name不为空，password=''）
        else:
            raise_exception(exp_msg=_("invalid account or password"),
                            exp_num=ncTShareMgntError.NCT_INVALID_ACCOUNT_OR_PASSWORD)

        if isinstance(domain, ncTUsrmDomainInfo):
            server_ip = domain.ipAddress
            if (":" in server_ip) and ("[" not in server_ip):
                server_ip = f"[{server_ip}]"
            port = domain.port
            domain_name = domain.name
            use_ssl = domain.useSSL
        # 其余情况为domain为ncTUsrmFailoverDomainInfo
        else:
            server_ip = domain.address
            if (":" in server_ip) and ("[" not in server_ip):
                server_ip = f"[{server_ip}]"
            port = domain.port
            domain_name = ''
            use_ssl = domain.useSSL

        return LdapManage(server_ip,
                            admin,
                            pwd,
                            port,
                            domain_name,
                            use_ssl,
                            pwd_enc)


    def init_available_domain_pool(self, domain_id = -1):
        """
        初始化可用域池
        """
        with self.available_domain_pool_mutex:
            if domain_id == -1:
                # id为-1时，初始化所有的域
                id_list = self.__get_all_primary_domain_id()
            else:
                id_list = [domain_id]

            for domain_id in id_list:
                domain_list = self.get_full_domain_by_id(domain_id)
                AVAILABLE_DOMAIN_POOL[domain_id] = domain_list

    def get_full_domain_by_id(self, domain_id):
        """
        根据主域的id获取所有的域控信息，并把备用域的结构体转化为主域结构体的形式
        """
        domain_info = self.get_domain_by_id(domain_id)
        domain_list = [domain_info]
        failover_domains = self.get_failover_domains(domain_id)
        for failover_domain in failover_domains:
            # 备用域格式转换为主域格式，更新可用域列表时，id与主域的id保持一致
            domain = copy.deepcopy(domain_info)
            domain.id = failover_domain.parentId
            domain.ipAddress = failover_domain.address
            domain.port = failover_domain.port
            domain.adminName = failover_domain.adminName
            domain.password = failover_domain.password
            domain.useSSL = failover_domain.useSSL
            domain_list.append(domain)
        return domain_list

    def check_failover_domain_available(self, failover_domains, parent_domain_id = ''):
        """
        检查备用域是否可用
        """
        # 检查输入的备用域是否重复
        tmp_list = []
        for each in failover_domains:
            if parent_domain_id and each.parentId != parent_domain_id:
                raise_exception("Parameters error")

            each_address = each.address.lower()
            if each_address not in tmp_list:
                tmp_list.append(each_address)
            else:
                raise_exception(exp_msg=_("IDS_DUPLICATED_FAILOVER_DOMAIN")%each.address,
                                exp_num=ncTShareMgntError.NCT_DUPLICATED_FAILOVER_DOMAIN)

        for failover_domain in failover_domains:
            # 检查备用域信息是否正确
            self.__check_failover_domain_config(failover_domain)

    def edit_failover_domains(self, failover_domains, parent_domain_id):
        """
        编辑备用域
        """
        # 检查备用域是否可用
        self.check_failover_domain_available(failover_domains, parent_domain_id)

        self.__edit_failover_domain_db(failover_domains, parent_domain_id)

        self.init_available_domain_pool(parent_domain_id)

    def get_failover_domains(self, parent_domain_id):
        """
        根据首选域id获取备用域信息
        """
        sql = """
        SELECT * FROM `t_failover_domain` WHERE `f_parent_domain_id` = %s
        """
        result = self.r_db.all(sql, parent_domain_id)
        failover_infos = []
        for row in result:
            failover_info = ncTUsrmFailoverDomainInfo()
            failover_info.id = row['f_domain_id']
            failover_info.parentId = row['f_parent_domain_id']
            failover_info.address = row['f_ip_address']
            failover_info.port = row['f_port']
            failover_info.adminName = row['f_administrator']
            failover_info.password = row['f_password']
            failover_info.useSSL = row['f_use_ssl']
            failover_infos.append(failover_info)
        return failover_infos

    def get_available_domain_by_id(self, domain_id):
        """
        根据首选域id获取可用域
        """
        ldap_manage, domain_info = self.connect_domain(domain_id)
        return domain_info

    def get_ldap_and_domain(self, domain_id, domain_pool, user_name =None, password = None):
        """
        从domain_pool中获取可用的域
        domain_pool可能来自缓存，也可能来自数据库
        """
        if not domain_pool or not domain_id:
            return

        result = None
        new_pool = domain_pool[:]
        for domain_info in domain_pool:
            try:
                ldap_manage = self.__get_ldap_manage(domain_info, user_name, password)
                result = (ldap_manage, domain_info)
                break
            except ncTException as ex:
                # 如果连接域控有问题，移除该域;其他错误，抛出
                if ex.errID in [ncTShareMgntError.NCT_DOMAIN_ERROR, ncTShareMgntError.NCT_DOMAIN_SERVER_UNAVAILABLE]:
                    new_pool.remove(domain_info)
                    ShareMgnt_Log("Temporarily remove %s from domain control pool"%(domain_info.ipAddress))
                else:
                    raise ex

        # 更新缓存
        with self.available_domain_pool_mutex:
            AVAILABLE_DOMAIN_POOL[domain_id] = new_pool
        return result

    def connect_domain(self, domain_id, user_name = None, password = None):
        """
        根据首选域id获取可用域和ldap
        result为ldap_manage，domain_info
        """
        # 获取缓存中的域
        domain_pool = AVAILABLE_DOMAIN_POOL.get(int(domain_id))
        result = self.get_ldap_and_domain(domain_id, domain_pool, user_name, password)
        if result:
            return result

        # 缓存中为空，获取数据库中的域
        domain_pool = self.get_full_domain_by_id(domain_id)
        result = self.get_ldap_and_domain(domain_id, domain_pool, user_name, password)
        if result:
            return result
        # 缓存、数据库中的域都不可用，报错
        raise_exception(exp_msg=_("domain server unavailable"),
                        exp_num=ncTShareMgntError.NCT_DOMAIN_SERVER_UNAVAILABLE,
                        exp_detail=domain_pool[0].name)

class InitAvailableDomainPoolThread(threading.Thread):
    """
    初始化可用域线程
    """
    def __init__(self):
        """
        初始化
        """
        super(InitAvailableDomainPoolThread, self).__init__()
        self.domain_mange = DomainManage()
        self.config_manage = ConfigManage()
        self.terminate = False

    def get_interval(self):
        return int(self.config_manage.get_config('get_available_domains_interval'))

    def close(self):
        """
        关闭
        """
        self.terminate = True

    def run(self):
        """
        执行
        """
        ShareMgnt_Log("**************** initialize available domain pool thread start *****************")

        while True:
            try:
                interval = self.get_interval()
                if not self.terminate:
                    self.domain_mange.init_available_domain_pool()
            except Exception as e:
                ShareMgnt_Log("initialize available domain pool thread run error: %s", str(e))

            time.sleep(interval)

        ShareMgnt_Log("**************** initialize available domain pool thread end *****************")
