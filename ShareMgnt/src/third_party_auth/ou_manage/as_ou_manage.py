#!/usr/bin/python3
# -*- coding:utf-8 -*-

import json
import uuid
import time
from datetime import datetime
from eisoo.tclients import TClient
from src.common import global_info
from src.common.http import pub_nsq_msg
from src.common.db.connector import ConnectorManager, safe_cursor
from src.common.lib import (check_name,
                            check_name2,
                            encrypt_pwd,
                            generate_group_str,
                            is_code_string,
                            raise_exception)
from src.third_party_auth.ou_manage.as_sql import *
from src.modules.user_manage import UserManage, TOPIC_DEPARTMENT_USER_ADD, TOPIC_DEPARTMENT_USER_REMOVE
from src.modules.config_manage import ConfigManage
from src.modules.department_manage import DepartmentManage, TOPIC_DEPT_DELETE, TOPIC_DEPART_MANAGER_MODIFIED, TOPIC_DEPART_STATUS_MODIFIED
from src.common.sharemgnt_logger import ShareMgnt_Log
from src.common.business_date import BusinessDate
from src.common.eacp_log import eacp_log
from src.third_party_auth.ou_manage.base_ou_manage import (BaseOuManage,
                                                           OuInfo,
                                                           UserInfo)
from ShareMgnt.constants import (NCT_USER_ADMIN,
                                 NCT_USER_AUDIT,
                                 NCT_USER_SYSTEM,
                                 NCT_DEFAULT_ORGANIZATION,
                                 NCT_UNDISTRIBUTE_USER_GROUP,
                                 ncTUsrmUserStatus)
from ShareMgnt.ttypes import (ncTShareMgntError,
                              ncTUsrmUserType,
                              ncTUsrmOSSInfo)
from EThriftException.ttypes import ncTException
from EVFS.ttypes import ncTEVFSError

INIT_PRIORITY = 999
TOPIC_USER_CREATE = "core.user_management.user.created"
TOPIC_ORG_NAME_MODIFY = "core.org.name.modify"
TOPIC_DEPT_CREATED = "core.user_management.dept.created"
TOPIC_USER_STATUS_CHANGED = "user_management.user.status.changed"
TOPIC_USER_MODIFIED = "user_management.user.modified"


class ASOuManage(BaseOuManage):
    """
    AnyShare组织结构处理器
    """

    def __init__(self, b_eacplog=False):
        """
        初始化函数
        """
        super(ASOuManage, self).__init__()
        self.depart_manage = DepartmentManage()
        self.user_manage = UserManage()
        self.config_manage = ConfigManage()
        self.user_create_status = 0
        self.user_csf_level = self.config_manage.get_min_csf_level()
        self.user_csf_level2 = self.config_manage.get_min_csf_level2()
        self.b_eacplog = b_eacplog
        self.expire_time = -1

    def ou_eacp_log(self, op_type, msg, level=global_info.LOG_LEVEL_INFO, ex_msg=None):
        """
        记录组织操作日志
        参数：
            loglevel - {str} 日志级别
            NCT_LL_ALL = 0,    // 所有日志级别
            NCT_LL_INFO = 1,   // 信息
            NCT_LL_WARN = 2,   // 警告

            optype - {str} 操作类型
            NCT_SOT_ALL = 0,        // 所有操作
            NCT_SOT_CREATE = 1,     // 新建操作
            NCT_SOT_MODIFY = 2,     // 修改操作
            NCT_SOT_DELETE = 3,     // 删除操作
            NCT_OOT_ADD_USER_TO_DEP = 4,    // 添加用户到部门
            NCT_OOT_MOVE_USER_AND_DEP = 5,  // 迁移用户和部门
            NCT_OOT_DISABLE = 6,            // 禁用
            NCT_OOT_ENABLE = 7,             // 启用
        """
        eacp_log(_("IDS_SYNCER"),
                        global_info.LOG_TYPE_MANAGE,
                        global_info.USER_TYPE_INTER,
                        level,
                        op_type,
                         msg,
                         ex_msg,
                         raise_ex=True)

    def init_sync_config(self, server_info):
        """
        初始化全局的同步选项
        1.创建用户时是否创建个人文档, 0 表示启用，1 表示禁用
        2.创建用户时默认为启用/还是禁用状态
        """
        self.admin_list = list(
            self.user_manage.get_all_admin_account().values())

        # 0 表示启用，1 表示禁用
        if isinstance(server_info, dict):
            # 第三方组织结构同步的配置是字典，根据userCreateStatus来判断启用/禁用
            if "userCreateStatus" in server_info and not server_info["userCreateStatus"]:
                self.user_create_status = ncTUsrmUserStatus.NCT_STATUS_DISABLE
            else:
                self.user_create_status = ncTUsrmUserStatus.NCT_STATUS_ENABLE

            # 根据validPeriod来设置用户账号有效期
            if "validPeriod" in server_info and isinstance(server_info["validPeriod"], int) \
                    and server_info["validPeriod"] != -1:
                self.expire_time = int(
                    BusinessDate.time() + server_info["validPeriod"] * 24 * 3600)

            # 根据配置设置第三方同步默认用户密级
            csf_config = json.loads(self.config_manage.get_config("csf_level_enum"))
            csf_level = csf_config.values()
            if server_info.get("userCsfLevel", ""):
                self.user_csf_level = server_info["userCsfLevel"]
            if self.user_csf_level not in csf_level:
                # 如果第三方同步的密级不在用户密级枚举中，则使用用户密级最小值
                self.user_csf_level = self.config_manage.get_min_csf_level()
            ShareMgnt_Log('user_csf_level: %d, 创建用户的默认同步密级' %
                          self.user_csf_level)
            csf2_config = json.loads(self.config_manage.get_config("csf_level2_enum"))
            csf2_level = csf2_config.values()
            if server_info.get("userCsfLevel2", ""):
                self.user_csf_level2 = server_info["userCsfLevel2"]
            if self.user_csf_level2 not in csf2_level:
                # 如果第三方同步的密级2不在用户密级2枚举中，则使用用户密级2最小值
                self.user_csf_level2 = self.config_manage.get_min_csf_level2()
            ShareMgnt_Log('user_csf_level2: %d, 创建用户的默认同步密级2' %
                          self.user_csf_level2)
        else:
            # 域控配置为ncTUsrmDomainInfo，根据userEnableStatus参数确定用户默认创建状态
            if server_info.config.userEnableStatus is False:
                self.user_create_status = ncTUsrmUserStatus.NCT_STATUS_DISABLE
            else:
                self.user_create_status = ncTUsrmUserStatus.NCT_STATUS_ENABLE

            # 根据validPeriod来设置用户账号有效期
            if server_info.config.validPeriod != -1:
                self.expire_time = int(
                    BusinessDate.time() + server_info.config.validPeriod * 24 * 3600)

        if self.user_create_status == ncTUsrmUserStatus.NCT_STATUS_ENABLE:
            ShareMgnt_Log('user_create_status: %d, 创建用户时设置为启用状态' %
                          self.user_create_status)
        else:
            ShareMgnt_Log('user_create_status: %d, 创建用户时设置为禁用状态' %
                          self.user_create_status)

    def get_root_id(self):
        """
        获取组织id
        """
        return NCT_DEFAULT_ORGANIZATION

    def check_user_info(self, user_info, b_create=True):
        """
        检查用户信息
        """
        if(not user_info.login_name or
                not check_name2(user_info.login_name) or
                user_info.login_name.lower() in self.admin_list):
            raise_exception(exp_msg=_("login name illegal"),
                            exp_num=ncTShareMgntError.NCT_INVALID_LOGIN_NAME)

        user_info.email = self.user_manage.is_email_valid(
            user_info.login_name, user_info.email)
        user_info.idcard_number = self.user_manage.is_idcardNumber_valid(
            user_info.idcard_number, user_info.login_name)
        if self.user_manage.check_user_exists_by_idcardNumber_loginName(user_info.idcard_number, user_info.login_name) == False:
            if self.user_manage.get_olduserinfo_by_loginName(user_info.idcard_number):
                user_info.idcard_number = self.user_manage.get_olduserinfo_by_loginName(
                    user_info.idcard_number)['f_idcard_number']
            else:
                user_info.idcard_number = ""
        if user_info.type != ncTUsrmUserType.NCT_USER_TYPE_THIRD:
            user_info.tel_number = self.user_manage.is_teleNumber_valid(
                user_info.tel_number, user_info.login_name)
        if not user_info.display_name:
            user_info.display_name = user_info.login_name
        else:
            user_info.display_name = user_info.display_name.strip()
            if not check_name2(user_info.display_name):
                raise_exception(exp_msg=_("display name illegal"),
                                exp_num=ncTShareMgntError.NCT_INVALID_DISPLAY_NAME)

        if not user_info.third_id:
            user_info.third_id = ''

        if not user_info.password:
            user_info.password = ''

        csf_config = json.loads(self.config_manage.get_config("csf_level_enum"))
        csf_level = csf_config.values()
        if user_info.csf_level:
            if user_info.csf_level not in csf_level:
                ShareMgnt_Log('用户："%s"设置密级不合法，请检查系统密级: third_id=%s',
                              user_info.login_name,
                              user_info.third_id)
                user_info.csf_level = None
        else:
            if user_info.type != ncTUsrmUserType.NCT_USER_TYPE_DOMAIN and b_create:
                user_info.csf_level = self.user_csf_level

        csf2_config = json.loads(self.config_manage.get_config("csf_level2_enum"))
        csf2_level = csf2_config.values()
        if user_info.csf_level2:
            if user_info.csf_level2 not in csf2_level:
                ShareMgnt_Log('用户："%s"设置密级2不合法，请检查系统密级2: third_id=%s',
                              user_info.login_name,
                              user_info.third_id)
                user_info.csf_level2 = None
        else:
            if user_info.type != ncTUsrmUserType.NCT_USER_TYPE_DOMAIN and b_create:
                user_info.csf_level2 = self.user_csf_level2

        if user_info.position is not None:
            user_info.position = self.user_manage.check_user_position(user_info.position)

        if user_info.code is not None:
            user_info.code = self.check_third_party_user_code(user_info.code, user_info.third_id)

    def check_third_party_user_code(self, code, third_id=None):
        """
        检查用户编码格式 以及是否唯一
        """
        # 除去前面的空格，末尾的空格
        striped_code = code.lstrip()
        striped_code = striped_code.rstrip()

        if striped_code == "":
            return striped_code

        if not is_code_string(striped_code):
            raise_exception(exp_msg=_("IDS_INVALID_USER_CODE"),
                            exp_num=ncTShareMgntError.NCT_INVALID_USER_CODE)


        select_sql = """
        select f_third_party_id from t_user where f_code = %s
        """
        result = self.r_db.one(select_sql, striped_code)
        if result and result['f_third_party_id'] != third_id:
            raise_exception(exp_msg=_("IDS_DUPLICATED_USER_CODE"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_USER_CODE)
        return striped_code

    def get_unique_display_name(self, display_name, user_id=None):
        """
        检查系统中是否存在相同的显示名.
        参数：display_name:需要检查的用户显示名
              user_id: 排除检查的用户id
        返回值：如果系统中存在相同的显示名，则返回当前显示名加索引后的名字;否则，返回原显示名.
        """
        return self.user_manage.get_unique_displayname(display_name, user_id)

    def get_md5_password(self, password):
        """
        获取md5加密的密码
        """
        md5_password = ''
        if password:
            md5_password = encrypt_pwd(password)
        return md5_password

    def is_init_password(self, password):
        """
        检查是否是初始密码
        """
        return True if encrypt_pwd(password) == self.user_manage.user_default_password.md5_pwd else False

    def get_unique_depart_name(self, parent_id, ou_info, exclude_id=None):
        """
        获取唯一部门名
        """
        unique_name = ou_info.ou_name
        try:
            self.depart_manage.check_name_in_sub_departs(
                parent_id, ou_info.ou_name, exclude_id)
        except ncTException as e:
            if e.errID == ncTShareMgntError.NCT_DEPARTMENT_HAS_EXIST:
                unique_name = "%s(%s)" % (
                    ou_info.ou_name, ou_info.third_id[0:17])
            else:
                raise e
        return unique_name

    def compare_ou_info(self, old_ou_info, new_ou_info):
        """
        比较组织部门信息是否一致
        """
        if not new_ou_info or not old_ou_info:
            return False

        if new_ou_info.ou_name.strip() != old_ou_info.ou_name.strip():
            return False

        # 部门排序码如果是默认值则不覆盖
        if new_ou_info.priority and new_ou_info.priority != global_info.DEFAULT_DEPART_PRIORITY and new_ou_info.priority != old_ou_info.priority:
            return False

        if new_ou_info.remark is not None and new_ou_info.remark != old_ou_info.remark:
            return False

        if new_ou_info.code is not None and new_ou_info.code != old_ou_info.code:
            return False

        if new_ou_info.status is not None and new_ou_info.status != old_ou_info.status:
            return False

        return True

    def compare_user_info(self, old_user_info, new_user_info):
        """
        比较用户信息是否一致
        """
        if not new_user_info or not old_user_info:
            return False

        if (new_user_info.display_name.rstrip() != old_user_info.display_name.rstrip() or
                new_user_info.login_name.rstrip() != old_user_info.login_name.rstrip() or
                new_user_info.dn != old_user_info.dn or
                new_user_info.type != old_user_info.type):
            return False

        # 根据用户类型判断是否需要比较用户密码
        if new_user_info.type != ncTUsrmUserType.NCT_USER_TYPE_LOCAL:
            if new_user_info.password != old_user_info.password:
                return False

        if new_user_info.priority and new_user_info.priority != old_user_info.priority:

            # 域控用户如果是默认999权值时认为没有修改权值，不会触发更新用户操作
            if not (new_user_info.type == ncTUsrmUserType.NCT_USER_TYPE_DOMAIN and new_user_info.priority == 999):
                return False

        if new_user_info.email and new_user_info.email != old_user_info.email:
            return False

        if new_user_info.third_attr and new_user_info.third_attr != old_user_info.third_attr:
            return False

        if new_user_info.idcard_number and new_user_info.idcard_number != old_user_info.idcard_number:
            return False

        if new_user_info.tel_number and new_user_info.tel_number != old_user_info.tel_number:
            return False

        if new_user_info.type != ncTUsrmUserType.NCT_USER_TYPE_DOMAIN:
            if (new_user_info.csf_level and new_user_info.csf_level != old_user_info.csf_level) or old_user_info.csf_level != self.user_csf_level:
                return False

        if new_user_info.position is not None and new_user_info.position != old_user_info.position:
            return False

        if new_user_info.code is not None and new_user_info.code != old_user_info.code:
            return False

        if new_user_info.csf_level2 and new_user_info.csf_level2 != old_user_info.csf_level2:
            return False

        return True

    def chec_user_exists(self, third_user_id):
        """
        判断用户是否存在
        """
        if third_user_id:
            result = self.r_db.one(check_user_exist_by_id_sql, third_user_id)
            if result:
                if result['cnt']:
                    return True
        return False

    def check_user_exists_by_name(self, login_name):
        """
        根据登录名判断用户是否存在
        """

        if login_name:
            result = self.r_db.one(check_user_exist_by_name_sql, login_name)
            if result:
                if result['cnt']:
                    return True
        return False

    def check_depart_exists(self, third_ou_id):
        """
        检查部门是否存在
        """
        if third_ou_id:
            result = self.r_db.one(check_dept_exist_by_id_sql, third_ou_id)
            if result:
                if result['cnt']:
                    return True
        return False

    def check_user_is_responsible(self, user_id, depart_id):
        """
        检查用户是否是某部门的管理员
        """
        return self.user_manage.check_is_responsible_person_of_depart(user_id, depart_id, raise_ex=False)

    def check_user_belong_other_depart(self, user_id, depart_id):
        """
        检查用户是否属于其他部门
        """
        depart_path = self.depart_manage.get_department_path_by_dep_id(depart_id)
        if depart_path == '':
            ShareMgnt_Log('departmentinfo illegal depart_path is None, depart_id:%s, user_id: %s',
                          depart_id, user_id)
            raise_exception(exp_msg=_("departmentinfo illegal"),
                            exp_num=ncTShareMgntError.
                            NCT_ORG_OR_DEPART_NOT_EXIST)
        result = self.r_db.one(
            check_user_belong_other_depart_sql, user_id, depart_path)
        return True if result['cnt'] else False

    def check_user_belong_other_depart_same_org(self, user_id, depart_id):
        """
        检查用户是否属于同一个组织中的其他部门
        """
        ou_id = self.get_belong_org_id(depart_id)
        depart_path = self.depart_manage.get_department_path_by_dep_id(depart_id)
        if depart_path == '':
            ShareMgnt_Log('departmentinfo illegal depart_path is None, depart_id:%s, user_id: %s',
                          depart_id, user_id)
            raise_exception(exp_msg=_("departmentinfo illegal"),
                            exp_num=ncTShareMgntError.
                            NCT_ORG_OR_DEPART_NOT_EXIST)
        result = self.r_db.one(
            check_user_belong_other_dept_same_ou_sql, user_id, ou_id + "%", depart_path)
        return True if result['cnt'] else False

    def get_all_depart_path_node_ids(self, depart_ids):
        """
        根据部门id, 获取部门到根节点的所有部门id集合
        """
        all_node_ids = set()
        if depart_ids is None:
            return all_node_ids

        for d_id in depart_ids:
            path_ids = self.depart_manage.get_dept_path_to_root(d_id)
            for node_id in path_ids:
                all_node_ids.add(node_id)
        return all_node_ids

    def get_manager_ids_for_update(self, user_id, depart_id):
        """
        获取需要更新配额空间的管理员
        """
        # 获取当前组织下, 除depart_id外用户所属的所有部门到根节点的路径
        ou_id = self.get_belong_org_id(depart_id)
        belong_depart_ids = self.get_belong_depart_ids(user_id, ou_id)

        # 获取当前部门到组织的部门路径
        path_ids = self.depart_manage.get_dept_path_to_root(depart_id)
        path_ids.reverse()

        all_path_node_ids = set()
        all_manager_ids = []
        for d_id in path_ids:
            manager_ids = self.depart_manage.get_depart_mgr_ids(d_id)
            for manager_id in manager_ids:
                # 获取除当前部门外的所有所属部门到根节点的路径节点id集合
                if not all_path_node_ids:
                    all_path_node_ids = self.get_all_depart_path_node_ids(
                        belong_depart_ids)

                # 判断用户是否属于当前部门或其子部门下, 如果是，则从当前节点到
                # 根节点路径上部门管理员都不需要更新
                if all_path_node_ids and d_id in all_path_node_ids:
                    break
                else:
                    if manager_id not in manager_ids:
                        all_manager_ids.append(manager_id)

        return all_manager_ids

    def get_belong_depart_ids(self, user_id, ou_id):
        """
        根据用户id，获取某一组织下所属的所有部门id
        """
        depart_ids = []
        results = self.r_db.all(
            select_belong_depart_in_ou_sql, user_id, ou_id + "%")
        for result in results:
            depart_ids.append(result["f_path"].split("/")[-1])
        return depart_ids

    def check_depart_belong_ou(self, depart_id, ou_id):
        """
        检查部门是否属于某个组织
        """
        result = self.r_db.one(check_depart_belong_ou_sql, depart_id, ou_id)
        return True if result['cnt'] else False

    def get_depart_id(self, third_ou_id):
        """
        根据第三方id获取部门id
        """
        if third_ou_id == NCT_UNDISTRIBUTE_USER_GROUP:
            return NCT_UNDISTRIBUTE_USER_GROUP

        if third_ou_id:
            result = self.r_db.one(select_third_id_sql, third_ou_id)
            if result:
                return result['f_department_id']

    def get_depart_oss_id(self, depart_id):
        """
        根据第三方id获取对象存储id
        """
        if depart_id:
            result = self.r_db.one(select_dept_oss_id_sql, depart_id)
            if result:
                return result['f_oss_id']
            else:
                return ""
        else:
            return ""

    def get_depart_info_by_id(self, depart_id):
        """
        根据部门id获取部门信息(部门信息包括第三方id)
        """
        if depart_id:
            if depart_id == NCT_UNDISTRIBUTE_USER_GROUP:
                ou_info = OuInfo()
                ou_info.ou_name = '未分配用户组'
                ou_info.depart_id = NCT_UNDISTRIBUTE_USER_GROUP
                ou_info.third_id = NCT_UNDISTRIBUTE_USER_GROUP
                ou_info.priority = global_info.DEFAULT_DEPART_PRIORITY
                ou_info.depart_path = NCT_UNDISTRIBUTE_USER_GROUP
                return ou_info
            else:
                result = self.r_db.one(
                    select_depart_by_depart_id_sql, depart_id)
                if result:
                    ou_info = OuInfo()
                    ou_info.ou_name = result['f_name']
                    ou_info.depart_id = result['f_department_id']
                    ou_info.third_id = result['f_third_party_id']
                    ou_info.priority = result['f_priority']
                    ou_info.depart_path = result['f_path']
                    return ou_info

    def get_belong_org_id(self, depart_id):
        """
        根据部门id获取部门所在的组织id
        """
        if depart_id:
            result = self.r_db.one(select_depart_ou_id_sql, depart_id)
            if result:
                return result['f_ou_id']

    def get_ou(self, third_ou_id):
        """
        获取组织部门信息
        """
        if third_ou_id:
            result = self.r_db.one(select_depart_by_third_id_sql, third_ou_id)
            if result:
                ou_info = OuInfo()
                ou_info.depart_id = result['f_department_id']
                ou_info.ou_name = result['f_name']
                ou_info.priority = result['f_priority']
                ou_info.third_id = third_ou_id

                status = result['f_status']
                ou_info.status = True
                if status == 2:
                    ou_info.status = False
                ou_info.manager_id = result['f_manager_id']
                ou_info.code = result['f_code']
                ou_info.remark = result['f_remark']
                return ou_info

    def get_sub_ous_by_depart_id(self, depart_id):
        """
        根据部门id获取子部门信息
        """
        if not depart_id:
            raise_exception(exp_msg=_("depart not exists"),
                            exp_num=ncTShareMgntError.
                            NCT_DEPARTMENT_NOT_EXIST)

        ou_infos = []
        depart_str = "/____________________________________"
        depart_path = self.depart_manage.get_department_path_by_dep_id(depart_id)
        if depart_path == '':
            ShareMgnt_Log(
                'departmentinfo illegal depart_path is None, depart_id:%s', depart_id)
            raise_exception(exp_msg=_("departmentinfo illegal"),
                            exp_num=ncTShareMgntError.
                            NCT_ORG_OR_DEPART_NOT_EXIST)
        results = self.r_db.all(select_sub_depart_sql,
                                depart_path + depart_str)
        if results:
            for result in results:
                if result['f_department_id'] != depart_id:
                    ou_info = OuInfo()
                    ou_info.third_id = result['f_third_party_id']
                    ou_info.ou_name = result['f_name']
                    ou_info.depart_id = result['f_department_id']
                    ou_info.priority = result['f_priority']
                    ou_info.remark = result['f_remark']
                    ou_info.code = result['f_code']
                    ou_info.manager_id = result['f_manager_id']

                    if result['f_status'] == 1:
                        ou_info.status = True
                    else:
                        ou_info.status = False
                    ou_infos.append(ou_info)

        return ou_infos

    def get_sub_ous_by_third_id(self, third_ou_id):
        """
        根据第三方id获取子组织部门
        """
        # 检查参数
        if not third_ou_id:
            raise_exception(exp_msg=_("parameter is none"),
                            exp_num=ncTShareMgntError.NCT_PARAMETER_IS_NULL)

        depart_id = self.get_depart_id(third_ou_id)
        return self.get_sub_ous_by_depart_id(depart_id)

    def get_user(self, third_user_id):
        """
        获取用户信息
        """
        if third_user_id:
            result = self.r_db.one(select_sub_user_by_dept_third_id_sql,
                                   third_user_id)
            if result:
                user_info = UserInfo()
                user_info.user_id = result['f_user_id']
                user_info.login_name = result['f_login_name']
                user_info.display_name = result['f_display_name']
                user_info.email = result['f_mail_address']
                user_info.idcard_number = result['f_idcard_number']
                user_info.tel_number = result['f_tel_number']
                user_info.status = result['f_status']
                user_info.dn = result['f_domain_path']
                user_info.password = result['f_password']
                user_info.type = result['f_auth_type']
                user_info.priority = result['f_priority']
                user_info.csf_level = result['f_csf_level']
                user_info.third_id = third_user_id
                user_info.position = result['f_position']
                user_info.code = result['f_code']
                user_info.manager_id = result['f_manager_id']

                return user_info

    def get_user_id_by_third_id(self, third_id):
        """
        通过特定third_id列表获取user_id
        """
        users_id = []
        if not third_id:
            third_id = ','.join(third_id)
            third_id_sql = '(' + third_id + ')'
            result = self.r_db.one(select_user_id_by_third_id_sql,
                                   third_id_sql)
            if result:
                users_id.append(result['f_user_id'])

        return users_id

    def get_sub_users_by_depart_id(self, depart_id):
        """
        根据部门id获取子用户
        """
        if not depart_id:
            raise_exception(exp_msg=_("depart not exists"),
                            exp_num=ncTShareMgntError.
                            NCT_DEPARTMENT_NOT_EXIST)

        user_infos = []
        depart_path = self.depart_manage.get_department_path_by_dep_id(depart_id)
        if depart_path == '':
            ShareMgnt_Log(
                'departmentinfo illegal depart_path is None, depart_id:%s', depart_id)
            raise_exception(exp_msg=_("departmentinfo illegal"),
                            exp_num=ncTShareMgntError.
                            NCT_ORG_OR_DEPART_NOT_EXIST)
        results = self.r_db.all(select_sub_user_sql,
                                depart_path,
                                NCT_USER_ADMIN,
                                NCT_USER_AUDIT,
                                NCT_USER_SYSTEM)
        if results:
            for result in results:
                user_info = UserInfo()
                user_info.user_id = result['f_user_id']
                user_info.third_id = result['f_third_party_id']
                user_info.login_name = result['f_login_name']
                user_info.display_name = result['f_display_name']
                user_info.email = result['f_mail_address']
                user_info.idcard_number = result['f_idcard_number']
                user_info.tel_number = result['f_tel_number']
                user_info.status = result['f_status']
                user_info.password = result['f_password']
                user_info.dn = result['f_domain_path']
                user_info.type = result['f_auth_type']
                user_info.priority = result['f_priority']
                user_info.third_attr = result['f_third_party_attr']
                user_info.csf_level = result['f_csf_level']
                user_info.code = result['f_code']
                user_info.position = result['f_position']
                user_info.csf_level2 = result['f_csf_level2']
                user_infos.append(user_info)
        return user_infos

    def get_sub_users_by_third_id(self, third_ou_id):
        """
        获取子用户
        """
        # 检查参数
        if not third_ou_id:
            raise_exception(exp_msg=_("parameter is none"),
                            exp_num=ncTShareMgntError.NCT_PARAMETER_IS_NULL)

        depart_id = self.get_depart_id(third_ou_id)
        return self.get_sub_users_by_depart_id(depart_id)

    def add_ou(self, parent_id, ou_info, progress_info):
        """
        增加部门
        """
        progress_info.synced_num += 1
        # 判断是添加组织还是部门
        is_org = 0
        if parent_id == -1 or ou_info.is_enterprise:
            is_org = 1

        # 检查参数
        if not parent_id or not ou_info:
            progress_info.failed_num += 1
            raise_exception(exp_msg=_("parameter is none"),
                            exp_num=ncTShareMgntError.NCT_PARAMETER_IS_NULL)

        # 检查部门名是否合法
        if not check_name(ou_info.ou_name):
            self.ou_eacp_log(global_info.LOG_OP_TYPE_CREATE,
                          _("IDS_CREATE_DEPART_FAILED") % ou_info.ou_name,
                          global_info.LOG_LEVEL_INFO, _("IDS_INVALID_DEPARTMENT_NAME"))

            progress_info.failed_num += 1
            ShareMgnt_Log('新建 部门"%s":%s 失败, 部门名不合法 %d/%d',
                          ou_info.ou_name,
                          ou_info.third_id,
                          progress_info.synced_num,
                          progress_info.total_num)

            return

        org_id = parent_id
        if not is_org:
            try:
                # 检查父部门是否存在
                self.depart_manage.check_depart_exists(parent_id, True)
            except Exception as ex:
                self.ou_eacp_log(global_info.LOG_OP_TYPE_CREATE,
                              _("IDS_CREATE_DEPART_FAILED") % ou_info.ou_name,
                              global_info.LOG_LEVEL_INFO,
                              _('IDS_PARENT_DEPARTMENT_NOT_EXISTS'))

                progress_info.failed_num += 1
                ShareMgnt_Log('新建 部门"%s":%s 失败，父部门不存在   %d/%d',
                              ou_info.ou_name,
                              ou_info.third_id,
                              progress_info.synced_num,
                              progress_info.total_num)
                return

            # 检查并获取部门所在组织id
            org_id = self.depart_manage.get_ou_by_depart_id(parent_id)
            parent_path = self.depart_manage.get_department_path_by_dep_id(parent_id)
            if parent_path == '':
                ShareMgnt_Log(
                    'departmentinfo illegal depart_path is None, depart_id:%s', depart_id)
                raise_exception(exp_msg=_("departmentinfo illegal"),
                                exp_num=ncTShareMgntError.
                                NCT_ORG_OR_DEPART_NOT_EXIST)
            if not org_id:
                self.ou_eacp_log(global_info.LOG_OP_TYPE_CREATE,
                              _("IDS_CREATE_DEPART_FAILED") % ou_info.ou_name,
                              global_info.LOG_LEVEL_INFO,
                              _('IDS_OU_OF_PARENT_DEPARTMENT_NOT_EXISTS'))

                progress_info.failed_num += 1
                ShareMgnt_Log('新建 部门"%s":%s 失败, 父部门所在组织不存在   %d/%d',
                              ou_info.ou_name,
                              ou_info.third_id,
                              progress_info.synced_num,
                              progress_info.total_num)
                return

            # 获取唯一部门名
            ou_info.ou_name = self.get_unique_depart_name(parent_id, ou_info)

        depart_id = str(uuid.uuid1())

        if is_org:
            org_id = depart_id
            depart_path = org_id
        else:
            depart_path = parent_path + "/" + depart_id

        # 检查排序权重
        if not ou_info.priority:
            ou_info.priority = global_info.DEFAULT_DEPART_PRIORITY

        # 如果部门的对象存储id存在，则取该部门的对象存储id，否则取父部门的对象存储id
        if ou_info.oss_id:
            oss_id = ou_info.oss_id
        else:
            oss_id = self.get_depart_oss_id(str(parent_id))


        # 检查remark
        if ou_info.remark:
            ou_info.remark = self.depart_manage._is_remark_valid(ou_info.remark)
        else:
            ou_info.remark = ''

        # 检查code
        if ou_info.code:
            ou_info.code = self.check_third_party_depart_code(ou_info.code)
        else:
            ou_info.code = ''

        status = 1
        if ou_info.status is not None and ou_info.status == False:
            status = 2

        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()

        try:
            # 依次插入组织或部门
            cursor.execute(insert_depart_sql,
                           (depart_id,
                            ou_info.type,
                            ou_info.ou_name,
                            is_org,
                            ou_info.third_id,
                            ou_info.dn,
                            ou_info.priority,
                            oss_id,
                            depart_path,
                            ou_info.remark,
                            status,
                            ou_info.code))

            # 插入部门关系
            if not is_org:
                cursor.execute(insert_depart_relation_sql,
                               (depart_id, parent_id))

            # 插入组织搜索关系
            cursor.execute(insert_depart_ou_sql, (depart_id, org_id))

            conn.commit()
            # 发送创建部门消息
            pub_nsq_msg(TOPIC_DEPT_CREATED,{"id":depart_id,"name":ou_info.ou_name})
            # 记录操作日志
            self.ou_eacp_log(global_info.LOG_OP_TYPE_CREATE,
                          _('IDS_CREATE_DEPARTMENT_SUCCESS') % ou_info.ou_name)

            progress_info.added_num += 1
            ShareMgnt_Log('新建 部门"%s":%s成功   %d/%d',
                          ou_info.ou_name,
                          ou_info.third_id,
                          progress_info.synced_num,
                          progress_info.total_num)

            return depart_id
        except Exception as ex:
            conn.rollback()
            self.ou_eacp_log(global_info.LOG_OP_TYPE_CREATE,
                          _('IDS_CREATE_DEPART_FAILED') % ou_info.ou_name,
                          global_info.LOG_LEVEL_INFO,
                          _('IDS_DATABASE_ERROR'))

            progress_info.failed_num += 1
            ShareMgnt_Log('新建 部门"%s":%s 异常: ex=%s   %d/%d',
                          ou_info.ou_name,
                          ou_info.third_id,
                          str(ex),
                          progress_info.synced_num,
                          progress_info.total_num)

    def move_ou(self, parent_id, ou_info, progress_info):
        """
        移动部门到其他部门，包括部门下的所有用户和子部门
        """
        progress_info.synced_num += 1

        try:
            # 处理目的部门下已存在 ou_info.name的情况
            self.update_ou(parent_id, ou_info, progress_info)

            src_dept_id = self.get_depart_id(ou_info.third_id)
            self.depart_manage.move_department(src_dept_id, parent_id)

            progress_info.moved_num += 1

            # 记录操作日志
            self.ou_eacp_log(global_info.LOG_OP_TYPE_MOVE,
                          _('IDS_MOVE_DEPARTMENT_SUCCESS') % ou_info.ou_name)

            ShareMgnt_Log('移动 部门"%s":%s成功   %d/%d',
                          ou_info.ou_name,
                          ou_info.third_id,
                          progress_info.synced_num,
                          progress_info.total_num)
        except Exception as ex:
            self.ou_eacp_log(global_info.LOG_OP_TYPE_MOVE,
                          _('IDS_MOVE_DEPARTMENT_FAILED') % ou_info.ou_name,
                          global_info.LOG_LEVEL_INFO,
                          _('IDS_DATABASE_ERROR'))

            progress_info.failed_num += 1
            ShareMgnt_Log('移动 部门"%s":%s 异常: ex=%s   %d/%d',
                          ou_info.ou_name,
                          ou_info.third_id,
                          str(ex),
                          progress_info.synced_num,
                          progress_info.total_num)

    def update_ou(self, parent_id, ou_info, progress_info):
        """
        更新组织部门
        """
        progress_info.synced_num += 1

        if not parent_id or not ou_info:
            progress_info.failed_num += 1
            return False

        # 检查部门名是否合法
        if not check_name(ou_info.ou_name):
            self.ou_eacp_log(global_info.LOG_OP_TYPE_SET,
                          _('IDS_EDIT_DEPARTMENT_FAILED') % ou_info.ou_name,
                          global_info.LOG_LEVEL_INFO,
                          _('IDS_INVALID_DEPARTMENT_NAME'))

            progress_info.failed_num += 1
            ShareMgnt_Log('更新 部门"%s":%s 失败, 部门名不合法   %d/%d',
                          ou_info.ou_name,
                          ou_info.third_id,
                          progress_info.synced_num,
                          progress_info.total_num)
            return False

        # 检查父部门是否存在
        try:
            self.depart_manage.check_depart_exists(parent_id, True)
        except Exception:
            self.ou_eacp_log(global_info.LOG_OP_TYPE_SET,
                          _('IDS_EDIT_DEPARTMENT_FAILED') % ou_info.ou_name,
                          global_info.LOG_LEVEL_INFO,
                          _('IDS_PARENT_DEPARTMENT_NOT_EXISTS'))

            progress_info.failed_num += 1
            ShareMgnt_Log('更新 部门"%s":%s 失败, 父部门不存在  %d/%d',
                          ou_info.ou_name,
                          ou_info.third_id,
                          progress_info.synced_num,
                          progress_info.total_num)
            return False

        # 检查部门是否存在
        as_ou_info = self.get_ou(ou_info.third_id)
        if not as_ou_info:
            self.ou_eacp_log(global_info.LOG_OP_TYPE_SET,
                          _('IDS_EDIT_DEPARTMENT_FAILED') % ou_info.ou_name,
                          global_info.LOG_LEVEL_INFO,
                          _('IDS_DEPARTMENT_NOT_EXISTS'))

            progress_info.failed_num += 1
            ShareMgnt_Log('更新 部门"%s:%s"失败, 部门不存在  %d/%d',
                          ou_info.ou_name,
                          ou_info.third_id,
                          progress_info.synced_num,
                          progress_info.total_num)
            return False

        # 获取唯一部门名
        ou_info.ou_name = self.get_unique_depart_name(parent_id,
                                                      ou_info,
                                                      as_ou_info.depart_id)

        # 备注检查
        if ou_info.remark is not None:
            ou_info.remark = self.depart_manage._is_remark_valid(ou_info.remark)
        else:
            ou_info.remark = as_ou_info.remark

        # code检查
        if ou_info.code is not None:
            ou_info.code = self.check_third_party_depart_code(ou_info.code, ou_info.third_id)
        else:
            ou_info.code = as_ou_info.code

        b_status = as_ou_info.status
        if ou_info.status is not None:
            b_status = ou_info.status

        status = 1
        if b_status == False:
            status = 2

        # 再次检查经过重名处理的部门名和旧的部门名是否相同，如果相同，则不需要更新部门名称
        if as_ou_info.ou_name == ou_info.ou_name and as_ou_info.priority == ou_info.priority \
            and as_ou_info.remark == ou_info.remark and as_ou_info.code == ou_info.code \
            and as_ou_info.status == ou_info.status and as_ou_info.manager_id == ou_info.manager_id:
            return True

        # 设置优先级
        if not ou_info.priority:
            ou_info.priority = as_ou_info.priority

        self.w_db.query(update_depart_sql,
                        ou_info.ou_name,
                        ou_info.dn,
                        ou_info.priority,
                        ou_info.remark,
                        status,
                        ou_info.code,
                        ou_info.third_id)

        # 发送显示名更改nsq消息
        if as_ou_info.ou_name != ou_info.ou_name:
            sql = """
            SELECT `f_department_id` FROM `t_department` WHERE `f_third_party_id` = %s
            """
            db_object = self.r_db.one(sql, ou_info.third_id)
            if isinstance(ou_info.ou_name, bytes):
                ou_info.ou_name = bytes.decode(ou_info.ou_name)
            pub_nsq_msg(TOPIC_ORG_NAME_MODIFY, {
                        "id": db_object["f_department_id"], "new_name": ou_info.ou_name, "type": "department"})

        # 发送状态变更消息
        if as_ou_info.status != ou_info.status:
            pub_nsq_msg(TOPIC_DEPART_STATUS_MODIFIED, {"ids": [as_ou_info.depart_id], "status": ou_info.status})

        # 记录操作日志
        self.ou_eacp_log(global_info.LOG_OP_TYPE_SET,
                      _('IDS_EDIT_DEPARTMENT_SUCCESS') % ou_info.ou_name)

        progress_info.updated_num += 1
        ShareMgnt_Log('更新 部门"%s":%s成功  %d/%d',
                      ou_info.ou_name,
                      ou_info.third_id,
                      progress_info.synced_num,
                      progress_info.total_num)

        return True

    def check_third_party_depart_code(self, code, third_id=None):
        """
        检查部门编码格式 以及是否唯一
        """
        # 除去前面的空格，末尾的空格
        striped_code = code.lstrip()
        striped_code = striped_code.rstrip()

        if striped_code == "":
            return striped_code

        if not is_code_string(striped_code):
            raise_exception(exp_msg=_("IDS_INVALID_DEPART_CODE"),
                            exp_num=ncTShareMgntError.NCT_INVALID_DEPART_CODE)


        select_sql = """
        select f_third_party_id from t_department where f_code = %s
        """
        result = self.r_db.one(select_sql, striped_code)
        if result and result['f_third_party_id'] != third_id:
            raise_exception(exp_msg=_("IDS_DUPLICATED_DEPART_CODE"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_DEPART_CODE)
        return striped_code

    def delete_ou(self, third_ou_id,
                  get_user_disable_status_func,
                  dept_progress_info, user_progress_info):
        """
        删除部门
        """
        dept_progress_info.synced_num += 1
        if not third_ou_id:
            dept_progress_info.failed_num += 1
            return False

        ou_info = self.get_ou(third_ou_id)
        if not ou_info:
            dept_progress_info.failed_num += 1
            ShareMgnt_Log('删除 部门失败，部门不存在：third_id=%s   %d/%d',
                          third_ou_id,
                          dept_progress_info.synced_num,
                          dept_progress_info.total_num)
            return False

        depart_id = self.get_depart_id(third_ou_id)

        # 获取部门的所有一级子部门和子用户
        sub_departs = self.get_sub_ous_by_third_id(third_ou_id)
        sub_users = self.get_sub_users_by_third_id(third_ou_id)

        # 从部门移除一级子用户
        for user in sub_users:
            disable_flag = get_user_disable_status_func(user.third_id)
            self.remove_user_from_ou(
                depart_id, user.third_id, disable_flag, user_progress_info)

        for depart in sub_departs:
            self.delete_ou(depart.third_id,
                           get_user_disable_status_func,
                           dept_progress_info,
                           user_progress_info)

        # 检查子用户和子部门是否删完
        sub_departs = self.get_sub_ous_by_third_id(third_ou_id)
        sub_users = self.get_sub_users_by_third_id(third_ou_id)

        if sub_users or sub_departs:
            return

        # 删除部门、部门关系、部门索引
        try:
            conn = ConnectorManager.get_db_conn()
            cursor = conn.cursor()
            for sql in del_depart_sql_list:
                cursor.execute(sql, (depart_id,))
            conn.commit()
            conn.close()
            # 发布部门删除nsq消息
            pub_nsq_msg(TOPIC_DEPT_DELETE, {"id": depart_id})
            # 记录操作日志
            self.ou_eacp_log(global_info.LOG_OP_TYPE_DELETE,
                          _('IDS_DELETE_DEPARTMENT_SUCCESS') % ou_info.ou_name)

            dept_progress_info.deleted_num += 1
            ShareMgnt_Log('删除 部门"%s":%s 成功   %d/%d',
                          ou_info.ou_name,
                          ou_info.third_id,
                          dept_progress_info.synced_num,
                          dept_progress_info.total_num)
            return True
        except Exception as ex:
            conn.rollback()
            self.ou_eacp_log(global_info.LOG_OP_TYPE_DELETE,
                          _('IDS_DELETE_DEPARTMENT_FAILED') % ou_info.ou_name,
                          global_info.LOG_LEVEL_INFO,
                          _('IDS_DATABASE_ERROR'))

            dept_progress_info.failed_num += 1
            ShareMgnt_Log('删除 部门"%s": %s异常ex=%s   %d/%d',
                          ou_info.ou_name,
                          ou_info.third_id,
                          str(ex),
                          dept_progress_info.synced_num,
                          dept_progress_info.total_num)
            return False

    def add_user(self, parent_id, user_info, progress_info):
        """
        添加用户
        """
        if not parent_id or not user_info:
            return

        # 检查用户是否存在
        b_exists = self.chec_user_exists(user_info.third_id)

        if not b_exists:
            return self.add_user_to_ou(parent_id, user_info, progress_info)
        else:
            return self.move_user_to_ou(parent_id, user_info, progress_info)

    def cover_user(self, user_id, user_info, old_name, progress_info):
        """
        覆盖用户信息
        """

        # 获取之前的用户信息
        sql = """
            SELECT `f_user_id`, `f_display_name`, `f_tel_number`, `f_mail_address` FROM `t_user` WHERE `f_user_id` = %s
        """
        db_obj = self.r_db.one(sql, user_id)

        # 本地用户不覆盖密码
        if user_info.type == ncTUsrmUserType.NCT_USER_TYPE_LOCAL:
            if user_info.csf_level:
                self.w_db.query(update_user_by_user_id_without_pwd_sql,
                                user_info.login_name,
                                user_info.display_name,
                                user_info.email,
                                user_info.third_id,
                                user_info.type,
                                user_info.dn,
                                user_info.server_type,
                                user_info.priority,
                                user_info.idcard_number,
                                user_info.tel_number,
                                user_info.csf_level,
                                user_info.position,
                                user_info.code,
                                user_info.csf_level2,
                                user_id)
            else:
                self.w_db.query(update_user_by_user_id_without_pwd_csf_sql,
                                user_info.login_name,
                                user_info.display_name,
                                user_info.email,
                                user_info.third_id,
                                user_info.type,
                                user_info.dn,
                                user_info.server_type,
                                user_info.priority,
                                user_info.idcard_number,
                                user_info.tel_number,
                                user_info.position,
                                user_info.code,
                                user_info.csf_level2,
                                user_id)
        # 如果是第三方用户，需要更新用户密级
        elif user_info.type == ncTUsrmUserType.NCT_USER_TYPE_THIRD and user_info.csf_level:
            self.w_db.query(update_user_by_user_id_contain_csf_sql,
                                user_info.login_name,
                                user_info.display_name,
                                user_info.password,
                                user_info.email,
                                user_info.third_id,
                                user_info.type,
                                user_info.dn,
                                user_info.server_type,
                                user_info.priority,
                                user_info.idcard_number,
                                user_info.tel_number,
                                user_info.csf_level,
                                user_info.position,
                                user_info.code,
                                user_info.csf_level2,
                                user_id)
        else:
            self.w_db.query(update_user_by_user_id_sql,
                            user_info.login_name,
                            user_info.display_name,
                            user_info.password,
                            user_info.email,
                            user_info.third_id,
                            user_info.type,
                            user_info.dn,
                            user_info.server_type,
                            user_info.priority,
                            user_info.idcard_number,
                            user_info.tel_number,
                            user_info.position,
                            user_info.code,
                            user_info.csf_level2,
                            user_id)

        self.ou_eacp_log(global_info.LOG_OP_TYPE_SET,
                      _('IDS_COVER_USER_SUCCESS') % (user_info.login_name, user_info.display_name))

        if old_name != user_info.display_name:
            # 发送用户显示名更新nsq消息
            pub_nsq_msg(TOPIC_ORG_NAME_MODIFY, {
                        "id": user_id, "new_name": user_info.display_name, "type": "user"})

        user_modify_info = {}
        if db_obj and db_obj['f_tel_number'] != user_info.tel_number:
            user_modify_info["new_telephone"] = user_info.tel_number
            if user_info.tel_number is None:
                user_modify_info["new_telephone"] = ""
        if db_obj and db_obj['f_mail_address'] != user_info.email:
            user_modify_info["new_email"] = user_info.email
            if user_info.email is None:
                user_modify_info["new_email"] = ""

        if len(user_modify_info) > 0:
            # 发送用户信息更新nsq消息
            user_modify_info["user_id"] = user_id
            pub_nsq_msg(TOPIC_USER_MODIFIED, user_modify_info)

        progress_info.updated_num += 1
        ShareMgnt_Log('覆盖 用户"%s" 成功: third_id=%s   %d/%d',
                      user_info.login_name,
                      user_info.third_id,
                      progress_info.synced_num,
                      progress_info.total_num)

    def add_user_to_ou(self, parent_id, user_info, progress_info):
        """
        增加用户到组织（用户不存在）
        """
        progress_info.synced_num += 1

        # 检查用户参数
        self.check_user_info(user_info)

        # 检查父部门,不存在,则默认添加到未分配用户组
        org_id = NCT_UNDISTRIBUTE_USER_GROUP
        b_exists = self.depart_manage.check_depart_exists(
            parent_id, True, False)
        if not b_exists:
            parent_id = NCT_UNDISTRIBUTE_USER_GROUP
        else:
            org_id = self.depart_manage.get_ou_by_depart_id(parent_id)

        ou_info = self.get_depart_info_by_id(parent_id)

        if user_info.code is None:
            user_info.code = ''

        if user_info.position is None:
            user_info.position = ''

        # 登录名重复，直接覆盖
        as_user = self.user_manage.get_user_by_loginname(user_info.login_name)
        if as_user and as_user.user.loginName.rstrip().lower() == user_info.login_name.rstrip().lower():
            user_info.display_name = self.get_unique_display_name(user_info.display_name,
                                                                  as_user.id)

            # 检查排序权重
            if not user_info.priority:
                user_info.priority = as_user.user.priority

            if not user_info.csf_level2:
                user_info.csf_level2 = as_user.user.csfLevel2

            self.cover_user(as_user.id, user_info,
                            as_user.user.displayName, progress_info)
            self.move_user_to_ou(parent_id, user_info, progress_info)
            user_uuid = as_user.id
        else:
            # 如果用户的oss_id存在，则取用户的，否则取父部门的对象存储id
            if user_info.oss_id:
                oss_id = user_info.oss_id
            else:
                oss_id = self.get_depart_oss_id(parent_id)

            # 生成uuid
            user_uuid = str(uuid.uuid1())

            # 用户状态先取自身启用禁用状态，没有再取默认状态
            if user_info.status is not None:
                user_status = ncTUsrmUserStatus.NCT_STATUS_ENABLE if user_info.status else ncTUsrmUserStatus.NCT_STATUS_DISABLE
            else:
                user_status = self.user_create_status

            # 获取唯一显示名
            user_info.display_name = self.get_unique_display_name(
                user_info.display_name)

            # 新建用户且启用时，检查授权数
            if user_status == ncTUsrmUserStatus.NCT_STATUS_ENABLE:
                if self.user_manage.is_user_num_overflow():
                    global_info.IMPORT_DISABLE_USER_NUM += 1
                    self.ou_eacp_log(global_info.LOG_OP_TYPE_SET,
                                  _("IDS_CREATE_DISABLE_USER_MSG") % (
                                      user_info.display_name, user_info.login_name),
                                  global_info.LOG_LEVEL_WARN,
                                  _("user num overflow"))
                    user_status = ncTUsrmUserStatus.NCT_STATUS_DISABLE

            # 本地用户，如未设置密码，则默认为初始密码
            if user_info.type == ncTUsrmUserType.NCT_USER_TYPE_LOCAL:
                if not user_info.password:
                    user_info.password = self.user_manage.user_default_password.md5_pwd

            # 检查排序权重
            if not user_info.priority:
                user_info.priority = INIT_PRIORITY

            user_info.status = user_status
            user_info.oss_id = oss_id

            # 获取用户最小密级
            min_csf_level = self.config_manage.get_min_csf_level()
            #获取用户最小密级2
            min_csf_level2 = self.config_manage.get_min_csf_level2()

            conn = ConnectorManager.get_db_conn()
            try:
                self.__add_user_to_db(
                    conn, user_uuid, user_info, parent_id, org_id, ou_info.depart_path, min_csf_level, min_csf_level2)

                # 记录操作日志
                self.ou_eacp_log(global_info.LOG_OP_TYPE_CREATE,
                              _('IDS_ADD_USER_TO_DEPARTMENT_SUCCESS') % (user_info.login_name, user_info.display_name, ou_info.ou_name))

                progress_info.added_num += 1
                ShareMgnt_Log('添加 用户"%s"到部门"%s":%s 成功   %d/%d',
                              user_info.login_name,
                              ou_info.ou_name,
                              ou_info.third_id,
                              progress_info.synced_num,
                              progress_info.total_num)
            except Exception as ex:
                self.ou_eacp_log(global_info.LOG_OP_TYPE_CREATE,
                              _('IDS_ADD_USER_TO_DEPARTMENT_FAILED') % (
                                  user_info.login_name, user_info.display_name, ou_info.ou_name),
                              global_info.LOG_LEVEL_INFO,
                              _('IDS_DATABASE_ERROR'))

                progress_info.failed_num += 1
                ShareMgnt_Log('添加 用户"%s"到部门"%s":%s 异常: %s   %d/%d',
                              user_info.login_name,
                              ou_info.ou_name,
                              ou_info.third_id,
                              str(ex),
                              progress_info.synced_num,
                              progress_info.total_num)
            finally:
                conn.close()
        return user_uuid

    def __add_user_to_db(self, conn, user_uuid, user_info, parent_id, org_id, path, min_csf_level, min_csf_level2):
        now = BusinessDate.now().strftime("%Y-%m-%d %H:%M:%S")
        with safe_cursor(conn) as cursor:
            if user_info.type == ncTUsrmUserType.NCT_USER_TYPE_DOMAIN:
                cursor.execute(insert_domain_user_sql,
                               (user_uuid, user_info.login_name,
                                user_info.display_name,
                                user_info.password,
                                user_info.email,
                                user_info.type,
                                user_info.status,
                                user_info.third_id,
                                user_info.dn,
                                user_info.server_type,
                                user_info.priority,
                                user_info.oss_id,
                                user_info.third_attr,
                                self.expire_time,
                                user_info.idcard_number,
                                user_info.tel_number,
                                user_info.csf_level or min_csf_level,
                                user_info.position,
                                user_info.code,
                                user_info.csf_level2 or min_csf_level2))
            else:
                cursor.execute(insert_user_sql,
                               (user_uuid, user_info.login_name,
                                user_info.display_name,
                                user_info.password,
                                user_info.email,
                                user_info.type,
                                user_info.status,
                                user_info.third_id,
                                user_info.dn,
                                user_info.server_type,
                                now,
                                now,
                                user_info.priority,
                                user_info.oss_id,
                                user_info.third_attr,
                                self.expire_time,
                                user_info.idcard_number,
                                user_info.tel_number,
                                user_info.csf_level or self.user_csf_level,
                                user_info.position,
                                user_info.code,
                                user_info.csf_level2 or self.user_csf_level2))

            # 用户部门关系
            cursor.execute(insert_user_depart_sql,
                           (user_uuid, parent_id, path))

            # 用户组织关系
            cursor.execute(insert_user_ou_sql, (user_uuid, org_id))

            # 用户联系人组
            cursor.execute(insert_group_sql, (str(uuid.uuid1()),
                           user_uuid, _("IDS_TMP_PERSON_GROUP")))

        # 添加用户自定义属性
        custom_attr, document_attr = {}, {}
        if user_info.doc_status is not None:
            document_attr["user_doc_lib_create_status"] = user_info.doc_status
        if user_info.space_size is not None:
            document_attr["space_quote"] = user_info.space_size
        if document_attr:
            custom_attr["document"] = document_attr
            self.user_manage.patch_user_custom_attr(user_uuid, custom_attr)

        pub_nsq_msg(TOPIC_USER_CREATE, {
                    "id": user_uuid, "name": user_info.display_name})
        pub_nsq_msg(TOPIC_DEPARTMENT_USER_ADD,{"id": user_uuid, "dept_paths": [path]})

    def move_user_to_ou(self, depart_id, user_info, progress_info):
        """
        增加用户到组织(用户已存在)
        """
        progress_info.synced_num += 1

        if not depart_id or not user_info:
            return

        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()

        # 检查用户是否存在
        as_user = self.get_user(user_info.third_id)
        if not as_user:
            progress_info.failed_num += 1
            ShareMgnt_Log('添加 用户"%s" 失败，用户不存在: third_id=%s   %d/%d',
                          user_info.login_name,
                          user_info.third_id,
                          progress_info.synced_num,
                          progress_info.total_num)
            return

        depart_path = self.depart_manage.get_department_path_by_dep_id(depart_id)
        if depart_path == '':
            ShareMgnt_Log(
                'departmentinfo illegal depart_path is None, depart_id:%s', depart_id)
            raise_exception(exp_msg=_("departmentinfo illegal"),
                            exp_num=ncTShareMgntError.
                            NCT_ORG_OR_DEPART_NOT_EXIST)
        # 判断用户是否在当前部门下
        cursor.execute(check_user_depart_sql,
                       (as_user.user_id, depart_path))
        result = cursor.fetchall()
        if not result:
            ou_info = self.get_depart_info_by_id(depart_id)
            org_id = self.depart_manage.get_ou_by_depart_id(depart_id)

            # 获取用户源部门的组织管理员id
            src_manager_ids = self.user_manage.get_parent_dept_responsbile_person(
                as_user.user_id)

            # 判断用户是否在未分配组
            b_belong_undistribute = self.check_is_belong_undistribute_user_group(as_user.user_id)

            try:
                # 插入用户部门关系
                cursor.execute(insert_user_depart_sql,
                               (as_user.user_id, depart_id, ou_info.depart_path))

                # 删除用户所属未分配关系
                cursor.execute(delete_user_depart_sql,
                               (as_user.user_id,
                                NCT_UNDISTRIBUTE_USER_GROUP))

                # 检查是否需要添加用户组织索引
                cursor.execute(
                    check_user_ou_sql, (as_user.user_id, org_id))
                result = cursor.fetchall()
                if not result:
                    cursor.execute(insert_user_ou_sql,
                                   (as_user.user_id, org_id))

                conn.commit()
                conn.close()

                # 如果是未分配组移动出来，用户状态设为启用
                if b_belong_undistribute and self.user_create_status == 0:
                    self.user_manage.set_user_status(as_user.user_id, True)
                pub_nsq_msg(TOPIC_DEPARTMENT_USER_ADD,{"id": as_user.user_id, "dept_paths": [ou_info.depart_path]})

                # 记录操作日志
                self.ou_eacp_log(global_info.LOG_OP_TYPE_MOVE,
                              _('IDS_MOVE_USER_TO_DEPARTMENT_SUCCESS') % (user_info.login_name, ou_info.ou_name))

                progress_info.moved_num += 1
                ShareMgnt_Log('添加 用户"%s"到部门"%s":"%s" 成功   %d/%d',
                              user_info.login_name,
                              ou_info.ou_name,
                              ou_info.third_id,
                              progress_info.synced_num,
                              progress_info.total_num)
                return as_user.user_id

            except Exception as ex:
                conn.rollback()

                self.ou_eacp_log(global_info.LOG_OP_TYPE_MOVE,
                              _('IDS_MOVE_USER_TO_DEPARTMENT_FAILED') % (
                                  user_info.login_name, ou_info.ou_name),
                              global_info.LOG_LEVEL_INFO,
                              _('IDS_DATABASE_ERROR'))

                progress_info.failed_num += 1
                ShareMgnt_Log('添加 用户"%s"到部门"%s":%s异常: %s   %d/%d',
                              user_info.login_name,
                              ou_info.ou_name,
                              ou_info.third_id,
                              str(ex),
                              progress_info.synced_num,
                              progress_info.total_num)

    def update_user(self, latest_user_info, user_info, progress_info):
        """
        更新用户
        """
        progress_info.synced_num += 1
        if not user_info:
            progress_info.failed_num += 1
            return False

        self.check_user_info(user_info, False)

        # 获取唯一显示名
        user_info.display_name = self.get_unique_display_name(user_info.display_name,
                                                              latest_user_info.user_id)

        # 这里再把经过重名处理的信息和旧的用户信息比较下，是否需要更新
        b_same = self.compare_user_info(latest_user_info, user_info)
        if b_same:
            return False

        # 设置优先级，如果优先级不存在或为默认值时使用旧优先级，防止覆盖手动修改的优先级
        if not user_info.priority or user_info.priority == 999:
            user_info.priority = latest_user_info.priority

        if user_info.code is None:
            user_info.code = latest_user_info.code

        if user_info.position is None:
            user_info.position = latest_user_info.position

        if user_info.csf_level2 is None:
            user_info.csf_level2 = latest_user_info.csf_level2

        # 登录名已存在，则直接覆盖
        result = self.r_db.one(check_loginname_excluede_third_id_sql,
                               user_info.login_name, user_info.third_id)
        if result:
            self.cover_user(result['f_user_id'], user_info,
                            result['f_display_name'], progress_info)
        else:
            try:
                # 获取之前的用户信息
                sql = """
                    SELECT `f_user_id`, `f_display_name`, `f_tel_number`, `f_mail_address` FROM `t_user` WHERE `f_third_party_id` = %s
                """
                db_obj = self.r_db.one(sql, user_info.third_id)

                # 如果是本地用户，不更新密码
                if user_info.type == ncTUsrmUserType.NCT_USER_TYPE_LOCAL:
                    if user_info.csf_level:
                        self.w_db.query(update_user_by_third_id_without_pwd_sql,
                                        user_info.login_name,
                                        user_info.display_name,
                                        user_info.email,
                                        user_info.dn,
                                        user_info.server_type,
                                        user_info.type,
                                        user_info.priority,
                                        user_info.third_attr,
                                        user_info.idcard_number,
                                        user_info.tel_number,
                                        user_info.csf_level,
                                        user_info.position,
                                        user_info.code,
                                        user_info.csf_level2,
                                        user_info.third_id)
                    else:
                        self.w_db.query(update_user_by_third_id_without_pwd_csf_sql,
                                        user_info.login_name,
                                        user_info.display_name,
                                        user_info.email,
                                        user_info.dn,
                                        user_info.server_type,
                                        user_info.type,
                                        user_info.priority,
                                        user_info.third_attr,
                                        user_info.idcard_number,
                                        user_info.tel_number,
                                        user_info.position,
                                        user_info.code,
                                        user_info.csf_level2,
                                        user_info.third_id)
                # 如果是第三方用户，需要更新用户密级
                elif user_info.type == ncTUsrmUserType.NCT_USER_TYPE_THIRD and user_info.csf_level:
                    self.w_db.query(update_user_by_third_id_contain_csf_sql,
                                        user_info.login_name,
                                        user_info.display_name,
                                        user_info.password,
                                        user_info.email,
                                        user_info.dn,
                                        user_info.server_type,
                                        user_info.type,
                                        user_info.priority,
                                        user_info.third_attr,
                                        user_info.idcard_number,
                                        user_info.tel_number,
                                        user_info.csf_level,
                                        user_info.position,
                                        user_info.code,
                                        user_info.csf_level2,
                                        user_info.third_id)
                else:
                    self.w_db.query(update_user_by_third_id_sql,
                                    user_info.login_name,
                                    user_info.display_name,
                                    user_info.password,
                                    user_info.email,
                                    user_info.dn,
                                    user_info.server_type,
                                    user_info.type,
                                    user_info.priority,
                                    user_info.third_attr,
                                    user_info.idcard_number,
                                    user_info.tel_number,
                                    user_info.position,
                                    user_info.code,
                                    user_info.csf_level2,
                                    user_info.third_id)

                if db_obj and db_obj['f_display_name'] != user_info.display_name:
                    # 发送显示名变更NSQ消息
                    pub_nsq_msg(TOPIC_ORG_NAME_MODIFY, {
                                "id": db_obj["f_user_id"], "new_name": user_info.display_name, "type": "user"})

                user_modify_info = {}
                if db_obj and db_obj['f_tel_number'] != user_info.tel_number:
                    user_modify_info["new_telephone"] = user_info.tel_number
                    if user_info.tel_number is None:
                        user_modify_info["new_telephone"] = ""
                if db_obj and db_obj['f_mail_address'] != user_info.email:
                    user_modify_info["new_email"] = user_info.email
                    if user_info.email is None:
                        user_modify_info["new_email"] = ""
                if len(user_modify_info) > 0:
                    user_modify_info["user_id"] = db_obj["f_user_id"]
                    pub_nsq_msg(TOPIC_USER_MODIFIED, user_modify_info)

                # 记录操作日志
                self.ou_eacp_log(global_info.LOG_OP_TYPE_SET,
                              _('IDS_EDIT_USER_SUCCESS') % (user_info.login_name, user_info.display_name))

                progress_info.updated_num += 1
                ShareMgnt_Log('修改 用户"%s"成功: third_id=%s   %d/%d',
                              user_info.login_name,
                              user_info.third_id,
                              progress_info.synced_num,
                              progress_info.total_num)
                return True

            except Exception as ex:
                self.ou_eacp_log(global_info.LOG_OP_TYPE_SET,
                              _('IDS_EDIT_USER_FAILED') % (
                                  user_info.login_name, user_info.display_name),
                              global_info.LOG_LEVEL_INFO,
                              _('IDS_DATABASE_ERROR'))

                progress_info.failed_num += 1
                ShareMgnt_Log('修改 用户"%s" 异常: third_id=%s, ex=%s   %d/%d',
                              user_info.login_name,
                              user_info.third_id,
                              str(ex),
                              progress_info.synced_num,
                              progress_info.total_num)
                return False

    def disable_user(self, user_info):
        """
        禁用用户
        """
        if user_info.status == ncTUsrmUserStatus.NCT_STATUS_ENABLE:
            self.user_manage.set_user_status(user_info.user_id, False)
            ShareMgnt_Log('禁用用户: user_third_id=%s(%s)',
                          user_info.user_id, user_info.display_name)

    def enable_user(self, user_info):
        """
        启用用户
        """
        if user_info.status == ncTUsrmUserStatus.NCT_STATUS_DISABLE:
            self.user_manage.set_user_status(user_info.user_id, True)
            ShareMgnt_Log('启用用户: user_third_id=%s(%s)',
                          user_info.user_id, user_info.display_name)

    def delete_user(self, third_user_id, progress_info):
        """
        删除用户
        """
        progress_info.synced_num += 1
        if not third_user_id:
            progress_info.failed_num += 1
            return
        # 检查用户是否存在
        user_info = self.get_user(third_user_id)
        if not user_info:
            progress_info.failed_num += 1
            ShareMgnt_Log('删除 用户 失败, 用户不存在: user_third_id=%s   %d/%d',
                          third_user_id,
                          progress_info.synced_num,
                          progress_info.total_num)
            return False

        try:
            # 调用uesrmanagement删除用户方法删除用户
            self.user_manage.delete_user(user_info.user_id)
            # 记录操作日志
            self.ou_eacp_log(global_info.LOG_OP_TYPE_DELETE,
                          _('IDS_DELETE_UESR_SUCCESS') % (
                              user_info.login_name, user_info.display_name),
                          global_info.LOG_LEVEL_WARN)

            progress_info.deleted_num += 1
            ShareMgnt_Log('删除 用户"%s" 成功: third_id=%s   %d/%d',
                          user_info.login_name,
                          third_user_id,
                          progress_info.synced_num,
                          progress_info.total_num)
            return True
        except Exception as ex:
            self.ou_eacp_log(global_info.LOG_OP_TYPE_DELETE,
                          _('IDS_DELETE_UESR_FAILED') % (
                              user_info.login_name, user_info.display_name),
                          global_info.LOG_LEVEL_INFO,
                          _('IDS_DATABASE_ERROR'))

            progress_info.failed_num += 1
            ShareMgnt_Log('删除 用户"%s" 异常: third_id=%s, ex=%s   %d/%d',
                          user_info.login_name,
                          third_user_id,
                          str(ex),
                          progress_info.synced_num,
                          progress_info.total_num)
            return False

    def remove_user_from_ou(self, depart_id, third_user_id, disable_flag, progress_info):
        """
        从部门移除用户
        """
        # 检查参数是否为空
        if not depart_id or not third_user_id:
            progress_info.failed_num += 1
            ShareMgnt_Log('从部门移除用户失败,部门或者用户id不存在:depart_id=%s, user_third_id=%s   %d/%d',
                          depart_id,
                          third_user_id,
                          progress_info.synced_num,
                          progress_info.total_num)
            return False

        # 检查部门
        ou_info = self.get_depart_info_by_id(depart_id)
        if not ou_info:
            # 部门不存在
            progress_info.failed_num += 1
            ShareMgnt_Log('从部门移除用户失败,部门不存在:depart_id=%s, user_third_id=%s   %d/%d',
                          depart_id,
                          third_user_id,
                          progress_info.synced_num,
                          progress_info.total_num)
            return False

        # 检查用户
        user_info = self.get_user(third_user_id)
        if not user_info:
            progress_info.failed_num += 1
            ShareMgnt_Log('从部门"%s":%s移除用户失败,用户不存在:user_third_id=%s   %d/%d',
                          ou_info.ou_name,
                          ou_info.third_id,
                          third_user_id,
                          progress_info.synced_num,
                          progress_info.total_num)
            return False

        # 如果用户属于其它部门，则不会移动到未分配
        b_move_undistribue = not self.check_user_belong_other_depart(
            user_info.user_id, depart_id)

        # 如果用户属于本组织中的其他部门, 则不删除用户索引关系
        b_del_ou = not self.check_user_belong_other_depart_same_org(
            user_info.user_id, depart_id)

        try:
            depart_path = self.depart_manage.get_department_path_by_dep_id(depart_id)
            if depart_path == '':
                ShareMgnt_Log(
                    'departmentinfo illegal depart_path is None, depart_id:%s', depart_id)
                raise_exception(exp_msg=_("departmentinfo illegal"),
                                exp_num=ncTShareMgntError.
                                NCT_ORG_OR_DEPART_NOT_EXIST)
            conn = ConnectorManager.get_db_conn()
            cursor = conn.cursor()

            # 删除用户部门关系
            cursor.execute(del_relation_sql,
                           (user_info.user_id, depart_path))

            # 删除用户组织索引
            if b_del_ou:
                cursor.execute(del_ou_sql, (user_info.user_id,))

            # 移动用户到未分配
            b_send_nsq_msg = False
            if b_move_undistribue:
                # 如果有禁用标识，则禁用用户
                if disable_flag:
                    b_send_nsq_msg = True
                    cursor.execute(set_user_status_sql, (1, user_info.user_id))

                cursor.execute(insert_relation_sql,
                               (user_info.user_id,
                                NCT_UNDISTRIBUTE_USER_GROUP, NCT_UNDISTRIBUTE_USER_GROUP))

            conn.commit()
            conn.close()

            # 记录操作日志
            self.ou_eacp_log(global_info.LOG_OP_TYPE_REMOVE,
                          _('IDS_REMOVE_USER_FROM_DEPARTMENT_SUCCESS') % (ou_info.ou_name, user_info.login_name, user_info.display_name))
            pub_nsq_msg(TOPIC_DEPARTMENT_USER_REMOVE, {
                            "id": user_info.user_id, "dept_paths": [depart_path]})
            if b_send_nsq_msg:
                pub_nsq_msg(TOPIC_USER_STATUS_CHANGED, {
                            "user_id": user_info.user_id, "status": False})

            progress_info.moved_num += 1
            ShareMgnt_Log('从部门"%s":%s 移除用户"%s(%s)"成功   %d/%d',
                          ou_info.ou_name,
                          ou_info.third_id,
                          user_info.login_name,
                          user_info.display_name,
                          progress_info.synced_num,
                          progress_info.total_num)

            return True
        except Exception as ex:
            conn.rollback()

            self.ou_eacp_log(global_info.LOG_OP_TYPE_REMOVE,
                          _('IDS_REMOVE_USER_FROM_DEPARTMENT_FAILED') % (
                              ou_info.ou_name, user_info.login_name, user_info.display_name),
                          global_info.LOG_LEVEL_INFO,
                          _('IDS_DATABASE_ERROR'))

            progress_info.failed_num += 1
            ShareMgnt_Log('从部门"%s":%s 移除用户"%s"异常: %s   %d/%d',
                          ou_info.ou_name,
                          ou_info.third_id,
                          user_info.login_name,
                          str(ex),
                          progress_info.synced_num,
                          progress_info.total_num)
            return False

    def sync_update_responsible_person_space(self):

        # 同步后更新组织管理员信息
        # 获取所有组织管理员id
        person_ids = []
        responsible_person_ids = self.r_db.all(select_responsible_person_id)
        # 根据组织管理员id获取所管理的所有用户
        if responsible_person_ids:
            for responsible_person_id in responsible_person_ids:
                managered_user_ids = self.depart_manage.get_user_ids_by_admin_id(
                    responsible_person_id["f_user_id"])

    def get_undistributed_users(self):
        # 获取未分配的用户（只包括第三方的）
        return self.get_sub_users_by_depart_id(NCT_UNDISTRIBUTE_USER_GROUP)

    def is_contain_moved_ou(self, third_ou_id, moved_depart_ids):
        # 检查third_ou_id对应的部门是否能被删除
        # moved_depart_ids表示移动过的部门
        # 如果third_ou_id所有的子部门和moved_depart_ids有交集，则不允许删除
        depart_id = self.get_depart_id(third_ou_id)
        all_depart_ids = self.depart_manage.get_all_departids(depart_id)

        ret_list = set(all_depart_ids) & set(moved_depart_ids)

        if len(ret_list):
            return True
        else:
            return False

    def check_is_belong_undistribute_user_group(self, user_id):
        """
        检查用户是否属于未分配组
        """
        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()

        cursor.execute(check_user_depart_sql,
                       (user_id, NCT_UNDISTRIBUTE_USER_GROUP))
        result = cursor.fetchall()

        return True if result else False

    def get_oss_info(self, oss_id):
        """
        获取对象存储信息
        """
        oss_info = get_oss_info(oss_info)
        if oss_info:
            return oss_info

    def update_manager(self, user_manager_infos, depart_manager_infos):
        """
        更新用户上级和部门负责人
        """
        # 获取所有的用户信息
        sql = """
            SELECT f_user_id, f_manager_id, f_third_party_id FROM t_user
        """
        user_infos = self.r_db.all(sql)

        # 获取所有的部门信息
        sql = """
            SELECT f_department_id, f_manager_id, f_third_party_id FROM t_department
        """
        depart_infos = self.r_db.all(sql)

        # 处理用户负责人
        user_third_id_id_map = {}
        user_id_third_id_map = {}
        user_id_manager_id_map = {}
        for user_info in user_infos:
            user_third_id_id_map[user_info["f_third_party_id"]] = user_info["f_user_id"]
            user_id_third_id_map[user_info["f_user_id"]] = user_info["f_third_party_id"]
            user_id_manager_id_map[user_info["f_user_id"]] = user_info["f_manager_id"]

        # 查看哪些用户新增了负责人
        user_update_manager_map = {}

        # 判断哪些用户变更的负责人
        for key, value in user_manager_infos.items():
            # 判断第三方用户是否在as存在
            if key not in user_third_id_id_map:
                ShareMgnt_Log("用户负责人新增错误，用户不存在: user_third_id=%s, manager_third_id=%s", key, value)
                continue

            # 判断负责人是否在as存在
            if value not in user_third_id_id_map and value != '':
                ShareMgnt_Log("用户负责人新增错误，负责人不存在: user_third_id=%s, manager_third_id=%s", key, value)
                continue

            dst_user_id = user_third_id_id_map[key]
            dst_manager_id = ''
            if value != '':
                dst_manager_id = user_third_id_id_map[value]
            if user_id_manager_id_map[dst_user_id] != dst_manager_id:
                # 记录变更负责人的用户
                user_update_manager_map[dst_user_id] = dst_manager_id

        # 处理部门负责人
        depart_third_id_id_map = {}
        depart_id_third_id_map = {}
        depart_id_manager_id_map = {}
        for depart_info in depart_infos:
            depart_third_id_id_map[depart_info["f_third_party_id"]] = depart_info["f_department_id"]
            depart_id_manager_id_map[depart_info["f_department_id"]] = depart_info["f_manager_id"]
            depart_id_third_id_map[depart_info["f_department_id"]] = depart_info["f_third_party_id"]

        depart_update_manager_map = {}
        # 判断哪些部门变更了负责人
        for key, value in depart_manager_infos.items():
            # 判断第三方部门是否在as存在
            if key not in depart_third_id_id_map:
                ShareMgnt_Log("部门负责人新增错误，部门不存在: depart_third_id=%s, manager_third_id=%s", key, value)
                continue

            # 判断负责人是否在as存在
            if value not in user_third_id_id_map and value != '':
                ShareMgnt_Log("部门负责人新增错误，负责人不存在: depart_third_id=%s, manager_third_id=%s", key, value)
                continue

            dst_depart_id = depart_third_id_id_map[key]
            dst_manager_id = ''
            if value != '':
                dst_manager_id = user_third_id_id_map[value]
            if depart_id_manager_id_map[dst_depart_id] != dst_manager_id:
                # 记录变更负责人的部门
                depart_update_manager_map[dst_depart_id] = dst_manager_id

        update_user_manager_sql = """
            UPDATE `t_user` SET `f_manager_id` = %s WHERE `f_user_id` = %s
        """
        update_depart_manager_sql = """
            UPDATE `t_department` SET `f_manager_id` = %s WHERE `f_department_id` = %s
        """

        # 更新用户负责人
        for user_id, manager_id in user_update_manager_map.items():
            self.w_db.query(update_user_manager_sql, manager_id, user_id)

        # 更新部门负责人
        for depart_id, manager_id in depart_update_manager_map.items():
            self.w_db.query(update_depart_manager_sql, manager_id, depart_id)
            pub_nsq_msg(TOPIC_DEPART_MANAGER_MODIFIED, {"depart_id": depart_id, "original_manager_id": depart_id_manager_id_map[depart_id], "current_manager_id": manager_id})









