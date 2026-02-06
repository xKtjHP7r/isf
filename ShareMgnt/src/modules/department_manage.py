#!/usr/bin/python3
# -*- coding:utf-8 -*-
"""This is spcae manage class"""
import uuid
import re
from src.common.global_info import (DEFAULT_DEPART_PRIORITY)
from collections import deque
from eisoo.tclients import TClient
from src.common.db.connector import DBConnector, ConnectorManager
from src.common.db.db_manager import get_db_name
from src.common.http import pub_nsq_msg
from src.modules.user_manage import UserManage, TOPIC_USER_MOVE, TOPIC_DEPARTMENT_USER_REMOVE, TOPIC_DEPARTMENT_USER_ADD
from src.modules.handle_task_thread import (CallableTask, HandleTaskThread)
from src.common.lib import (raise_exception,
                            check_is_uuid,
                            check_start_limit,
                            escape_key,
                            generate_group_str,
                            generate_search_order_sql,
                            check_email,
                            is_valid_string,
                            is_code_string,
                            remove_duplicate_item_from_list)
from src.modules.ossgateway import get_oss_info
from ShareMgnt.ttypes import (ncTUsrmDepartmentInfo,
                              ncTUsrmOrganizationInfo,
                              ncTUsrmDepartType,
                              ncTRootOrgInfo,
                              ncTDepartmentInfo,
                              ncTSearchUserInfo,
                              ncTLocateInfo,
                              ncTUsrmOSSInfo,
                              ncTShareMgntError,
                              ncTManageDeptInfo)
from ShareMgnt.constants import (NCT_USER_ADMIN,
                                 NCT_USER_AUDIT,
                                 NCT_USER_SYSTEM,
                                 NCT_USER_SECURIT,
                                 NCT_UNDISTRIBUTE_USER_GROUP,
                                 NCT_ALL_USER_GROUP,
                                 NCT_SYSTEM_ROLE_SUPPER,
                                 NCT_SYSTEM_ROLE_ADMIN,
                                 NCT_SYSTEM_ROLE_SECURIT,
                                 NCT_SYSTEM_ROLE_AUDIT,
                                 NCT_SYSTEM_ROLE_ORG_MANAGER,
                                 NCT_SYSTEM_ROLE_ORG_AUDIT)
from EFAST.ttypes import ncTGetPageDocParam

TOPIC_DEPT_DELETE = "core.dept.delete"
TOPIC_ORG_NAME_MODIFY = "core.org.name.modify"
TOPIC_DEPT_CREATED = "core.user_management.dept.created"
TOPIC_DEPT_MOVE = "user_management.dept.moved"
TOPIC_DEPART_STATUS_MODIFIED = "user_management.dept.status.modified"
TOPIC_DEPART_MANAGER_MODIFIED = "user_management.dept.manager.modified"

class DepartmentManage(DBConnector):
    """
    user manage
    """
    def __init__(self):
        super(DepartmentManage, self).__init__()
        self.user_manage = UserManage()
        self.handle_task_thread = HandleTaskThread()
        self.initAdminPwd = "e10adc3949ba59abbe56e057f20f883e"
        self.initSha2AdminPwd = "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92"

##################################################################################################
#                                            公用函数
##################################################################################################
    def check_organ_exists(self, organ_id, raise_ex=True):
        """
        检查组织结构是否存在
        Args:
            organ_id: string 组织结构UUID
            raise_ex: bool 是否抛出异常
        Return:
            bool 检查结束
        Raise:
            组织不存在
        """
        def raise_ex_func():
            """
            根据参数判断抛出异常
            """
            if raise_ex:
                raise_exception(exp_msg=_("organ not exists"),
                                exp_num=ncTShareMgntError.
                                NCT_ORGNIZATION_NOT_EXIST)

        if not check_is_uuid(organ_id):
            raise_ex_func()
            return False

        sql = """
        SELECT COUNT(*) AS cnt FROM `t_department`
        WHERE `f_department_id` = %s AND `f_is_enterprise` = 1
        """
        count = self.r_db.one(sql, organ_id)['cnt']
        if count != 1:
            raise_ex_func()
            return False
        return True

    def check_depart_exists(self, depart_id,
                            include_organ=False, raise_ex=True):
        """
        检查部门组织结构是否存在
        Args:
            depart_id: string 部门UUID
            include_organ: bool False 同时检查组织是否存在
            raise_ex: bool 是否抛出异常
        Return:
            bool 检查结束
        Raise:
            部门、组织不存在
        """
        def raise_ex_func():
            """
            根据参数判断抛出异常
            """
            if raise_ex:
                if include_organ:
                    raise_exception(exp_msg=_("depart or organ not exists"),
                                    exp_num=ncTShareMgntError.
                                    NCT_ORG_OR_DEPART_NOT_EXIST)
                else:
                    raise_exception(exp_msg=_("depart not exists"),
                                    exp_num=ncTShareMgntError.
                                    NCT_DEPARTMENT_NOT_EXIST)

        # 部门是未分配、全部用户不判断
        if (depart_id == NCT_UNDISTRIBUTE_USER_GROUP or depart_id == NCT_ALL_USER_GROUP):
            return True

        where = ""
        if not include_organ:
            where = " AND `f_is_enterprise` <> 1"

        sql = """
        SELECT COUNT(*) AS cnt FROM `t_department`
        WHERE `f_department_id` = %s {0}
        """.format(where)
        count = self.r_db.one(sql, depart_id)['cnt']
        if count != 1:
            raise_ex_func()
            return False
        return True

    def check_name_in_sub_departs(self, parent_id, name, exclude_dept_id=None):
        """
        检查父部门下的子部门是否存在同名部门
        Args:
            parent_id: string 父部门ID
            name: string 要检查的部门名
        Raise:
            部门已存在
        """

        parent_path = self.get_department_path_by_dep_id(parent_id)
        field = "/____________________________________"

        sql = """
        SELECT  COUNT(*) AS cnt FROM t_department
        WHERE f_path like %s
            AND f_name = %s
        """
        if exclude_dept_id:
            sql += "AND f_department_id != %s"
            count = self.r_db.one(sql, parent_path + field, name, exclude_dept_id)["cnt"]
        else:
            count = self.r_db.one(sql, parent_path + field, name)["cnt"]

        if count != 0:
            raise_exception(exp_msg=_("depart exists"),
                            exp_num=ncTShareMgntError.NCT_DEPARTMENT_HAS_EXIST)

    def is_depart_id_exists(self, depart_id):
        """
        检查部门id是否存在
        因为check_depart_exists会忽略掉虚拟的 所有用户组 和 未分配用户组
        所以不采用check_depart_exists
        Args:
            depart_id: string 部门UUID
        Return:
            bool: True表示存在，False表示不存在
        """
        sql = """
        SELECT COUNT(*) AS cnt FROM `t_department`
        WHERE `f_department_id` = %s
        """
        count = self.r_db.one(sql, depart_id)['cnt']
        if count == 1:
            return True
        else:
            return False

    def get_oss_info(self, dept_id, oss_id):
        """
        获取对象存储信息，如果组织或部门的对象存储信息为空，则设置为空
        """
        if oss_id:
            oss_info = get_oss_info(oss_id)
        else:
            oss_info = ncTUsrmOSSInfo()
        return oss_info


    def get_site_id(self, dept_id):
        """
        根据组织或部门的id获取站点id
        """
        sql = """
        SELECT `f_oss_id` FROM `t_department`
        WHERE `f_department_id` = %s
        """
        result = self.r_db.one(sql, dept_id)
        return result['f_oss_id']

    def get_oss_id_by_dept_id(self, dept_id):
        """
        根据组织或部门的id获取对象存储id
        """
        sql = """
        SELECT `f_oss_id` FROM `t_department`
        WHERE `f_department_id` = %s
        """
        result = self.r_db.one(sql, dept_id)
        return result['f_oss_id']

    def get_oss_id_by_user_id(self, user_id):
        """
        根据组织或部门的id获取对象存储id
        """
        sql = """
        SELECT `f_oss_id` FROM `t_user`
        WHERE `f_user_id` = %s
        """
        result = self.r_db.one(sql, user_id)
        return result['f_oss_id']

    def get_organ_id_by_name(self, organ_name, b_raise=True):
        """
        根据名称获取组织ID
        """
        sql = """
        SELECT `f_department_id` FROM `t_department`
        WHERE `f_name` = %s AND `f_is_enterprise` = 1
        """
        organ = self.r_db.one(sql, organ_name)
        if not organ:
            if b_raise:
                raise_exception(exp_msg=_("organ not exists"),
                                exp_num=ncTShareMgntError.
                                NCT_ORGNIZATION_NOT_EXIST)
            else:
                return
        return organ['f_department_id']

    def get_parent_id(self, depart_id):
        """
        获取指定部门的父部门ID
        Args:
            depart_id: string 要获取的部门ID
        Return:
            有父部门则返回ID，没有则返回空字符串
        """
        depart_path = self.get_department_path_by_dep_id(depart_id)
        id_list = depart_path.split('/')
        if len(id_list) > 1:
            return id_list[-2]

        return ""

    def get_ou_by_depart_id(self, depart_id):
        """
        获取部门所在的组织id
        """
        depart_path = self.get_department_path_by_dep_id(depart_id)
        if depart_path:
            id_list = depart_path.split('/')
            return id_list[0]

        return ''


    def get_ou_by_user_id(self, user_id):
        """
        获取用户所在的组织id
        """

        path_list = self.get_department_path_by_user_id(user_id)
        ou_ids = []
        if path_list:
            for path in path_list:
                if path != NCT_UNDISTRIBUTE_USER_GROUP:
                    id_list = path.split('/')
                    ou_ids.append(id_list[0])

        return ou_ids

    def check_user_depart_belong_same_ou(self, user_id, depart_id):
        """
        检查用户和部门是否属于同一个组织
        """
        depart_ou_id = self.get_ou_by_depart_id(depart_id)
        user_ou_ids = self.get_ou_by_user_id(user_id)
        if depart_ou_id and user_ou_ids:
            if depart_ou_id in user_ou_ids:
                return True
        return False

    def check_department_email(self, department_id, email):
        """
        检查组织/部门邮箱
        """
        # 允许组织/部门邮箱为空
        if not email:
            return

        # 检查邮箱名是否合法
        if len(email) > 128 or not check_email(email):
            raise_exception(exp_msg=_("email illegal"),
                            exp_num=ncTShareMgntError.NCT_INVALID_EMAIL)

        # 检查邮箱名是否冲突
        sql = """
        SELECT f_mail_address FROM t_department
        WHERE f_mail_address = %s and f_department_id != %s
        UNION
        SELECT f_mail_address FROM t_user
        WHERE f_mail_address = %s
        LIMIT 1
        """
        result = self.r_db.one(sql, email, department_id, email)
        if result:
            raise_exception(exp_msg=_("duplicate email"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_EMALI)

    def update_third_id_by_id(self, depart_id, third_id):
        """
        更新部门的第三方id
        """
        sql = """
        UPDATE `t_department`
        SET `f_third_party_id` = %s
        WHERE `f_department_id` = %s
        """
        self.w_db.query(sql, third_id, depart_id)

    def __update_depart(self, editParam):
        """
        更新部门或组织
        """
        # 获取部门信息
        tmpSql = """
        SELECT f_manager_id, f_name, f_status FROM t_department WHERE f_department_id = %s
        """
        result = self.r_db.one(tmpSql, editParam.departId)

        tmp = ""
        # 检查组织或部门名
        c = ''
        change_departName = False
        change_status = False
        change_manager = False
        param_list = []
        if editParam.departName:
            for s in editParam.departName:
                if s == '%':
                    c += '%'
                c += s
            tmp += ("f_name='%s'," % self.w_db.escape(c))

            if result and result['f_name'] != c:
                change_departName = True

        # 检查权重值
        if editParam.priority is not None:
            tmp += ("f_priority='%s'," % editParam.priority)

        # 邮箱变更
        if editParam.email is not None:
            tmp += ("f_mail_address='%s'," % self.w_db.escape(editParam.email))

        # 存储变更
        tmp += ("f_oss_id='%s'," % self.w_db.escape(editParam.ossId))

        # 负责人变更
        if editParam.managerID is not None:
            tmp += ("f_manager_id='%s'," % editParam.managerID)
            if result and result['f_manager_id'] != editParam.managerID:
                change_manager = True

        # code变更
        if editParam.code is not None:
            tmp += ("f_code='%s'," % editParam.code)

        # remark变更
        if editParam.remark is not None:
            tmp += ("f_remark=%s,")
            param_list.append(editParam.remark)

        # status变更
        if editParam.status is not None:
            status = 1
            if editParam.status == False:
                status = 2
            tmp += ("f_status='%s'," % status)
            if result and result['f_status'] != status:
                change_status = True

        if tmp == "":
            return

        tmp = tmp[:len(tmp) - 1]

        sql = """
        UPDATE t_department SET {0} WHERE f_department_id = %s
        """.format(tmp)
        param_list.append(editParam.departId)

        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()
        cursor.execute(sql, tuple(param_list))

        if editParam.departName and change_departName:
            # 发送部门显示名更新nsq消息
            pub_nsq_msg(TOPIC_ORG_NAME_MODIFY, {"id": editParam.departId, "new_name": c, "type": "department"})

        if change_status:
            # 发送部门变更消息
            self.update_sub_depart_status(editParam.departId, editParam.status)

        if change_manager:
             # 发送部门服务人变更消息
            pub_nsq_msg(TOPIC_DEPART_MANAGER_MODIFIED, {"depart_id": editParam.departId, "original_manager_id": result['f_manager_id'], "current_manager_id": editParam.managerID})

    def update_sub_depart_status(self, depart_id, status):
        """
        更新所有子部门状态
        """
        # 如果启用，只更新自己的状态
        if status == True:
            pub_nsq_msg(TOPIC_DEPART_STATUS_MODIFIED, {"ids": [depart_id], "status": status})
            return

        # 如果禁用，则更新所有子部门的状态
        # 获取所有需要变更状态的部门
        checkSql = """
        SELECT f_department_id FROM t_department WHERE f_path like %s and f_status = 1
        """
        results = self.r_db.all(checkSql, "%%%s%%" % depart_id)
        departIds = [result["f_department_id"] for result in results] + [depart_id]

        sql = """
        UPDATE t_department SET f_status = 2 WHERE f_path like %s
        """
        self.w_db.query(sql, "%%%s%%" % depart_id)

        # 发送部门状态变更消息
        pub_nsq_msg(TOPIC_DEPART_STATUS_MODIFIED, {"ids": departIds, "status": status})

    def check_user_in_depart(self, user_id, depart_id, raise_ex=True):
        """
        检查用户是否只属于这个部门
        Raise:
            用户不属于这个部门
        """
        depart_path = self.get_department_path_by_dep_id(depart_id)

        sql = """
            SELECT `f_user_id` FROM `t_user_department_relation`
            WHERE `f_user_id` = %s AND f_path = %s
            LIMIT 1
        """
        result = self.r_db.one(sql, user_id, depart_path)
        if not result:
            if raise_ex:
                raise_exception(exp_msg=_("user not in depart"),
                                exp_num=ncTShareMgntError.
                                NCT_USER_NOT_IN_DEPARTMENT)
            else:
                return False
        return True

    def check_user_in_depart_recur(self, user_id, departid):
        """
        递归检查用户是否属于某个部门
        True:
            用户属于这个部门及其子部门
        False:
            用户不属于这个部门及其子部门
        """
        user_info = self.user_manage.get_user_by_id(user_id)
        sql = """
        SELECT `f_parent_department_id` FROM `t_department_relation`
        WHERE f_department_id = %s
        LIMIT 1
        """
        parent_ids = user_info.user.departmentIds
        while len(parent_ids):
            if departid in parent_ids:
                return True

            tmp_ids = []
            for d_id in parent_ids:
                result = self.r_db.one(sql, d_id)
                if result:
                    tmp_ids.append(result['f_parent_department_id'])
            parent_ids = tmp_ids
        return False

##################################################################################################
#                                         组织管理
##################################################################################################
    def fetch_organ(self, db_organ, get_departs=True):
        """
        获取组织信息，转换为结构体
        """
        organ = ncTUsrmOrganizationInfo()
        organ.organizationId = db_organ['f_department_id']
        organ.organizationName = db_organ['f_name']
        organ.ossInfo = self.get_oss_info(organ.organizationId, db_organ['f_oss_id'])
        organ.email = db_organ['f_mail_address']
        organ.thirdId = db_organ.get('f_third_party_id', '')

        if get_departs:
            organ.departments = self.get_sub_depart(db_organ['f_department_id'])

        organ.responsiblePersons = self.get_depart_mgrs(db_organ['f_department_id'])
        return organ

    def _is_or_name_valid(self, name):
        """
        检查组织名称是否符合规则，返回最后的名称
        1.必须为utf8编码
        2.前后的空格会被除去，中间的空格会被保留
        3.不能包含 \ / : * ? " < > | \s 特殊字符，\s包括[\t\n\r\f\v]
        4.长度最大为128字节
        5.最后的..会被去除
        """
        if name is None:
            raise_exception(exp_msg=_("IDS_INVALID_ORG_NAME"),
                            exp_num=ncTShareMgntError.NCT_INVALID_ORG_NAME)

        # 除去前面的空格，末尾的空格和点
        striped_name = name.lstrip()
        striped_name = striped_name.rstrip(". ")

        if not is_valid_string(striped_name):
            raise_exception(exp_msg=_("IDS_INVALID_ORG_NAME"),
                            exp_num=ncTShareMgntError.NCT_INVALID_ORG_NAME)

        return striped_name

    def create_organization(self, addParam):
        """
        添加组织
        """
        striped_name = self._is_or_name_valid(addParam.orgName)

        sql = """
        SELECT COUNT(*) AS cnt FROM `t_department`
        WHERE `f_name` = %s AND `f_is_enterprise` = 1
        """
        count = self.r_db.one(sql, striped_name)["cnt"]

        if count > 0:
            raise_exception(exp_msg=_("organ exists"),
                            exp_num=ncTShareMgntError.NCT_ORGNIZATION_HAS_EXIST)

        if addParam.thirdId != "":
            if self.check_department_exists_by_thirdId(addParam.thirdId):
                raise_exception(exp_msg=_("thirdId already exists"),
                            exp_num=ncTShareMgntError.NCT_ORGNIZATION_HAS_EXIST)

        # 检查组织权重是否在[1， 999999]范围内
        if addParam.priority is not None:
            if addParam.priority < 1 or addParam.priority > DEFAULT_DEPART_PRIORITY:
                raise_exception(exp_msg=_("IDS_INVALID_ORGAN_PRIORITY"),
                                exp_num=ncTShareMgntError.NCT_INVALID_ORGAN_PRIORITY)

        # 检查邮箱是否合法
        if addParam.email is not None:
            addParam.email = addParam.email.strip()
            self.check_department_email("", addParam.email)
        else:
            addParam.email = ""

        # 检查对象存储
        if addParam.ossId is None or addParam.ossId == "null":
            addParam.ossId = ""
        else:
            self.check_oss_id(addParam.ossId)

        # 检查负责人
        if addParam.managerID is not None and addParam.managerID != "":
            if addParam.managerID in [NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT]:
                raise_exception(exp_msg=_("IDS_INVALID_MANAGER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

            bExist = self.user_manage.check_user_exists(addParam.managerID, False)
            if not bExist:
                raise_exception(exp_msg=_("IDS_INVALID_MANAGER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

        # 检查备注
        if addParam.remark is not None:
            addParam.remark = self._is_remark_valid(addParam.remark)

        # 检查部门编码
        if addParam.code is not None:
            addParam.code = self.check_depart_code(addParam.code)

        organ_uuid = self.add_depart_to_db(name=striped_name, oss_id=addParam.ossId, priority=addParam.priority, email=addParam.email,
                            thirdId=addParam.thirdId, managerID=addParam.managerID, remark=addParam.remark, code=addParam.code, status=addParam.status)
        return organ_uuid

    def check_depart_code(self, code, depart_id=None):
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
        select f_department_id from t_department where f_code = %s
        """
        result = self.r_db.one(select_sql, striped_code)
        if result and result['f_department_id'] != depart_id:
            raise_exception(exp_msg=_("IDS_DUPLICATED_DEPART_CODE"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_DEPART_CODE)
        return striped_code

    def _is_remark_valid(self, remark):
        """
        检查备注是否符合规则，返回最后的备注
        1.必须为utf8编码
        2.前后的空格会被除去，中间的空格会被保留
        3.不能包含 \ / : * ? " < > | 特殊字符
        4.长度最大为128字节
        """
        if not remark:
            return ""

        # 除去前面的空格，末尾的空格和点
        striped_remark = remark.strip()

        if not is_valid_string(striped_remark):
            raise_exception(exp_msg=_("IDS_INVALID_REMARK"),
                            exp_num=ncTShareMgntError.NCT_INVALID_REMARK)
        return striped_remark

    def edit_organization(self, editParam):
        """
        编辑组织
        """
        # 检查组织是否存在
        self.check_organ_exists(editParam.departId)

        # 检查组织名
        if editParam.departName is not None:
            striped_name = self._is_or_name_valid(editParam.departName)
            sql = """
            SELECT COUNT(*) AS cnt FROM `t_department`
            WHERE `f_name` = %s AND `f_department_id` != %s
                AND `f_is_enterprise` = 1
            """
            count = self.r_db.one(sql, striped_name, editParam.departId)["cnt"]
            if count > 0:
                raise_exception(exp_msg=_("organ exists"),
                                exp_num=ncTShareMgntError.NCT_ORGNIZATION_HAS_EXIST)
            editParam.departName = striped_name

        if editParam.priority is not None:
            if editParam.priority < 1 or editParam.priority > DEFAULT_DEPART_PRIORITY:
                raise_exception(exp_msg=_("IDS_INVALID_ORGAN_PRIORITY"),
                                exp_num=ncTShareMgntError.NCT_INVALID_ORGAN_PRIORITY)

        # 检查组织邮箱
        if editParam.email is not None:
            editParam.email = editParam.email.strip()
            self.check_department_email(editParam.departId, editParam.email)

        # 检查对象存储
        if editParam.ossId is None or editParam.ossId == "null":
            editParam.ossId = ""
        else:
            self.check_oss_id(editParam.ossId)

        # 检查负责人
        if editParam.managerID is not None and editParam.managerID != "":
            if editParam.managerID in [NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT]:
                raise_exception(exp_msg=_("IDS_INVALID_MANAGER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

            bExist = self.user_manage.check_user_exists(editParam.managerID, False)
            if not bExist:
                raise_exception(exp_msg=_("IDS_INVALID_MANAGER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

        # 检查备注
        if editParam.remark is not None:
            editParam.remark = self._is_remark_valid(editParam.remark)

        # 检查部门编码
        if editParam.code is not None:
            editParam.code = self.check_depart_code(editParam.code, editParam.departId)

        # 更新数据库表
        self.__update_depart(editParam)

    def get_organization(self, organ_id, sub_departs=True):
        """
        获取组织信息
        """
        self.check_organ_exists(organ_id)

        sql = """
        SELECT `f_department_id`, `f_third_party_id`, `f_name`, `f_oss_id`, `f_mail_address` FROM `t_department`
        WHERE `f_department_id` = %s
        """
        db_organ = self.r_db.one(sql, organ_id)
        organ = self.fetch_organ(db_organ, sub_departs)
        return organ

    def get_organization_by_Name(self, organ_name, sub_departs=True):
        """
        获取组织信息
        """
        sql = """
        SELECT `f_department_id`, `f_name`, `f_oss_id`, `f_mail_address` FROM `t_department`
        WHERE `f_name` = %s AND `f_is_enterprise` = 1
        """
        db_organ = self.r_db.one(sql, organ_name)
        if not db_organ:
                raise_exception(exp_msg=_("organ not exists"),
                                exp_num=ncTShareMgntError.
                                NCT_ORGNIZATION_NOT_EXIST)

        organ = self.fetch_organ(db_organ, sub_departs)
        return organ

    def set_responsible_person(self, user_id, depart_ids, manager_id=None):
        """
        设为组织或部门负责人
        manager_id: 设置部门负责人的管理员， 管理员只能设置他所能管辖范围内的部门给用户，不能修改不在他管辖范围内的配置
        """
        # 所选组织部门为空
        if not depart_ids:
            raise_exception(exp_msg=_("depart or organ not exists"),
                            exp_num=ncTShareMgntError.NCT_ORG_OR_DEPART_NOT_EXIST)

        # 检查部门是否存在
        for depart_id in depart_ids:
            self.check_depart_exists(depart_id, include_organ=True)

        self.user_manage.check_user_exists(user_id)

        # 获取该用户管辖的组织和部门
        select_sql = """
        SELECT `f_department_id`
        FROM `t_department_responsible_person`
        INNER JOIN `t_department`
        USING(f_department_id)
        WHERE `f_user_id` = %s
        """
        db_dest_depart_ids = self.r_db.all(select_sql, user_id)
        old_depart_ids = [result["f_department_id"] for result in db_dest_depart_ids]
        need_add_depart_ids = set(depart_ids) - set(old_depart_ids)
        need_delete_depart_ids = set(old_depart_ids) - set(depart_ids)
        need_change_depart_ids = need_add_depart_ids | need_delete_depart_ids
        # 无论添加和删除的部门都必须在管理员所能管理的范围内进行
        if need_change_depart_ids and manager_id:
            # 获取管理员所能管理的所有部门id
            manager_dept_ids = self.get_supervisory_all_departids(manager_id)

            # 如果受改变的不在管理员范围内抛错, 根据差集为空判断
            if need_change_depart_ids - set(manager_dept_ids):
                raise_exception(exp_msg=_("IDS_CANNOT_MANAGER_DEPARTMENT"),
                                exp_num=ncTShareMgntError.NCT_CANNOT_MANAGER_DEPARTMENT)

        # 使用事务进行添加和删除
        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()

        # 表内插入数据
        def insert_depart_responsible_person(cursor, user_id, dept):
            check_sql = """
            select f_department_id from t_department_responsible_person
            where f_user_id = %s and f_department_id = %s
            """
            cursor.execute(check_sql, (user_id, dept))
            result = cursor.fetchone()

            if not result:
                insert_sql = """
                INSERT INTO `t_department_responsible_person`
                (`f_user_id`, `f_department_id`)
                VALUES(%s, %s)
                """
                cursor.execute(insert_sql, (user_id, dept))

        delete_sql = """
        DELETE
        FROM `t_department_responsible_person`
        WHERE `f_user_id` = %s and `f_department_id` = %s
        """
        try:
            for dept in need_add_depart_ids:
                insert_depart_responsible_person(cursor, user_id, dept)
            for dept in need_delete_depart_ids:
                cursor.execute(delete_sql, (user_id, dept))
            conn.commit()

        except Exception as e:
            conn.rollback()
            raise_exception(exp_msg=str(e),
                            exp_num=ncTShareMgntError.NCT_DB_OPERATE_FAILED)
        finally:
            cursor.close()
            conn.close()

    def cancel_responsible_person(self, user_id, manager_id=None):
        """
        取消部门负责人
        """
        self.user_manage.check_user_exists(user_id)

        # 获取该用户管辖的组织和部门
        select_sql = """
        SELECT `f_department_id` FROM `t_department_responsible_person`
        INNER JOIN `t_department`
        USING(f_department_id)
        WHERE `f_user_id` = %s
        """
        db_dest_depart_ids = self.r_db.all(select_sql, user_id)
        old_depart_ids = [result["f_department_id"] for result in db_dest_depart_ids]
        if old_depart_ids and manager_id:
            # 获取管理员所能管理的所有部门id
            manager_dept_ids = self.get_supervisory_all_departids(manager_id)
            # 如果受改变的不在管理员范围内抛错, 根据差集为空判断
            if set(old_depart_ids) - set(manager_dept_ids):
                raise_exception(exp_msg=_("IDS_CANNOT_MANAGER_DEPARTMENT"),
                                exp_num=ncTShareMgntError.NCT_CANNOT_MANAGER_DEPARTMENT)

        delete_sql = """
        DELETE FROM `t_department_responsible_person`
        WHERE `f_user_id` = %s
        """
        self.w_db.query(delete_sql, user_id)

    def set_audit_person(self, user_id, depart_ids, manager_id=None):
        """
        设为组织或部门审计员
        """
        # 所选组织部门为空
        if not depart_ids:
            raise_exception(exp_msg=_("depart or organ not exists"),
                            exp_num=ncTShareMgntError.NCT_ORG_OR_DEPART_NOT_EXIST)

        # 检查部门是否存在
        for depart_id in depart_ids:
            self.check_depart_exists(depart_id, include_organ=True)

        self.user_manage.check_user_exists(user_id)

        # 获取该用户已审计的组织和部门
        select_sql = """
        SELECT `f_department_id`
        FROM `t_department_audit_person`
        INNER JOIN `t_department`
        USING(f_department_id)
        WHERE `f_user_id` = %s
        """
        results = self.r_db.all(select_sql, user_id)
        old_list = [result['f_department_id'] for result in results]

        # 获取需要添加的审计对象列表
        need_add_list = []
        for dept in depart_ids:
            if dept not in old_list:
                need_add_list.append(dept)

        # 获取需要删除的审计对象列表
        need_delete_list = []
        for dept in old_list:
            if dept not in depart_ids:
                need_delete_list.append(dept)
        need_change_depart_ids = set(need_add_list) | set(need_delete_list)
        if need_change_depart_ids and manager_id:
            manager_dept_ids = self.get_supervisory_all_departids(manager_id,
                                                                  NCT_SYSTEM_ROLE_ORG_AUDIT)

            # 如果受改变的不在管理员范围内抛错, 根据差集为空判断
            if need_change_depart_ids - set(manager_dept_ids):
                raise_exception(exp_msg=_("IDS_CANNOT_MANAGER_DEPARTMENT"),
                                exp_num=ncTShareMgntError.NCT_CANNOT_MANAGER_DEPARTMENT)

        # 使用事务进行添加和删除
        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()

        # 修改t_department_audit_person表信息
        def replace_dep_audit(cursor, user_id, dept):
            check_sql = """
            select f_user_id from t_department_audit_person where f_user_id = %s and f_department_id = %s
            """
            cursor.execute(check_sql, (user_id, dept))
            result = cursor.fetchone()

            if not result:
                insert_sql = """
                INSERT INTO `t_department_audit_person`
                (`f_user_id`, `f_department_id`)
                VALUES(%s, %s)
                """
                cursor.execute(insert_sql, (user_id, dept))

        delete_sql = """
        DELETE
        FROM `t_department_audit_person`
        WHERE `f_user_id` = %s and `f_department_id` = %s
        """
        try:
            for dept in need_add_list:
               replace_dep_audit(cursor, user_id, dept)
            for dept in need_delete_list:
                cursor.execute(delete_sql, (user_id, dept))
            conn.commit()

        except Exception as e:
            conn.rollback()
            raise_exception(exp_msg=str(e),
                            exp_num=ncTShareMgntError.NCT_DB_OPERATE_FAILED)
        finally:
            cursor.close()
            conn.close()

    def cancel_audit_person(self, user_id, manager_id=None):
        """
        取消部门审计员
        """
        self.user_manage.check_user_exists(user_id)

        # 获取该用户管辖的组织和部门
        select_sql = """
        SELECT `f_department_id` FROM `t_department_audit_person`
        INNER JOIN `t_department`
        USING(f_department_id)
        WHERE `f_user_id` = %s
        """
        db_dest_depart_ids = self.r_db.all(select_sql, user_id)
        old_depart_ids = [result["f_department_id"] for result in db_dest_depart_ids]
        if old_depart_ids and manager_id:
            # 获取管理员所能管理的所有部门id
            manager_dept_ids = self.get_supervisory_all_departids(manager_id,
                                                                  NCT_SYSTEM_ROLE_ORG_AUDIT)
            # 如果受改变的不在管理员范围内抛错, 根据差集为空判断
            if set(old_depart_ids) - set(manager_dept_ids):
                raise_exception(exp_msg=_("IDS_CANNOT_MANAGER_DEPARTMENT"),
                                exp_num=ncTShareMgntError.NCT_CANNOT_MANAGER_DEPARTMENT)

        delete_sql = """
        DELETE FROM `t_department_audit_person`
        WHERE `f_user_id` = %s
        """
        self.w_db.query(delete_sql, user_id)

    def get_supervisory_departs(self, user_id):
        """
        获取用户所管辖的组织、部门
        """
        self.user_manage.check_user_exists(user_id)

        sql = """
            SELECT `responsible_person`.`f_department_id` AS `f_department_id`
            FROM `t_department_responsible_person` AS `responsible_person`
            JOIN `t_department`
            ON `responsible_person`.`f_department_id` = `t_department`.`f_department_id`
            where `f_user_id` = %s
        """

        results = self.r_db.all(sql, user_id)

        department_infos = []

        if results:
            for result in results:
                department_id = result["f_department_id"]
                depart_info = self.get_department_info(department_id, True)
                department_infos.append(depart_info)

        return department_infos

    def fill_role_manage_departments(self, role_members):
        """
        填充组织管理员角色成员所管辖部门信息
        """
        if not role_members:
            return

        userid_map = {}
        for role_member in role_members:
            userid_map[role_member.userId] = role_member

        groupStr = generate_group_str(list(userid_map.keys()))
        if not groupStr:
            return

        sql = """
        SELECT  r.f_user_id, r.f_department_id, d.f_name
        FROM t_department as d
        JOIN t_department_responsible_person as r
        ON d.f_department_id = r.f_department_id
        where r.f_user_id in ({0})
        """.format(groupStr)

        results = self.r_db.all(sql)
        for result in results:
            role_member = userid_map[result['f_user_id']]
            if not role_member.manageDeptInfo:
                role_member.manageDeptInfo = ncTManageDeptInfo()
                role_member.manageDeptInfo.departmentIds = []
                role_member.manageDeptInfo.departmentNames = []

            role_member.manageDeptInfo.departmentIds.append(result['f_department_id'])
            role_member.manageDeptInfo.departmentNames.append(result['f_name'])

    def fill_role_audit_departments(self, role_members):
        """
        填充审计管理员角色成员所管辖部门信息
        """
        if not role_members:
            return

        userid_map = {}
        for role_member in role_members:
            userid_map[role_member.userId] = role_member

        groupStr = generate_group_str(list(userid_map.keys()))
        if not groupStr:
            return

        sql = """
        SELECT  r.f_user_id, r.f_department_id, d.f_name
        FROM t_department as d
        JOIN t_department_audit_person as r
        ON d.f_department_id = r.f_department_id
        where r.f_user_id in ({0})
        """.format(groupStr)

        results = self.r_db.all(sql)
        for result in results:
            role_member = userid_map[result['f_user_id']]
            # 解析用户所管辖的部门信息
            if not role_member.manageDeptInfo:
                role_member.manageDeptInfo = ncTManageDeptInfo()
                role_member.manageDeptInfo.departmentIds = []
                role_member.manageDeptInfo.departmentNames = []

            role_member.manageDeptInfo.departmentIds.append(result['f_department_id'])
            role_member.manageDeptInfo.departmentNames.append(result['f_name'])

    def get_depart_mgr_ids(self, depart_id):
        """
        获取当前部门所有负责人ID
        """
        sql = """
            SELECT `f_user_id` FROM `t_department_responsible_person`
            WHERE `f_department_id` = %s
        """
        results = self.r_db.all(sql, depart_id)
        user_ids = []
        for result in results:
            user_ids.append(result['f_user_id'])
        return user_ids

    def get_depart_mgrs(self, depart_id):
        """
        获取当前部门下所有负责人信息
        """
        uids = self.get_depart_mgr_ids(depart_id)
        users = []
        for uid in uids:
            sql = """
                SELECT `f_user_id`, `f_login_name`, `f_display_name`, `f_idcard_number`,
                `f_password`, `f_des_password`, `f_sha2_password`,`f_mail_address`,
                `f_auth_type`, `f_status`, `f_remark`, `f_priority`,
                `f_csf_level`, `f_pwd_control`, `f_oss_id`,
                `f_create_time`, `f_auto_disable_status`
                FROM `t_user`
                WHERE `f_user_id` = %s
            """
            db_user = self.r_db.one(sql, uid)

            sql_dept = """
                SELECT `f_name` FROM `t_department`
                WHERE `f_department_id` = %s
            """
            db_dept = self.r_db.one(sql_dept, depart_id)
            db_user['parentDepartId'] = depart_id
            db_user['f_name'] = db_dept['f_name']
            if db_user['f_password'] != '':
                db_user['originalPwd'] = True if self.initAdminPwd == db_user['f_password'] else False
            elif db_user['f_sha2_password'] != '':
                db_user['originalPwd'] = True if self.initSha2AdminPwd == db_user['f_sha2_password'] else False

            user = self.user_manage.fetch_user(db_user, False)

            users.append(user)
        return users

    def get_depart_parent_path_by_batch(self, depart_ids):
        """
        批量根据部门ID(组织ID)获取部门（组织）父路经
        """
        department_infos = []
        sql = """
        SELECT `f_name` from `t_department`
        WHERE `f_department_id` = %s
        """

        for depart_id in depart_ids:
            depart_info = ncTUsrmDepartmentInfo()
            depart_info.departmentId = depart_id
            parent_id = self.get_parent_id(depart_id)
            depart_info.parentDepartId = parent_id
            db_parent = self.r_db.one(sql, self.w_db.escape(parent_id))
            depart_info.parentDepartName = db_parent['f_name'] if db_parent else ""
            depart_info.parentPath = self.get_parent_path(depart_id) if depart_info.parentDepartId  else ""
            department_infos.append(depart_info)

        return department_infos

##################################################################################################
#                                          部门管理
##################################################################################################
    def _is_depart_name_valid(self, name):
        """
        检查部门名称是否符合规则，返回最后的名称
        1.必须为utf8编码
        2.前后的空格会被除去，中间的空格会被保留
        3.不能包含 \ / : * ? " < > | \s 特殊字符，\s包括[\t\n\r\f\v]
        4.长度最大为128字节
        5.最后的..会被去除
        """
        if name is None:
            raise_exception(exp_msg=_("IDS_INVALID_DEPART_NAME"),
                            exp_num=ncTShareMgntError.NCT_INVALID_DEPART_NAME)

        # 除去前面的空格，末尾的空格和点
        striped_name = name.lstrip()
        striped_name = striped_name.rstrip(". ")

        if not is_valid_string(striped_name):
            raise_exception(exp_msg=_("IDS_INVALID_DEPART_NAME"),
                            exp_num=ncTShareMgntError.NCT_INVALID_DEPART_NAME)

        return striped_name

    def check_department_exists_by_thirdId(self, thirdId):
        """"
        检查部门是否存在，返回检查结果
        """
        sql = """
        SELECT `f_department_id` FROM `t_department` WHERE `f_third_party_id` = %s
        LIMIT 1
        """
        result = self.r_db.one(sql, thirdId)
        if not result:
            return False
        return True

    def add_department(self, addParam):
        """
        新建部门
        """
        striped_name = self._is_depart_name_valid(addParam.departName)

        if addParam.thirdId != "":
            if self.check_department_exists_by_thirdId(addParam.thirdId):
                raise_exception(exp_msg=_("thirdId already exists"),
                            exp_num=ncTShareMgntError.NCT_DEPARTMENT_HAS_EXIST)

        # 检查父部门是否存在，包含检测组织
        self.check_depart_exists(addParam.parentId, True)

        # 检查父部门下的子部门是否存在同名部门
        self.check_name_in_sub_departs(addParam.parentId, striped_name)

        # 检查部门权重是否在[1， 999999]范围内
        if addParam.priority is not None:
            if addParam.priority < 1 or addParam.priority > DEFAULT_DEPART_PRIORITY:
                raise_exception(exp_msg=_("IDS_INVALID_DEPART_PRIORITY"),
                                exp_num=ncTShareMgntError.NCT_INVALID_DEPART_PRIORITY)

        # 检查邮箱是否合法
        if addParam.email is not None:
            addParam.email = addParam.email.strip()
            self.check_department_email("", addParam.email)

        # 检查对象存储
        if addParam.ossId is None or addParam.ossId == "null":
            addParam.ossId = ""
        else:
            self.check_oss_id(addParam.ossId)

        # 检查负责人
        if addParam.managerID is not None and addParam.managerID != "":
            if addParam.managerID in [NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT]:
                raise_exception(exp_msg=_("IDS_INVALID_MANAGER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

            bExist = self.user_manage.check_user_exists(addParam.managerID, False)
            if not bExist:
                raise_exception(exp_msg=_("IDS_INVALID_MANAGER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)
        # 检查备注
        if addParam.remark is not None:
            addParam.remark = self._is_remark_valid(addParam.remark)

        # 检查code
        if addParam.code is not None:
            addParam.code = self.check_depart_code(addParam.code)

        return self.add_depart_to_db(name=striped_name, oss_id=addParam.ossId,
                                     parent_id=addParam.parentId, priority=addParam.priority, email=addParam.email, thirdId=addParam.thirdId,
                                     managerID=addParam.managerID, code=addParam.code, remark=addParam.remark, status=addParam.status)


    def add_depart_to_db(self, name="", oss_id=None, parent_id=None, ou_info=None,
                         priority=None, email=None, third_ou_info=None, thirdId=None,
                         managerID=None, status=True, remark=None, code=None):
        """
        添加部门到数据库
        这个函数会生成UUID
        Args:
            name: string 部门名称
            parent_id: string 父部门组织ID，默认None。如果None，则添加为组织
            site_id: 站点id
            ou_info: ncTUsrmDomainOU None为添加本地部门
            third_ou_info: OuInfo第三方组织信息
        """
        depart_uuid = str(uuid.uuid1())
        is_organ = 1 if parent_id is None else 0
        full_path = ""

        # 获取组织/部门全路径
        if parent_id is None:
            full_path = depart_uuid
        else:
            full_path = self.get_department_path_by_dep_id(parent_id) + "/" + depart_uuid

        # 如果存储未设置，则为空
        if not oss_id:
            oss_id = ""

        if priority is None:
            priority = DEFAULT_DEPART_PRIORITY

        if email is None:
            email = ""

        if thirdId is None:
            thirdId = ""

        if managerID is None:
            managerID = ""

        if remark is None:
            remark = ""

        if code is None:
            code = ""

        status_data = 1
        if status == False:
            status_data = 2

        # 记录插入的部门名字，用于发送nsq消息
        departName = ""
        # 保存到部门表
        if ou_info is not None:
            self.w_db.insert(
                "t_department",
                {
                    "f_department_id": depart_uuid,
                    "f_auth_type": ncTUsrmDepartType.NCT_DEPART_TYPE_DOMAIN,
                    "f_name": ou_info.name,
                    "f_domain_path": ou_info.pathName,
                    "f_is_enterprise": is_organ,
                    "f_third_party_id": ou_info.objectGUID,
                    "f_oss_id": oss_id,
                    "f_mail_address": email,
                    "f_path": full_path,
                    "f_manager_id": managerID,
                    "f_code": code,
                    "f_remark": remark,
                    "f_status": status_data
                }
            )
            departName = ou_info.name
        elif third_ou_info is not None:
            self.w_db.insert(
                "t_department",
                {
                    "f_department_id": depart_uuid,
                    "f_auth_type": third_ou_info.type,
                    "f_name": third_ou_info.ou_name,
                    "f_is_enterprise": is_organ,
                    "f_third_party_id": third_ou_info.third_id,
                    "f_oss_id": third_ou_info.oss_id,
                    "f_priority": third_ou_info.priority,
                    "f_path": full_path,
                    "f_manager_id": managerID,
                    "f_code": code,
                    "f_remark": remark,
                    "f_status": status_data
                }
            )
            departName = third_ou_info.ou_name
        else:
            self.w_db.insert(
                "t_department",
                {
                    "f_department_id": depart_uuid,
                    "f_auth_type": ncTUsrmDepartType.NCT_DEPART_TYPE_LOCAL,
                    "f_name": name,
                    "f_is_enterprise": is_organ,
                    "f_third_party_id": thirdId,
                    "f_oss_id": oss_id,
                    "f_priority": priority,
                    "f_mail_address": email,
                    "f_path": full_path,
                    "f_manager_id": managerID,
                    "f_code": code,
                    "f_remark": remark,
                    "f_status": status_data
                }
            )
            departName = name
        pub_nsq_msg(TOPIC_DEPT_CREATED,{"id":depart_uuid,"name":departName})

        if not is_organ:
            # 保存到部门关系表
            self.w_db.insert("t_department_relation", {
                "f_department_id": depart_uuid,
                "f_parent_department_id": parent_id
            })

        # 添加索引
        data = {
            "f_department_id": depart_uuid,
            "f_ou_id": self.user_manage.get_organ_from_depart(depart_uuid),
        }
        self.w_db.insert("t_ou_department", data)

        return depart_uuid

    def add_third_depart_to_db(self, third_id, third_name, parent_id=None, oss_id=None, email=None):
        """
        添加第三方部门到数据库
        这个函数会生成UUID
        Args:
            third_id: string 第三方部门id
            third_name: string 第三方部门id
            name: string 部门名称
            parent_id: string 父部门῁组织ID，默认None。如果None，则添加为组织
            ou_info: ncTUsrmDomainOU None为添加本地部门
        """
        depart_uuid = str(uuid.uuid1())

        is_organ = 1 if parent_id is None else 0
        org_id = depart_uuid if parent_id is None \
            else self.user_manage.get_organ_from_depart(parent_id)
        full_path = ""

        # 获取组织/部门全路径
        if parent_id is None:
            full_path = org_id
        else:
            full_path = self.get_department_path(parent_id) + "/" + depart_uuid

        # 使用事务插入部门数据
        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()

        # 如果存储未设置，则为空
        if not oss_id:
            oss_id = ""

        if email is None:
            email = ""

        try:
            # 保存到部门表
            strsql = """
            INSERT INTO t_department
                (f_department_id, f_auth_type, f_name, f_is_enterprise, f_third_party_id, f_oss_id, f_mail_address, f_path)
            VALUES(%s, %s, %s, %s, %s, %s, %s, %s)
            """
            cursor.execute(strsql, (depart_uuid,
                                    ncTUsrmDepartType.NCT_DEPART_TYPE_THIRD,
                                    third_name,
                                    is_organ,
                                    third_id,
                                    oss_id,
                                    email,
                                    full_path))

            # 保存到部门关系表
            if not is_organ:
                strsql = """
                INSERT INTO t_department_relation
                    (f_department_id,f_parent_department_id)
                VALUES(%s, %s)
                """
                cursor.execute(strsql, (depart_uuid, parent_id))

            # 部门-组织关系表
            strsql = """
            INSERT INTO t_ou_department
                (f_department_id,f_ou_id)
            VALUES(%s, %s)
            """
            cursor.execute(strsql, (depart_uuid, org_id))

            conn.commit()
            pub_nsq_msg(TOPIC_DEPT_CREATED,{"id":depart_uuid,"name":third_name})
        except Exception as e:
            conn.rollback()
            raise_exception(exp_msg=str(e),
                            exp_num=ncTShareMgntError.NCT_DB_OPERATE_FAILED)
        return depart_uuid

    def edit_department(self, editParam):
        """
        编辑部门
        """
        # 检查部门是否存在
        self.check_depart_exists(editParam.departId)
        # 检查部门名
        if editParam.departName is not None:
            striped_name = self._is_depart_name_valid(editParam.departName)
            # 检查父部门下的子部门是否存在同名部门
            parent_id = self.get_parent_id(editParam.departId)
            if parent_id:
                self.check_name_in_sub_departs(parent_id, striped_name, editParam.departId)
            editParam.departName = striped_name

        if editParam.priority is not None:
            if editParam.priority < 1 or editParam.priority > DEFAULT_DEPART_PRIORITY:
                raise_exception(exp_msg=_("IDS_INVALID_DEPART_PRIORITY"),
                                exp_num=ncTShareMgntError.NCT_INVALID_DEPART_PRIORITY)

        # 检查邮箱是否合法
        if editParam.email is not None:
            editParam.email = editParam.email.strip()
            self.check_department_email(editParam.departId, editParam.email)

        # 检查对象存储
        if editParam.ossId is None or editParam.ossId == "null":
            editParam.ossId = ""
        else:
            self.check_oss_id(editParam.ossId)

        # 检查负责人
        if editParam.managerID is not None and editParam.managerID != "":
            if editParam.managerID == editParam.departId or editParam.managerID in [NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT]:
                raise_exception(exp_msg=_("IDS_INVALID_MANAGER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

            bExist = self.user_manage.check_user_exists(editParam.managerID, False)
            if not bExist:
                raise_exception(exp_msg=_("IDS_INVALID_MANAGER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

        # 检查remark
        if editParam.remark is not None:
            editParam.remark= self._is_remark_valid(editParam.remark)

        # 检查code
        if editParam.code is not None:
            editParam.code = self.check_depart_code(editParam.code, editParam.departId)

        # 更新数据库表
        self.__update_depart(editParam)

    def edit_department_oss(self, depart_id, oss_id):
        """
        编辑部门的对象存储
        """
        # 检查部门是否存在
        if oss_id is None or oss_id == "null":
            oss_id = ""
        self.check_depart_exists(depart_id, True)

        # 检查存储是否可用
        self.check_oss_id(oss_id)

        # 更新数据库表
        sql = """
        UPDATE `t_department` SET`f_oss_id` = %s
        WHERE `f_department_id` = %s
        """
        self.w_db.query(sql, oss_id, depart_id)

    def move_department(self, src_depart_id, dest_depart_id):
        """
        移动部门到其他部门，包括部门下的所有用户和子部门
        Args:
            src_depart_id:被移动的部门id
            dest_depart_id:目的部门id
        """
        # 目的部门不能是未分配用户组和所用户组
        if dest_depart_id == NCT_UNDISTRIBUTE_USER_GROUP:
            raise_exception(exp_msg=_("cann't move departmen to undistribute group."),
                            exp_num=ncTShareMgntError.NCT_CANNOT_MOVE_DEPARTMENT_TO_UNDISTRIBUTE_GROUP)

        if dest_depart_id == NCT_ALL_USER_GROUP:
            raise_exception(exp_msg=_("cann't move department to all user group."),
                            exp_num=ncTShareMgntError.NCT_CANNOT_MOVE_DEPARTMENT_TO_ALL_GROUP)

        # 检查部门是否存在
        try:
            self.check_depart_exists(src_depart_id, True)
        except Exception:
            raise_exception(exp_msg=_("src department not exist."),
                            exp_num=ncTShareMgntError.NCT_SRC_DEPARTMENT_NOT_EXIST)

        try:
            # 目的部门可以是组织
            self.check_depart_exists(dest_depart_id, True)
        except Exception:
            raise_exception(exp_msg=_("dest department not exist."),
                            exp_num=ncTShareMgntError.NCT_DEST_DEPARTMENT_NOT_EXIST)

        # 如果是移动到父部门下，则不处理
        if dest_depart_id == self.get_parent_id(src_depart_id):
            return

        # 检查目的部门下是否存在同名子部门
        try:
            src_depart_info = self.get_department_info(src_depart_id, True)
            self.check_name_in_sub_departs(dest_depart_id, src_depart_info.departmentName)
        except Exception:
            raise_exception(exp_msg=_("a same name subdepartment exists in dest deapartment"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_SUBDEP_EXIST_IN_DEP)

        # 获取部门所有子部门id包括自身
        depart_ids = self.get_all_departids(src_depart_id)

        # 不能移动到子部门
        if dest_depart_id in depart_ids:
            raise_exception(exp_msg=_("cann't move department to it's subdepartment"),
                            exp_num=ncTShareMgntError.NCT_CANNOT_MOVE_DEPARTMENT_TO_CHILDREN)

        # 获取组织信息
        src_org_id = self.user_manage.get_organ_from_depart(src_depart_id)
        dest_org_id = self.user_manage.get_organ_from_depart(dest_depart_id)

        b_org = self.check_organ_exists(src_depart_id, False)

        # 获取被移动部门下所有用户id
        user_infos = []
        user_infos += self.get_all_users_info_of_depart(src_depart_id, 0, -1, False)

        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()
        try:
            # 移动部门，则修改父部门id
            if not b_org:
                str_sql = """
                    UPDATE `t_department_relation`
                    SET `f_parent_department_id` = %s
                    WHERE `f_department_id` = %s
                """
                cursor.execute(str_sql, (dest_depart_id, src_depart_id))

                # 移动部门，修改移动部门的权重为默认权重
                str_sql = """
                    UPDATE `t_department`
                    SET `f_priority` = %s
                    WHERE `f_department_id` = %s
                """
                cursor.execute(str_sql, (DEFAULT_DEPART_PRIORITY, src_depart_id))
            # 移动组织则插入部门关系
            else:
                insert_sql = """
                    INSERT INTO `t_department_relation`
                        (`f_department_id`, `f_parent_department_id`)
                    VALUES(%s, %s)
                """
                cursor.execute(insert_sql, (src_depart_id, dest_depart_id))

                update_dept_sql = """
                    UPDATE `t_department`
                    SET `f_is_enterprise` = 0
                    WHERE `f_department_id` = %s
                """
                cursor.execute(update_dept_sql, (src_depart_id,))

            # 修改部门搜索索引关系
            str_sql = """
                UPDATE `t_ou_department`
                SET `f_ou_id` = %s
                WHERE `f_department_id` = %s AND `f_ou_id` = %s
            """
            for depart_id in depart_ids:
                cursor.execute(str_sql, (dest_org_id, depart_id, src_org_id))

            # 修改用户的搜素索引关系
            update_ou_user_sql = """
                UPDATE `t_ou_user`
                SET `f_ou_id` = %s
                WHERE `f_user_id` = %s AND `f_ou_id` = %s
            """

            insert_ou_user_sql = """
                INSERT INTO `t_ou_user` (`f_user_id`, `f_ou_id`)
                VALUES(%s, %s)
            """

            for user in user_infos:
                # 检查原组织中是否应该删除
                b_del_ou = True
                for d_id in user.user.departmentIds:
                    if d_id != src_depart_id:
                        tmp_ou_id = self.get_ou_by_depart_id(d_id)
                        if tmp_ou_id == src_org_id:
                            b_del_ou = False

                if b_del_ou:
                    cursor.execute(update_ou_user_sql, (dest_org_id, user.id, src_org_id))
                else:
                    cursor.execute(insert_ou_user_sql, (user.id, dest_depart_id))

            new_path = self.get_department_path_by_dep_id(dest_depart_id) + "/" + src_depart_id
            old_path = self.get_department_path_by_dep_id(src_depart_id)
            update_full_path_sql = """
            UPDATE `t_department` SET `f_path` = replace(f_path, %s, %s)
            """
            # 更新当前部门以及所有子部门全路径path值
            cursor.execute(update_full_path_sql, (old_path, new_path))

            update_user_full_path_sql = """
            UPDATE `t_user_department_relation` SET `f_path` = replace(f_path, %s, %s)
            """

            # 更新当前部门下所有用户的部门全路径path值
            cursor.execute(update_user_full_path_sql, (old_path, new_path))

            conn.commit()

            # 发送部门被移动消息
            pub_nsq_msg(TOPIC_DEPT_MOVE, {
                                "id": src_depart_id, "old_path": old_path, "new_path": new_path})

            # 获取所有被移动的用户，并且发送用户被移动消息
            get_moved_users_sql = f"""
            select f_user_id, f_path from {get_db_name('sharemgnt_db')}.t_user_department_relation where f_path like %s
            """
            results = self.r_db.all(get_moved_users_sql, new_path + "%%")
            for result in results:
                new_direct_path = result['f_path']
                old_direct_path = result['f_path'].replace(new_path, old_path)
                pub_nsq_msg(TOPIC_USER_MOVE, {
                                "id": result['f_user_id'], "old_dept_path": old_direct_path, "new_dept_path": new_direct_path})

        except Exception as ex:
            conn.rollback()
            raise_exception(exp_msg=str(ex),
                            exp_num=ncTShareMgntError.NCT_DB_OPERATE_FAILED)

    def sort_department(self, userId, src_depart_id, dest_down_depart_id):
        """
        对部门进行权重排序
        只能在同级部门下移动
        args:
            src_depart_id:被移动的部门id
            dest_down_depart_id:插入位置下面的部门
        """
        # 检查源部门是否存在
        try:
            self.check_depart_exists(src_depart_id, True)
        except Exception:
            raise_exception(exp_msg=_("src department not exist."),
                            exp_num=ncTShareMgntError.NCT_SRC_DEPARTMENT_NOT_EXIST)

        # 检查目的位置是否合法
        try:
            if dest_down_depart_id:
                self.check_depart_exists(dest_down_depart_id, True)
        except Exception:
            raise_exception(exp_msg=_("dest position is illegal."),
                            exp_num=ncTShareMgntError.NCT_DEST_POSTION_ILLEGAL)

        # 检查移动到的位置是否合法
        # 不能移动未分配用户组和所有用户组,且目的部门不能是未分配用户组和所有用户组
        if dest_down_depart_id == NCT_UNDISTRIBUTE_USER_GROUP or src_depart_id == NCT_UNDISTRIBUTE_USER_GROUP:
            raise_exception(exp_msg=_("cann't move department above undistribute group."),
                            exp_num=ncTShareMgntError.NCT_CANNOT_MOVE_DEPARTMENT_TO_UNDISTRIBUTE_GROUP)

        if dest_down_depart_id == NCT_ALL_USER_GROUP or src_depart_id == NCT_ALL_USER_GROUP:
            raise_exception(exp_msg=_("cann't move department above all user group."),
                            exp_num=ncTShareMgntError.NCT_CANNOT_MOVE_DEPARTMENT_TO_ALL_GROUP)

        # 组织管理员，检查源部门和移动的位置是否在在管理的组织下
        if userId not in [NCT_USER_ADMIN, NCT_USER_SYSTEM]:
            orgs = self.get_supervisory_root_org(userId, roleId=None)
            orgs_id = [org.id for org in orgs]

            src_path = self.get_department_path_by_dep_id(src_depart_id).split("/")
            dest_path = self.get_department_path_by_dep_id(dest_down_depart_id).split("/")

            if  len(set(src_path) & set (orgs_id)) == 0:
                raise_exception(exp_msg=_("IDS_CANNOT_MANAGER_DEPARTMENT"),
                            exp_num=ncTShareMgntError.NCT_CANNOT_MANAGER_DEPARTMENT)

            if dest_down_depart_id:
                if  len(set(dest_path) & set (orgs_id)) == 0:
                    raise_exception(exp_msg=_("IDS_CANNOT_MANAGER_DEPARTMENT"),
                        exp_num=ncTShareMgntError.NCT_CANNOT_MANAGER_DEPARTMENT)

        # 只能在同级部门下移动
        src_parent_depart_id = self.get_parent_id(src_depart_id)
        if dest_down_depart_id:
            dest_parent_depart_id = self.get_parent_id(dest_down_depart_id)
        else:
            dest_parent_depart_id = src_parent_depart_id

        if src_parent_depart_id != dest_parent_depart_id:
            raise_exception(exp_msg=_("cann't move department out of the same level department or organization."),
                                       exp_num=ncTShareMgntError.NCT_NOT_IN_ORIGINAL_DEPARTMENT)

        # 获取子部门
        if src_parent_depart_id:
            sub_departs = self.get_sub_departments(src_parent_depart_id)
        else:
            sub_departs = self.get_supervisory_root_org(userId, roleId=None)

        # 部门重新排序
        depart_sorted_list = []
        for depart in sub_departs:
            if depart.id == src_depart_id:
                continue
            if depart.id == dest_down_depart_id:
                depart_sorted_list.append(src_depart_id)
                depart_sorted_list.append(dest_down_depart_id)
            else:
                depart_sorted_list.append(depart.id)
        if dest_down_depart_id == "":
            depart_sorted_list.append(src_depart_id)

        # 把新的权重写入数据库
        condition_sql = ""
        where_sql= ""
        for depart_id in depart_sorted_list:
            tmp = """
            WHEN '{0}' THEN {1}
            """.format(self.w_db.escape(depart_id), depart_sorted_list.index(depart_id)+1)
            condition_sql = " ".join([condition_sql, tmp])

            end_tmp = """
            '{0}'
            """.format(self.w_db.escape(depart_id))
            where_sql = (where_sql + "," + end_tmp) if where_sql else end_tmp

        sql = """
        UPDATE t_department SET f_priority =
        (CASE f_department_id
        {0}
        end
        ) WHERE f_department_id IN ({1})
        """.format(condition_sql, where_sql)
        self.w_db.query(sql)

    def fetch_departs(self, db_departs):
        """
        将部门数据库信息填充到结构体
        """
        if not isinstance(db_departs, list):
            db_departs = list(db_departs)

        depart_list = []
        for db_depart in db_departs:
            depart = ncTUsrmDepartmentInfo()
            depart.departmentId = db_depart['f_department_id']
            depart.departmentName = db_depart['f_name']
            depart.parentDepartId = db_depart['parent_id']
            depart.parentDepartName = db_depart['parent_name']
            depart.responsiblePersons = self.get_depart_mgrs(db_depart['f_department_id'])
            depart.ossInfo = self.get_oss_info(depart.departmentId, db_depart['f_oss_id'])
            depart.email = db_depart['f_mail_address']
            depart_list.append(depart)
        return depart_list

    def get_one_level_sub_depart(self, depart_id):
        """
        获取部门下的子部门
        只获取一级部门
        """
        self.check_depart_exists(depart_id, True)
        return self.get_sub_depart(depart_id, False)

    def get_sub_depart_id_by_name(self, parent_id, name):
        """
        根据子部门名获取部门下的子部门id
        Args:
            parent_id: string 父部门ID
            name:  要获取的部门名
        """
        sql = """
        SELECT `depart`.`f_department_id` FROM `t_department` AS `depart`
        JOIN `t_department_relation` AS `relation`
        ON `depart`.`f_department_id` = `relation`.`f_department_id`
        WHERE `relation`.`f_parent_department_id` = %s
            AND `depart`.`f_name` = %s
        """
        result = self.r_db.one(sql, parent_id, name)
        if result:
            return result['f_department_id']

    def get_sub_depart(self, depart_id, get_all=True):
        """
        获取部门下的子部门
        Args:
            depart_id: string 要获取的部门ID
            get_all: bool 是否获取所有子部门
        """
        sql = """
        SELECT `depart`.`f_department_id`, `depart`.`f_name`, `depart`.`f_oss_id`, `depart`.`f_mail_address`,
            `relation`.`f_parent_department_id` AS `parent_id`,
            `parent`.`f_name` AS `parent_name`
        FROM `t_department_relation` AS `relation`
        JOIN `t_department` AS `depart`
        ON `depart`.`f_department_id` = `relation`.`f_department_id`
        JOIN `t_department` AS `parent`
        ON `parent`.`f_department_id` = `relation`.`f_parent_department_id`
        WHERE `relation`.`f_parent_department_id` = %s
        order by `depart`.`f_priority`, upper(`depart`.`f_name`)
        """
        db_departs = self.r_db.all(sql, depart_id)
        depart_list = self.fetch_departs(db_departs)

        # 不需要获取所有子部门，只返回一级子部门
        if not get_all:
            return depart_list

        # 转换为双端队列，提高处理性能
        depart_queue = deque(depart_list)
        try:
            while True:
                depart = depart_queue.popleft()
                # 从数据库获取这个部门的子部门
                db_departs = self.r_db.all(sql, depart.departmentId)
                sub_departs = self.fetch_departs(db_departs)
                # 将获取到的子部门添加到队列，后面继续获取
                depart_queue.extend(sub_departs)
                # 将查到的子部门放到返回列表
                depart_list.extend(sub_departs)
        # 所有部门的子部门获取完毕，则会丢出异常
        except IndexError:
            return depart_list

    def add_user_to_department(self, user_ids, depart_id):
        """
        添加用户到部门
        """
        if depart_id == NCT_UNDISTRIBUTE_USER_GROUP:
            raise_exception(exp_msg=_("cann't move user to undistribute group."),
                            exp_num=ncTShareMgntError.
                            NCT_CANNOT_IMPORT_DOMAIN_USER_TO_UNDISTRIBUTE)

        self.check_depart_exists(depart_id, True)

        # 去重
        user_ids = list(set(user_ids))

        # 用户不存在，忽略
        exist_users = []
        for uid in user_ids:
            if not self.user_manage.check_user_exists(uid, False):
                continue

            exist_users.append(uid)

            # 如果已经在当前部门，则忽略
            sql = """
            SELECT `f_user_id` FROM `t_user_department_relation`
            WHERE `f_user_id` = %s AND `f_department_id` = %s
            LIMIT 1
            """
            result = self.r_db.one(sql, uid, depart_id)
            if result:
                continue

            # 获取用户所在部门全路径
            department_path = self.get_department_path_by_dep_id(depart_id)
            self.w_db.insert("t_user_department_relation", {
                "f_user_id": uid,
                "f_department_id": depart_id,
                "f_path": department_path
            })

            # 删除该用户所属未分配组的关系
            sql = """
            DELETE FROM `t_user_department_relation`
            WHERE `f_user_id` = %s AND `f_department_id` = %s
            """
            self.w_db.query(sql, uid, NCT_UNDISTRIBUTE_USER_GROUP)

            # 添加搜索索引
            ou_id = self.user_manage.get_organ_from_depart(depart_id)
            str_sql = """
            SELECT count(*) AS cnt
            FROM `t_ou_user`
            WHERE `f_user_id` = %s AND `f_ou_id` = %s
            """
            result = self.r_db.one(str_sql, uid, ou_id)
            if result and result['cnt'] == 0:
                self.w_db.insert("t_ou_user", {"f_user_id": uid, "f_ou_id": ou_id})

            pub_nsq_msg(TOPIC_DEPARTMENT_USER_ADD,{"id": uid, "dept_paths": [department_path]})

        return exist_users

    def remove_user_from_department(self, user_ids, depart_id):
        """
        从部门移除用户
        """
        self.check_depart_exists(depart_id, True)
        # 去重
        user_ids = list(set(user_ids))

        failed_user_id = []

        def del_search_index(uid, ou_id):
            """
            删除搜索索引
            """
            sql = """
            DELETE FROM `t_ou_user`
            WHERE `f_user_id` = %s AND `f_ou_id` = %s
            """
            self.w_db.query(sql, uid, ou_id)

        # 用户不存在，忽略
        for uid in user_ids:
            if not self.check_user_in_depart(uid, depart_id, False):
                failed_user_id.append(uid)
                continue

            # 获取用户所在部门全路径
            department_path = self.get_department_path_by_dep_id(depart_id)
            # 用户-部门关系不存在，忽略
            # 删除此关系
            sql = """
            DELETE FROM `t_user_department_relation`
            WHERE `f_user_id` = %s AND `f_path` = '{0}'
            """.format(department_path)
            self.w_db.query(sql, uid)

            ou_id = self.get_ou_by_depart_id(depart_id)
            # 查询此用户是否还属于其他部门，不属于了，则添加一条未分配用户的记录

            # 发送用户从部门移除消息
            pub_nsq_msg(TOPIC_DEPARTMENT_USER_REMOVE, {
                        "id": uid, "dept_paths": [department_path]})
            path_list = self.get_department_path_by_user_id(uid)
            if len(path_list) > 0:
                # 判断索引是否需要删除
                # 如果用户已经不属于部门，则需要删除索引
                # 否则保留索引
                want_del = True
                for path in path_list:
                    oid = self.get_ou_id_by_depart_path(path)
                    if oid == ou_id:
                        want_del = False
                        break
                if want_del:
                    del_search_index(uid, ou_id)
                continue

            self.w_db.insert("t_user_department_relation", {
                "f_user_id": uid,
                "f_department_id": NCT_UNDISTRIBUTE_USER_GROUP,
                "f_path": NCT_UNDISTRIBUTE_USER_GROUP
            })

            # 已经不在所有部门，删除搜索索引
            del_search_index(uid, ou_id)

        # 返回不成功的用户id
        return failed_user_id

    def move_user_to_department(self, user_ids, src_depart_id, dest_depart_id):
        """
        移动用户到其他部门
        Args:
            user_ids:用户id列表
            src_depart_id:源部门id
            dest_depart_id:目的部门id
        """
        if src_depart_id == dest_depart_id:
            return []

        # 去重
        user_ids = list(set(user_ids))

        # 用户不能移动到未分配用户组
        if dest_depart_id == NCT_UNDISTRIBUTE_USER_GROUP:
            raise_exception(exp_msg=_("cann't move user to undistribute group."),
                            exp_num=ncTShareMgntError.
                            NCT_CANNOT_MOVE_USER_TO_UNDISTRIBUTE_GROUP)

        # 检查部门是否存在
        try:
            self.check_depart_exists(src_depart_id, True)
        except Exception:
            raise_exception(exp_msg=_("src department not exist."),
                            exp_num=ncTShareMgntError.
                            NCT_SRC_DEPARTMENT_NOT_EXIST)

        try:
            self.check_depart_exists(dest_depart_id, True)
        except Exception:
            raise_exception(exp_msg=_("dest department not exist"),
                            exp_num=ncTShareMgntError.
                            NCT_DEST_DEPARTMENT_NOT_EXIST)

        # 获取搜索部门索引
        src_org_id = self.user_manage.get_organ_from_depart(src_depart_id)
        dest_org_id = self.user_manage.get_organ_from_depart(dest_depart_id)

        not_exists_user_ids = []

        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()

        for user_id in user_ids:
            # 用户不属于当前部门则，跳过
            if not self.check_user_in_depart(user_id, src_depart_id, False):
                not_exists_user_ids.append(user_id)
                continue

            # 判断目的部门是否已有此用户
            has_exist = self.check_user_in_depart(user_id, dest_depart_id, False)

            src_depart_path = self.get_department_path_by_dep_id(src_depart_id)
            dest_depart_path = self.get_department_path_by_dep_id(dest_depart_id)
            try:
                if has_exist:
                    str_delet_sql = """
                    DELETE FROM `t_user_department_relation`
                    WHERE `f_user_id` = %s AND `f_path` = %s
                    """
                    cursor.execute(str_delet_sql, (user_id, src_depart_path))
                else:
                    # 更新用户部门信息
                    str_path_sql = """
                    UPDATE `t_user_department_relation`
                    SET `f_department_id` = %s, `f_path` = %s
                    WHERE `f_user_id` = %s AND `f_path` = %s
                    """
                    cursor.execute(str_path_sql, (dest_depart_id, dest_depart_path, user_id, src_depart_path))

                if src_org_id != NCT_UNDISTRIBUTE_USER_GROUP:
                    str_sql = """
                    UPDATE `t_ou_user`
                    SET `f_ou_id` = %s
                    WHERE `f_user_id` = %s AND `f_ou_id` = %s
                    """
                    cursor.execute(str_sql, (dest_org_id, user_id, src_org_id))
                else:
                    str_sql = """
                    INSERT INTO `t_ou_user`
                    (`f_user_id`,`f_ou_id`)
                    VALUES(%s, %s)
                    """
                    cursor.execute(str_sql, (user_id, dest_org_id))

                conn.commit()
                if not has_exist:
                    pub_nsq_msg(TOPIC_USER_MOVE, {
                                "id": user_id, "old_dept_path": src_depart_path, "new_dept_path": dest_depart_path})

            except Exception as ex:
                conn.rollback()
                raise_exception(exp_msg=str(ex),
                                exp_num=ncTShareMgntError.
                                NCT_DB_OPERATE_FAILED)

        return not_exists_user_ids

    def get_mgr_id_by_depart_id(self, depart_id):
        """
        获取当前部门及其父部门的所有管理员id
        """
        # 获取当前部门及其父部门的ID
        manager_ids = []
        all_depart_ids = self.get_dept_path_to_root(depart_id)
        if len(all_depart_ids) == 0:
            return manager_ids

        # 获取当前部门及其父部门所有管理员ID
        groupStr = generate_group_str(all_depart_ids)
        sql = """
        SELECT distinct f_user_id FROM t_department_responsible_person WHERE f_department_id in ({0})
        """.format(groupStr)

        results = self.r_db.all(sql)
        for result in results:
            manager_ids.append(result['f_user_id'])

        return manager_ids

    def get_parent_path(self, depart_id):
        """
        获取部门父路径
        """
        parent_depart_names = []
        depart_path = self.get_department_path_by_dep_id(depart_id)
        parent_path = self.get_parent_department_path(depart_path)
        id_list = parent_path.split('/')

        groupStr = generate_group_str(id_list)

        sql = """
        SELECT f_name FROM t_department WHERE f_department_id in ({0}) ORDER BY FIELD(f_department_id, {0})
        """.format(groupStr)

        result = self.r_db.all(sql)
        parent_depart_names = [name['f_name'] for name in result]
        if parent_depart_names:
            return '/'.join(parent_depart_names)
        return ''

    def get_department_info(self, depart_id, b_include_org=False, include_parent_path=False):
        """
        获取部门信息
        """
        self.check_depart_exists(depart_id, b_include_org)

        sql = """
        SELECT `f_name`, `f_oss_id`, `f_mail_address` , `f_path`, `f_third_party_id`, `f_manager_id`, `f_code`, `f_remark`, `f_status` FROM `t_department`
        WHERE `f_department_id` = %s
        """
        db_depart = self.r_db.one(sql, depart_id)

        depart_info = ncTUsrmDepartmentInfo()
        depart_info.departmentId = depart_id
        depart_info.departmentName = '未分配用户' if db_depart == [] else db_depart['f_name']
        depart_info.ossInfo = self.get_oss_info(depart_id, db_depart['f_oss_id'])
        depart_info.responsiblePersons = self.get_depart_mgrs(depart_id)
        depart_info.email = db_depart['f_mail_address']
        depart_info.thirdId = db_depart['f_third_party_id']
        depart_info.code = db_depart['f_code']
        depart_info.remark = db_depart['f_remark']
        depart_info.managerID = db_depart['f_manager_id']
        depart_info.status = True
        if db_depart['f_status'] == 2:
            depart_info.status = False
        depart_info.managerDisplayName = self.user_manage.get_displayname_by_userid(db_depart['f_manager_id'])

        parent_path = self.get_parent_department_path(db_depart['f_path'])
        sql = """
        SELECT `f_department_id`, `f_name`
        FROM `t_department`
        WHERE `f_path` = %s
        LIMIT 1
        """
        db_parent = self.r_db.one(sql, parent_path)
        depart_info.parentDepartId = db_parent['f_department_id'] if db_parent else ""
        depart_info.parentDepartName = db_parent['f_name'] if db_parent else ""
        if include_parent_path:
            depart_info.parentPath = self.get_parent_path(depart_id)
        return depart_info

    def get_department_info_by_third_id(self, third_id):
        """
        获取部门信息
        """
        sql = """
        SELECT `f_department_id`, `f_name`, `f_oss_id`, `f_mail_address`
        FROM `t_department`
        WHERE `f_third_party_id` = %s
        """
        db_depart = self.r_db.one(sql, third_id)
        if not db_depart:
            raise_exception(exp_msg=_("depart or organ not exists"),
                            exp_num=ncTShareMgntError.NCT_ORG_OR_DEPART_NOT_EXIST)
        depart_info = ncTUsrmDepartmentInfo()
        depart_info.departmentId = db_depart['f_department_id']
        depart_info.departmentName = '未分配用户' if db_depart == [] else db_depart['f_name']
        depart_info.ossInfo = self.get_oss_info(depart_info.departmentId, db_depart['f_oss_id'])
        depart_info.responsiblePersons = self.get_depart_mgrs(depart_info.departmentId)
        depart_info.email = db_depart['f_mail_address']

        sql = """
        SELECT `parent`.`f_department_id`, `parent`.`f_name`
        FROM `t_department` AS `parent`
        JOIN `t_department_relation` AS `relation`
        ON `parent`.`f_department_id` = `relation`.`f_parent_department_id`
        WHERE `relation`.`f_department_id` = %s
        LIMIT 1
        """
        db_parent = self.r_db.one(sql, depart_info.departmentId)
        depart_info.parentDepartId = db_parent['f_department_id'] if db_parent else ""
        depart_info.parentDepartName = db_parent['f_name'] if db_parent else ""

        return depart_info

    def get_department_info_by_name(self, name):
        """
        通过部门层级名获取部门信息
        """
        name_levels = name.split("/")
        sql = """
        SELECT `f_department_id`
        FROM `t_department`
        WHERE `f_name` = %s AND `f_is_enterprise` = 1
        """
        result = self.r_db.one(sql, name_levels[0])
        if result:
            parent_id = result["f_department_id"]
            sql = """
            SELECT d.f_department_id
            FROM t_department as d
            INNER JOIN t_department_relation as r
            ON d.f_department_id = r.f_department_id
            WHERE f_name = %s and f_parent_department_id = %s
            """
            for name in name_levels[1:]:
                result = self.r_db.one(sql, name, parent_id)
                if not result:
                    break
                parent_id = result["f_department_id"]
            if (name == name_levels[-1]) and result:
                depart_id = result["f_department_id"]
                return self.get_department_info(depart_id, True)

        raise_exception(exp_msg=_("organ not exists"),
                        exp_num=ncTShareMgntError.NCT_ORGNIZATION_NOT_EXIST)

    def get_department_tree(self, organ_id):
        """
        获取指定组织的部门树
        """
        self.check_organ_exists(organ_id)

        # 获取组织，并检查组织是否存在
        sql = """
        SELECT `f_department_id`, `f_name` FROM `t_department`
        WHERE `f_department_id` = %s AND `f_is_enterprise` = 1
        """
        db_organ = self.r_db.one(sql, organ_id)
        if not db_organ:
            raise_exception(exp_msg=_("organ not exists"),
                            exp_num=ncTShareMgntError.
                            NCT_ORGNIZATION_NOT_EXIST)

        return self.get_sub_depart(db_organ['f_department_id'])

    def get_dept_path_to_root(self, dept_id):
        """
        获取部门到组织的路径 (包括部门)
        """
        sql = """
        SELECT `f_path` FROM `t_department`
        WHERE `f_department_id` = %s
        """
        result = self.r_db.one(sql, dept_id)

        all_depart_ids = []
        if result:
            all_depart_ids = result['f_path'].split('/')
        return all_depart_ids

    def get_dept_path_name_to_root(self, dept_id):
        """
        获取部门到组织的路径名称(包括部门)
        比如由部门B的id获取完整路径['A/C/B']
        """
        sql = """
        SELECT
            `t_department_relation`.`f_parent_department_id`,
            `t_department`.`f_name`
        FROM
            `t_department_relation`
            INNER JOIN `t_department`
            ON ( `t_department_relation`.`f_department_id` = `t_department`.`f_department_id` )
        WHERE
            `t_department`.`f_department_id` = %s
        """
        path_names = []
        tmp_id = dept_id
        while tmp_id:
            result = self.r_db.one(sql, tmp_id)
            if result:
                tmp_id = result['f_parent_department_id']
                tmp_name = result['f_name']
                path_names.insert(0, tmp_name)
            else:
                sql = """
                SELECT `f_name` from `t_department` WHERE `f_department_id` = %s
                """
                org_name = self.r_db.one(sql, tmp_id)
                path_names.insert(0, org_name.get('f_name'))

                tmp_id = None

        return path_names

    def is_descendant_of_ids(self, depart_id, ids):
        """
        判断depart_id是否是ids的子部门
        """
        parent_id = self.get_parent_id(depart_id)
        while parent_id != "":
            if parent_id in ids:
                return True
            parent_id = self.get_parent_id(parent_id)
        return False

    def __get_user_role_id(self, user_id):
        """
        获取用户角色id
        """
        from src.modules.role_manage import RoleManage
        return RoleManage().get_user_role_id(user_id)

    def get_supervisory_ids(self, user_id, roleId=None):
        """
        用户在指定角色中所能看到的所有部门
        若用户是超级管理员、系统管理员、安全管理员、 审计管理员，可以看所有组织
        若用户是组织审计员，并且获取的是组织审计员角色，则只能看组织审计员的范围
        若用户是组织管理员，则能看组织管理员的范围
        """
        user_roles = self.__get_user_role_id(user_id)
        can_see_all_roles = [NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_ADMIN,
                             NCT_SYSTEM_ROLE_SECURIT, NCT_SYSTEM_ROLE_AUDIT]

        can_see_roles = []
        can_see_roles.extend(can_see_all_roles)
        can_see_roles.append(NCT_SYSTEM_ROLE_ORG_MANAGER)
        can_see_roles.append(NCT_SYSTEM_ROLE_ORG_AUDIT)

        # 若用户不能看组织结构，则返回空
        if not (set(user_roles) & set(can_see_roles)):
            return []

        # 超级管理员、系统管理员、安全管理员、审计管理员可以看到所有组织
        if set(user_roles) & set(can_see_all_roles):
            sql = """
                SELECT f_department_id
                FROM t_department WHERE f_is_enterprise = 1
                """
            results = self.r_db.all(sql)
            return [result['f_department_id'] for result in results]
        # 组织审计员看自身审计范围内部门
        elif (NCT_SYSTEM_ROLE_ORG_AUDIT in user_roles and roleId == NCT_SYSTEM_ROLE_ORG_AUDIT):
            sql = """
                SELECT f_department_id FROM t_department_audit_person
                WHERE f_user_id = '{0}'
                """.format(self.w_db.escape(user_id))
        # 组织管理员看自身管辖内部门, 组织审计员有自身范围，不使用组织管理员范围
        elif NCT_SYSTEM_ROLE_ORG_MANAGER in user_roles:
            sql = """
                SELECT f_department_id FROM t_department_responsible_person
                WHERE f_user_id = '{0}'
                """.format(self.w_db.escape(user_id))
        else:
            return []
        results = self.r_db.all(sql)

        # 非超级管理员、系统管理员、安全管理员、审计管理员，只返回根部门
        ids = [result['f_department_id'] for result in results]
        rootids = []
        for depart_id in ids:
            if self.is_descendant_of_ids(depart_id, ids) is False:
                rootids.append(depart_id)
        return rootids

    def get_sub_department_count(self, depart_id):
        """
        获取子部门数目
        """
        sql = """
        select count(f_relation_id) as cnt from t_department_relation
        where f_parent_department_id=%s
        """
        result = self.r_db.one(sql, depart_id)
        return int(result['cnt'])

    def get_sub_user_count(self, depart_id):
        """
        获取部门的子用户数
        """
        sql = """
        select count(f_relation_id) as cnt from t_user_department_relation
        where f_department_id=%s
        """
        result = self.r_db.one(sql, depart_id)
        return int(result['cnt'])

    def get_org_infos_by_ids(self, org_ids):
        """
        根据org的id列表获取组织信息
        """
        groupStr = generate_group_str(org_ids)
        sql = """
        SELECT f_department_id, f_name, f_is_enterprise, f_oss_id, f_mail_address
        FROM t_department
        WHERE f_department_id IN ({0})
        ORDER BY f_priority, upper(f_name)
        """.format(groupStr)
        results = self.r_db.all(sql)
        org_infos = []
        for record in results:
            info = ncTRootOrgInfo()
            info.isOrganization = (record['f_is_enterprise'] == 1)
            info.id = record['f_department_id']
            info.name = record['f_name']
            info.ossInfo = self.get_oss_info(info.id, record['f_oss_id'])
            info.responsiblePersons = self.get_depart_mgrs(record['f_department_id'])
            info.subDepartmentCount = self.get_sub_department_count(record['f_department_id'])
            info.subUserCount = self.get_sub_user_count(record['f_department_id'])
            info.email = record['f_mail_address']

            org_infos.append(info)
        return org_infos

    def get_root_org_by_user_id(self, user_id):
        """
        根据用户id获取根组织
        """
        self.user_manage.check_user_exists(user_id)
        sql = """
        SELECT `f_ou_id` FROM `t_ou_user`
        WHERE `f_user_id` = %s
        """
        results = self.r_db.all(sql, user_id)

        if results:
            org_ids = [res['f_ou_id'] for res in results]
            return self.get_org_infos_by_ids(org_ids)
        else:
            info = ncTRootOrgInfo()
            info.id = NCT_UNDISTRIBUTE_USER_GROUP
            info.name = _("undistributed user")
            info.isOrganization = True
            info.ossInfo = ncTUsrmOSSInfo()
            return [info]

    def get_supervisory_root_org(self, user_id, roleId=None):
        """
        获取用户管理的根组织或部门
        """
        self.user_manage.check_user_exists(user_id)
        org_ids = self.get_supervisory_ids(user_id, roleId)

        if len(org_ids) == 0:
            return []
        else:
            return self.get_org_infos_by_ids(org_ids)

    def get_sub_departments(self, depart_id):
        """
        获取部门的子部门
        """
        self.check_depart_exists(depart_id, True)

        sql = ("select f_department_id, f_name, f_oss_id, f_mail_address from t_department "
               "where f_department_id in "
               "(select f_department_id from t_department_relation"
               " where f_parent_department_id = %s) "
               "order by f_priority, upper(f_name)")

        results = self.r_db.all(sql, depart_id)

        departs = []
        for record in results:
            info = ncTDepartmentInfo()
            info.id = record['f_department_id']
            info.name = record['f_name']
            info.responsiblePersons = self.get_depart_mgrs(record['f_department_id'])
            info.subDepartmentCount = self.get_sub_department_count(record['f_department_id'])
            info.subUserCount = self.get_sub_user_count(record['f_department_id'])
            info.ossInfo = self.get_oss_info(info.id, record['f_oss_id'])
            info.email = record['f_mail_address']
            departs.append(info)

        return departs

    def search_supervisory_users(self, user_id, key, start, limit, roleId=None):
        """
        搜索用户管理的用户
        """
        self.user_manage.check_user_exists(user_id)
        limit_statement = check_start_limit(start, limit)
        if key == "":
            return []

        user_roles = self.__get_user_role_id(user_id)
        can_see_all_roles = [NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_ADMIN,
                             NCT_SYSTEM_ROLE_SECURIT, NCT_SYSTEM_ROLE_AUDIT]

        can_see_roles = []
        can_see_roles.extend(can_see_all_roles)
        can_see_roles.append(NCT_SYSTEM_ROLE_ORG_MANAGER)
        can_see_roles.append(NCT_SYSTEM_ROLE_ORG_AUDIT)

        # 若用户不能看组织结构，则返回空
        if not (set(user_roles) & set(can_see_roles)):
            return []

        esckey = "%%%s%%" % escape_key(key)
        # 增加搜索排序子句
        order_by_str = generate_search_order_sql(['f_login_name', 'f_display_name'])

        if set(user_roles) & set(can_see_all_roles):
            sql = """
            select f_user_id,f_login_name,f_display_name, f_csf_level from t_user
            where (f_login_name like %s or f_display_name like %s) and
            f_user_id not in ('{0}', '{1}', '{2}', '{3}')
            order by {4}, upper(f_display_name)
            {5}
            """.format(NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM,
                       NCT_USER_SECURIT, order_by_str, limit_statement)

        else:
            departids = self.get_supervisory_all_departids(user_id, roleId)
            if(len(departids) == 0):
                return []
            groupStr = generate_group_str(departids)
            sql = """
            select distinct t_user.f_user_id,
            t_user.f_login_name,
            t_user.f_display_name,
            t_user.f_csf_level,
            t_user.f_priority
            from t_user inner join t_user_department_relation
            on (t_user.f_user_id = t_user_department_relation.f_user_id)
            where t_user_department_relation.f_department_id in ({0}) and
            """.format(groupStr)
            sql += """(t_user.f_login_name like %s
            or t_user.f_display_name like %s)
            and t_user.f_user_id not in ('{0}', '{1}', '{2}', '{3}')
            order by t_user.f_priority, {4}, upper(t_user.f_display_name)
            {5}""".format(NCT_USER_ADMIN, NCT_USER_AUDIT,
                NCT_USER_SYSTEM, NCT_USER_SECURIT, order_by_str, limit_statement)

        results = self.r_db.all(sql, esckey, esckey,
                                escape_key(key), escape_key(key), esckey, esckey)

        userinfos = []
        for record in results:
            info = ncTSearchUserInfo()
            info.id = record['f_user_id']
            info.loginName = record['f_login_name']
            info.displayName = record['f_display_name']
            info.csfLevel = record['f_csf_level']
            info.departmentIds = []
            info.departmentNames = []
            userinfos.append(info)

        # 填充所属部门id和名称
        self.user_manage.fill_user_departments(userinfos)

        # 填充所属部门路径
        for userinfo in userinfos:
            userinfo.departmentPaths = []
            for index, depids in enumerate(userinfo.departmentIds):
                parentpath = self.get_parent_path(depids)
                if parentpath != '':
                    userinfo.departmentPaths.append(parentpath + '/' + userinfo.departmentNames[index])
                else:
                    userinfo.departmentPaths.append(userinfo.departmentNames[index])

        return userinfos

    def get_supervisory_user_ids(self, manager_id, roleId=None):
        """
        获取用户所能管理的所有用户信息
        """
        user_ids = set()
        self.user_manage.check_user_exists(manager_id)

        # 如果是超级管理员、系统管理员从所有用户中获取
        user_roles = self.__get_user_role_id(manager_id)
        if set(user_roles) & set([NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_ADMIN]):
            sql = """
            SELECT f_user_id,f_login_name,f_display_name
            FROM t_user
            WHERE f_user_id not in ('{0}', '{1}', '{2}', '{3}')
            order by f_priority, upper(f_display_name)
            """.format(NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT)

        else:
            departids = self.get_supervisory_all_departids(manager_id, roleId)
            if(len(departids) == 0):
                return []
            groupStr = generate_group_str(departids)
            sql = """
            SELECT distinct t_user.f_user_id, t_user.f_login_name, t_user.f_display_name, t_user.f_priority
            FROM t_user inner join t_user_department_relation
            on (t_user.f_user_id = t_user_department_relation.f_user_id)
            WHERE t_user_department_relation.f_department_id in ({0}) and
            """.format(groupStr)
            sql += """  t_user.f_user_id not in ('{0}', '{1}', '{2}', '{3}')
            order by t_user.f_priority, upper(t_user.f_display_name)
            """.format(NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT)

        results = self.r_db.all(sql)

        for record in results:
            user_ids.add(record['f_user_id'])

        return user_ids

    def locate_user(self, manager_id, user_id):
        """
        定位用户
        """
        self.user_manage.check_user_exists(manager_id)
        self.user_manage.check_user_exists(user_id)

        depart_ids = []
        parent_ids = self.user_manage.get_belong_depart_id(user_id)
        if parent_ids != "":
            for parent_id in parent_ids:
                depart_ids.append(parent_id)
                parent_id = self.get_parent_id(parent_id)

        if len(depart_ids) == 0:
            return []

        groupStr = generate_group_str(depart_ids)

        sql = """
        select distinct f_department_id,f_name
        from t_department
        where f_department_id in ({0})
        """.format(groupStr)
        results = self.r_db.all(sql)

        locateinfos = []
        for record in results:
            info = ncTLocateInfo()
            info.departId = record['f_department_id']
            info.departName = record['f_name']
            locateinfos.append(info)

        return locateinfos

    def get_all_departids(self, departid):
        """
        获取所有部门id, 包括自身,子部门id
        """

        departids = []
        if departid == NCT_UNDISTRIBUTE_USER_GROUP:
            return departids

        depart_path = self.get_department_path_by_dep_id(departid)
        if depart_path == '':
            return departids

        sql = """
        select f_department_id from t_department
        where f_path like '{0}%%'
        """.format(depart_path)

        results = self.r_db.all(sql)

        for record in results:
            departids.append(record['f_department_id'])
        return departids

    def get_sub_departids(self, departid):
        """
        获取部门id的子部门id，不包括自身
        """
        departids = []
        sql = """
        select f_department_id from t_department_relation
         where f_parent_department_id = %s
        """
        results = self.r_db.all(sql, departid)

        subDepartids = []
        for record in results:
            subDepartids.append(record['f_department_id'])
        departids.extend(subDepartids)

        while subDepartids:
            where_clause = []
            for dept_id in subDepartids:
                tmp = "'%s'" % self.w_db.escape(dept_id)
                where_clause.append(tmp)
            where_clause = 'WHERE f_parent_department_id in ( ' + ','.join(where_clause) + ')'
            sql = """
            select f_department_id from t_department_relation
            {0}
            """.format(where_clause)
            results = self.r_db.all(sql)

            subDepartids = []
            for record in results:
                subDepartids.append(record['f_department_id'])
            departids.extend(subDepartids)

        return departids

    def get_supervisory_all_departids(self, manager_id, roleId=None):
        """
        获取用户管理的所有的部门id
        """
        ids = self.get_supervisory_ids(manager_id, roleId)

        # 再获取这些部门的子部门id
        departids = []
        for depart_id in ids:
            departids += self.get_all_departids(depart_id)

        return departids

    def get_user_count_of_depart(self, depart_id):
        """
        根据部门获取用户总数
        """
        self.check_depart_exists(depart_id, True)

        sql = """
        SELECT COUNT(*) AS cnt FROM `t_user_department_relation`
        WHERE `f_department_id` = %s
        AND `f_user_id` != '{0}'
        AND `f_user_id` != '{1}'
        AND `f_user_id` != '{2}'
        AND `f_user_id` != '{3}'
        """.format(NCT_USER_ADMIN, NCT_USER_SYSTEM,
                   NCT_USER_AUDIT, NCT_USER_SECURIT)
        db_count = self.r_db.one(sql, depart_id)

        return db_count['cnt']

    def get_users_of_depart(self, depart_id, start, limit,
                            need_check_depart_exist=True, only_user_id=False):
        """
        获取指定组织/部门下的用户
        """
        if depart_id == NCT_ALL_USER_GROUP:
            return self.user_manage.get_all_users(start, limit)

        # 检查部门是否存在
        if need_check_depart_exist:
            self.check_depart_exists(depart_id, True)
        db_users = []
        limit_statement = check_start_limit(start, limit)

        depart_path = self.get_department_path_by_dep_id(depart_id)
        sql = """
        SELECT `user`.`f_user_id`, `user`.`f_login_name`,
            `user`.`f_display_name`, `user`.`f_remark`, `user`.`f_password`,
            `user`.`f_des_password`, `user`.`f_sha2_password`, `user`.`f_mail_address`,
            `user`.`f_tel_number`, `user`.`f_idcard_number`, `user`.`f_expire_time`,
            `user`.`f_auth_type`, `user`.`f_status`,
            `user`.`f_priority`, `user`.`f_csf_level`,
            `user`.`f_pwd_control`, `user`.`f_oss_id`,
            `user`.`f_freeze_status`,
            `user`.`f_create_time`,
            `user`.`f_auto_disable_status`,
            `dept`.`f_name`,
            `user`.`f_code`,
            `user`.`f_manager_id`,
            `user`.`f_position`,
            `user`.`f_csf_level2`
        FROM `t_user` AS `user`
        JOIN `t_user_department_relation` AS `relation`
            ON `relation`.`f_user_id` = `user`.`f_user_id`
        JOIN `t_department` AS `dept`
            ON `relation`.`f_path` = `dept`.`f_path`
        WHERE `relation`.`f_path` = %s
            AND `relation`.`f_user_id` != '{0}'
            AND `relation`.`f_user_id` != '{1}'
            AND `relation`.`f_user_id` != '{2}'
            AND `relation`.`f_user_id` != '{3}'
        ORDER BY f_priority, upper(`f_display_name`)
        {4}
        """.format(NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM,
                   NCT_USER_SECURIT, limit_statement)

        if depart_id == NCT_UNDISTRIBUTE_USER_GROUP:
            sql = """
            SELECT `user`.`f_user_id`, `user`.`f_login_name`,
                `user`.`f_display_name`, `user`.`f_remark`, `user`.`f_password`,
                `user`.`f_des_password`, `user`.`f_sha2_password`, `user`.`f_mail_address`,
                `user`.`f_tel_number`, `user`.`f_idcard_number`, `user`.`f_expire_time`,
                `user`.`f_auth_type`, `user`.`f_status`,
                `user`.`f_priority`, `user`.`f_csf_level`,
                `user`.`f_pwd_control`, `user`.`f_oss_id`,
                `user`.`f_freeze_status`,
                `user`.`f_create_time`,
                `user`.`f_auto_disable_status`,
                `user`.`f_code`,
                `user`.`f_manager_id`,
                `user`.`f_position`,
                `user`.`f_csf_level2`
            FROM `t_user` AS `user`
            JOIN `t_user_department_relation` AS `relation`
                ON `relation`.`f_user_id` = `user`.`f_user_id`
            WHERE `relation`.`f_path` = %s
                AND `relation`.`f_user_id` != '{0}'
                AND `relation`.`f_user_id` != '{1}'
                AND `relation`.`f_user_id` != '{2}'
                AND `relation`.`f_user_id` != '{3}'
            ORDER BY f_priority,upper(`f_display_name`)
            {4}
            """.format(NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM,
                       NCT_USER_SECURIT, limit_statement)
        db_users = self.r_db.all(sql, depart_path)

        users = []
        if only_user_id:
            return [db_user['f_user_id'] for db_user in db_users]

        for db_user in db_users:
            if '-1' == depart_id:
                db_user['f_name'] = _("undistributed user")
            if db_user['f_password'] != '':
                db_user['originalPwd'] = True if self.initAdminPwd == db_user['f_password'] else False
            elif db_user['f_sha2_password'] != '':
                db_user['originalPwd'] = True if self.initSha2AdminPwd == db_user['f_sha2_password'] else False
            db_user['parentDepartId'] = depart_id
            users.append(self.user_manage.convert_user_info(db_user))
        # 填充用户所属部门信息
        self.user_manage.fill_user_departments(users)
        # 填充用户角色信息
        self.user_manage.fill_user_roles(users)
        # 填充用户部门负责人信息
        self.user_manage.fill_user_managers(users)
        return users

    def get_all_users_info_of_depart(self, depart_id, start, limit,
                            need_check_depart_exist=True, only_user_id=False):
        """
        获取指定组织/部门下的所有用户
        """
        if depart_id == NCT_ALL_USER_GROUP:
            return self.user_manage.get_all_users(start, limit)

        # 检查部门是否存在
        if need_check_depart_exist:
            self.check_depart_exists(depart_id, True)
        db_users = []
        limit_statement = check_start_limit(start, limit)

        depart_path = self.get_department_path_by_dep_id(depart_id)
        sql = """
        SELECT `user`.`f_user_id`, `user`.`f_login_name`,
            `user`.`f_display_name`, `user`.`f_remark`, `user`.`f_password`,
            `user`.`f_des_password`, `user`.`f_sha2_password`, `user`.`f_mail_address`,
            `user`.`f_tel_number`, `user`.`f_idcard_number`, `user`.`f_expire_time`,
            `user`.`f_auth_type`, `user`.`f_status`,
            `user`.`f_priority`, `user`.`f_csf_level`, `user`.`f_csf_level2`,
            `user`.`f_pwd_control`, `user`.`f_oss_id`,
            `user`.`f_freeze_status`,
            `user`.`f_create_time`,
            `user`.`f_auto_disable_status`,
            `dept`.`f_name`
        FROM `t_user` AS `user`
        JOIN `t_user_department_relation` AS `relation`
            ON `relation`.`f_user_id` = `user`.`f_user_id`
        JOIN `t_department` AS `dept`
            ON `relation`.`f_path` = `dept`.`f_path`
        WHERE `relation`.`f_path` like %s
            AND `relation`.`f_user_id` != '{0}'
            AND `relation`.`f_user_id` != '{1}'
            AND `relation`.`f_user_id` != '{2}'
            AND `relation`.`f_user_id` != '{3}'
        ORDER BY f_priority, upper(`f_display_name`)
        {4}
        """.format(NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM,
                   NCT_USER_SECURIT, limit_statement)
        path_data = depart_path + '%%'

        if depart_id == NCT_UNDISTRIBUTE_USER_GROUP:
            sql = """
            SELECT `user`.`f_user_id`, `user`.`f_login_name`,
                `user`.`f_display_name`, `user`.`f_remark`, `user`.`f_password`,
                `user`.`f_des_password`, `user`.`f_sha2_password`, `user`.`f_mail_address`,
                `user`.`f_tel_number`, `user`.`f_idcard_number`, `user`.`f_expire_time`,
                `user`.`f_auth_type`, `user`.`f_status`,
                `user`.`f_priority`, `user`.`f_csf_level`, `user`.`f_csf_level2`,
                `user`.`f_pwd_control`, `user`.`f_oss_id`,
                `user`.`f_freeze_status`,
                `user`.`f_create_time`,
                `user`.`f_auto_disable_status`
            FROM `t_user` AS `user`
            JOIN `t_user_department_relation` AS `relation`
                ON `relation`.`f_user_id` = `user`.`f_user_id`
            WHERE `relation`.`f_path` = %s
                AND `relation`.`f_user_id` != '{0}'
                AND `relation`.`f_user_id` != '{1}'
                AND `relation`.`f_user_id` != '{2}'
                AND `relation`.`f_user_id` != '{3}'
            ORDER BY f_priority, upper(`f_display_name`)
            {4}
            """.format(NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM,
                       NCT_USER_SECURIT, limit_statement)
            path_data = depart_path
        db_users = self.r_db.all(sql, path_data)

        users = []
        if only_user_id:
            return [db_user['f_user_id'] for db_user in db_users]

        for db_user in db_users:
            if '-1' == depart_id:
                db_user['f_name'] = _("undistributed user")
            if db_user['f_password'] != '':
                db_user['originalPwd'] = True if self.initAdminPwd == db_user['f_password'] else False
            elif db_user['f_sha2_password'] != '':
                db_user['originalPwd'] = True if self.initSha2AdminPwd == db_user['f_sha2_password'] else False
            db_user['parentDepartId'] = depart_id
            users.append(self.user_manage.convert_user_info(db_user))
        # 填充用户所属部门信息
        self.user_manage.fill_user_departments(users)
        # 填充用户角色信息
        self.user_manage.fill_user_roles(users)
        return users

    def get_all_users_of_depart(self, depart_id, only_user_id =True):
        """
        获取指定组织或部门下的所有用户
        """
        user_ids = []
        # 去重
        if only_user_id:
            depart_path = self.get_department_path_by_dep_id(depart_id)
            if depart_path == '':
                return user_ids
            sql = """
            SELECT f_user_id from t_user_department_relation
            WHERE f_path like '{0}%%'
            """.format(depart_path)
            db_users = self.r_db.all(sql)

            for db_user in db_users:
                user_ids.append(db_user['f_user_id'])

            return list(set(user_ids))
        else:
            user_ids.extend(self.get_all_users_info_of_depart(depart_id, 0, -1, False, only_user_id))

            return list(remove_duplicate_item_from_list(user_ids, lambda user: user.id))

    def __search_for_all_users_count(self, search_key):
        """
        从所有用户搜索
        """

        sql = """
        SELECT count(*) as cnt
        FROM t_user
        WHERE f_user_id <> '{0}'
            AND f_user_id <> '{1}'
            AND f_user_id <> '{2}'
            AND f_user_id <> '{3}'
        AND (f_login_name LIKE %s ESCAPE '\\\\' OR f_display_name LIKE %s ESCAPE '\\\\' OR f_remark LIKE %s ESCAPE '\\\\')
        """.format(NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT)

        result = self.r_db.one(sql, search_key, search_key, search_key)
        return result['cnt']

    def __search_for_all_users(self, search_key, start, limit):
        """
        从所有用户搜索
        """
        limit_statement = check_start_limit(start, limit)
        sql = """
        SELECT u.`f_user_id`, u.`f_login_name`, u.`f_display_name`, u.`f_remark`, u.`f_password`,
            u.`f_des_password`, u.`f_sha2_password`,u.`f_mail_address`, u.`f_auth_type`, u.`f_status`,
            u.`f_priority`, u.`f_csf_level`, u.`f_pwd_control`, u.`f_oss_id`, u.`f_freeze_status`,
            u.`f_create_time`, u.`f_auto_disable_status`
        FROM `t_user` as u
        WHERE u.`f_user_id` <> '{0}'
            AND u.`f_user_id` <> '{1}'
            AND u.`f_user_id` <> '{2}'
            AND u.`f_user_id` <> '{3}'
        AND (`f_login_name` LIKE %s ESCAPE '\\\\' OR `f_display_name` LIKE %s ESCAPE '\\\\' OR `f_remark` LIKE %s ESCAPE '\\\\')
        """.format(NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT)

        # 增加搜索排序子句
        order_by_str = generate_search_order_sql(['f_login_name', 'f_display_name'])
        sql = sql + ' order by ' + order_by_str

        # 添加limit子句
        sql = sql + limit_statement

        # 去掉searchkey中的%
        convert_key = search_key[1:-1]

        db_users = self.r_db.all(sql, search_key, search_key, search_key,
                                 convert_key, convert_key, search_key, search_key)

        user_ids = []
        for db_user in db_users:
            user_ids.append(db_user['f_user_id'])
            if db_user['f_password'] != '':
                db_user['originalPwd'] = True if self.initAdminPwd == db_user['f_password'] else False
            elif db_user['f_sha2_password'] != '':
                db_user['originalPwd'] = True if self.initSha2AdminPwd == db_user['f_sha2_password'] else False

        # 获取用户的直属部门和ID
        user_deps_infos = self.user_manage.get_users_parent_deps(user_ids)
        for db_user in db_users:
            user_id = db_user['f_user_id']
            if user_id in user_deps_infos:
                db_user['depart_ids'] = '|'.join(user_deps_infos[user_id]['depart_ids'])
                db_user['depart_names'] = '|'.join(user_deps_infos[user_id]['depart_names'])
            else:
                db_user['depart_ids'] = None
                db_user['depart_names'] = None

        return db_users

    def search_department_of_users(self, depart_id, search_key, start,
                                   limit, search_elem="f_login_name"):
        """
        搜索用户
        """
        # 转义通配符
        search_key = search_key.strip().replace("%", "\\%").replace("_", "\\_")
        search_key = "%%%s%%" % search_key
        db_users = []
        # 全部用户不在关系表，所以需要单独处理
        if depart_id == NCT_ALL_USER_GROUP:
            db_users = self.__search_for_all_users(search_key, start, limit)
        elif depart_id == NCT_UNDISTRIBUTE_USER_GROUP:
            db_users = self.__search_users_from_undistribute_department(search_key, start, limit)
        else:
            # 判断部门是否存在
            self.check_depart_exists(depart_id, True)
            db_users = self.__search_users_from_all_department(depart_id, search_key, start, limit)

        users = []
        for db_user in db_users:
            users.append(self.user_manage.convert_user_info(db_user))
        # 填充用户所属部门信息
        if depart_id != NCT_ALL_USER_GROUP:
            self.user_manage.fill_user_departments(users)
        # 填充用户角色信息
        self.user_manage.fill_user_roles(users)
        return users

    def __count_search_users_from_all_department(self, depart_id, search_key, start, limit):
        """
        从所有部门中搜索用户
        """
        depart_id_search = "%%%s%%" % depart_id

        sql = """
        SELECT COUNT(distinct u.f_user_id) as cnt
        FROM `t_user` AS `u`
        INNER JOIN `t_user_department_relation` AS `relation`
            ON `relation`.`f_user_id` = `u`.`f_user_id`
        WHERE `u`.`f_user_id` != '{0}'
            AND `u`.`f_user_id` != '{1}'
            AND `u`.`f_user_id` != '{2}'
            AND `u`.`f_user_id` != '{3}'
            AND `relation`.`f_path` LIKE %s
            AND (`u`.`f_login_name` LIKE %s ESCAPE '\\\\' OR `u`.`f_display_name` LIKE %s ESCAPE '\\\\' OR `u`.`f_remark` LIKE %s ESCAPE '\\\\')
        """.format(NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT)
        db_count = self.r_db.one(sql, depart_id_search, search_key, search_key, search_key)
        return db_count['cnt']

    def __count_search_users_from_undistribute_department(self, search_key, start, limit):
        """
        从未分配中搜索用户
        """
        where_clause = "AND `relation`.`f_department_id` in ('-1')"
        sql = """
        SELECT COUNT(*) AS cnt
        FROM `t_user` AS `u`
        INNER JOIN `t_user_department_relation` AS `relation`
            ON `relation`.`f_user_id` = `u`.`f_user_id`
            {0}
            AND (`u`.`f_login_name` LIKE %s ESCAPE '\\\\' OR `u`.`f_display_name` LIKE %s ESCAPE '\\\\' OR `u`.`f_remark` LIKE %s ESCAPE '\\\\')
        WHERE `u`.`f_user_id` != '{1}'
            AND `u`.`f_user_id` != '{2}'
            AND `u`.`f_user_id` != '{3}'
            AND `u`.`f_user_id` != '{4}'
        """.format(where_clause, NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT)
        db_count = self.r_db.one(sql, search_key, search_key, search_key)
        return db_count['cnt']

    def __search_users_from_undistribute_department(self, search_key, start, limit):
        """
        从未分配中搜索用户
        """
        limit_statement = check_start_limit(start, limit)

        where_clause = "AND `relation`.`f_department_id` in ('-1')"
        sql = """
        SELECT u.`f_user_id`, u.`f_login_name`, u.`f_display_name`, u.`f_remark`,
            u.`f_password`, u.`f_des_password`, u.`f_sha2_password`, u.`f_mail_address`, u.`f_auth_type`,
            u.`f_status`, u.`f_priority`, u.`f_csf_level`, u.`f_pwd_control`, u.`f_oss_id`,
            u.`f_create_time`, u.`f_auto_disable_status`
        FROM `t_user` AS u
        INNER JOIN `t_user_department_relation` AS `relation`
            ON `relation`.`f_user_id` = `u`.`f_user_id`
            {0}
            AND (`u`.`f_login_name` LIKE %s ESCAPE '\\\\' OR `u`.`f_display_name` LIKE %s ESCAPE '\\\\' OR `u`.`f_remark` LIKE %s ESCAPE '\\\\')
        WHERE `u`.`f_user_id` != '{1}'
            AND `u`.`f_user_id` != '{2}'
            AND `u`.`f_user_id` != '{3}'
            AND `u`.`f_user_id` != '{4}'
        """.format(where_clause, NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT,
                   search_key, search_key)

        # 增加搜索排序子句
        order_by_str = generate_search_order_sql(['f_login_name', 'f_display_name'])
        sql = sql + ' order by ' + order_by_str

        # 添加limit子句
        sql = sql + limit_statement

        # 去掉searchkey中的%
        convert_key = search_key[1:-1]

        db_users = self.r_db.all(sql, search_key, search_key, search_key,
                                 convert_key, convert_key, search_key, search_key)

        for db_user in db_users:
            if db_user['f_password'] != '':
                db_user['originalPwd'] = True if self.initAdminPwd == db_user['f_password'] else False
            elif db_user['f_sha2_password'] != '':
                db_user['originalPwd'] = True if self.initSha2AdminPwd == db_user['f_sha2_password'] else False
            db_user['f_name'] = _("undistributed user")
            db_user['parentDepartId'] = NCT_UNDISTRIBUTE_USER_GROUP
        return db_users

    def __search_users_from_all_department(self, depart_id, search_key, start, limit):
        """
        从部门中搜索用户(包括子部门)
        """
        limit_statement = check_start_limit(start, limit)

        sql = """
        SELECT distinct u.`f_user_id`, u.`f_login_name`, u.`f_display_name`, u.`f_remark`,
            u.`f_password`, u.`f_des_password`, u.`f_sha2_password`, u.`f_mail_address`, u.`f_auth_type`,
            u.`f_status`, u.`f_priority`, u.`f_csf_level`, u.`f_pwd_control`, u.`f_oss_id`,
            u.`f_create_time`, u.`f_auto_disable_status`
        FROM `t_user` AS u
        INNER JOIN `t_user_department_relation` AS `relation`
            ON `relation`.`f_user_id` = `u`.`f_user_id`
        WHERE `u`.`f_user_id` != '{0}'
            AND `u`.`f_user_id` != '{1}'
            AND `u`.`f_user_id` != '{2}'
            AND `u`.`f_user_id` != '{3}'
            AND `relation`.`f_path` LIKE %s
            AND (`u`.`f_login_name` LIKE %s ESCAPE '\\\\' OR `u`.`f_display_name` LIKE %s ESCAPE '\\\\' OR `u`.`f_remark` LIKE %s ESCAPE '\\\\')
        """.format(NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT)

        # 增加搜索排序子句
        order_by_str = generate_search_order_sql(['f_login_name', 'f_display_name'])
        sql = sql + ' order by ' + order_by_str

        # 添加limit子句
        sql = sql + limit_statement

        # 去掉searchkey中的%
        convert_key = search_key[1:-1]

        depart_id_search = "%%%s%%" % depart_id

        db_users = self.r_db.all(sql, depart_id_search, search_key, search_key, search_key,
                                 convert_key, convert_key, search_key, search_key)

        for db_user in db_users:
            if db_user['f_password'] != '':
                db_user['originalPwd'] = True if self.initAdminPwd == db_user['f_password'] else False
            elif db_user['f_sha2_password'] != '':
                db_user['originalPwd'] = True if self.initSha2AdminPwd == db_user['f_sha2_password'] else False

        return db_users

    def search_department_of_users_by_displayname(self, depart_id, search_key, start, limit):
        """
        以用户显示名为搜索项搜索用户
        """
        return self.search_department_of_users(depart_id, search_key,
                                               start, limit, "f_display_name")

    def search_depart_by_name(self, parent_id, depart_name):
        """
        根据子部门名，获取某个部门下的子部门或孙子部门的id，返回找到的第一个部门名
        """
        depart_id = self.get_sub_depart_id_by_name(parent_id, depart_name)
        if depart_id:
            return depart_id
        else:
            sub_departs = self.get_one_level_sub_depart(parent_id)
            for depart in sub_departs:
                depart_id = self.search_depart_by_name(depart.departmentId, depart_name)
                if depart_id:
                    return depart_id

    def search_depart_by_key(self, user_id, key, start, limit, roleId=None):
        """
        根据关键字搜索部门
        """
        self.user_manage.check_user_exists(user_id)
        if key == "":
            return []

        limit_statement = check_start_limit(start, limit)

        esckey = "%%%s%%" % escape_key(key)

        user_roles = self.__get_user_role_id(user_id)
        can_see_all_roles = [NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_ADMIN,
                             NCT_SYSTEM_ROLE_SECURIT, NCT_SYSTEM_ROLE_AUDIT]

        can_see_roles = []
        can_see_roles.extend(can_see_all_roles)
        can_see_roles.append(NCT_SYSTEM_ROLE_ORG_MANAGER)
        can_see_roles.append(NCT_SYSTEM_ROLE_ORG_AUDIT)

        # 若用户不能看组织结构，则返回空
        if not (set(user_roles) & set(can_see_roles)):
            return []

        if set(user_roles) & set(can_see_all_roles):
            sql = """
            SELECT DISTINCT `f_department_id`, `f_name`
            FROM `t_department`
            WHERE `f_name` LIKE %s
            ORDER BY upper(`f_name`)
            """
        else:
            departids = self.get_supervisory_all_departids(user_id, roleId)
            if(len(departids) == 0):
                return []
            groupStr = generate_group_str(departids)
            sql = """
            SELECT DISTINCT `f_department_id`, `f_name`
            FROM `t_department`
            WHERE f_department_id in ({0}) AND `f_name` LIKE %s
            ORDER BY upper(`f_name`)
            """.format(groupStr)

        # 添加limit子句
        sql = sql + limit_statement

        depart_ids = self.r_db.all(sql, esckey)
        result = []
        for depart_id in depart_ids:
            result.append(self.get_department_info(depart_id['f_department_id'], True, True))
        return result

    def count_serach_department_of_users(self, depart_id, search_key):
        """
        获取搜索到的用户数
        """
        self.check_depart_exists(depart_id, include_organ=True)
        count = 0
        search_key = search_key.strip().replace("%", "\\%").replace("_", "\\_")
        search_key = "%%%s%%" % search_key

        if depart_id == NCT_ALL_USER_GROUP:
            count = self.__search_for_all_users_count(search_key)
        elif depart_id == NCT_UNDISTRIBUTE_USER_GROUP:
            count = self.__count_search_users_from_undistribute_department(search_key, 0, -1)
        else:
            count = self.__count_search_users_from_all_department(depart_id, search_key, 0, -1)

        return count

    def get_third_id2dept_id_dict(self):
        """
        获取第三方id和部门id的影射
        """
        sql = """
        select f_third_party_id,f_department_id from t_department
        where f_auth_type = 3
        """
        results = self.r_db.all(sql)

        ret_infos = {}
        for result in results:
            ret_infos[result["f_third_party_id"]] = result["f_department_id"]

        return ret_infos

    def get_user_ids_by_admin_id(self, responsible_person_id):
        """
        根据组织管理员id获取其管辖范围内的所有用户id
        """
        # 获取用户管理的所有部门id
        select_sql = """
            SELECT `f_department_id` FROM `t_department_responsible_person`
            WHERE `f_user_id` = %s
        """
        results = self.r_db.all(select_sql, responsible_person_id)

        user_ids = []
        for result in results:
            tmp_user_ids = self.get_all_users_of_depart(result['f_department_id'])
            # 同一个用户只占用一次管理员的空间，求并集
            user_ids = list(set(tmp_user_ids).union(user_ids))

        return user_ids

    def get_deepest_departs(self, depart_id):
        """
        获取部门下的最深层部门
        depart_id: string 要获取的部门ID
        # 组织结构如下：
        # ——组织结构
        #   |——一级部门1
        #    |    |——二级部门1
        #    |    |——二级部门2
        #    |    |——二级部门3
        #    |        |——三级部门1
        #    |        |——三级部门2
        #    |        |——三级部门3
        #    |——一级部门2
        返回结果：
        [一级部门2,二级部门1,二级部门2,三级部门1,三级部门2,三级部门3]
        """
        sql = """
        SELECT `depart`.`f_department_id`, `depart`.`f_name`, `depart`.`f_oss_id`, `depart`.`f_mail_address`,
            `relation`.`f_parent_department_id` AS `parent_id`,
            `parent`.`f_name` AS `parent_name`
        FROM `t_department_relation` AS `relation`
        JOIN `t_department` AS `depart`
        ON `depart`.`f_department_id` = `relation`.`f_department_id`
        JOIN `t_department` AS `parent`
        ON `parent`.`f_department_id` = `relation`.`f_parent_department_id`
        WHERE `relation`.`f_parent_department_id` = %s
        order by `depart`.`f_priority`, upper(`depart`.`f_name`)
        """
        db_departs = self.r_db.all(sql, depart_id)
        depart_list = self.fetch_departs(db_departs)
        deepest_depart_list = list()

        # 转换为双端队列，提高处理性能
        depart_queue = deque(depart_list)
        try:
            while True:
                depart = depart_queue.popleft()
                # 从数据库获取这个部门的子部门
                db_departs = self.r_db.all(sql, depart.departmentId)

                # 将没有子部门的最深层部门加入返回队列
                if not db_departs:
                    deepest_depart_list.append(depart)

                sub_departs = self.fetch_departs(db_departs)
                # 将获取到的子部门添加到队列，后面继续获取
                depart_queue.extend(sub_departs)

        # 所有部门的子部门获取完毕，则会丢出异常
        except IndexError:
            return deepest_depart_list

    def get_depart_tree_of_user(self, user_id):
        """
        获取用户相关的部门树
        organ1--
            |--depart1
            |        |--user1
            |        |--user2
            |
            |--depart2
            |      |--user1
            |      |--user3
            |
        organ2--
            |
            |--depart3
            |         |--user1
            |         |--user3
            |
            |--depart4
                   |--user3
                   |--user4

        user3相关的部门树如下:
        organ1--
            |
            |--depart2
            |
        organ2--
            |
            |--depart3
            |
            |--depart4
        """

        depart_tree = {}
        depart_ids = self.user_manage.get_departments_from_user(user_id).departmentIds

        while depart_ids:
            parent_ids = []
            for depart_id in depart_ids:

                # 未分配用户没有部门
                if depart_id == NCT_UNDISTRIBUTE_USER_GROUP:
                    continue

                depart_info = self.get_department_info(depart_id, b_include_org=True)
                depart_info.subDepartIds = []

                # 没有添加过该部门
                if depart_id not in depart_tree:
                    # 添加部门进部门树
                    depart_tree[depart_id] = depart_info

                # 存在父部门时将父部们加入部门树
                if depart_info.parentDepartId:
                    parent_ids.append(depart_info.parentDepartId)

                    # 部门树没有该部门的父部门时添加父部门
                    if depart_info.parentDepartId not in depart_tree:
                        depart_tree[depart_info.parentDepartId] = self.get_department_info(depart_info.parentDepartId, b_include_org=True)

                        # 初始化子部门ID列表
                        depart_tree[depart_info.parentDepartId].subDepartIds = [depart_id]
                    else:
                        depart_tree[depart_info.parentDepartId].subDepartIds.append(depart_id)

            depart_ids = parent_ids
        return depart_tree

    def check_oss_id(self, ossId):
        """
        检查存储id
        """
        ossInfo = get_oss_info(ossId)

        # 空字符串合法，代表没有配置存储
        if ossId and not ossInfo:
            raise_exception(exp_msg=(_("IDS_OSS_NOT_EXIST") % (ossId)),
                            exp_num=ncTShareMgntError.NCT_OSS_NOT_EXIST)

        if ossId and ossInfo.enabled is False:
            raise_exception(exp_msg=_("IDS_OSS_HAS_BEEN_DISABLED"),
                            exp_num=ncTShareMgntError.NCT_OSS_HAS_BEEN_DISABLED)

    def get_batch_depart_infos_by_id(self, batch_departids):
        """
        根据部门id批量获取部门信息，返回dict
        """
        if len(batch_departids) == 0:
            return {}

        sql = """
        SELECT f_department_id,f_name FROM t_department WHERE f_department_id IN (%s)
        """ % ','.join(["'{0}'".format(value) for value in batch_departids])

        ret_dict = {}
        results = self.r_db.all(sql)
        for info in results:
            ret_dict[info["f_department_id"]] = info

        return ret_dict

    def delete_responsible_person(self, user_id, depart_id):
        """
        删除部门负责人
        """

        self.user_manage.check_user_exists(user_id)
        self.check_depart_exists(depart_id, include_organ=True)

        sql = """
        DELETE FROM `t_department_responsible_person`
        WHERE `f_user_id` = %s AND `f_department_id` = %s
        """
        self.w_db.query(sql, user_id, depart_id)

    def get_depart_id_by_name(self, name, parent_id):
        """
        根据部门名获取部门id
        """
        sql = """
        SELECT `depart`.`f_department_id` FROM `t_department` AS `depart`
        JOIN `t_department_relation` AS `relation`
        ON `depart`.`f_department_id` = `relation`.`f_department_id`
        WHERE `relation`.`f_parent_department_id` = %s
            AND `depart`.`f_name` = %s
        """
        result = self.r_db.one(sql, parent_id, name)
        if result:
            return result['f_department_id']

    def get_department_path_by_dep_id(self, depart_id):
        """
        根据部门id获取部门全路径
        """
        if depart_id == NCT_UNDISTRIBUTE_USER_GROUP:
            return '-1'

        sql = """
        SELECT `f_path` FROM `t_department` WHERE `f_department_id` = %s
        """
        result = self.r_db.one(sql, depart_id)

        if result:
            return result['f_path']

        return ''

    def get_department_path_by_user_id(self, user_id):
        """
        根据用户id获取部门全路径
        """
        sql = """
        SELECT `f_path` FROM `t_user_department_relation`
        WHERE `f_user_id` = %s
        """

        results = self.r_db.all(sql, user_id)
        path_list = []
        if results:
            path_list = [res['f_path'] for res in results]

        return path_list

    def get_parent_department_path(self, depart_path):
        """
        根据当前部门路径获取直系父部门路径
        """
        if depart_path:
            end_pos = depart_path.rfind('/')
            parent_path = depart_path[:end_pos]

            return parent_path

        return ''
    def get_ou_id_by_depart_path(self, depart_path):
        """
        根据当前部门path获取直接组织id
        """
        if depart_path:
            id_list = depart_path.split('/')
            return id_list[0]

        return ''

    def check_has_belong_to_dep(self, src_depart_id, dest_depart_id):
        """
        判断部门是否为另外一个部门或者其下子部门
        """
        all_dept_ids = self.get_all_departids(src_depart_id)

        if dest_depart_id in all_dept_ids:
            return True

        return False
