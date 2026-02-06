#!/usr/bin/python3
# -*- coding:utf-8 -*-
import os
import json
import time
from datetime import datetime
from collections import deque
from src.third_party_auth.third_config_manage import ThirdConfigManage
from src.common.business_date import BusinessDate
from src.common import global_info
from src.common.http import pub_nsq_msg
from src.common.sharemgnt_logger import ShareMgnt_Log
from src.common.db.connector import DBConnector
from src.common.lib import (raise_exception,
                            check_email,
                            check_name)
from src.modules.ossgateway import get_oss_info
from src.modules.department_manage import DepartmentManage
from src.modules.user_manage import UserManage
from src.modules.config_manage import ConfigManage
from src.modules.role_manage import RoleManage
from ShareMgnt.constants import (NCT_UNDISTRIBUTE_USER_GROUP,
                                 NCT_ALL_USER_GROUP,
                                 NCT_USER_ADMIN)
from ShareMgnt.ttypes import *
from EThriftException.ttypes import ncTException
import threading

TOPIC_ORG_NAME_MODIFY = "core.org.name.modify"
TOPIC_USER_MODIFIED = "user_management.user.modified"


class ThirdImportManage(DBConnector):
    """
    第三方用户导入管理类
    """
    import_mutex = threading.Lock()

    def __init__(self):
        """
        初始化
        """
        self.ou_manage = None
        self.depart_manage = DepartmentManage()
        self.user_manage = UserManage()
        self.config_manage = ConfigManage()

    def __init_ou_module(self):
        """
        根据app_id获取用户管理组件
        """
        self.ou_manage = None

        try:
            # 二进制服务运行，从third_party_auth导入
            import_ou_str = "from third_party_auth.ou_manage import *"
            exec(import_ou_str)
        except ImportError:
            # 源码调试时，需要从src目录导入
            import_src_ou_str = "from src.third_party_auth.ou_manage import *"
            exec(import_src_ou_str)

        third_info = ThirdConfigManage().get_third_party_info_auth()
        if third_info and third_info.enabled and third_info.config:
            # 导入第三方插件组织结构解析模块
            ou_module = "/sysvol/plugin/auth_"+ str(third_info.indexId) +"/ou_module.py"
            if os.path.exists(ou_module):
                import_str = "from ou_module import *"
                exec(import_str)
            config = json.loads(third_info.config)

            if "ouModule" in self.server_info:
                self.ou_manage = locals()[config['ouModule']]()

    @property
    def server_info(self):
        """
        服务配置
        """
        third_info = ThirdConfigManage().get_third_party_info_auth()
        if third_info and third_info.enabled and third_info.config:
            return json.loads(third_info.config)

    def get_root_node(self, user_id):
        """
        获取第三方根组织节点
        """
        self.__init_ou_module()
        if not self.ou_manage:
            raise_exception(exp_msg=_("third party auth no open"),
                            exp_num=ncTShareMgntError.NCT_THIRD_PARTY_AUTH_NOT_OPEN)

        # 根据用户id获取第三方id
        if user_id != NCT_USER_ADMIN:
            third_id = self.user_manage.get_third_id_by_user_id(user_id)
        else:
            third_id = NCT_USER_ADMIN
        return self.ou_manage.get_root_node(self.server_info, third_id)

    def expand_node(self, third_id):
        """
        展开第三方节点
        """
        self.__init_ou_module()
        if not self.ou_manage:
            raise_exception(exp_msg=_("third party auth no open"),
                            exp_num=ncTShareMgntError.NCT_THIRD_PARTY_AUTH_NOT_OPEN)

        return self.ou_manage.expand_node(self.server_info, third_id)

    def import_ous(self, ous, users, option, responsiblePersonId):
        """
        导入选择的第三方组织结构和用户树
        """
        self.__init_ou_module()
        if not self.ou_manage:
            raise_exception(exp_msg=_("third party auth no open"),
                            exp_num=ncTShareMgntError.NCT_THIRD_PARTY_AUTH_NOT_OPEN)

        if ThirdImportManage.import_mutex.locked():
            raise_exception(exp_msg=_("THIRD_IMPORTING"),
                            exp_num=ncTShareMgntError.NCT_THREAD_IMPORTING)

        # 导入第三方用户组织方法加锁
        with ThirdImportManage.import_mutex:
            ou_depart_tree, ou_user_tree = self.ou_manage.get_selected_ous(self.server_info,
                                                                           ous, users)
            self.start_import_ous(
                ou_depart_tree, ou_user_tree, option, responsiblePersonId)

    def check_option(self, option):
        """
        检查导入选项
        """
        # 检查部门存在
        self.depart_manage.check_depart_exists(option.departmentId, True)
        if option.departmentId == NCT_UNDISTRIBUTE_USER_GROUP:
            raise_exception(exp_msg=_("could not import to undistribute"),
                            exp_num=ncTShareMgntError.
                            NCT_CANNOT_IMPORT_DOMAIN_USER_TO_UNDISTRIBUTE)

        if option.departmentId == NCT_ALL_USER_GROUP:
            raise_exception(exp_msg=_("could not import to all group"),
                            exp_num=ncTShareMgntError.
                            NCT_CANNOT_IMPORT_DOMAIN_USER_TO_ALL)

        # 检查用户账号有效期
        if option.expireTime is None:
            option.expireTime = -1
        else:
            if option.expireTime != -1 and option.expireTime < int(BusinessDate.time()):
                raise_exception(exp_msg=_("IDS_INVALID_DATE"),
                                exp_num=ncTShareMgntError.NCT_INVALID_DATETIME)

    def check_ous(self, ou_depart_tree, option):
        """
        检查部门是否已经导入过
        如果没有导入过或者已经导入过，并且导入目的一致，则通过验证
        如果导入目的不一致，则异常
        """
        # 只导入用户，部门会是空的
        if not ou_depart_tree:
            return

        #
        # 获取勾选的组织、部门中的第一级部门进行验证，
        # 部门第一级采用父部门不在队列或为-1
        #
        if "-1" in ou_depart_tree:
            self.check_ou_import_already("-1", option)
        else:
            for ou_id in ou_depart_tree:
                # 判断一级部门
                if ou_depart_tree[ou_id].parent_id in ou_depart_tree and \
                        ou_depart_tree[ou_id].parent_id != ou_id:
                    continue
                self.check_ou_import_already(ou_id, option)

    def check_ou_import_already(self, ou_id, option):
        """
        判断组织是否已经导入
        """
        sql = """
            SELECT `f_path`
            FROM `t_department`
            WHERE `f_third_party_id` = %s
            LIMIT 1
            """
        result = self.r_db.one(sql, ou_id)
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

    def start_import_ous(self, ou_depart_tree, ou_user_tree, option, responsible_person_id):
        """
        开始导入选择的第三方组织结构和用户树
        """
        ShareMgnt_Log('start import ous, ou_tree: %s, user_tree: %s, option: %s',
                      str(ou_depart_tree), str(ou_user_tree), str(option))

        # 检查导入选项
        self.check_option(option)

        # 检查组织管理员
        if (not self.user_manage.check_is_responsible_person(responsible_person_id) and
                not RoleManage().check_is_role_supper(responsible_person_id)):
            raise_exception(exp_msg=_("IDS_INVALID_MANAGER_ID"),
                            exp_num=ncTShareMgntError.NCT_INVALID_MANAGER_ID)

        # 检查部门
        self.check_ous(ou_depart_tree, option)

        # 初始化导入进度
        global_info.init_import_variable()
        global_info.IMPORT_TOTAL_NUM = len(ou_depart_tree) + len(ou_user_tree)

        option.oss_id = ""

        # 创建组织
        self.add_ous(ou_depart_tree, option)

        # 获取内置管理员名称列表
        self.admin_list = list(
            self.user_manage.get_all_admin_account().values())

        # 创建用户
        self.add_users(ou_user_tree, option, responsible_person_id)

    def add_ous(self, ou_depart_tree, option):
        """
        添加组织
        """
        # 获取root组织id
        src_ou_ids = []

        # 当-1存在时，-1即为根组织
        if "-1" in ou_depart_tree:
            src_ou_ids.insert(0, "-1")
        else:
            for ou_id in ou_depart_tree:
                # 获取第一级部门
                ou_info = ou_depart_tree[ou_id]
                if ou_info.parent_id not in ou_depart_tree or \
                        ou_info.parent_id == ou_id:
                    src_ou_ids.append(ou_id)

        # 按广度优先获取所有需要同步的第三方ouid
        ou_queue = deque(src_ou_ids)
        try:
            while True:
                ou_id = ou_queue.popleft()
                sub_ou_ids = ou_depart_tree[ou_id].sub_third_ou_ids
                src_ou_ids.extend(sub_ou_ids)
                ou_queue.extend(sub_ou_ids)
        # 所有子部门获取完毕，则会丢异常
        except IndexError:
            pass

        for ou_id in src_ou_ids:
            try:
                ou_info = ou_depart_tree[ou_id]
                depart_id = NCT_UNDISTRIBUTE_USER_GROUP
                # 如果是第一级部门
                if "-1" in ou_depart_tree:
                    # 如果-1在组织树里则将所有没有父部门的放置-1目录下
                    ou_info.parent_id = ou_info.parent_id if ou_info.parent_id else "-1"

                # 如果自身id与父部门id相同，则认为是根组织放到目的部门下
                if ou_id == ou_info.parent_id:
                    depart_id = option.departmentId
                else:
                    sql = """
                    SELECT `f_department_id` FROM `t_department`
                    WHERE `f_third_party_id` = %s
                    """
                    db_depart = self.r_db.one(sql, ou_info.parent_id)
                    if not db_depart:
                        depart_id = option.departmentId
                    else:
                        depart_id = db_depart['f_department_id']

                # 检查部门名是否合法
                if not check_name(ou_info.ou_name):
                    msg = _("IDS_INVALID_DEPART_NAME")
                    raise Exception(msg)

                self.add_ou(ou_info, depart_id, option.oss_id)
                global_info.IMPORT_SUCCESS_NUM += 1

            except Exception as ex:
                # 接口调用异常
                if isinstance(ex, ncTException):
                    msg = "add ou failed: %s,%s, error: %s" % \
                        (ou_info.third_id, ou_info.ou_name, ex.expMsg)
                # 其他异常
                else:
                    msg = "add ou failed: %s,%s, error: %s" % \
                        (ou_info.third_id, ou_info.ou_name, str(ex))

                ShareMgnt_Log(msg)

                global_info.IMPORT_FAIL_NUM += 1
                global_info.IMPORT_FAIL_INFO.append(msg)

    def add_ou(self, ou_info, parent_id, oss_id):
        """
        添加组织单元
        """
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
        result = self.r_db.one(sql, parent_path + depart_str, ou_info.ou_name)

        # 存在同名部门，覆盖
        if result:
            sql = """
            UPDATE `t_department`
            SET `f_auth_type` = %s, `f_third_party_id` = %s
            WHERE `f_department_id` = %s
            """
            self.w_db.query(sql, ou_info.type,
                            ou_info.third_id, result['f_department_id'])

            ShareMgnt_Log('edit ou success(name exists), %s,%s',
                          ou_info.third_id, ou_info.ou_name)
        else:
            sql = """
            SELECT f_department_id FROM `t_department`
            WHERE `f_third_party_id` = %s
            """
            # 已导入过部门，则更新
            db_object = self.r_db.one(sql, ou_info.third_id)

            if db_object:
                sql = """
                UPDATE `t_department`
                SET `f_name` = %s
                WHERE `f_third_party_id` = %s
                """
                self.w_db.query(sql, ou_info.ou_name, ou_info.third_id)
                self.depart_manage.move_department(
                    db_object["f_department_id"], parent_id)

                # 发送部门显示名更新nsq消息
                pub_nsq_msg(TOPIC_ORG_NAME_MODIFY, {
                            "id": db_object["f_department_id"], "new_name": ou_info.ou_name, "type": "department"})

                ShareMgnt_Log('edit ou success(guid exists), %s,%s',
                              ou_info.third_id, ou_info.ou_name)
            else:
                self.depart_manage.add_depart_to_db(oss_id=oss_id,
                                                    parent_id=parent_id,
                                                    third_ou_info=ou_info)
                ShareMgnt_Log('add ou success, %s,%s',
                              ou_info.third_id, ou_info.ou_name)

    def add_users(self, ou_user_tree, option, responsible_person_id):
        """
        添加用户
        Args:
            ou_user_tree: 第三方用户树
            option: ncTUsrmImportOption 导入选项
        """
        def select_depart(third_ou_id):
            """
            为用户选择一个部门进行导入
            """
            sql = """
            SELECT `f_department_id` FROM `t_department`
            WHERE `f_third_party_id` = %s
            """
            db_depart = self.r_db.one(sql, third_ou_id)
            if not db_depart:
                depart_id = option.departmentId
            else:
                depart_id = db_depart['f_department_id']

            return depart_id

        for third_id in ou_user_tree:
            # 判断是否终止
            if global_info.IMPORT_IS_STOP:
                return
            try:
                user = ou_user_tree[third_id]
                # 用户是禁用状态，直接跳过， 不导入
                if user.status is False:
                    raise Exception('user is disable cannot import')

                # 是否导入邮箱
                if option.userEmail is False:
                    user.email = ''

                # 如果邮箱非法，设置为""
                user.email = user.email.strip()
                if (len(user.email) > 128 or not check_email(user.email)):
                    user.email = ""

                # 是否导入身份证号
                if option.userIdcardNumber is False:
                    user.idcard_number = ''

                # 自动选择导入部门
                depart_id = ""
                # 用户所属部门路径不是根路径
                if user.third_ou_ids and user.third_ou_ids[0] not in ["", "-1", None]:
                    depart_id = select_depart(user.third_ou_ids[0])
                # 否则使用导入选项中的部门
                else:
                    depart_id = option.departmentId

                # 是否导入显示名
                if option.userDisplayName is False:
                    user.display_name = user.login_name

                # 显示名为空，则默认为登录名
                if not user.display_name:
                    user.display_name = user.login_name

                # 检查是否存在同账号用户
                as_user = self.user_manage.get_user_by_loginname(
                    user.login_name)

                if as_user:
                    if option.userCover is True:
                        if user.login_name.lower() in self.admin_list:
                            raise_exception(exp_msg=_("duplicate login name"),
                                            exp_num=ncTShareMgntError.
                                            NCT_DUPLICATED_LOGIN_NAME)

                        # 覆盖用户
                        self.cover_user(as_user, user, depart_id)
                else:
                    # 第三方id相同，用户已存在，但是登录名变了, 这种情况下强制覆盖，不允许存在两个相同的第三方id，或考虑不导入
                    as_user = self.user_manage.get_user_by_third_id(
                        third_id, False)
                    if as_user:
                        self.cover_user(as_user, user, depart_id)
                    else:
                        if not user.oss_id:
                            user.oss_id = option.oss_id
                        user.expire_time = option.expireTime
                        self.add_user(user, depart_id, responsible_person_id)

                global_info.IMPORT_SUCCESS_NUM += 1
            except Exception as ex:
                if isinstance(ex, ncTException):
                    msg = "add user failed: %s,%s,%s, error: %s" % \
                        (user.third_id, user.login_name,
                         user.display_name, ex.expMsg)
                    if ex.expMsg == _("user num overflow"):
                        global_info.IMPORT_IS_STOP = True
                    if ex.errID == ncTShareMgntError.NCT_SPACE_ALLOCATED_FOR_USER_EXCEEDS_THE_MAX_LIMIT:
                        global_info.IMPORT_IS_STOP = True
                else:
                    msg = "add user failed: %s,%s,%s, error: %s" % \
                        (user.third_id, user.login_name, user.display_name, str(ex))

                ShareMgnt_Log(msg)

                global_info.IMPORT_FAIL_NUM += 1
                global_info.IMPORT_FAIL_INFO.append(msg)

    def cover_user(self, as_user, user_info, depart_id):
        """
        覆盖用户
        """
        # 检查显示名
        displayName = user_info.display_name
        if displayName.lower() != as_user.user.displayName.lower():
            displayName = self.user_manage.get_unique_displayname(
                displayName, as_user.id)

        # 检查密码不为None
        if user_info.password is None:
            user_info.password = ''

        # 检查用户信息
        user = ncTUsrmUserInfo()
        user.loginName = user_info.login_name
        user.displayName = displayName
        user.email = user_info.email
        user.idcardNumber = user_info.idcard_number
        user.userType = user_info.type
        self.user_manage.check_user(user)

        # 用户已存在，修改
        sql = """
        UPDATE `t_user` SET `f_login_name` = %s, `f_display_name` = %s,
        `f_password` = %s, `f_mail_address` = %s, `f_idcard_number` = %s,
        `f_pwd_timestamp` = %s, `f_third_party_id` = %s,
        `f_auth_type` = %s, `f_priority`=%s, `f_sha2_password`=%s
        WHERE `f_user_id` = %s
        """
        now = BusinessDate.now().strftime("%Y-%m-%d %H:%M:%S")
        self.w_db.query(sql,
                        user.loginName,
                        user.displayName,
                        user_info.password,
                        user.email,
                        user.idcardNumber,
                        now,
                        user_info.third_id,
                        user.userType,
                        user_info.priority,
                        "",
                        as_user.id)

        # 补充部门关联关系
        self.update_relation(as_user.id, depart_id)
        self.set_third_user_department_relation(
            as_user.id, user_info.third_ou_ids)

        if as_user.user.displayName != user.displayName:
            # 发送用户显示名更新nsq消息
            pub_nsq_msg(TOPIC_ORG_NAME_MODIFY, {
                        "id": as_user.id, "new_name": user.displayName, "type": "user"})

        user_modify_info = {}
        if as_user.user.email != user.email:
            user_modify_info["new_email"] = user.email

        if len(user_modify_info) > 0:
            # 发送用户信息更新nsq消息
            user_modify_info["user_id"] = as_user.id
            pub_nsq_msg(TOPIC_USER_MODIFIED, user_modify_info)

        # 记录日志
        ShareMgnt_Log("edit user success: %s,%s",
                      user_info.login_name,
                      user_info.display_name)

    def update_relation(self, user_id, depart_id):
        """
        更新已存在第三方用户的用户-部门关系
        Args:
            user_id: string 要更新的用户
            depart_id: string 要将用户分配到的部门
        """
        self.depart_manage.add_user_to_department([user_id], depart_id)

    def add_user(self, user_info, depart_id, responsible_person_id):
        """
        添加用户
        """
        user_info.display_name = \
            self.user_manage.get_unique_displayname(user_info.display_name)

        # 构造结构体
        add_user = ncTUsrmAddUserInfo()
        add_user.user = ncTUsrmUserInfo()
        add_user.user.loginName = user_info.login_name
        add_user.user.displayName = user_info.display_name
        add_user.user.email = user_info.email
        add_user.user.idcardNumber = user_info.idcard_number
        add_user.user.userType = user_info.type
        add_user.user.departmentIds = [depart_id]
        add_user.user.ossInfo = get_oss_info(
            user_info.oss_id) or ncTUsrmOSSInfo()
        add_user.user.expireTime = user_info.expire_time

        add_user.user.objectGUID = user_info.third_id
        add_user.user.priority = user_info.priority
        add_user.user.csfLevel = self.config_manage.get_min_csf_level()
        add_user.user.csfLevel2 = self.config_manage.get_min_csf_level2()
        add_user.user.pwdControl = 0
        if user_info.password:
            add_user.md5Password = user_info.password

        self.user_manage.check_user(add_user.user)

        user_id = self.user_manage.add_user_to_db(add_user)

        self.set_third_user_department_relation(
            user_id, user_info.third_ou_ids)

        ShareMgnt_Log("add user success: %s,%s,%s",
                      user_id, user_info.login_name, user_info.display_name)

    def set_third_user_department_relation(self, user_id, third_ou_ids):
        """
        补充设置用户部门关联关系
        """
        if not third_ou_ids:
            return
        # 根据第三方组织id获取部门信息
        for third_ou_id in third_ou_ids:
            sql = """
            SELECT `f_department_id` FROM `t_department`
            WHERE `f_third_party_id` = %s
            """
            db_depart = self.r_db.one(sql, third_ou_id)
            if db_depart:
                self.depart_manage.add_user_to_department(
                    [user_id], db_depart['f_department_id'])
