#!/usr/bin/python3
# -*- coding:utf-8 -*-
"""This is user manage class"""
import uuid
import csv
import datetime
import re
import json
import os
import collections
import time
import requests
import ulid
from eisoo.tclients import TClient
from src.common import global_info
from src.common.db.connector import DBConnector, ConnectorManager, safe_cursor
from src.common.db.db_manager import get_db_name
from src.common.lib import (raise_exception,
                            check_start_limit,
                            check_is_uuid,
                            check_email,
                            encrypt_pwd,
                            sha2_encrypt,
                            check_is_valid_password,
                            check_is_strong_password,
                            escape_format_percent,
                            generate_group_str,
                            ntlm_md4,
                            check_tel_number,
                            is_valid_string2,
                            is_valid_string,
                            merge_dicts)
from src.modules.ossgateway import get_oss_info
from src.common.eacp_log import eacp_log
from src.common.http import send_request, pub_nsq_msg
from src.common.sharemgnt_logger import ShareMgnt_Log
from src.common.encrypt.simple import (des_encrypt, des_decrypt,
                                       eisoo_rsa_decrypt,
                                       des_encrypt_with_padzero,
                                       des_decrypt_with_padzero)
from src.common.business_date import BusinessDate
from src.modules.config_manage import ConfigManage
from src.modules.vcode_manage import VcodeManage
from ShareMgnt.ttypes import (ncTUsrmGetUserInfo,
                              ncTUsrmUserInfo,
                              ncTUsrmUserType,
                              ncTUsrmUserStatus,
                              ncTShareMgntError,
                              ncTUsrmAddUserInfo,
                              ncTUsrmPasswordConfig,
                              ncTUsrmOSSInfo,
                              ncTUsrmDirectDeptInfo,
                              ncTUsrmPwdControlConfig,
                              ncTSimpleUserInfo,
                              ncTRoleInfo,
                              ncTVcodeType)
from ShareMgnt.constants import (NCT_UNDISTRIBUTE_USER_GROUP,
                                 NCT_USER_ADMIN,
                                 NCT_USER_AUDIT,
                                 NCT_USER_SYSTEM,
                                 NCT_USER_SECURIT,
                                 NCT_SYSTEM_ROLE_SUPPER,
                                 NCT_SYSTEM_ROLE_ADMIN,
                                 NCT_SYSTEM_ROLE_ORG_MANAGER)
from EThriftException.ttypes import ncTException
from EFAST.ttypes import ncTGetPageDocParam
from src.common.lib import (is_code_string)

USER_DISABLED = 0x00000040
USER_EXPIRE_DISABLED = 0x00000002   # 用户账号过期被禁用
MIN_STRONG_PWD_LENGTH = 8
MIN_STRONG_PWD_LENGTH2 = 10 #涉密要求的最小值
MAX_STRONG_PWD_LENGTH = 99
ACTIVE_INTERVAL_SECONDS = 300

TOPIC_USER_CREATE = "core.user_management.user.created"
TOPIC_USER_DELETE = "core.user.delete"
TOPIC_USER_FREEZE = "core.user.freeze"
TOPIC_USER_UNREALNAME = "core.user.unrealname"
TOPIC_ORG_NAME_MODIFY = "core.org.name.modify"
TOPIC_USER_MOVE = "user_management.user.moved"
TOPIC_DEPARTMENT_USER_ADD = "user_management.department.user.added"
TOPIC_DEPARTMENT_USER_REMOVE = "user_management.department.user.removed"
TOPIC_USER_CUSTOM_ATTR_MODIFIED = "user_management.user.custom_attr.modified"
TOPIC_USER_PASSWORD_MODIFIED = "user_management.user.password.modified"
TOPIC_USER_STATUS_CHANGED = "user_management.user.status.changed"
TOPIC_USER_MODIFIED = "user_management.user.modified"


class UserDefaultPassword:
    def __init__(self):
        """
        init
        """
        self.des_pwd = ""
        self.ntlm_pwd = ""
        self.sha2_pwd = ""
        self.md5_pwd = ""


class UserManage(DBConnector):
    """
    user manage
    """
    # 定义用户默认密码
    __user_default_passwd = None
    __admin_default_passwd = None

    def __init__(self):
        """
        init
        """
        self.config_manage = ConfigManage()
        self.vcode_manage = VcodeManage()
        self.tmp_group = _("IDS_TMP_PERSON_GROUP")
        self.initAdminPwd = "e10adc3949ba59abbe56e057f20f883e"
        self.initSha2AdminPwd = "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92"
        # 增加保留的admin账号，因为日志记录会包含admin显示账号信息
        self.remain_accounts = ['admin', 'audit', 'system', 'security']

    def _is_login_name_valid(self, name):
        """
        检查用户名是否符合规则
        1.必须为utf8编码
        2.不能包含 \ / * ? " < > | 特殊字符
        3.不能包含whitespace字符 \s，包括[\t\n\r\f\v]
        4.长度最大为128字节
        5.中广核去除:字符检查
        """
        if name is None:
            raise_exception(exp_msg=_("IDS_INVALID_LOGIN_NAME"),
                            exp_num=ncTShareMgntError.NCT_INVALID_LOGIN_NAME)

        # 正则匹配，不能包含 \ / * ? " < > | \s 特殊字符，且长度为[1, 128]
        # 不要问老子这里为什么不用is_valid_name，他么的AT测试用例竟然要求前后有
        # 空格的账号添加报错，他么的过滤前后空格不是基本操作吗？
        # 2019-07-23 袁晨思
        if re.match(r'^[^\\\/\*\?\"\<\>\|\s]{1,128}$', name) is None:
            raise_exception(exp_msg=_("IDS_INVALID_LOGIN_NAME"),
                            exp_num=ncTShareMgntError.NCT_INVALID_LOGIN_NAME)

        return name

    def _is_display_name_valid(self, name):
        """
        检查显示名是否符合规则，返回最后的名称
        1.utf8编码或者为unicode
        2.前后的空格会被除去，中间的空格会被保留
        3.不能包含 \ / * ? " < > | 特殊字符
        4.长度最大为128字节
        5.最后的..会被去除
        6.中广核去除:字符检查
        """
        if name is None:
            raise_exception(exp_msg=_("IDS_INVALID_DISPLAY_NAME"),
                            exp_num=ncTShareMgntError.NCT_INVALID_DISPLAY_NAME)

        # 除去前面的空格，末尾的空格和点
        striped_name = name.lstrip()
        striped_name = striped_name.rstrip(". ")

        if not is_valid_string2(striped_name):
            raise_exception(exp_msg=_("IDS_INVALID_DISPLAY_NAME"),
                            exp_num=ncTShareMgntError.NCT_INVALID_DISPLAY_NAME)

        return striped_name

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

    def is_email_valid(self, login_name, email):
        """
        检查用户邮箱
        """
        # 允许用户邮箱为空
        if not email:
            if self.get_olduserinfo_by_loginName(login_name):
                striped_email = self.get_olduserinfo_by_loginName(login_name)[
                    'f_mail_address']
                return striped_email
            else:
                return ""
        striped_email = email.strip()
        # 检查邮箱名是否合法
        if len(email) > 100 or not check_email(email):
            if self.get_olduserinfo_by_loginName(login_name):
                striped_email = self.get_olduserinfo_by_loginName(login_name)[
                    'f_mail_address']
                return striped_email
            else:
                return ""

        # 检查邮箱名是否冲突
        sql = """
        SELECT f_mail_address FROM t_user
        WHERE f_mail_address = %s and f_login_name != %s
        UNION
        SELECT f_mail_address FROM t_department
        WHERE f_mail_address = %s
        LIMIT 1
        """
        result = self.r_db.one(sql, striped_email, login_name, email)
        if result:
            if self.get_olduserinfo_by_loginName(login_name):
                striped_email = self.get_olduserinfo_by_loginName(login_name)[
                    'f_mail_address']
                return striped_email
            else:
                return ""
        return striped_email

    def add_is_idcardNumber_valid(self, idcardNumber, raiseExcep=False):
        """
        编辑检查身份证号是否符合规则，返回最后的身份证号
        1.必须为utf8编码
        2.前后的空格会被除去
        3.符合身份证号码的规则
        4.长度18字节
        """
        if not idcardNumber:
            return ""

        striped_idcardNumber = idcardNumber.strip()
        # 正则匹配，符合多种身份证号的规则
        if re.match(r'^[A-Za-z0-9/()-]{8,18}$', striped_idcardNumber) is None:
            if raiseExcep:
                raise_exception(exp_msg=_("IDS_INVALID_IDCARDNUMBER"),
                                exp_num=ncTShareMgntError.NCT_INVALID_IDCARDNUMBER)
            else:
                return ""

        striped_idcardNumber = bytes.decode(des_encrypt_with_padzero(global_info.des_key,
                                                                     striped_idcardNumber,
                                                                     global_info.des_key))

        return striped_idcardNumber

    def is_idcardNumber_valid(self, idcardNumber, loginName):
        """
        检查身份证号是否符合规则，返回最后的身份证号
        1.必须为utf8编码
        2.前后的空格会被除去
        3.符合身份证号码的规则
        4.长度18字节
        """
        # 如果身份证号不合法，保留之前数据
        if not idcardNumber:
            if self.get_olduserinfo_by_loginName(loginName):
                striped_idcardNumber = self.get_olduserinfo_by_loginName(loginName)[
                    'f_idcard_number']
                return striped_idcardNumber
            else:
                return ""

        striped_idcardNumber = idcardNumber.strip()
        # 正则匹配，符合多种身份证号的规则
        if re.match(r'^[A-Za-z0-9/()-]{8,18}$', striped_idcardNumber) is None:
            if self.get_olduserinfo_by_loginName(loginName):
                striped_idcardNumber = self.get_olduserinfo_by_loginName(loginName)[
                    'f_idcard_number']
                return striped_idcardNumber
            else:
                return ""
        striped_idcardNumber = bytes.decode(des_encrypt_with_padzero(global_info.des_key,
                                                                     striped_idcardNumber,
                                                                     global_info.des_key))

        return striped_idcardNumber

    def is_teleNumber_valid(self, teleNumber, loginName):
        """
        检查电话号是否符合规则，返回最后的电话号
        1.前后的空格会被除去
        2.符合电话号码的规则
        3.检查电话号码是否冲突
        """
        # 如果电话号码不合法，保留之前数据
        if not teleNumber:
            if self.get_olduserinfo_by_loginName(loginName):
                striped_teleNumber = self.get_olduserinfo_by_loginName(loginName)[
                    'f_tel_number']
                return striped_teleNumber
            else:
                return ""

        striped_teleNumber = teleNumber.strip()
        # 正则匹配，符合电话号码与手机号码的规择
        if re.match(r'^\d{1,20}$', striped_teleNumber) is None:
            if self.get_olduserinfo_by_loginName(loginName):
                striped_teleNumber = self.get_olduserinfo_by_loginName(loginName)[
                    'f_tel_number']
                return striped_teleNumber
            else:
                return ""

        sql = """
        SELECT `f_login_name` FROM `t_user`
        WHERE `f_tel_number` = %s
        AND `f_login_name` <> %s
        """
        result = self.r_db.one(sql, striped_teleNumber, loginName)
        if result:
            if self.get_olduserinfo_by_loginName(loginName):
                striped_teleNumber = self.get_olduserinfo_by_loginName(loginName)[
                    'f_tel_number']
                return striped_teleNumber
            else:
                return ""
        return striped_teleNumber

    def check_user_exists_by_idcardNumber_loginName(self, idcardNumber, loginName):
        """
        根据身份证检查用户是否存在
        """
        if not idcardNumber:
            return

        sql = """
        SELECT f_idcard_number FROM t_user
        WHERE f_idcard_number = %s and f_login_name != %s
        LIMIT 1
        """
        result = self.r_db.one(sql, idcardNumber, loginName)
        if result:
            raise_exception(exp_msg=_("duplicate idcardNumber"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_IDCARDNUMBER)
            return False
        return True

    def check_duplicate_email_by_login_name(self, email, login_name):
        if not email:
            return

        sql = """
        SELECT f_mail_address FROM t_user
        WHERE f_mail_address = %s and f_login_name != %s
        LIMIT 1
        """
        result = self.r_db.one(sql, email, login_name)
        if result:
            raise_exception(exp_msg=_("duplicate email"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_EMALI)

    def check_user_exists_by_idcardNumber_id(self, idcardNumber, userId):
        """
        根据身份证检查用户是否存在
        """
        if not idcardNumber:
            return

        sql = """
        SELECT f_idcard_number FROM t_user
        WHERE f_idcard_number = %s and f_user_id != %s
        LIMIT 1
        """
        result = self.r_db.one(sql, idcardNumber, userId)
        if result:
            raise_exception(exp_msg=_("duplicate idcardNumber"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_IDCARDNUMBER)
            return False
        return True

    def check_user(self, user, raiseExcep=False):
        """
        检测用户结构体合法性
        """
        # 检查用户登录名
        user.loginName = self._is_login_name_valid(user.loginName)

        # 检查用户显示名
        if not user.displayName:
            user.displayName = user.loginName
        user.displayName = self._is_display_name_valid(user.displayName)

        # 检查用户备注
        user.remark = self._is_remark_valid(user.remark)

        # 检查用户身份证号
        if not user.idcardNumber:
            user.idcardNumber = ""
        else:
            user.idcardNumber = self.add_is_idcardNumber_valid(
                user.idcardNumber, raiseExcep)
            self.check_user_exists_by_idcardNumber_loginName(
                user.idcardNumber, user.loginName)

        # 检查用户邮箱
        if not user.email:
            user.email = ""
        else:
            user.email = user.email.strip()
            if (len(user.email) > 128 or not check_email(user.email)):
                raise_exception(exp_msg=_("email illegal"),
                                exp_num=ncTShareMgntError.NCT_INVALID_EMAIL)
        # 默认本地用户
        if user.userType is None:
            user.userType = ncTUsrmUserType.NCT_USER_TYPE_LOCAL
        else:
            if user.userType not in list(ncTUsrmUserType._NAMES_TO_VALUES.values()):
                raise_exception(exp_msg=_("user type illegal"),
                                exp_num=ncTShareMgntError.NCT_INVALID_USER_TYPE)

        # 检查用户的排序权重
        if user.priority is not None:
            if user.priority < 1 or user.priority > 999:
                raise_exception(exp_msg=_("IDS_INVALID_USER_PRIORITY"),
                                exp_num=ncTShareMgntError.NCT_INVALID_USER_PRIORITY)

        # 检查用户部门
        if not user.departmentIds:
            user.departmentIds = [NCT_UNDISTRIBUTE_USER_GROUP]

        # 检查密级
        if user.csfLevel is not None:
            self.check_user_csflevel(user.csfLevel)

        # 检查密级2
        if user.csfLevel2 is not None:
            self.check_user_csflevel2(user.csfLevel2)

        # 检查存储
        if user.ossInfo is not None:
            if user.ossInfo.ossId is None or user.ossInfo.ossId == "null":
                user.ossInfo.ossId = ""
            self.check_oss_id(user.ossInfo.ossId)

        # 检查手机合法性
        if user.telNumber:
            self.check_user_tel_number(user.loginName, user.telNumber)

        # 检查用户账号有效期
        if user.expireTime is None:
            user.expireTime = -1
        else:
            if user.expireTime != -1 and user.expireTime < int(BusinessDate.time()):
                raise_exception(exp_msg=_("IDS_INVALID_DATE"),
                                exp_num=ncTShareMgntError.NCT_INVALID_DATETIME)

    def base_confict_with_lib_name(self, name):
        sql = f"""
        select f_id from {get_db_name('sharemgnt_db')}.t_reserved_name where f_name = %s
        """
        result = self.r_db.one(sql, name)
        if result:
            return True
        return False

    def is_confict_with_lib_name(self, name):
        """
        """
        if (self.base_confict_with_lib_name(name)):
            raise_exception(exp_msg=_("duplicate display name with doc-lib"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_DISPLAY_NAME_WITH_DOC_LIB)

    def is_confict_with_lib_name_display_name_error(self, name):
        """
        """
        if (self.base_confict_with_lib_name(name)):
            raise_exception(exp_msg=_("duplicate display name"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_DISPLAY_NAME)

    def check_user_display_name_exists(self, user_id, display_name):
        """
        检查用户显示名称是否存在
        """
        sql = """
        SELECT f_display_name FROM t_user
        WHERE f_display_name = %s and f_user_id != %s
        LIMIT 1
        """
        result = self.r_db.one(sql, display_name, user_id)
        if result:
            raise_exception(exp_msg=_("duplicate display name"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_DISPLAY_NAME)

    def check_user_email(self, user_id, email):
        """
        检查用户邮箱
        """
        # 允许用户邮箱为空
        if not email:
            return

        # 检查邮箱名是否合法
        if len(email) > 100 or not check_email(email):
            raise_exception(exp_msg=_("email illegal"),
                            exp_num=ncTShareMgntError.NCT_INVALID_EMAIL)

        # 检查邮箱名是否冲突
        sql = """
        SELECT f_mail_address FROM t_user
        WHERE f_mail_address = %s and f_user_id != %s
        UNION
        SELECT f_mail_address FROM t_department
        WHERE f_mail_address = %s
        LIMIT 1
        """
        result = self.r_db.one(sql, email, user_id, email)
        if result:
            raise_exception(exp_msg=_("duplicate email"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_EMALI)

    def check_user_priority(self, priority):
        """
        检查用户排序权重
        """
        if priority < 1 or priority > 999:
            raise_exception(exp_msg=_("IDS_INVALID_USER_PRIORITY"),
                            exp_num=ncTShareMgntError.NCT_INVALID_USER_PRIORITY)

    def check_user_csflevel2(self, csflevel2):
        """
        检查用户密级2
        """
        csf_levels = self.config_manage.get_csf_levels2()
        max_csf_level = max(csf_levels.values())
        min_csf_level = min(csf_levels.values())
        csf_level_list = list(csf_levels.values())
        if csflevel2 < min_csf_level or csflevel2 > max_csf_level or csflevel2 not in csf_levels.values():
            raise_exception(exp_msg=(_("IDS_INVALID_CSF_LEVEL2") % csf_level_list),
                            exp_num=ncTShareMgntError.NCT_INVALID_CSF_LEVEL2)

    def check_user_csflevel(self, csflevel):
        """
        检查用户密级
        """
        csf_levels = self.config_manage.get_csf_levels()
        max_csf_level = max(csf_levels.values())
        min_csf_level = min(csf_levels.values())
        csf_level_list = list(csf_levels.values())
        if csflevel < min_csf_level or csflevel > max_csf_level or csflevel not in csf_levels.values():
            raise_exception(exp_msg=(_("IDS_INVALID_CSF_LEVEL") % csf_level_list),
                            exp_num=ncTShareMgntError.NCT_INVALID_CSF_LEVEL)

    def check_user_exists_by_account(self, account):
        """
        根据用户名检查用户是否存在
        Args:
            account: 登录名
        """
        sql = """
        SELECT `f_login_name` FROM `t_user`
        WHERE `f_login_name` = %s
        LIMIT 1
        """
        result = self.r_db.one(sql, account)

        if result:
            errDetail = {}
            errDetail['type'] = 'user'
            raise_exception(exp_msg=_("duplicate login name"),
                             exp_num=ncTShareMgntError.NCT_DUPLICATED_LOGIN_NAME,
                             exp_detail=json.dumps(errDetail, ensure_ascii=False))

        # 检查与应用账户重名
        sql = f"""
        SELECT COUNT(*) AS cnt FROM `{get_db_name("user_management")}`.`t_app` WHERE `f_name` = %s
        """
        exist = self.r_db.one(sql, account)['cnt']

        if exist:
            errDetail = {}
            errDetail['type'] = 'app'
            raise_exception(exp_msg=_("duplicate login name"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_LOGIN_NAME,
                            exp_detail=json.dumps(errDetail, ensure_ascii=False))

    def check_user_exists_by_name(self, login_name, display_name,
                                  email, uid='', raise_ex=True):
        """
        根据用户名检查用户是否存在
        Args:
            login_name: 登录名
            display_name:显示名
            email: 邮箱
            uid: 用户ID，使用这个参数，将排除这个用户
                 使用这个参数会同时调用 self.check_user_exists 进行检测
            raise_ex: 如果存在，是否丢出异常
        """
        where = ''
        if uid:
            self.check_user_exists(uid)
            where = "AND `f_user_id` != '{0}'".format(self.w_db.escape(uid))

        # 过滤 邮箱为空的情况
        if not email:
            sql = """
            SELECT `f_login_name`, `f_display_name`, `f_mail_address` FROM `t_user`
            WHERE (`f_login_name` = %s OR `f_display_name` = %s) {0}
            LIMIT 1
            """.format(where)
            result = self.r_db.one(sql, login_name, display_name)
            cnt = 0
        else:
            sql = """
            SELECT `f_login_name`, `f_display_name`, `f_mail_address` FROM `t_user`
            WHERE (`f_login_name` = %s OR `f_display_name` = %s OR `f_mail_address` = %s) {0}
            LIMIT 1
            """.format(where)
            sql_cnt = """
            SELECT COUNT(*) AS cnt FROM `t_department`
            WHERE `f_mail_address` = %s
            """
            result = self.r_db.one(sql, login_name, display_name, email)
            cnt = self.r_db.one(sql_cnt, email)['cnt']

        # 检查与应用账户重名
        sql = f"""
        SELECT COUNT(*) AS cnt FROM `{get_db_name("user_management")}`.`t_app` WHERE `f_name` = %s
        """
        exist = self.r_db.one(sql, login_name)['cnt']

        if result or cnt or exist:
            if raise_ex:
                errDetail = {}
                if result and result['f_login_name'].lower() == login_name.lower():
                    errDetail['type'] = 'user'
                    raise_exception(exp_msg=_("duplicate login name"),
                                    exp_num=ncTShareMgntError.NCT_DUPLICATED_LOGIN_NAME,
                                    exp_detail=json.dumps(errDetail, ensure_ascii=False))

                if exist:
                    errDetail['type'] = 'app'
                    raise_exception(exp_msg=_("duplicate login name"),
                                    exp_num=ncTShareMgntError.NCT_DUPLICATED_LOGIN_NAME,
                                    exp_detail=json.dumps(errDetail, ensure_ascii=False))

                if result and result['f_display_name'].lower() == display_name.lower():
                    raise_exception(exp_msg=_("duplicate display name"),
                                    exp_num=ncTShareMgntError.
                                    NCT_DUPLICATED_DISPLAY_NAME)

                if cnt or result['f_mail_address'].lower() == email.lower():
                    raise_exception(exp_msg=_("duplicate email"),
                                    exp_num=ncTShareMgntError.
                                    NCT_DUPLICATED_EMALI)
            return True
        else:
            return False

    def check_user_exists(self, user_id, raise_ex=True):
        """
        检查用户是否存在，返回检查结果
        """
        result = None
        if check_is_uuid(user_id):
            sql = """
            SELECT `f_user_id` FROM `t_user` WHERE `f_user_id` = %s
            LIMIT 1
            """
            result = self.r_db.one(sql, user_id)

        if not result:
            if raise_ex:
                raise_exception(exp_msg=_("user not exists"),
                                exp_num=ncTShareMgntError.
                                NCT_USER_NOT_EXIST)
            return False
        return True

    def check_user_exists_by_thirdId(self, thirdId):
        """"
        检查用户是否存在，返回检查结果
        """
        sql = """
        SELECT `f_user_id` FROM `t_user` WHERE `f_third_party_id` = %s
        LIMIT 1
        """
        result = self.r_db.one(sql, thirdId)

        if not result:
            return False
        return True

    def check_user_belong_same_ou(self, user_id1, user_id2):
        """
        判断两个用户是否在同一个组织下
        """
        sql = """
        SELECT `f_ou_id` FROM `f_ou_user` WHERE `f_user_id` = %s
        """
        result1s = self.r_db.all(sql, user_id1)
        result2s = self.r_db.all(sql, user_id2)
        if result1s and result2s:
            ou1_ids = [res['f_ou_id'] for res in result1s]
            ou2_ids = [res['f_ou_id'] for res in result2s]

            if set(ou1_ids) & set(ou2_ids):
                return True

        return False

    def is_user_id_exists(self, user_id):
        """
        检查用户是否存在
        """
        sql = """
        SELECT `f_user_id` FROM `t_user` WHERE `f_user_id` = %s
        LIMIT 1
        """
        result = self.r_db.one(sql, user_id)

        return True if result else False

    def check_dispalyname_confict(self, display_name, user_id=None):
        """
        检查名字是否和用户显示名及文档库名冲突
        """
        where = ''
        if user_id:
            where = "AND `f_user_id` != '{0}'".format(
                self.w_db.escape(user_id))

        sql = """
        SELECT `f_user_id` FROM `t_user`
        WHERE `f_display_name` = %s {0}
        """.format(where)
        if self.r_db.one(sql, display_name):
            return True

        # 检查是否和文档库名称冲突
        if (self.base_confict_with_lib_name(display_name)):
            return True
        return False

    def get_unique_displayname(self, display_name, user_id=None):
        """
        检查系统中是否存在相同的显示名.
        参数：
              isplay_name: 需要检查的用户显示名
              user_id: 排除检查的用户id
        返回值：如果系统中存在相同的显示名，则返回当前显示名加索引后的名字;否则，返回原显示名.
        """

        index = 0
        unique_name = display_name
        while True:
            index += 1
            if self.check_dispalyname_confict(unique_name, user_id):
                suffix = str(index) if index / 10 else '0' + str(index)
                unique_name = display_name + suffix
            else:
                break
        return unique_name

    def is_des_encrypt(self):
        """
        是否记录des密码
        """
        sql = """
        SELECT `f_value`
        FROM `t_sharemgnt_config`
        WHERE `f_key` = %s
        """
        result = self.r_db.one(sql, 'enable_des_password')
        return False if result['f_value'] == '0' else True

    def get_organ_from_depart(self, depart_id):
        """
        根据部门ID获取组织ID
        用于添加搜索索引
        """
        sql = """
        SELECT `f_path` FROM `t_department` WHERE `f_department_id` = %s
        """
        result = self.r_db.one(sql, depart_id)
        if result:
            id_list = result['f_path'].split('/')
            return id_list[0]

        return depart_id

    def get_organ_from_depart_v2(self, depart_id):
        """
        根据部门ID获取组织ID
        用于添加搜索索引
        该接口用于添加用户接口实现，保证其多条sql操作为事务操作
        """
        sql = """
        SELECT `f_path` FROM `t_department` WHERE `f_department_id` = %s
        """
        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()

        cursor.execute(sql, (depart_id,))
        result = cursor.fetchone()
        if result:
            id_list = result['f_path'].split('/')
            return id_list[0]

        return depart_id

    def get_olduserinfo_by_loginName(self, loginName):
        """
        根据部门ID获取组织ID
        用于添加搜索索引
        """
        sql = """
            SELECT * FROM `t_user` WHERE `f_login_name` = %s
            """
        result = self.r_db.one(sql, loginName)
        return result

    def get_belong_depart_id(self, user_id):
        """
        获取用户所属的部门
        """
        sql = """
        select f_department_id
        from t_user_department_relation
        where f_user_id = %s
        order by f_relation_id
        """
        result = self.r_db.all(sql, user_id)
        if result:
            if len(result) == 0:
                return ""
            else:
                return [item['f_department_id'] for item in result]
        else:
            return ""

    def get_belong_depart_path(self, user_id):
        """
        获取用户所属的部门
        """
        sql = """
        select f_path
        from t_user_department_relation
        where f_user_id = %s
        order by f_relation_id
        """
        result = self.r_db.all(sql, user_id)
        if result:
            if len(result) == 0:
                return ""
            else:
                return [item['f_path'] for item in result]
        else:
            return ""

    def get_departid_by_name(self, depart_name):
        """
        根据部门名获取部门id
        """
        sql = """
        SELECT `f_department_id` FROM `t_department`
        WHERE `f_name` = %s
        """

        result = self.r_db.one(sql, depart_name)
        if result:
            return result['f_department_id']

    def get_all_path_dept_id(self, user_id):
        """
        根据用户id获取其到组织路径上的所有父部门id
        """
        # 获取用户所属部门 部门全路径path
        direct_dept_paths = self.get_belong_depart_path(user_id)
        all_paths = set()

        for dept_path in direct_dept_paths:
            id_list = dept_path.split('/')
            all_paths.update(id_list)

        return all_paths

    def add_user(self, user, responsible_person_id):
        """
        新建用户
        param: ncTAddUserParam
        """
        # 数据验证
        ShareMgnt_Log("add_user begin")
        ShareMgnt_Log(user.user)
        self.check_user(user.user, True)
        self.check_user_exists(responsible_person_id)

        # 新建账号不能和保留管理员账号重复
        if user.user.loginName in self.remain_accounts:
            raise_exception(exp_msg=_("remain admin account"),
                            exp_num=ncTShareMgntError.NCT_REMAIN_ADMIN_ACCOUNT)

        if user.user.loginName.lower() in list(self.get_all_admin_account().values()):
            raise_exception(exp_msg=_("account confict with admin"),
                            exp_num=ncTShareMgntError.NCT_ACCOUNT_CONFICT_WITH_ADMIN)

        # 检查用户密码
        if user.password:
            user.password = user.password.strip()
            sha2_pwd = sha2_encrypt(user.password)
            if sha2_pwd != self.user_default_password.sha2_pwd:
                self.check_password_valid(user.password)
            user.sha2Password = sha2_pwd
            user.ntlmPassword = ntlm_md4(user.password)

            if self.is_des_encrypt() or user.user.pwdControl:
                user.desPassword = bytes.decode(des_encrypt(global_info.des_key,
                                                            user.password,
                                                            global_info.des_key))
        else:
            user.sha2Password = self.user_default_password.sha2_pwd
            user.ntlmPassword = self.user_default_password.ntlm_pwd

            if self.is_des_encrypt() or user.user.pwdControl:
                user.desPassword = self.user_default_password.des_pwd

        # 检查用户登录名、显示名
        self.check_user_exists_by_name(
            user.user.loginName, user.user.displayName, user.user.email)

        if user.user.thirdId != "":
            if self.check_user_exists_by_thirdId(user.user.thirdId):
                raise_exception(exp_msg=_("thirdId already exists"),
                                exp_num=ncTShareMgntError.NCT_USER_HAS_EXIST)
            user.user.objectGUID = user.user.thirdId

        # 检查是否和文档库、归档库冲突
        self.is_confict_with_lib_name(user.user.displayName)

        # 如果未分配部门在列表中，则不进行判断，并只保存未分配部门
        if NCT_UNDISTRIBUTE_USER_GROUP in user.user.departmentIds:
            user.user.departmentIds = [NCT_UNDISTRIBUTE_USER_GROUP]
        else:
            # 判断部门是否存在
            ids_condition = []
            for depart_id in user.user.departmentIds:
                # 存在非法ID，则异常
                if not check_is_uuid(depart_id):
                    raise_exception(exp_msg=_("depart not exists"),
                                    exp_num=ncTShareMgntError.NCT_DEPARTMENT_NOT_EXIST)

                # 将合法的ID放到条件列表
                ids_condition.append("'%s'" % depart_id)

            # 如果条件列表存在，则进行判断
            if len(ids_condition) > 0:
                ids_condition = ",".join(ids_condition)
                sql = """
                SELECT COUNT(*) AS cnt FROM t_department
                WHERE f_department_id in ({0})
                """.format(ids_condition)
                count = self.r_db.one(sql)['cnt']
                # 如果通过条件查出来的总数与实际总数不符，则异常
                if count != len(user.user.departmentIds):
                    raise_exception(exp_msg=_("depart not exists"),
                                    exp_num=ncTShareMgntError.NCT_DEPARTMENT_NOT_EXIST)

        # 如果密级为空，则取用户密级最小值
        if user.user.csfLevel is None:
            user.user.csfLevel = self.config_manage.get_min_csf_level()

        # 如果密级2为空，则取用户密级2最小值
        if user.user.csfLevel2 is None:
            user.user.csfLevel2 = self.config_manage.get_min_csf_level2()

        # 如果排序权重为空，则取999
        if user.user.priority is None:
            user.user.priority = 999

        # 设置密码管控
        user.user.pwdControl = 1 if user.user.pwdControl else 0

        # 如果存储为空合法，表示没有归属对象存储使用站点默认存储
        if user.user.ossInfo is None:
            ossInfo = ncTUsrmOSSInfo()
            user.user.ossInfo = ossInfo

        # 检查用户上级
        if user.user.managerID is not None and user.user.managerID != "":
            if user.user.managerID in [NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT]:
                raise_exception(exp_msg=_("IDS_INVALID_MANAGER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

            bExist = self.check_user_exists(user.user.managerID, False)
            if not bExist:
                raise_exception(exp_msg=_("IDS_INVALID_MANAGER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

        user.user.telNumber = user.user.telNumber.strip() if user.user.telNumber else ""

        # 检查用户编码
        if user.user.code is not None:
            user.user.code = self.check_user_code(user.user.code)

        # 检查用户岗位
        if user.user.position is not None:
            user.user.position = self.check_user_position(user.user.position)

        ShareMgnt_Log(user.user)
        return self.add_user_to_db(user)

    def check_user_position(self, position):
        """
        检查用户岗位
        """
        # 除去前面的空格，末尾的空格
        position = position.lstrip()
        position = position.rstrip()

        if len(position) > 50:
            raise_exception(exp_msg=_("IDS_INVALID_USER_POSITION"),
                        exp_num=ncTShareMgntError.NCT_INVALID_USER_POSITION)
        return position

    def check_user_code(self, code, user_id=None):
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
        select f_user_id from t_user where f_code = %s
        """
        result = self.r_db.one(select_sql, striped_code)
        if result and result['f_user_id'] != user_id:
            raise_exception(exp_msg=_("IDS_DUPLICATED_USER_CODE"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_USER_CODE)
        return striped_code

    def __get_user_role_id(self, user_id):
        """
        获取用户角色id
        """
        from src.modules.role_manage import RoleManage
        return RoleManage().get_user_role_id(user_id)

    def get_parent_dept_responsbile_person(self, user_id):
        """
        根据用户id获取所有上级部门的组织管理员
        """
        from src.modules.department_manage import DepartmentManage
        # 获取用户所有的父部门id
        parent_dept_ids = self.get_all_path_dept_id(user_id)

        # 获取用户所有父部门的组织管理员id
        responsible_person_ids = []
        if len(parent_dept_ids) == 0:
            return responsible_person_ids

        groupStr = generate_group_str(parent_dept_ids)
        sql = """
        SELECT distinct f_user_id FROM t_department_responsible_person WHERE f_department_id in ({0})
        """.format(groupStr)

        results = self.r_db.all(sql)
        for result in results:
            responsible_person_ids.append(result['f_user_id'])

        return responsible_person_ids

    def get_status_before_add(self, user, user_uuid):
        """
        更新用户状态前先检查授权
        """
        # 如果传入字符为unicode，转为utf-8，防止记日志时出现编码错误
        if isinstance(user.user.displayName, str):
            display_name = user.user.displayName.encode('utf-8')
        else:
            display_name = user.user.displayName
        if isinstance(user.user.loginName, str):
            login_name = user.user.loginName.encode('utf-8')
        else:
            login_name = user.user.loginName
        # 启用用户需要检查授权，授权人数已满时将变为禁用状态，创建仍成功，记录日志
        userStatus = user.user.status if user.user.status is not None else ncTUsrmUserStatus.NCT_STATUS_ENABLE
        if userStatus == ncTUsrmUserStatus.NCT_STATUS_ENABLE:
            if self.is_user_num_overflow():
                global_info.IMPORT_DISABLE_USER_NUM += 1
                eacp_log(user_uuid,
                        global_info.LOG_TYPE_MANAGE,
                        global_info.USER_TYPE_AUTH,
                        global_info.LOG_LEVEL_WARN,
                        global_info.LOG_OP_TYPE_SET,
                         _("IDS_CREATE_DISABLE_USER_MSG") % (
                             display_name, login_name),
                         raise_ex=True)
                userStatus = ncTUsrmUserStatus.NCT_STATUS_DISABLE
        return userStatus

    def patch_user_custom_attr(self, user_id, custom_attr):
        # 查询当前用户id是否有用户自定义属性
        select_sql = """
        SELECT `f_custom_attr` FROM `t_user_custom_attr`
        WHERE `f_user_id` = %s
        """
        result = self.r_db.one(select_sql, user_id)
        if result:
            old_custom_attr = json.loads(result["f_custom_attr"])
            marge_custom_attr = merge_dicts(old_custom_attr, custom_attr)
            update_sql = """
            update `t_user_custom_attr` set `f_custom_attr` = %s
            where `f_user_id` = %s
            """
            self.w_db.query(update_sql, json.dumps(marge_custom_attr), user_id)
            ShareMgnt_Log(update_sql, json.dumps(marge_custom_attr), user_id)
        else:
            custom_attr_id = str(ulid.new())
            insert_sql = """
            insert into `t_user_custom_attr` values (%s, %s, %s)
            """
            self.w_db.query(insert_sql, custom_attr_id, user_id, json.dumps(custom_attr))
            ShareMgnt_Log(insert_sql, custom_attr_id, user_id, json.dumps(custom_attr))

    def add_user_to_db(self, user):
        """
        添加用户到数据库
        域控导入需要本接口，所以公开
        """
        # 生成UUID
        user_uuid = str(uuid.uuid1())

        sha2Password = user.sha2Password if hasattr(
            user, 'sha2Password') else ""
        md5Password = user.md5Password if hasattr(user, 'md5Password') else ""
        desPassword = user.desPassword if hasattr(user, 'desPassword') else ""
        ntlmPassword = user.ntlmPassword if hasattr(
            user, 'ntlmPassword') else ""
        guid = user.user.objectGUID if hasattr(user.user, 'objectGUID') else ""

        # 检查server_type和dnPath
        ldap_type = 0
        if hasattr(user.user, 'server_type'):
            if user.user.server_type is not None:
                ldap_type = user.user.server_type

        # 检查domian_path
        dn_path = ""
        if hasattr(user.user, 'dnPath'):
            if user.user.dnPath is not None and user.user.dnPath != "null":
                dn_path = user.user.dnPath

        # 获取用户禁用状态
        userStatus = self.get_status_before_add(user, user_uuid)

        # 检查用户上级
        managerID = ''
        if user.user.managerID is not None:
            managerID = user.user.managerID

        userCode = ''
        if user.user.code is not None:
            userCode = user.user.code

        userPosi = ''
        if user.user.position is not None:
            userPosi = user.user.position

        with safe_cursor(self.w_db) as cursor:
            # 插入用户信息
            insert_user_sql = """
            INSERT INTO `t_user`
            (`f_user_id`, `f_login_name`, `f_display_name`, `f_remark`, `f_idcard_number`,
            `f_password`, `f_des_password`, `f_ntlm_password`, `f_mail_address`, `f_auth_type`,
            `f_status`, `f_pwd_timestamp`, `f_pwd_error_latest_timestamp`,
            `f_third_party_id`, `f_domain_path`, `f_ldap_server_type`,
            `f_priority`, `f_csf_level`, `f_pwd_control`, `f_oss_id`, `f_tel_number`,
            `f_expire_time`, `f_sha2_password`, `f_manager_id`, `f_code`, `f_position`, `f_csf_level2`)
            VALUES(%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
                    %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
            """
            now = BusinessDate.now().strftime("%Y-%m-%d %H:%M:%S")
            cursor.execute(insert_user_sql, (user_uuid,
                           user.user.loginName,
                           user.user.displayName,
                           user.user.remark,
                           user.user.idcardNumber,
                           md5Password, desPassword, ntlmPassword,
                           user.user.email,
                           user.user.userType, int(
                               userStatus), now, now, guid, dn_path, ldap_type,
                           user.user.priority, user.user.csfLevel,
                           user.user.pwdControl, user.user.ossInfo.ossId, user.user.telNumber,
                           user.user.expireTime, sha2Password, managerID, userCode, userPosi,
                           user.user.csfLevel2))

            # 保存部门关系、索引
            relation_values = []
            index_value = []
            path = ''
            path_list = []
            for depart_id in user.user.departmentIds:
                from src.modules.department_manage import DepartmentManage
                path = DepartmentManage().get_department_path_by_dep_id(depart_id)
                path_list.append(path)
                relation_values.append((user_uuid, depart_id, path))
                # 未分配部门忽略
                if depart_id != NCT_UNDISTRIBUTE_USER_GROUP:
                    index_value.append((user_uuid,
                                        self.get_organ_from_depart_v2(depart_id)))
            insert_relation_sql = """
            INSERT INTO `t_user_department_relation`
            (`f_user_id`, `f_department_id`, `f_path`)
            VALUES (%s, %s, %s)
            """
            cursor.executemany(insert_relation_sql, relation_values)

            # 创建临时联系人组
            group_id = str(uuid.uuid1())
            sql = """
            INSERT INTO `t_person_group` VALUES (%s, %s, %s, 0)
            """
            cursor.execute(sql, (group_id, user_uuid, self.tmp_group))

            # 添加索引
            if len(index_value) > 0:
                sql = """
                INSERT INTO `t_ou_user`
                    (`f_user_id`, `f_ou_id`)
                VALUES (%s, %s)
                """
                cursor.executemany(sql, index_value)
        try:
            pub_nsq_msg(TOPIC_USER_CREATE, {
                        "id": user_uuid, "name": user.user.displayName})
            pub_nsq_msg(TOPIC_DEPARTMENT_USER_ADD, {
                        "id": user_uuid, "dept_paths": path_list})

        except Exception as ex:
            sql_list = [
                # 删掉用户所有组
                """
                DELETE FROM `t_person_group` WHERE `f_user_id` = %s
                """,
                # 删除用户关系表
                """
                DELETE FROM `t_user_department_relation` WHERE `f_user_id` = %s
                """,
                # 删除用户信息
                """
                DELETE FROM `t_user` WHERE `f_user_id` = %s
                """,
                # 删除索引信息
                """
                DELETE FROM `t_ou_user` WHERE `f_user_id` = %s
                """,
                # 删除用户自定义配置
                """
                DELETE FROM `t_user_custom_attr` WHERE `f_user_id` = %s
                """
            ]
            for sql in sql_list:
                self.w_db.query(sql, user_uuid)

            raise_exception(exp_msg=str(ex),
                            exp_num=ncTShareMgntError.
                            NCT_DB_OPERATE_FAILED)

        return user_uuid

    def delete_user(self, user_id):
        """
        删除用户
        """
        self.check_user_exists(user_id)

        # 是否为激活用户（登录过的用户）
        activate_status = self.get_activate_status(user_id)

        # 获取用户所属部门 部门全路径path
        direct_dept_paths = self.get_belong_depart_path(user_id)

        sql_list = [
            # 删除用户组成员
            f"""
            DELETE FROM {get_db_name("user_management")}.t_group_member WHERE `f_member_id` = %s
            """,
            # 删除用户关系表
            """
            DELETE FROM `t_user_department_relation` WHERE `f_user_id` = %s
            """,
            # 删除用户信息
            """
            DELETE FROM `t_user` WHERE `f_user_id` = %s
            """,
            # 删除索引信息
            """
            DELETE FROM `t_ou_user` WHERE `f_user_id` = %s
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
            DELETE FROM `t_department_responsible_person` WHERE `f_user_id` = %s
            """,
            # 删除内外链共享模板
            """
            DELETE FROM `t_link_template` WHERE `f_sharer_id` = %s
            """,
            # 删除文件抓取策略
            """
            DELETE FROM `t_file_crawl_strategy` WHERE `f_user_id` = %s
            """,
            # 删除自动归档策略
            """
            DELETE FROM `t_doc_auto_archive_strategy` WHERE `f_obj_id` = %s
            """,
            # 删除用户角色关系
            """
            DELETE FROM `t_user_role_relation` WHERE `f_user_id` = %s
            """,
            # 删除自动清理策略
            """
            DELETE FROM `t_doc_auto_clean_strategy` WHERE `f_obj_id` = %s
            """,
            # 删除本地同步策略
            """
            DELETE FROM `t_local_sync_strategy` WHERE `f_obj_id` = %s
            """,
            # 删除用户自定义配置
            """
            DELETE FROM `t_user_custom_attr` WHERE `f_user_id` = %s
            """
        ]

        for sql in sql_list:
            self.w_db.query(sql, user_id)

        # 更新用户总数、激活用户数
        self.update_user_count(activate_status)

        # 发布用户删除nsq消息
        pub_nsq_msg(TOPIC_USER_DELETE, {"id": user_id})

        # 发布用户从部门移除nsq消息
        pub_nsq_msg(TOPIC_DEPARTMENT_USER_REMOVE, {
                    "id": user_id, "dept_paths": direct_dept_paths})

    def get_all_user_count(self):
        """
        获取用户总数
        """
        sql = """
        SELECT COUNT(*) AS cnt FROM `t_user`
        WHERE  `f_user_id` != %s
        AND `f_user_id` != %s
        AND `f_user_id` != %s
        AND `f_user_id` != %s
        """
        count = self.r_db.one(
            sql, NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT)
        return count["cnt"]

    def is_user_num_overflow(self):

        return False

    def get_all_admin_account(self):
        """
        获取所有管理员账号
        """
        sql = """
        SELECT f_login_name, f_user_id
        FROM `t_user`
        WHERE `f_user_id` in (%s, %s, %s, %s)
        """
        results = self.r_db.all(sql, NCT_USER_ADMIN,
                                NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT)
        admin_account_map = {}
        for result in results:
            admin_account_map[result["f_user_id"]
                              ] = result["f_login_name"].lower()

        for remain_account in self.remain_accounts:
            if remain_account not in list(admin_account_map.values()):
                admin_account_map[remain_account] = remain_account

        return admin_account_map

    def get_all_user_count_belong_ou(self, ou_id):
        """
        获取属于一个组织下的用户总数
        """
        sql = """
        SELECT COUNT(*) AS cnt
        FROM `t_ou_user`
        WHERE `f_ou_id` = %s
        """
        count = self.r_db.one(sql, ou_id)
        return count["cnt"]

    def fetch_user(self, db_user, get_quota=True):
        """
        将数据库用户信息转换为Thrift结构体
        非必填数据没有，将使用默认数据
        Args:
            db_user: dict 数据库查询结果
            get_quota: bool 是否获取配额信息
        Return:
            ncTUsrmGetUserInfo
        """
        user_info = ncTUsrmGetUserInfo()
        user_info.id = db_user['f_user_id']
        user_info.originalPwd = db_user.get('originalPwd', '')

        user = ncTUsrmUserInfo()
        user.loginName = db_user['f_login_name']
        user.email = db_user['f_mail_address']
        user.remark = db_user['f_remark']
        user.idcardNumber = db_user['f_idcard_number']
        user.displayName = db_user.get('f_display_name', '')
        user.userType = db_user.get(
            'f_auth_type', ncTUsrmUserType.NCT_USER_TYPE_LOCAL)
        if db_user['f_status'] == 0 and db_user['f_auto_disable_status'] == 0:
            user.status = 0
        else:
            user.status = 1
        user.priority = db_user['f_priority']
        user.csfLevel = db_user['f_csf_level']
        user.pwdControl = db_user['f_pwd_control']
        user.freezeStatus = db_user.get('f_freeze_status', False)
        user.telNumber = db_user.get('f_tel_number')
        user.thirdId = db_user.get('f_third_party_id', '')

        # 增加用户创建时间字段
        user.createTime = int(db_user['f_create_time'].timestamp()) if 'f_create_time' in db_user else 0

        # 如果对象存储为空，则对象存储信息默认为空
        if not db_user['f_oss_id']:
            user.ossInfo = ncTUsrmOSSInfo()
        else:
            user.ossInfo = get_oss_info(db_user['f_oss_id'])

        # 增加用户有效期字段
        if 'f_expire_time' in db_user:
            user.expireTime = db_user['f_expire_time']
        user_info.user = user

        depart_id = db_user.get('parentDepartId', '')
        depart_name = db_user.get('f_name', '')
        if depart_id:
            user_info.directDeptInfo = self.__get_direct_dept_info(
                depart_id, depart_name)

        self.__get_departments_from_user(user_info.id, user)
        from src.modules.role_manage import RoleManage
        user_info.user.roles = RoleManage().get_user_role(user_info.id)

        # 获取code
        if 'f_code' in db_user:
            user_info.user.code = db_user['f_code']

        # 获取上级
        user_info.user.managerID = ''
        user_info.user.managerDisplayName = ''
        if 'f_manager_id' in db_user:
            user_info.user.managerID = db_user['f_manager_id']
            if user_info.user.managerID != '':
                user_info.user.managerDisplayName = self.get_displayname_by_userid(user_info.user.managerID)

        # 获取岗位
        if 'f_position' in db_user:
            user_info.user.position = db_user['f_position']

        # 获取密级2
        if 'f_csf_level2' in db_user:
            user_info.user.csfLevel2 = db_user['f_csf_level2']

        return user_info

    def __get_direct_dept_info(self, depart_id, depart_name):
        """
        获取直属部门信息
        """
        from src.modules.department_manage import DepartmentManage
        directDeptInfo = ncTUsrmDirectDeptInfo()
        directDeptInfo.departmentId = depart_id
        directDeptInfo.departmentName = depart_name
        directDeptInfo.ids = []
        directDeptInfo.responsiblePersons = []

        # 获取管理员信息
        uids = DepartmentManage().get_depart_mgr_ids(depart_id)
        for uid in uids:
            sql = """
            SELECT `f_user_id`, `f_login_name`, `f_display_name`, `f_status` FROM `t_user`
            WHERE `f_user_id` = %s
            """
            db_user = self.r_db.one(sql, uid)
            user = ncTSimpleUserInfo()
            user.id = db_user['f_user_id']
            user.displayName = db_user['f_display_name']
            user.loginName = db_user['f_login_name']
            user.status = db_user['f_status']
            directDeptInfo.responsiblePersons.append(user)

        return directDeptInfo

    def get_departments_from_user(self, user_id):
        """
        根据用户ID获取其所属部门
        """
        self.check_user_exists(user_id)
        user = ncTUsrmUserInfo()
        self.__get_departments_from_user(user_id, user)
        return user

    def __get_departments_from_user(self, uid, user):
        """
        根据用户ID获取其所属部门
        """
        if uid in [NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT]:
            user.departmentIds = [NCT_UNDISTRIBUTE_USER_GROUP]
            user.departmentNames = [_("undistributed user")]

        id_list = []
        name_list = []
        code_list = []

        sql = """
        SELECT `f_path` AS `path` FROM `t_user_department_relation`
        WHERE `f_user_id` = %s
        """
        db_depart_list = self.r_db.all(sql, uid)
        for db_depart in db_depart_list:
            # 如果用户为未分配用户，只需要保存未分配即可
            if db_depart['path'] == NCT_UNDISTRIBUTE_USER_GROUP:
                id_list = [NCT_UNDISTRIBUTE_USER_GROUP]
                name_list = [_("undistributed user")]
                code_list = ['']
                break
            else:
                sql = """
                SELECT `f_name`, `f_department_id`, `f_code` FROM `t_department`
                WHERE `f_path` = %s
                """
                result = self.r_db.one(sql, db_depart['path'])
                if result:
                    id_list.append(result['f_department_id'])
                    name_list.append(result['f_name'])
                    code_list.append(result['f_code'])

                else:
                    # 如果没有查到部门，则修正用户-部门关系表数据
                    sql = """
                    DELETE FROM `t_user_department_relation`
                    WHERE `f_user_id` = %s AND `f_path` = %s
                    """
                    self.w_db.query(sql, uid, db_depart['path'])

        # 如果并没有保存部门ID，则向用户-部门关系表添加数据
        # 并且设置用户为未分配人员
        if not id_list:
            self.w_db.insert("t_user_department_relation", {
                "f_user_id": uid,
                "f_department_id": NCT_UNDISTRIBUTE_USER_GROUP,
                "f_path": NCT_UNDISTRIBUTE_USER_GROUP
            })

            id_list = [NCT_UNDISTRIBUTE_USER_GROUP]
            name_list = [_("undistributed user")]
            code_list = ['']
        user.departmentIds = id_list
        user.departmentNames = name_list
        user.departmentCodes = code_list

    def get_all_users(self, start, limit):
        """
        获取当前页的用户
        返回数据形式
        [ncTUsrmGetUserInfo, ncTUsrmGetUserInfo...]
        """
        limit_statement = check_start_limit(start, limit)
        sql = """
        SELECT u.`f_user_id`, u.`f_login_name`, u.`f_display_name`, u.`f_remark`, u.`f_password`,
            u.`f_des_password`, u.`f_sha2_password`, u.`f_mail_address`, u.`f_auth_type`, u.`f_status`,
            u.`f_tel_number`, u.`f_idcard_number`, u.`f_expire_time`, u.`f_priority`,
            u.`f_csf_level`, u.`f_pwd_control`, u.`f_oss_id`, u.`f_freeze_status`,
            u.`f_create_time`, u.`f_auto_disable_status`, u.`f_code`, u.`f_position`, u.`f_manager_id`,
            u.`f_csf_level2`
        FROM `t_user` as u
        WHERE u.`f_user_id` <> '{0}'
            AND u.`f_user_id` <> '{1}'
            AND u.`f_user_id` <> '{2}'
            AND u.`f_user_id` <> '{3}'
            AND 1=%s
        ORDER BY u.f_priority, u.f_display_name
        {4}
        """.format(NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM,
                   NCT_USER_SECURIT, limit_statement)

        db_user_list = self.r_db.all(sql, 1)

        # python3 字典本身就是有序的,在此不需要重新初始化一个有序字典
        user_dict = {user_info["f_user_id"]: user_info for user_info in db_user_list}

        userIDs = []
        for userInfo in db_user_list:
            userIDs.append(userInfo['f_user_id'])

        if len(userIDs) > 0:
            users_dep_infos = self.get_users_parent_deps(userIDs)
            for user_id in userIDs:
                if user_id in users_dep_infos:
                    user_dict[user_id]['depart_ids'] = '|'.join(
                        users_dep_infos[user_id]['depart_ids'])
                    user_dict[user_id]['depart_names'] = '|'.join(
                        users_dep_infos[user_id]['depart_names'])
                    user_dict[user_id]['depart_codes'] = '|'.join(
                        users_dep_infos[user_id]['depart_codes'])
                else:
                    user_dict[user_id]['depart_ids'] = None
                    user_dict[user_id]['depart_names'] = None
                    user_dict[user_id]['depart_codes'] = None

        result = []
        for db_user in user_dict.values():
            if db_user['f_password'] != '':
                db_user['originalPwd'] = True if self.initAdminPwd == db_user['f_password'] else False
            elif db_user['f_sha2_password'] != '':
                db_user['originalPwd'] = True if self.initSha2AdminPwd == db_user['f_sha2_password'] else False
            result.append(self.convert_user_info(db_user))

        # 填充用户所属角色信息
        self.fill_user_roles(result)

        # 填充用户的直属上级
        self.fill_user_managers(result)

        return result

    def convert_user_info(self, db_user):
        """
        """
        def convert_idcard(
            num): return num[:3] + len(num[3:-4]) * '*' + num[-4:] if num else ''
        user_info = ncTUsrmGetUserInfo()
        user_info.id = db_user['f_user_id']
        user_info.originalPwd = db_user.get('originalPwd', '')

        user = ncTUsrmUserInfo()
        user_info.user = user
        user.loginName = db_user['f_login_name']
        user.email = db_user.get('f_mail_address', '')
        user.displayName = db_user.get('f_display_name', '')
        user.telNumber = db_user.get('f_tel_number', '')
        user.idcardNumber = convert_idcard(
            self.get_origin_idcardnumber(db_user))
        user.expireTime = db_user.get('f_expire_time')
        user.remark = db_user.get('f_remark', '')
        user.userType = db_user.get(
            'f_auth_type', ncTUsrmUserType.NCT_USER_TYPE_LOCAL)
        user.freezeStatus = db_user.get('f_freeze_status', False)
        if db_user['f_status'] == 0 and db_user['f_auto_disable_status'] == 0:
            user.status = 0
        else:
            user.status = 1
        user.priority = db_user.get('f_priority')
        user.csfLevel = db_user.get('f_csf_level')
        user.pwdControl = db_user.get('f_pwd_control')

        # 增加用户创建时间字段
        user.createTime = int(db_user['f_create_time'].timestamp()) if 'f_create_time' in db_user else 0

        # 解析用户所属部门id
        if 'depart_ids' in db_user:
            if db_user['depart_ids']:
                user.departmentIds = db_user['depart_ids'].split('|')
            else:
                user.departmentIds = [NCT_UNDISTRIBUTE_USER_GROUP]
        if 'depart_names' in db_user:
            if db_user['depart_names']:
                user.departmentNames = db_user['depart_names'].split('|')
            else:
                user.departmentNames = [_("undistributed user")]
        if 'depart_codes' in db_user:
            if db_user['depart_codes']:
                user.departmentCodes = db_user['depart_codes'].split('|')
            else:
                user.departmentCodes = ['']
        if 'parentDepartId' in db_user:
            user_info.directDeptInfo = ncTUsrmDirectDeptInfo()
            user_info.directDeptInfo.departmentId = db_user['parentDepartId']
            user_info.directDeptInfo.departmentName = db_user['f_name']
            user_info.directDeptInfo.ids = []
            user_info.directDeptInfo.responsiblePersons = []

        # 解析用户对象存储信息
        if not db_user['f_oss_id']:
            user.ossInfo = ncTUsrmOSSInfo()
        else:
            user.ossInfo = get_oss_info(db_user['f_oss_id'])

        # 解析用户编号
        if 'f_code' in db_user:
            user.code = db_user['f_code']

        # 解析用户上级ID
        if 'f_manager_id' in db_user:
            user.managerID = db_user['f_manager_id']

        # 解析用户岗位
        if 'f_position' in db_user:
            user.position = db_user['f_position']

        # 解析用户密级2
        if 'f_csf_level2' in db_user:
            user.csfLevel2 = db_user['f_csf_level2']

        return user_info

    def fill_user_departments(self, user_infos):
        """
        填充用户所属部门信息
        """
        if not user_infos:
            return

        userid_set = []
        for user_info in user_infos:
            userid_set.append(user_info.id)

        if len(userid_set) == 0:
            return

        depart_info_map = self.get_users_parent_deps(userid_set)

        # 解析用户所属部门信息
        for user_id in userid_set:
            if user_id not in depart_info_map:
                depart_info_map[user_id] = {}
                depart_info_map[user_id]['depart_names'] = [
                    _("undistributed user")]
                depart_info_map[user_id]['depart_ids'] = [
                    NCT_UNDISTRIBUTE_USER_GROUP]
                depart_info_map[user_id]['depart_codes'] = ['']

        for user_info in user_infos:
            if user_info.id in depart_info_map:
                if hasattr(user_info, "user"):
                    user_info.user.departmentIds = depart_info_map[user_info.id]['depart_ids']
                    user_info.user.departmentNames = depart_info_map[user_info.id]['depart_names']
                    user_info.user.departmentCodes = depart_info_map[user_info.id]['depart_codes']
                else:
                    user_info.departmentIds = depart_info_map[user_info.id]['depart_ids']
                    user_info.departmentNames = depart_info_map[user_info.id]['depart_names']
                    user_info.departmentCodes = depart_info_map[user_info.id]['depart_codes']

    def get_origin_idcardnumber(self, db_user):
        idcard_number = db_user.get('f_idcard_number')
        if not idcard_number:
            return ''
        origin_idcard_number = bytes.decode(des_decrypt_with_padzero(global_info.des_key,
                                                                     idcard_number,
                                                                     global_info.des_key))
        return origin_idcard_number[:18]

    def get_user_by_id(self, user_id, origin_idcard=True):
        """
        根据用户ID获取用户信息
        """
        self.check_user_exists(user_id)
        sql = """
        SELECT `f_user_id`, `f_third_party_id`, `f_login_name`, `f_display_name`, `f_password`, `f_idcard_number`,
        `f_des_password`, `f_sha2_password`, `f_mail_address`, `f_auth_type`, `f_status`, `f_remark`,
        `f_priority`, `f_csf_level`, `f_pwd_control`, `f_oss_id`, `f_freeze_status`,
        `f_create_time`, `f_auto_disable_status`, `f_tel_number`,
        `f_expire_time`, `f_code`, `f_manager_id`, `f_position`, `f_csf_level2`
        FROM `t_user`
        WHERE `f_user_id` = %s
        """
        db_user = self.r_db.one(sql, user_id)
        if db_user['f_password'] != '':
            db_user['originalPwd'] = True if self.initAdminPwd == db_user['f_password'] else False
        elif db_user['f_sha2_password'] != '':
            db_user['originalPwd'] = True if self.initSha2AdminPwd == db_user['f_sha2_password'] else False
        db_user['f_idcard_number'] = self.get_origin_idcardnumber(db_user)
        if not origin_idcard and db_user['f_idcard_number']:
            db_user['f_idcard_number'] = db_user['f_idcard_number'][:3] + \
                '*' * 4 + db_user['f_idcard_number'][-4:]
        result = self.fetch_user(db_user)
        return result

    def get_user_by_loginname(self, login_name, throw_ex=False):
        """
        根据用户名获取用户信息
        """
        sql = """
        SELECT `f_user_id`, `f_login_name`, `f_display_name`, `f_password`, `f_idcard_number`,
        `f_des_password`, `f_sha2_password`, `f_mail_address`, `f_auth_type`, `f_status`, `f_remark`,
        `f_priority`, `f_csf_level`, `f_pwd_control`, `f_oss_id`,
        `f_create_time`, `f_auto_disable_status`, `f_tel_number`, `f_csf_level2`
        FROM `t_user`
        WHERE `f_login_name` = %s
        """
        db_user = self.r_db.one(sql, login_name)
        if not db_user and throw_ex:
            raise_exception(exp_msg=_("user not exists"),
                            exp_num=ncTShareMgntError.
                            NCT_USER_NOT_EXIST)

        if db_user:
            if db_user['f_password'] != '':
                db_user['originalPwd'] = True if self.initAdminPwd == db_user['f_password'] else False
            elif db_user['f_sha2_password'] != '':
                db_user['originalPwd'] = True if self.initSha2AdminPwd == db_user['f_sha2_password'] else False
            db_user['f_idcard_number'] = self.get_origin_idcardnumber(db_user)
            result = self.fetch_user(db_user)
            return result

    def get_user_by_third_id(self, third_id, throw_ex=True):
        """
        根据第三方id获取用户信息
        """
        sql = """
        SELECT `f_user_id`, `f_login_name`, `f_display_name`, `f_password`, `f_idcard_number`,
        `f_des_password`, `f_sha2_password`, `f_mail_address`, `f_auth_type`, `f_status`, `f_remark`,
        `f_priority`, `f_csf_level`, `f_pwd_control`, `f_oss_id`,
        `f_create_time`, `f_auto_disable_status`, `f_tel_number`
        FROM `t_user`
        WHERE `f_third_party_id` = %s
        """
        db_user = self.r_db.one(sql, third_id)
        if not db_user and throw_ex:
            raise_exception(exp_msg=_("user not exists"),
                            exp_num=ncTShareMgntError.NCT_USER_NOT_EXIST)

        if db_user:
            if db_user['f_password'] != '':
                db_user['originalPwd'] = True if self.initAdminPwd == db_user['f_password'] else False
            elif db_user['f_sha2_password'] != '':
                db_user['originalPwd'] = True if self.initSha2AdminPwd == db_user['f_sha2_password'] else False
            db_user['f_idcard_number'] = self.get_origin_idcardnumber(db_user)
            result = self.fetch_user(db_user)
            return result

    def get_user_by_mail(self, mail):
        """
        根据用户名获取用户信息
        """
        sql = """
        SELECT `f_user_id`, `f_login_name`, `f_display_name`, `f_password`, `f_idcard_number`,
        `f_des_password`, `f_sha2_password`, `f_mail_address`, `f_auth_type`, `f_status`, `f_remark`,
        `f_priority`, `f_csf_level`, `f_pwd_control`, `f_oss_id`,
        `f_create_time`, `f_auto_disable_status`
        FROM `t_user`
        WHERE `f_mail_address` = %s
        """
        db_user = self.r_db.one(sql, mail)
        if db_user:
            if db_user['f_password'] != '':
                db_user['originalPwd'] = True if self.initAdminPwd == db_user['f_password'] else False
            elif db_user['f_sha2_password'] != '':
                db_user['originalPwd'] = True if self.initSha2AdminPwd == db_user['f_sha2_password'] else False
            db_user['f_idcard_number'] = self.get_origin_idcardnumber(db_user)
            result = self.fetch_user(db_user)
            return result

    def get_userid_by_loginname(self, login_name):
        """
        根据用户名获取用户id
        """
        sql = """
        SELECT `f_user_id` FROM `t_user` WHERE `f_login_name` = %s
        """
        result = self.r_db.one(sql, login_name)
        return result['f_user_id'] if result else ''

    def get_displayname_by_userid(self, user_id):
        """
        根据用户id获取用户名
        """
        sql = """
        SELECT `f_display_name` FROM `t_user` WHERE `f_user_id` = %s
        """
        result = self.r_db.one(sql, user_id)
        return result['f_display_name'] if result else ''

    def modify_user_space(self, user_id, user_space, responsible_person_id):
        """
        修改用户配额空间
        """
        # 检查配额
        if user_space is not None:
            if user_space <= 0:
                raise_exception(exp_msg=_("IDS_INVALID_USER_SPACE"),
                                exp_num=ncTShareMgntError.NCT_INVALID_USER_SPACE)

        # 如果需要修改配额
        if user_space is not None:
            self.check_user_exists(responsible_person_id)

            # 添加用户自定义属性
            custom_attr = {"document": {"space_quote": user_space}}
            self.patch_user_custom_attr(user_id, custom_attr)

            pub_nsq_msg(TOPIC_USER_CUSTOM_ATTR_MODIFIED, {"user_id": user_id})


    def edit_user(self, param, responsible_person_id):
        """
        编辑用户
        param: ncTEditUserParam
        """
        # 检查用户是否存在
        self.check_user_exists(param.id)

        # 检查登录名是否合法
        if param.account is not None:
            # 检查登录名是否合法
            param.account = self._is_login_name_valid(param.account)

            # 检查账号是否存在
            self.check_user_exists_by_account(param.account)

            # 新建账号不能和保留管理员账号重复
            if param.account in self.remain_accounts:
                raise_exception(exp_msg=_("remain admin account"),
                                exp_num=ncTShareMgntError.NCT_REMAIN_ADMIN_ACCOUNT)

            if param.account.lower() in list(self.get_all_admin_account().values()):
                raise_exception(exp_msg=_("account confict with admin"),
                                exp_num=ncTShareMgntError.NCT_ACCOUNT_CONFICT_WITH_ADMIN)

        # 检查显示名是否合法
        if param.displayName is not None:
            param.displayName = self._is_display_name_valid(param.displayName)
            self.check_user_display_name_exists(param.id, param.displayName)
            self.is_confict_with_lib_name_display_name_error(param.displayName)

        # 检查用户身份证号合法性
        if param.idcardNumber is not None:
            param.idcardNumber = self.add_is_idcardNumber_valid(
                param.idcardNumber, True)
            self.check_user_exists_by_idcardNumber_id(
                param.idcardNumber, param.id)

        # 检查邮箱是否合法
        if param.email is not None:
            param.email = param.email.strip()
            self.check_user_email(param.id, param.email)

        # 检查排序权重
        if param.priority is not None:
            self.check_user_priority(param.priority)

        # 检查密级
        if param.csfLevel is not None:
            self.check_user_csflevel(param.csfLevel)

        # 检查密级2
        if param.csfLevel2 is not None:
            self.check_user_csflevel2(param.csfLevel2)

        # 检查密码合法性
        if param.pwd:
            if sha2_encrypt(param.pwd) != self.user_default_password.sha2_pwd:
                self.check_password_valid(param.pwd)

        # 检查密码管控
        if param.pwdControl is not None:
            sql = """
            SELECT `f_pwd_control` FROM `t_user` WHERE `f_user_id` = %s
            """
            result = self.r_db.one(sql, param.id)
            if param.pwdControl:
                self.modify_control_password(param.id, param.pwd)
            elif result['f_pwd_control'] == 1 or (param.pwd and sha2_encrypt(param.pwd) == self.user_default_password.sha2_pwd):
                self.reset_password(param.id)
            param.pwdControl = 1 if param.pwdControl else 0

        # 检查用户手机号
        if param.telNumber:
            param.telNumber = param.telNumber.strip()
            # 检查手机号合法性
            if not check_tel_number(param.telNumber):
                raise_exception(exp_msg=_("IDS_INVALID_TEL_NUMBER"),
                                exp_num=ncTShareMgntError.NCT_INVALID_TEL_NUMBER)
            # 检查手机号是否冲突
            self.check_tel_number_confict_by_id(param.id, param.telNumber)

        # 检查用户账号有效期
        if param.expireTime is not None and param.expireTime != -1 and  \
                param.expireTime < int(BusinessDate.time()):
            raise_exception(exp_msg=_("IDS_INVALID_DATE"),
                            exp_num=ncTShareMgntError.NCT_INVALID_DATETIME)

        # 检查上级有效性：上级必须是普通用户，且不能是自己，不能是管理员
        if param.managerID is not None and param.managerID != "":
            if param.id == param.managerID or param.managerID in [NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT]:
                raise_exception(exp_msg=_("IDS_INVALID_MANAGER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

            bExist = self.check_user_exists(param.managerID, False)
            if not bExist:
                raise_exception(exp_msg=_("IDS_INVALID_MANAGER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

        # 检查用户编码
        if param.code is not None:
            param.code = self.check_user_code(param.code, param.id)

        # 检查用户岗位
        if param.position is not None:
            param.position = self.check_user_position(param.position)

        # 构造修改用户基本信息的sql
        tmp = ""
        change_Name = False
        param_list = []
        user_modify_info = {}

        # 判断用户显示名是否发生变化
        tmpSql = """
        SELECT f_display_name, f_mail_address, f_tel_number FROM t_user WHERE f_user_id = %s
        """
        result = self.r_db.one(tmpSql, param.id)
        if param.displayName:
            tmp += ("f_display_name='%s'," %
                    escape_format_percent(self.w_db.escape(param.displayName)))

            if result and result['f_display_name'] != param.displayName:
                change_Name = True
        if param.remark is not None:
            param.remark = self._is_remark_valid(param.remark)
            tmp += ("f_remark= %s,")
            param_list.append(param.remark)
        if param.idcardNumber is not None:
            tmp += ("f_idcard_number='%s'," %
                    self.w_db.escape(param.idcardNumber))
        if param.email is not None:
            tmp += ("f_mail_address='%s'," % self.w_db.escape(param.email))
            if result and result['f_mail_address'] != param.email:
                user_modify_info["new_email"] = param.email
        if param.priority:
            tmp += ("f_priority=%d," % param.priority)
        if param.csfLevel:
            tmp += ("f_csf_level=%d," % param.csfLevel)
        if param.csfLevel2:
            tmp += ("f_csf_level2=%d," % param.csfLevel2)
        if param.ossId is not None and param.ossId != "null":
            self.check_oss_id(param.ossId)
            tmp += ("f_oss_id='%s'," % self.w_db.escape(param.ossId))
        if param.pwdControl is not None:
            tmp += ("f_pwd_control=%d," % param.pwdControl)
        if param.telNumber is not None:
            tmp += ("f_tel_number='%s'," % self.w_db.escape(param.telNumber))
            if result and result['f_tel_number'] != param.telNumber:
                user_modify_info["new_telephone"] = param.telNumber
        if param.expireTime is not None:
            tmp += ("f_expire_time='%s'," % param.expireTime)
        if param.managerID is not None:
            tmp += ("f_manager_id='%s'," % param.managerID)
        if param.account is not None:
            tmp += ("f_login_name='%s'," % self.w_db.escape(param.account))
        if param.code is not None:
            tmp += ("f_code='%s'," % param.code)
        if param.position is not None:
            tmp += ("f_position=%s,")
            param_list.append(param.position)
        if tmp == "":
            return

        # 去掉末尾的,
        tmp = tmp[0:len(tmp) - 1]

        sql = """
        update t_user set {0} where f_user_id = %s
        """.format(tmp)
        param_list.append(param.id)

        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()
        cursor.execute(sql, tuple(param_list))

        # 过期禁用用户自动启用
        self.__enable_expired_user(param.id)

        if param.displayName and change_Name:
            # 发送用户显示名更新nsq消息
            pub_nsq_msg(TOPIC_ORG_NAME_MODIFY, {
                        "id": param.id, "new_name": param.displayName, "type": "user"})

        if len(user_modify_info) > 0:
            # 发送用户信息更新nsq消息
            user_modify_info["user_id"] = param.id
            pub_nsq_msg(TOPIC_USER_MODIFIED, user_modify_info)

    def check_adminId(self, adminId):
        """
        检查是否是管理员id
        """
        adminId_list = [NCT_USER_ADMIN, NCT_USER_AUDIT,
                        NCT_USER_SYSTEM, NCT_USER_SECURIT]
        if adminId not in adminId_list:
            raise_exception(exp_msg=_("IDS_USER_NOT_SUPER_ADMIN"),
                            exp_num=ncTShareMgntError.NCT_USER_NOT_SUPER_ADMIN)

    def edit_admin_account(self, adminId, account):
        """
        编辑内置管理员账号
        @param adminId: 管理员账号id
        @param account: 管理员账号名
        """
        # 检查是否是管理员
        self.check_adminId(adminId)

        # 检查名字是否合法
        account = self._is_login_name_valid(account)

        # 检查名字是否重名
        user_info = self.get_user_by_loginname(account)
        if user_info and adminId != user_info.id:
            raise_exception(exp_msg=_("duplicate login name"),
                            exp_num=ncTShareMgntError.NCT_DUPLICATED_LOGIN_NAME)

        # 构造修改用户基本信息的sql
        sql = """
        update t_user
        set f_login_name = %s
        where f_user_id = %s
        """
        self.w_db.query(sql, account, adminId)

    def edit_user_priority(self, userId, priority):
        """
        编辑用户的排序权重
        """
        # 检查用户是否存在
        self.check_user_exists(userId)

        # 检查用户权重是否合法
        self.check_user_priority(priority)

        # 修改用户
        sql = """
        UPDATE `t_user` SET `f_priority` = %s
        WHERE `f_user_id` = %s
        """
        self.w_db.query(sql, priority, userId)

    def edit_user_oss_id(self, userId, ossId):
        """
        编辑用户的对象存储
        """
        # 检查用户是否存在
        self.check_user_exists(userId)
        # 检查存储是否可用
        if ossId is None or ossId == "null":
            ossId = ""
        else:
            self.check_oss_id(ossId)

        # 修改用户
        sql = """
        UPDATE `t_user` SET `f_oss_id` = %s
        WHERE `f_user_id` = %s
        """
        self.w_db.query(sql, ossId, userId)

    def modify_password(self, account, old_password, new_password, option=None):
        """
        修改密码
        """
        # 如果是默认密码则报错
        if sha2_encrypt(new_password) == self.user_default_password.sha2_pwd:
            raise_exception(exp_msg=_("password is initial"),
                            exp_num=ncTShareMgntError.
                            NCT_PASSWORD_IS_INITIAL)

        # 登录验证码
        uuid = vcode = ""
        isForgetPwd = False
        if option:
            uuid = option.uuid if option.uuid else uuid
            vcode = option.vcode if option.vcode else vcode
            isForgetPwd = option.isForgetPwd if option.isForgetPwd else isForgetPwd

        # 检查用户是否存在
        sql = """
        SELECT `f_user_id`, `f_auth_type`, `f_password`, `f_sha2_password`, `f_pwd_control`, `f_pwd_error_cnt` FROM `t_user`
        WHERE `f_login_name` = %s
        """
        db_user = self.r_db.one(sql, account)
        if not db_user and self.config_manage.get_custom_config_of_bool("id_card_login_status"):
            account = bytes.decode(des_encrypt_with_padzero(global_info.des_key,
                                                            account,
                                                            global_info.des_key))
            sql = """
            SELECT `f_login_name`, `f_user_id`, `f_auth_type`, `f_sha2_password`, `f_password`, `f_pwd_control`, `f_pwd_error_cnt` FROM `t_user`
            WHERE `f_idcard_number` = %s
            """
            db_user = self.r_db.one(sql, account)
            if db_user:
                account = db_user['f_login_name']

        try:
            # 校验验证码
            if isForgetPwd:
                self.vcode_manage.verify_vcode_info(
                    uuid, vcode, ncTVcodeType.NUM_VCODE, delete_after_check=False)
            else:
                if uuid or self.vcode_manage.is_user_need_check_vcode(db_user):
                    self.vcode_manage.verify_vcode_info(
                        uuid, vcode, ncTVcodeType.IMAGE_VCODE, delete_after_check=False)
            if not db_user:
                raise_exception(exp_msg=_("invalid account or password"),
                                exp_num=ncTShareMgntError.
                                NCT_CHECK_PASSWORD_FAILED)

            # 只有本地用户才能修改密码
            if db_user["f_auth_type"] != 1:
                raise_exception(exp_msg=_("IDS_CANT_MODIFY_NON_LOCAL_USER_PASSWORD"),
                                exp_num=ncTShareMgntError.
                                NCT_CANNOT_MODIFY_NONLOCAL_USER_PASSWORD)

            # 开启密码管控的用户不能修改密码
            if db_user['f_pwd_control']:
                raise_exception(exp_msg=_("IDS_CANNOT_MODIFY_CONTROL_PASSWORD"),
                                exp_num=ncTShareMgntError.
                                NCT_CANNOT_MODIFY_CONTROL_PASSWORD)

            user_info = self.get_user_by_loginname(account)
            user_id = user_info.id
            pwd_config = self.get_password_config()

            # 检查用户是否被锁定
            if pwd_config.lockStatus:
                if self.check_user_locked(user_id):
                    self.handle_pwd_lock_except(user_id)
            if not isForgetPwd:
                # 检查用户旧密码是否正确
                if (db_user['f_password'] != "" and encrypt_pwd(old_password) != db_user['f_password'].lower()) or (db_user['f_sha2_password'] != "" and sha2_encrypt(old_password) != db_user['f_sha2_password']):
                    if pwd_config.lockStatus:
                        # 更新用户登录锁定信息
                        self.modify_pwd_lock_info(user_id, False)

                        # 密码错误异常处理
                        try:
                            self.handle_pwd_err_except(user_id)
                        except ncTException as ex:
                            if ex.errID == ncTShareMgntError.NCT_INVALID_ACCOUNT_OR_PASSWORD:
                                raise_exception(exp_msg=_("invalid account or password"),
                                                exp_num=ncTShareMgntError.NCT_CHECK_PASSWORD_FAILED)
                            else:
                                raise ex
                    else:
                        raise_exception(exp_msg=_("invalid account or password"),
                                        exp_num=ncTShareMgntError.
                                        NCT_CHECK_PASSWORD_FAILED)
                # 密码正确，则重置错误信息
                else:
                    self.modify_pwd_lock_info(user_id, True)

            # 检查密码合法性
            self.check_password_valid(new_password)

            des_pwd = ''
            if self.is_des_encrypt():
                des_pwd = bytes.decode(des_encrypt(global_info.des_key,
                                                   new_password,
                                                   global_info.des_key))

            # 记录用户的ntlm密码
            ntlm_password = ntlm_md4(new_password)

            sql = """
            UPDATE `t_user`
            SET `f_password` = %s,
            `f_des_password` = %s,
            `f_ntlm_password` = %s,
            `f_pwd_timestamp`= %s,
            `f_sha2_password`= %s
            WHERE `f_login_name` = %s
            """
            now = BusinessDate.now().strftime("%Y-%m-%d %H:%M:%S")
            self.w_db.query(sql, "", des_pwd, ntlm_password,
                            now, sha2_encrypt(new_password), account)
            if isForgetPwd:
                self.vcode_manage.delete_vcode_info(uuid)
        except ncTException as ex:
            if uuid and not isForgetPwd:
                self.vcode_manage.delete_vcode_info(uuid)
            if ex.errID == ncTShareMgntError.NCT_CHECK_PASSWORD_FAILED and    \
                    self.vcode_manage.get_vcode_config().isEnable and not self.get_password_config().lockStatus:
                if db_user:
                    self.modify_pwd_lock_info(db_user['f_user_id'], False)

            if db_user:
                # 用户名或密码错误时，会更新用户密码错误次数，db_user 中存放的是未更改的数据，需要重新读取
                sql = """
                SELECT `f_pwd_error_cnt` FROM `t_user`
                WHERE `f_login_name` = %s
                """
                db_user = self.r_db.one(sql, account)
            detail = {}
            if ex.errDetail:
                detail = json.loads(ex.errDetail)
            detail["isShowStatus"] = self.vcode_manage.is_user_need_display_vcode(
                db_user)
            raise_exception(exp_msg=ex.expMsg, exp_num=ex.errID,
                            exp_detail=json.dumps(detail))

    def check_password_valid(self, new_password):
        """
        """
        # 检查密码合法性
        pwd_config = self.get_password_config()
        if not pwd_config.strongStatus:
            b_valid = check_is_valid_password(new_password)
            if not b_valid:
                raise_exception(exp_msg=_("invalid password"),
                                exp_num=ncTShareMgntError.
                                NCT_INVALID_PASSWORD)
        else:
            b_valid = check_is_strong_password(new_password)
            if not b_valid:
                raise_exception(exp_msg=_("invalid strong password"),
                                exp_num=ncTShareMgntError.
                                NCT_INVALID_STRONG_PASSWORD)

    def modify_control_password(self, uid, new_password):
        """
        修改管控密码
        """
        # 检查用户是否存在
        sql = """
        SELECT `f_auth_type`,`f_password`, `f_pwd_control`, `f_des_password`, `f_sha2_password` FROM `t_user`
        WHERE `f_user_id` = %s
        """
        result = self.r_db.one(sql, uid)
        if not result:
            raise_exception(exp_msg=_("invalid account or password"),
                            exp_num=ncTShareMgntError.
                            NCT_CHECK_PASSWORD_FAILED)

        # 只有本地用户才能修改密码
        if result["f_auth_type"] != 1:
            raise_exception(exp_msg=_("IDS_CANT_MODIFY_NON_LOCAL_USER_PASSWORD"),
                            exp_num=ncTShareMgntError.
                            NCT_CANNOT_MODIFY_NONLOCAL_USER_PASSWORD)

        # 判断密码是否需要修改
        if result["f_des_password"] and result["f_password"] != "" and result["f_password"] == encrypt_pwd(new_password):
            return
        elif result["f_des_password"] and result["f_sha2_password"] != "" and result["f_sha2_password"] == sha2_encrypt(new_password):
            return

        self.modify_pwd_lock_info(uid, True)

        # 检查密码合法性
        self.check_password_valid(new_password)

        # 开启密码管控的用户需要存储des加密密码
        des_pwd = bytes.decode(des_encrypt(global_info.des_key,
                                           new_password,
                                           global_info.des_key))

        # 记录用户的ntlm密码
        ntlm_password = ntlm_md4(new_password)

        sql = """
        UPDATE `t_user`
        SET `f_password` = %s,
        `f_des_password` = %s,
        `f_ntlm_password` = %s,
        `f_pwd_timestamp`= %s,
        `f_sha2_password`= %s
        WHERE `f_user_id` = %s
        """
        now = BusinessDate.now().strftime("%Y-%m-%d %H:%M:%S")
        self.w_db.query(sql, "", des_pwd, ntlm_password,
                        now, sha2_encrypt(new_password), uid)
         # 修改用户密码发送修改密码消息
        pub_nsq_msg(TOPIC_USER_PASSWORD_MODIFIED, {"user_id": uid})

    def reset_password(self, user_id):
        """
        重置密码
        """
        des_pwd = ''
        if self.is_des_encrypt():
            des_pwd = self.user_default_password.des_pwd

        self.check_user_exists(user_id)
        sql = """
        UPDATE `t_user`
        SET `f_password` = '',
        `f_des_password` = %s,
        `f_sha2_password` = %s,
        `f_ntlm_password` = %s,
        `f_pwd_timestamp`= %s,
        `f_pwd_error_latest_timestamp` = %s,
        `f_pwd_error_cnt` = 0
        WHERE `f_user_id` = %s
        """
        now = BusinessDate.now().strftime("%Y-%m-%d %H:%M:%S")
        self.w_db.query(sql, des_pwd, self.user_default_password.sha2_pwd,
                        self.user_default_password.ntlm_pwd, now, now, user_id)
        # 修改用户密码发送修改密码消息
        pub_nsq_msg(TOPIC_USER_PASSWORD_MODIFIED, {"user_id": user_id})

    def reset_all_password(self, new_password):
        """
        重置除管理员外的所有用户密码
        """
        pwd = sha2_encrypt(new_password)
        des_pwd = ''
        if self.is_des_encrypt():
            des_pwd = bytes.decode(des_encrypt(global_info.des_key,
                                               new_password,
                                               global_info.des_key))

        # 记录用户的ntlm密码
        ntlm_password = ntlm_md4(new_password)

        sql = """
        UPDATE `t_user`
        SET `f_password` = '{0}',
        `f_des_password` = '{1}',
        `f_ntlm_password` = '{2}',
        `f_sha2_password` = '{3}',
        `f_pwd_timestamp`= %s,
        `f_pwd_error_latest_timestamp` = %s,
        `f_pwd_error_cnt` = 0
        WHERE `f_user_id` <> '{4}'
        AND `f_user_id` <> '{5}'
        AND `f_user_id` <> '{6}'
        AND `f_user_id` <> '{7}'
        """.format("",
                   des_pwd,
                   ntlm_password,
                   pwd,
                   NCT_USER_ADMIN,
                   NCT_USER_AUDIT,
                   NCT_USER_SYSTEM,
                   NCT_USER_SECURIT)
        now = BusinessDate.now().strftime("%Y-%m-%d %H:%M:%S")
        self.w_db.query(sql, now, now)

    def get_des_password(self, key, user_id):
        """
        根据key获取用户des解密的密码
        """
        self.check_user_exists(user_id)
        sql = """
        SELECT `f_auth_type`,`f_des_password`
        FROM `t_user`
        WHERE `f_user_id` = %s
        """
        result = self.r_db.one(sql, user_id)

        if not result:
            raise_exception(exp_msg=_("user not exists"),
                            exp_num=ncTShareMgntError.
                            NCT_USER_NOT_EXIST)

        if result["f_auth_type"] != 1:
            raise_exception(exp_msg=_("IDS_CANT_MODIFY_NON_LOCAL_USER_PASSWORD"),
                            exp_num=ncTShareMgntError.
                            NCT_CANNOT_MODIFY_NONLOCAL_USER_PASSWORD)

        decrypy_pwd = ''
        decrypy_pwd = bytes.decode(des_decrypt(
            key, result['f_des_password'], key))
        return decrypy_pwd

    def set_user_status(self, user_id, status):
        """
        设置用户状态
        """
        sql = """
        SELECT `f_status`, `f_auto_disable_status`, `f_oss_id`
        FROM `t_user`
        WHERE `f_user_id` = %s
        """
        result = self.r_db.one(sql, user_id)
        if not result:
            raise_exception(exp_msg=_("user not exists"),
                            exp_num=ncTShareMgntError.
                            NCT_USER_NOT_EXIST)

        # 用户账号已过期，需要先重新设置有效期
        if result["f_auto_disable_status"] & USER_EXPIRE_DISABLED:
            raise_exception(exp_msg=_("IDS_USER_ACCOUNT_HAS_EXPIRED"),
                            exp_num=ncTShareMgntError.NCT_USER_ACCOUNT_HAS_EXPIRED)

        # 用户已经被第三方认证系统删除，不允许设置状态
        if result['f_status'] == ncTUsrmUserStatus.NCT_STATUS_DELETE:
            raise_exception(exp_msg=_("can not set delete user status"),
                            exp_num=ncTShareMgntError.
                            NCT_USER_HAS_BEEN_DELETED)

        auto_disable_status = result['f_auto_disable_status']
        if status:
            # 启用用户需要检查用户授权数
            if user_id not in (NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT):
                if self.is_user_num_overflow():
                    raise_exception(exp_msg=_("user num overflow"),
                                    exp_num=ncTShareMgntError.NCT_USER_NUM_OVERFLOW)

            status = ncTUsrmUserStatus.NCT_STATUS_ENABLE
            auto_disable_status = 0

        else:
            status = ncTUsrmUserStatus.NCT_STATUS_DISABLE

        # 获取当前时间
        lastRequestTime = BusinessDate.now().strftime('%Y-%m-%d %H:%M:%S')

        sql = """
        UPDATE `t_user`
        SET `f_status` = %s, `f_auto_disable_status` = %s, `f_last_request_time` = %s
        WHERE `f_user_id` = %s
        """
        self.w_db.query(sql, status, auto_disable_status, lastRequestTime, user_id)

        if status == ncTUsrmUserStatus.NCT_STATUS_ENABLE:
            pub_nsq_msg(TOPIC_USER_STATUS_CHANGED, {"user_id": user_id, "status": True})
        else:
            pub_nsq_msg(TOPIC_USER_STATUS_CHANGED, {"user_id": user_id, "status": False})

    def set_user_auto_disable_status(self, user_id, status):
        """
        设置用户自动禁用标志
        """
        sql = """
        UPDATE `t_user`
        SET `f_auto_disable_status` = `f_auto_disable_status` | %s
        WHERE `f_user_id` = %s
        """
        self.w_db.query(sql, status, user_id)
        # 自动禁用用户发送用户禁用消息
        bStatus = False
        if status == 0:
            bStatus = True
        pub_nsq_msg(TOPIC_USER_STATUS_CHANGED, {"user_id": user_id, "status": bStatus})

    def check_user_status(self, user_id):
        """
        检查用户状态，是否启用/密码是否过期
        """
        sql = """
        SELECT `f_status`, `f_auto_disable_status`, `f_pwd_control`, `f_auth_type`
        FROM `t_user`
        WHERE `f_user_id` = %s
        """
        db_user = self.r_db.one(sql, user_id)
        if not db_user:
            raise_exception(exp_msg=_("user not exists"),
                            exp_num=ncTShareMgntError.
                            NCT_USER_NOT_EXIST)

        if db_user['f_status'] != 0 or db_user['f_auto_disable_status'] != 0:
            raise_exception(exp_msg=_("user is disabled"),
                            exp_num=ncTShareMgntError.NCT_USER_DISABLED)

        # 判断用户类型，用户类型为本地用户时受用户密码有效期管控
        if db_user['f_auth_type'] == ncTUsrmUserType.NCT_USER_TYPE_LOCAL:
            pwdExpire = self.check_password_expire(user_id)
            if pwdExpire:
                if db_user['f_pwd_control']:
                    raise_exception(exp_msg=_("IDS_CONTROLED_PASSWORD_EXPIRED"),
                                    exp_num=ncTShareMgntError.NCT_CONTROLED_PASSWORD_EXPIRE)
                raise_exception(exp_msg=_("password expire"),
                                exp_num=ncTShareMgntError.NCT_PASSWORD_EXPIRE)

    def get_trisystem_status(self):
        """
        检查是否开启三权分立.
        返回值：True：三权分立开启
                False:三权分立关闭
        """
        return int(self.config_manage.get_config("enable_tri_system_status"))

    def set_trisystem_status(self, enable):
        """
        开启或关闭三权分立系统.
        参数：enable:
                    True:开启
                    False:关闭
        """
        # 三权分立一旦开启，超级管理员角色自动变成系统管理员，关闭则清空三权分立角色
        from src.modules.role_manage import RoleManage
        RoleManage().change_supper_role(enable)

        self.config_manage.set_config('enable_tri_system_status', enable)

    def check_register_info(self, register_id, cert_id, real_name):
        """
        检查注册信息：
        参数：
        返回值:
            可以自注册，返回班级号
            不能自注册, 抛异常
        """
        with open("/sysvol/conf/CUG/students.csv", 'r') as f:
            csv_reader = csv.reader(f)
            for row in csv_reader:
                if csv_reader.line_num == 1:
                    continue
                if (len(row) == 4 and row[0] == register_id):
                    if row[3] != cert_id:
                        raise_exception(_('certificate ID invalid'))
                    if row[1] != real_name:
                        raise_exception(_('real name do not exist'))
                    return row[2]

        raise_exception(_('register ID invalid'))

    def get_register_departid(self, class_id):
        """
        获取自注册用户的部门
        """
        if len(class_id) < 6:
            return NCT_UNDISTRIBUTE_USER_GROUP

        sql = """
        SELECT `f_department_id`
        FROM `t_department`
        WHERE `f_third_party_id` = %s
        """
        result = self.r_db.one(sql, str(class_id))

        if result:
            return result['f_department_id']

        return NCT_UNDISTRIBUTE_USER_GROUP

    def self_registration(self, register_id, cert_id, real_name, pwd):
        """
        用户自注册
        """
        # 检查用户注册信息并获取班级号
        class_id = self.check_register_info(register_id, cert_id, real_name)

        # 根据班级号获取用户部门id
        depart_id = self.get_register_departid(class_id)

        # 获取唯一显示名
        display_name = self.get_unique_displayname(real_name)

        # 检查显示名，如果显示名存在相同，则显示名后添加序号,如xx01,xx02
        user_info = ncTUsrmAddUserInfo()
        user = ncTUsrmUserInfo()
        user_info.password = pwd
        user.loginName = register_id
        user.displayName = display_name
        user.departmentIds = [depart_id]
        user.status = 0

        user_info.user = user

        # 添加用户到部门
        try:
            result = self.add_user(user_info, NCT_USER_ADMIN)
            return result
        except Exception as ex:
            if ex.errID == 20105:
                object.__setattr__(ex, 'expMsg', _("user has been registed"))
            raise ex

    def set_password_config(self, pwd_config):
        """
        设置用户密码配置信息
        """
        update_config_sql = """
        UPDATE `t_sharemgnt_config`
        SET `f_value` = %s
        WHERE `f_key` = %s
        """

        update_login_timestamp_sql = """
        UPDATE `t_user`
        SET `f_pwd_timestamp` = %s
        """

        update_pwd_err_timestatmp_sql = """
        UPDATE `t_user`
        SET `f_pwd_error_latest_timestamp` = %s,
        `f_pwd_error_cnt` = 0
        """

        get_system_protection_level_sql = f"""
        SELECT f_value
        FROM {get_db_name('policy_mgnt')}.t_policies
        WHERE f_name = 'system_protection_levels'
        """
        result = self.r_db.one(get_system_protection_level_sql)
        secret_mode_status = json.loads(result["f_value"])["level"]

        latest_pwd_config = self.get_password_config()

        if pwd_config.strongStatus is not None:
            if secret_mode_status and not pwd_config.strongStatus:
                raise_exception(exp_msg="password must is strong password",
                                exp_num=500)
            elif secret_mode_status and not latest_pwd_config.strongStatus:
                self.w_db.query(update_config_sql, 1, 'strong_pwd_status')
            else:
                strong_status = 1 if pwd_config.strongStatus else 0
                self.w_db.query(update_config_sql, strong_status, 'strong_pwd_status')
        if (pwd_config.strongPwdLength is not None) and (pwd_config.strongStatus):
            if secret_mode_status and ((pwd_config.strongPwdLength < MIN_STRONG_PWD_LENGTH2) or  \
                    (pwd_config.strongPwdLength > MAX_STRONG_PWD_LENGTH)):
                raise_exception(exp_msg=_("IDS_INVALID_STRONG_PWD_LENGTH2"),
                                exp_num=ncTShareMgntError.NCT_INVALID_STRONG_PWD_LENGTH)
            elif (pwd_config.strongPwdLength < MIN_STRONG_PWD_LENGTH) or  \
                    (pwd_config.strongPwdLength > MAX_STRONG_PWD_LENGTH):
                raise_exception(exp_msg=_("IDS_INVALID_STRONG_PWD_LENGTH"),
                                exp_num=ncTShareMgntError.NCT_INVALID_STRONG_PWD_LENGTH)
            self.w_db.query(update_config_sql,
                            pwd_config.strongPwdLength, 'strong_pwd_length')

        if pwd_config.passwdErrCnt is not None:
            if (secret_mode_status and (pwd_config.passwdErrCnt < 1 or pwd_config.passwdErrCnt > 5)):
                raise_exception(exp_msg=_("IDS_INVALID_PASSWORD_ERR_CNT_2"),
                                exp_num=ncTShareMgntError.NCT_INVALID_PASSWORD_ERR_CNT)
            elif pwd_config.passwdErrCnt < 1 or pwd_config.passwdErrCnt > 99:
                raise_exception(exp_msg=_("IDS_INVALID_PASSWORD_ERR_CNT"),
                                exp_num=ncTShareMgntError.NCT_INVALID_PASSWORD_ERR_CNT)
            self.w_db.query(update_config_sql,
                            pwd_config.passwdErrCnt, 'pwd_err_cnt')

        if pwd_config.expireTime is not None:
            latest_expire_time = latest_pwd_config.expireTime
            if pwd_config.expireTime != latest_expire_time:
                if secret_mode_status and secret_mode_status == 1 and (pwd_config.expireTime == -1 or pwd_config.expireTime > 30):
                    raise_exception(exp_msg="classified expire_time must less than 30",
                                exp_num=500)
                elif secret_mode_status and secret_mode_status == 2 and (pwd_config.expireTime == -1 or pwd_config.expireTime > 7):
                    raise_exception(exp_msg="confidential expire_time must less than 7",
                                exp_num=500)
                elif secret_mode_status and secret_mode_status == 3 and (pwd_config.expireTime == -1 or pwd_config.expireTime > 3):
                    raise_exception(exp_msg="confidential expire_time must less than 3",
                                exp_num=500)
                else:
                    self.w_db.query(update_config_sql, pwd_config.expireTime, 'pwd_expire_time')

                # 重置所有用户的密码时效
                now = BusinessDate.now().strftime("%Y-%m-%d %H:%M:%S")
                self.w_db.query(update_login_timestamp_sql, now)

        if pwd_config.lockStatus is not None:
            if pwd_config.lockStatus != latest_pwd_config.lockStatus:
                lock_status = 1 if pwd_config.lockStatus else 0
                self.w_db.query(update_config_sql,
                                lock_status, 'enable_pwd_lock')

                # 重置所有用户的密码锁定信息
                now = BusinessDate.now().strftime("%Y-%m-%d %H:%M:%S")
                self.w_db.query(update_pwd_err_timestatmp_sql, now)

        if pwd_config.passwdLockTime is not None:
            self.w_db.query(update_config_sql,
                            pwd_config.passwdLockTime, 'pwd_lock_time')

    def get_password_config(self):
        """
        获取用户密码配置信息
        """
        sql = """
        SELECT f_key,f_value FROM t_sharemgnt_config
        WHERE f_key in ('pwd_expire_time','strong_pwd_status', 'strong_pwd_length',
                          'enable_pwd_lock', 'pwd_err_cnt', 'pwd_lock_time')
        """

        res = {}
        for row in self.r_db.all(sql):
            res[row['f_key']] = int(row['f_value'])

        if 'pwd_expire_time' in res and 'strong_pwd_status' in res:
            pwd_config = ncTUsrmPasswordConfig()
            pwd_config.expireTime = res['pwd_expire_time']
            pwd_config.strongStatus = True if res['strong_pwd_status'] == 1 else False
            pwd_config.strongPwdLength = res['strong_pwd_length']
            pwd_config.lockStatus = True if res['enable_pwd_lock'] == 1 else False
            pwd_config.passwdErrCnt = res['pwd_err_cnt'] if 'pwd_err_cnt' in res else 3
            pwd_config.passwdLockTime = res['pwd_lock_time']
            return pwd_config

    def check_password_expire(self, user_id):
        """
        检查用户密码是否过期
        """
        pwd_config = self.get_password_config()
        expire_time = pwd_config.expireTime
        if expire_time == -1:
            return False

        sql = """
        SELECT TIMESTAMPDIFF(HOUR, `f_pwd_timestamp`, %s) as hour_delta
        FROM `t_user` WHERE `f_user_id` = %s
        """
        now = BusinessDate.now().strftime("%Y-%m-%d %H:%M:%S")
        result = self.r_db.one(sql, now, user_id)
        if result:
            hour_delta = result['hour_delta']
            day_delta = hour_delta / 24

            if day_delta < expire_time:
                return False
            else:
                return True

    def modify_pwd_lock_info(self, user_id, b_sucess):
        """
        修改用户密码锁定信息
        """
        pwd_config = self.get_password_config()

        update_sql = """
        UPDATE `t_user`
        SET `f_pwd_error_cnt` = %s,
        `f_pwd_error_latest_timestamp` = %s
        WHERE `f_user_id` = %s
        """

        lock_info_sql = """
        SELECT `f_pwd_error_latest_timestamp`,
        `f_pwd_error_cnt` AS cnt
        FROM `t_user` WHERE `f_user_id` = %s
        """

        # 默认为0，表示成功
        now_cnt = 0

        # 如果失败
        if not b_sucess:
            result = self.r_db.one(lock_info_sql, user_id)
            if result:
                cnt = result['cnt']
                time_delta = BusinessDate.now() - result['f_pwd_error_latest_timestamp']
                minutes = time_delta.seconds / 60

                # 两次失败间隔在5分钟内
                if minutes < 5 and cnt != 0:
                    now_cnt = cnt + 1
                else:
                    now_cnt = 1

        # 更新最新锁定信息
        now = BusinessDate.now().strftime("%Y-%m-%d %H:%M:%S")
        self.w_db.query(update_sql, now_cnt, now, user_id)

    def get_pwd_error_info(self, user_id):
        """
        获取用户密码锁定的信息
        """
        lock_info_sql = """
        SELECT `f_pwd_error_latest_timestamp`,
        `f_pwd_error_cnt` AS cnt
        FROM `t_user` WHERE `f_user_id` = %s
        """
        result = self.r_db.one(lock_info_sql, user_id)
        if result:
            err_cnt = int(result['cnt'])
            time_delta = BusinessDate.now() - result['f_pwd_error_latest_timestamp']
            minutes = time_delta.seconds // 60
            lock_time = int(self.config_manage.get_config("pwd_lock_time"))
            minutes = -1 if minutes >= lock_time else lock_time - minutes
            return err_cnt, minutes

    def handle_pwd_err_except(self, user_id):
        """
        处理密码错误异常
        """
        err_cnt, remain_minutes = self.get_pwd_error_info(user_id)
        pwd_config = self.get_password_config()
        max_err_cnt = pwd_config.passwdErrCnt
        if err_cnt < max_err_cnt:
            raise_exception(exp_msg=_("invalid account or password"),
                            exp_num=ncTShareMgntError.NCT_INVALID_ACCOUNT_OR_PASSWORD)

        # 密码错误次数过多之后被锁定账号
        try:
            self.handle_pwd_lock_except(user_id, remain_minutes)
        except ncTException as ex:
            if ex.errID == ncTShareMgntError.NCT_ACCOUNT_LOCKED:
                object.__setattr__(ex, 'errID', ncTShareMgntError.NCT_PWD_THIRD_FAILED)
            raise ex

    def handle_pwd_lock_except(self, user_id, remain_minutes=None):
        """
        处理密码锁定异常
        """
        # 时间超过设置的解锁时间后，自动解锁
        if remain_minutes is None:
            err_cnt, remain_minutes = self.get_pwd_error_info(user_id)
        detail = {}
        detail["remainlockTime"] = remain_minutes
        raise_exception(exp_msg=_("account has been locked") %
                        str(remain_minutes),
                        exp_num=ncTShareMgntError.NCT_ACCOUNT_LOCKED,
                        exp_detail=json.dumps(detail))

    def check_user_locked(self, user_id):
        """
        检查用户是否被锁定
        """
        err_cnt, minutes = self.get_pwd_error_info(user_id)
        pwd_config = self.get_password_config()
        if not pwd_config.lockStatus:
            return False

        b_locked = True if err_cnt >= pwd_config.passwdErrCnt else False

        # 锁定时间到期则需要解锁
        if minutes == -1:
            self.modify_pwd_lock_info(user_id, True)
            b_locked = False

        return b_locked

    def check_is_responsible_person(self, user_id):
        """
        检查用户是否是管理员
        """
        check_sql = """
        SELECT COUNT(*) AS cnt from `t_department_responsible_person`
        WHERE `f_user_id` = %s
        """
        result = self.r_db.one(check_sql, user_id)
        if result['cnt'] < 1:
            return False
        else:
            return True

    def check_user_responsible_in_other_ou(self, user_id, exclude_depart_id):
        """
        检查用户是否是其他组织部门的管理员
        """
        check_user_responsible_in_other_ou_sql = """
            SELECT COUNT(*) AS cnt FROM `t_department_responsible_person`
            WHERE `f_user_id` = %s AND `f_department_id` != %s
        """
        result = self.r_db.one(
            check_user_responsible_in_other_ou_sql, user_id, exclude_depart_id)
        return True if result['cnt'] else False

    def check_is_responsible_person_of_depart(self, user_id, depart_id, raise_ex=True):
        """
        检查用户是否是指定部门管理员
        """
        check_sql = """
        SELECT `f_user_id` from `t_department_responsible_person`
        WHERE `f_user_id` = %s AND `f_department_id` = %s
        """
        result = self.r_db.one(check_sql, user_id, depart_id)
        if result:
            if raise_ex:
                raise_exception(exp_msg=_("IDS_RESPONSIBLE_PERSON_EXIST"),
                                exp_num=ncTShareMgntError.NCT_RESPONSIBLE_PERSON_EXIST)
            else:
                return True
        return False

    def get_third_id2user_id_dict(self):
        """
        获取第三方id和用户id的影射
        """
        sql = """
        select f_third_party_id,f_user_id from t_user
        where f_auth_type = 3
        """
        results = self.r_db.all(sql)

        ret_infos = {}
        for result in results:
            ret_infos[result["f_third_party_id"]] = result["f_user_id"]

        return ret_infos

    def update_t_user_fields(self, user_id, fields):
        """
        更新t_user表中的用户字段信息
        """
        sql = """
        update t_user set {0} where f_user_id = %s
        """ .format(fields)
        self.w_db.query(sql, user_id)

    def set_password_control(self, user_id, param):
        """
        设置用户的密码管控信息
        """
        self.check_user_exists(user_id)

        # 进行rsa解密
        password = param.password
        if password:
            password = bytes.decode(eisoo_rsa_decrypt(password))

        # 为用户解锁
        if not param.lockStatus:
            self.modify_pwd_lock_info(user_id, True)

        # 设置密码密码
        if param.pwdControl:
            # 如果是初始密码，则报错
            if sha2_encrypt(password) == self.user_default_password.sha2_pwd:
                raise_exception(exp_msg=_("password is initial"),
                                exp_num=ncTShareMgntError.
                                NCT_PASSWORD_IS_INITIAL)

            self.modify_control_password(user_id, password)
        elif password == "123456":
            self.reset_password(user_id)

        param.pwdControl = 1 if param.pwdControl else 0

        # 更新用户密码管控状态
        tmp = ("f_pwd_control=%d" % param.pwdControl)
        self.update_t_user_fields(user_id, tmp)

    def get_password_control(self, user_id):
        """
        获取用户的密码管控配置
        """

        self.check_user_exists(user_id)
        user_pwd_control_config = ncTUsrmPwdControlConfig()

        t_user_info = self.get_t_user_info_by_id(user_id)
        user_pwd_control_config.pwdControl = t_user_info['f_pwd_control']
        if user_pwd_control_config.pwdControl:
            user_pwd_control_config.password = bytes.decode(des_decrypt(global_info.des_key,
                                                                        t_user_info.get(
                                                                            'f_des_password', ''),
                                                                        global_info.des_key))
        user_pwd_control_config.lockStatus = self.check_user_locked(user_id)

        return user_pwd_control_config

    def get_t_user_info_by_id(self, user_id):
        """
        通过用户id获取用户基本信息
        """
        sql = """
        SELECT *
        FROM `t_user`
        WHERE `f_user_id` = %s
        """
        result = self.r_db.one(sql, user_id)
        return result

    @property
    def user_default_password(self):
        """
        获取用户默认密码
        """
        pwd_info = UserDefaultPassword()
        sql = f"""
        select `key`, `value` from `{get_db_name("user_management")}`.`option` where `key` in ('user_defalut_des_password',
            'user_defalut_ntlm_password', 'user_defalut_sha2_password', 'user_defalut_md5_password') and 1=%s
        """
        results = self.r_db.all(sql, 1)
        for result in results:
            if result['key'] == 'user_defalut_des_password':
                pwd_info.des_pwd = result['value']
            elif result['key'] == 'user_defalut_ntlm_password':
                pwd_info.ntlm_pwd = result['value']
            elif result['key'] == 'user_defalut_sha2_password':
                pwd_info.sha2_pwd = result['value']
            elif result['key'] == 'user_defalut_md5_password':
                pwd_info.md5_pwd = result['value']

        return pwd_info

    @property
    def admin_default_passwd(self):
        if UserManage.__admin_default_passwd is None:
            UserManage.__admin_default_passwd = ""
        return UserManage.__admin_default_passwd

    def get_export_user_info(self):
        """
        获取导出用户信息
        """
        # 获取所有用户信息
        sql = """
        SELECT `f_user_id`, `f_login_name`, `f_display_name`, `f_password`,
               `f_mail_address`, `f_status`
        FROM `t_user`
        WHERE `f_user_id` <> %s
        AND `f_user_id` <> %s
        AND `f_user_id` <> %s
        AND `f_user_id` <> %s
        """

        db_user_list = self.r_db.all(sql, NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM,
                                     NCT_USER_SECURIT)

        result = []
        for db_user in db_user_list:
            db_user_tmp = {}
            db_user_item = {}
            db_user_item['f_login_name'] = db_user['f_login_name']
            db_user_item['f_display_name'] = db_user['f_display_name']
            db_user_item['f_password'] = db_user['f_password']
            db_user_item['f_mail_address'] = db_user['f_mail_address']
            db_user_item['f_status'] = db_user['f_status']

            db_user_tmp[db_user['f_user_id']] = db_user_item
            result.append(db_user_tmp)

        return result

    def get_export_dept_info(self):
        """
        获取需要导出的部门信息
        """
        # 获取部门信息
        sql = """
            SELECT t_department.f_department_id, t_department.f_name,
                   relation.f_parent_department_id`
            FROM t_department
            LEFT JOIN t_department_relation AS relation
            ON t_department.f_department_id = relation.f_department_id
        """

        db_dept_list = self.r_db.all(sql)

        result = []
        for db_dept in db_dept_list:
            db_dept_tmp = {}
            db_dept_item = {}
            db_dept_item['f_name'] = db_dept['f_name']
            db_dept_item['f_parent_department_id'] = db_dept['f_parent_department_id']

            # 获取部门下所有用户id
            sql = """
                SELECT `f_user_id`
                FROM `t_user_department_relation`
                WHERE `f_department_id` = %s
            """
            sub_userids = self.r_db.all(sql, db_dept['f_department_id'])
            db_dept_item['user_ids'] = [sub_userid['f_user_id']
                                        for sub_userid in sub_userids]
            db_dept_tmp[db_dept['f_department_id']] = db_dept_item
            result.append(db_dept_tmp)

        return result

    def get_tenant_manager_ou_id(self, user_id):
        """
        获取租户管理的组织id, 租户模式下只会存在一个管理的组织id
        """
        sql = """
        SELECT `f_department_id`
        FROM `t_department_responsible_person`
        WHERE `f_user_id` = %s
        """
        user_id = self.r_db.one(sql, user_id)
        if user_id:
            return user_id['f_department_id']
        return

    def get_online_user_count(self):
        """
        获取客户端在线人数
        """
        end_time = BusinessDate.now()
        end_time_str = end_time.strftime("%Y-%m-%d %H:%M:%S")

        start_time = end_time - \
            datetime.timedelta(seconds=ACTIVE_INTERVAL_SECONDS)
        start_time_str = start_time.strftime("%Y-%m-%d %H:%M:%S")

        sql = """
        SELECT COUNT(f_user_id) AS cnt
        FROM `t_user`
        WHERE (`f_last_client_request_time` > %s and `f_last_client_request_time` <= %s and `f_last_client_request_time` != `f_create_time`)
        """
        online_user_db = self.r_db.one(sql, start_time_str, end_time_str)
        count = 0
        if online_user_db:
            count = online_user_db['cnt']
        return count

    def update_user_last_request_time(self, userId, lastRequestTime=None):
        """
        更新用户上一次请求时间
        """
        # 修改用户
        if not lastRequestTime:
            lastRequestTime = BusinessDate.now().strftime('%Y-%m-%d %H:%M:%S')
        sql = """
        UPDATE `t_user`
        SET `f_last_request_time` = %s
        WHERE `f_user_id` = %s
        """
        self.w_db.query(sql, lastRequestTime, userId)

    def update_user_last_client_request_time(self, userId, lastRequestTime=None):
        """
        更新用户上一次客户端请求时间
        """
        # 修改用户
        if not lastRequestTime:
            lastRequestTime = BusinessDate.now().strftime('%Y-%m-%d %H:%M:%S')
        sql = """
        UPDATE `t_user`
        SET `f_last_client_request_time` = %s
        WHERE `f_user_id` = %s
        """
        self.w_db.query(sql, lastRequestTime, userId)

    def set_user_freeze_status(self, userId, freezeStatus):
        """
        冻结|解冻用户
        """
        # 检查用户是否存在
        self.check_user_exists(userId)

        sql = """
        UPDATE `t_user`
        SET `f_freeze_status` = %s
        WHERE `f_user_id` = %s
        """
        self.w_db.query(sql, int(freezeStatus), userId)

        if freezeStatus:
            # 发布用户冻结nsq消息
            pub_nsq_msg(TOPIC_USER_FREEZE, {"id": userId})

        pub_nsq_msg(TOPIC_USER_MODIFIED, {"user_id": userId, "frozen": freezeStatus})

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

    def set_user_real_name_status(self, userId, status):
        """
        设置用户实名状态
        """
        # 检查用户是否存在
        self.check_user_exists(userId)

        sql = """
        UPDATE `t_user`
        SET `f_real_name_auth_status` = %s
        WHERE `f_user_id` = %s
        """
        self.w_db.query(sql, int(status), userId)

        if not status:
            # 发布用户未实名nsq消息
            pub_nsq_msg(TOPIC_USER_UNREALNAME, {"id": userId})

    def get_batch_user_infos_by_id(self, batch_userids, first_item=""):
        """
        根据用户id批量获取用户信息，返回dict
        """
        if len(batch_userids) == 0:
            return {}

        groupStr = generate_group_str(batch_userids)
        sql = """
        SELECT f_user_id,f_login_name,f_display_name,f_mail_address,f_tel_number,f_third_party_attr
        FROM t_user
        WHERE f_user_id IN ({0})
        ORDER BY case when f_user_id = %s then 0 else 1 end,
                f_priority, upper(`f_display_name`)
        """.format(groupStr)

        ret_dict = collections.OrderedDict()
        results = self.r_db.all(sql, first_item)
        for info in results:
            ret_dict[info["f_user_id"]] = info

        return ret_dict

    def check_tel_number_confict(self, loginName, telNumber):
        """
        """
        if not telNumber:
            return

        sql = """
        SELECT `f_login_name` FROM `t_user`
        WHERE `f_tel_number` = %s
        AND `f_login_name` <> %s
        """
        result = self.r_db.one(sql, telNumber, loginName)
        if result:
            raise_exception(exp_msg=_("IDS_TEL_NUMBER_EXISTS"),
                            exp_num=ncTShareMgntError.NCT_TEL_NUMBER_EXISTS)

    def check_tel_number_confict_by_id(self, userId, telNumber):
        """
        """
        if not telNumber:
            return

        sql = """
        SELECT `f_user_id` FROM `t_user`
        WHERE `f_tel_number` = %s
        AND `f_user_id` <> %s
        """
        result = self.r_db.one(sql, telNumber, userId)
        if result:
            raise_exception(exp_msg=_("IDS_TEL_NUMBER_EXISTS"),
                            exp_num=ncTShareMgntError.NCT_TEL_NUMBER_EXISTS)

    def check_user_tel_number(self, loginName, telNumber):
        """
        """
        if not telNumber:
            return

        telNumber = telNumber.strip()

        # 检查手机号合法性
        if not check_tel_number(telNumber):
            raise_exception(exp_msg=_("IDS_INVALID_TEL_NUMBER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_TEL_NUMBER)

        # 检查手机号是否冲突
        self.check_tel_number_confict(loginName, telNumber)

    def get_activate_status(self, user_id):
        """
        是否为激活用户
        """
        sql = """
        SELECT `f_activate_status`
        FROM `t_user`
        WHERE `f_user_id` = %s
        """
        result = self.r_db.one(sql, user_id)
        return True if result and int(result["f_activate_status"]) else False

    def update_user_count(self, activate_status):
        """
        更新用户总数、激活用户数
        """
        select_sql = """
        SELECT f_time FROM %s
        WHERE f_time = '%s'
        """
        now = BusinessDate.now()

        if activate_status:
            current_day = now.strftime("%Y-%m-%d")
            result = self.r_db.one(select_sql % (
                "t_active_user_day", current_day))
            if result:
                sql = """
                UPDATE t_active_user_day
                SET f_activate_count = f_activate_count + 1
                WHERE f_time = '%s'
                """
            else:
                sql = """
                INSERT INTO t_active_user_day (f_activate_count, f_time)
                VALUES ('1', '%s')
                """
            self.w_db.query(sql % current_day)

        current_month = now.strftime("%Y-%m")
        current_year = now.strftime("%Y")
        table_dict = {"t_active_user_month": current_month,
                      "t_active_user_year": current_year}
        for key, value in list(table_dict.items()):
            result = self.r_db.one(select_sql % (key, value))
            if result:
                sql = "UPDATE %s SET f_total_count = f_total_count + 1"
                if activate_status:
                    sql += ", f_activate_count = f_activate_count + 1 "
                sql += " WHERE f_time = '%s'"
            else:
                sql = """
                INSERT INTO %s (f_total_count, f_activate_count, f_time)
                VALUES (1, '{0}', '%s')
                """.format(int(activate_status))
            self.w_db.query(sql % (key, value))

    def fill_role_member_departments(self, role_members):
        """
        填充角色成员部门信息
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
            SELECT u.`f_user_id`, u.`f_display_name`, r.`f_department_id`, d.`f_name`
            FROM `t_user` as u
            LEFT JOIN `t_user_department_relation` as r
            ON r.f_user_id = u.f_user_id
            LEFT JOIN t_department as d
            ON d.f_department_id = r.f_department_id
            WHERE u.`f_user_id` in ({0}) AND 1=%s
            ORDER BY u.f_priority, u.f_display_name
        """.format(groupStr)

        results = self.r_db.all(sql, 1)

        for result in results:
            role_member = userid_map[result['f_user_id']]
            role_member.displayName = result['f_display_name']

            if not role_member.departmentIds:
                role_member.departmentIds = []
            if not role_member.departmentNames:
                role_member.departmentNames = []

            if result['f_department_id'] != '-1':
                role_member.departmentIds.append(result['f_department_id'])
                role_member.departmentNames.append(result['f_name'])

        for userInfo in userid_map.values():
            if len(userInfo.departmentIds) == 0 and len(userInfo.departmentNames) == 0:
                userInfo.departmentIds = [NCT_UNDISTRIBUTE_USER_GROUP]
                userInfo.departmentNames = [_("undistributed user")]

    def fill_user_roles(self, user_infos):
        """
        填充用户角色信息
        """
        if not user_infos:
            return

        userid_set = set()
        for user_info in user_infos:
            userid_set.add(user_info.id)

        groupStr = generate_group_str(userid_set)
        if not groupStr:
            return

        sql = """
            SELECT r.f_user_id, t.f_role_id, t.f_name
            FROM `t_role` as t
            INNER JOIN t_user_role_relation as r
            ON t.f_role_id = r.f_role_id
            WHERE r.f_user_id in ({0}) AND 1=%s
            ORDER BY t.f_priority
        """.format(groupStr)

        results = self.r_db.all(sql, 1)

        role_info_map = {}
        for result in results:
            user_id = result['f_user_id']
            # 解析用户角色信息
            info = ncTRoleInfo()
            info.name = result['f_name']
            info.id = result['f_role_id']
            if user_id not in role_info_map:
                role_info_map[user_id] = []
            role_info_map[user_id].append(info)

        for user_info in user_infos:
            if hasattr(user_info, "user"):
                if user_info.id in role_info_map:
                    user_info.user.roles = role_info_map[user_info.id]
                else:
                    user_info.user.roles = []

    def fill_user_managers(self, user_infos):
        """
        填充用户上级信息
        """
        if not user_infos:
            return

        userid_set = set()
        for user_info in user_infos:
            if user_info.user.managerID != "":
                userid_set.add(user_info.user.managerID)
            else:
                user_info.user.managerDisplayName = ''

        if len(userid_set) == 0:
            return

        groupStr = generate_group_str(userid_set)
        if not groupStr:
            return

        sql = """
            SELECT `f_user_id`, `f_display_name` FROM `t_user` WHERE f_user_id in ({0}) AND 1=%s ORDER BY f_priority
        """.format(groupStr)

        results = self.r_db.all(sql, 1)

        user_id_name_map = {}
        for result in results:
            user_id_name_map[result['f_user_id']] = result['f_display_name']

        for user_info in user_infos:
            # 如果没有上级，则跳过
            if user_info.user.managerID == "":
                user_info.user.managerDisplayName = ""
                continue

            if user_info.user.managerID in user_id_name_map:
                user_info.user.managerDisplayName = user_id_name_map[user_info.user.managerID]
            else:
                user_info.user.managerDisplayName = ""

    def get_third_id_by_user_id(self, user_id):
        """
        根据用户id获取第三方id
        """
        third_id = ""
        sql = """
        SELECT `f_third_party_id`
        FROM `t_user`
        WHERE `f_user_id` = %s
        """
        result = self.r_db.one(sql, user_id)
        if result:
            third_id = result["f_third_party_id"]
        return third_id

    def set_user_expire_time(self, userId, expireTime):
        """
        设置用户账号有效期
        """
        self.check_user_exists(userId)

        # 检查用户账号有效期
        if expireTime is None:
            expireTime = -1
        if expireTime != -1 and expireTime < int(BusinessDate.time()):
            raise_exception(exp_msg=_("IDS_INVALID_DATE"),
                            exp_num=ncTShareMgntError.NCT_INVALID_DATETIME)

        # 更新用户账号有效期
        sql = """
        UPDATE `t_user`
        SET `f_expire_time` = %s
        WHERE `f_user_id` = %s
        """
        self.w_db.query(sql, expireTime, userId)

        # 过期禁用用户自动启用
        self.__enable_expired_user(userId)

    def __enable_expired_user(self, userId):
        """
        自动启用过期用户
        """
        select_sql = """
        SELECT `f_login_name`, `f_display_name`, `f_status`, `f_auto_disable_status`, `f_expire_time`
        FROM `t_user`
        WHERE `f_user_id` = %s
        """

        # 恢复过期禁用标志位
        update_sql = """
        UPDATE `t_user`
        SET `f_auto_disable_status` = `f_auto_disable_status` & %s
        WHERE `f_user_id` = %s
        """

        result = self.r_db.one(select_sql, userId)
        if result["f_expire_time"] == -1 or result["f_expire_time"] > int(BusinessDate.time()):
            self.w_db.query(update_sql, ~USER_EXPIRE_DISABLED, userId)

            # 自动启用用户记录管理日志
            if (result["f_status"] == ncTUsrmUserStatus.NCT_STATUS_ENABLE) and \
                    (result["f_auto_disable_status"] == USER_EXPIRE_DISABLED):
                msg = _("IDS_ENABLE_EXPIRED_USER") % (
                    result["f_display_name"], result["f_login_name"])
                ex_msg = _("IDS_ENABLE_EXPIRED_USER_EXMSG")
                eacp_log(_("IDS_SYSTEM"),
                        global_info.LOG_TYPE_MANAGE,
                        global_info.USER_TYPE_INTER,
                        global_info.LOG_LEVEL_INFO,
                        global_info.LOG_OP_TYPE_SET,
                         msg,
                         ex_msg,
                         raise_ex=True)

    def need_quick_start(self, user_id, os_type):
        """
        获取用户是否显示“快速入门”
        现阶段只支持控制台
        os_type固定为8
        """
        self.check_user_exists(user_id)
        # os_type固定为8，代表控制台
        os_type = 8

        sql = """
        SELECT `f_user_document_read_status` FROM `t_user` WHERE `f_user_id` = %s
        """
        result = self.w_db.one(sql, user_id)
        return False if result['f_user_document_read_status'] & (1 << os_type) else True

    def set_quick_start_status(self, user_id, status, os_type):
        """
        设置是否显示用户“快速入门”的状态
        这里不管status传什么都是设置为不显示“快速入门”
        os_type固定为8
        """
        self.check_user_exists(user_id)
        # os_type固定为8，代表控制台
        os_type = 8

        sql = """
        UPDATE `t_user`
        SET `f_user_document_read_status` = f_user_document_read_status | %s
        WHERE f_user_id = %s
        """
        self.w_db.query(sql, 1 << os_type, user_id)

    def get_users_parent_deps(self, user_ids):
        """
        根据用户名数组获取用户的直属部门ID和Name
        """
        user_infos = {}
        if len(user_ids) == 0:
            return user_infos

        str_user_ids = generate_group_str(user_ids)
        userRelationSql = """
            SELECT r.f_user_id,  r.f_department_id, u.f_code, u.f_name FROM
                t_user_department_relation AS r
                left join t_department AS u on r.f_department_id = u.f_department_id
            where
                r.f_user_id in ({0})
            """.format(str_user_ids)

        userRelationList = self.r_db.all(userRelationSql)

        # 整理数据
        for user_info in userRelationList:
            user_id = user_info['f_user_id']

            # 保存数据
            if user_info['f_department_id'] != '-1':
                # 初始化
                if user_id not in user_infos:
                    user_infos[user_id] = {}
                    user_infos[user_id]['depart_ids'] = []
                    user_infos[user_id]['depart_names'] = []
                    user_infos[user_id]['depart_codes'] = []

                user_infos[user_id]['depart_ids'].append(
                    user_info['f_department_id'])
                user_infos[user_id]['depart_names'].append(user_info['f_name'])
                user_infos[user_id]['depart_codes'].append(user_info['f_code'])

        return user_infos
