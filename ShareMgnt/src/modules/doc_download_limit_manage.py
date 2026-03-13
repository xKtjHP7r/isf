#!/usr/bin/python3
# -*- coding:utf-8 -*-
"""
用户文档下载限制管理类
"""
import sys
import uuid
import time
import datetime
from eisoo.tclients import TClient
from src.modules.user_manage import UserManage
from src.modules.department_manage import DepartmentManage
from src.modules.oem_manage import OEMManage
from src.modules.config_manage import ConfigManage
from src.modules.handle_task_thread import (CallableTask, HandleTaskThread)
from EThriftException.ttypes import ncTException
from ShareMgnt.ttypes import (ncTShareMgntError)
from ShareMgnt.constants import (NCT_SYSTEM_ROLE_AUDIT)
from EVFS.ttypes import (ncTUserDownloadLimitInfo)
from src.common.db.connector import DBConnector
from src.common.db.connector import ConnectorManager
from src.common.lib import (raise_exception, escape_key,
                            check_start_limit, generate_search_order_sql)
from src.common.nc_senders import (email_send_html_content)
from src.common.global_info import (IS_SINGLE, NC_EVFS_NAME_IOC_DATAEXCHANGE_ID)
from src.common.business_date import BusinessDate
from src.modules.role_manage import RoleManage


USER_OBJ = 1
DEPART_OBJ = 2

MAX_DOC_DOWNLOAD_LIMIT_VALUE = sys.maxsize


class DocDownloadLimitManage(DBConnector):
    def __init__(self):
        """
        """
        self.user_manage = UserManage()
        self.dept_manage = DepartmentManage()
        self.oem_manage = OEMManage()
        self.handle_task_thread = HandleTaskThread()
        self.config_manage = ConfigManage()
        self.role_manage = RoleManage()

    def __check_user_list(self, userList):
        """
        检查用户列表中用户是否存在
        """
        if not userList:
            return
        for user_obj in userList:
            self.user_manage.check_user_exists(user_obj.objectId, True)

    def __check_dept_list(self, deptList):
        """
        检查部门列表中用户是否存在
        """
        if not deptList:
            return

        for dept_obj in deptList:
            self.dept_manage.check_depart_exists(dept_obj.objectId, True)

    def __check_limit_value(self, limitValue):
        """
        检查限制值是否有效
        """
        if limitValue != -1 and limitValue <= 0:
            raise_exception(exp_msg=_("IDS_INVALID_DOC_DOWNLOAD_LIMIT_VALUE"),
                            exp_num=ncTShareMgntError.NCT_INVALID_DOC_DOWNLOAD_LIMIT_VALUE)

    def __remove_duplicate_obj(self, objList):
        """
        去重重复项
        """
        obj_id_set = set()
        obj_list = []
        for obj in objList:
            if obj.objectId not in obj_id_set:
                obj_list.append(obj)
                obj_id_set.add(obj.objectId)

        return obj_list

    def add(self, limitInfo):
        """
        添加一条文档下载量限制
        """
        global IS_SINGLE
        if not IS_SINGLE:
            with TClient('ShareMgntSingle') as client:
                return client.Usrm_AddDocDownloadLimitInfo(limitInfo)

        # 检查限制对象是否配置
        if not limitInfo.userInfos and not limitInfo.depInfos:
            raise_exception(exp_msg=_("IDS_DOC_DOWNLOAD_LIMIT_OBJECT_NOT_SET"),
                            exp_num=ncTShareMgntError.NCT_DOC_DOWNLOAD_LIMIT_OBJECT_NOT_SET)

        # 检查用户列表中用户是否存在
        if limitInfo.userInfos:
            limitInfo.userInfos = self.__remove_duplicate_obj(limitInfo.userInfos)
            self.__check_user_list(limitInfo.userInfos)

        # 检查部门列表中部门是否存在
        if limitInfo.depInfos:
            limitInfo.depInfos = self.__remove_duplicate_obj(limitInfo.depInfos)
            self.__check_dept_list(limitInfo.depInfos)

        # 检查限制值
        self.__check_limit_value(limitInfo.limitValue)

        # 生成一条唯一id
        limitInfo.id = str(uuid.uuid1())

        # 增加任务更新 EFAST 中用户的下载限制值配置
        self.add_update_user_doc_download_limit_task()

        return limitInfo.id

    def add_update_user_doc_download_limit_task(self):
        """
        添加更新文件下载量任务
        """
        task = CallableTask()
        task.module_name = "doc_download_limit_manage"
        task.function_name = "update_efast_download_limit"
        self.handle_task_thread.add(task)

        # 三权分立模式下，如果audit开启了接收通知，则当securit修改用户阈值时，audit收到邮件通知
        if (self.user_manage.get_trisystem_status() and
                self.config_manage.get_ddl_email_notify_mode_status()):
            # 获取audit角色下所有用户邮箱
            toEmailList = self.role_manage.get_role_mails(NCT_SYSTEM_ROLE_AUDIT)
            if 0 != len(toEmailList):
                product_name = self.oem_manage.get_config_by_option('shareweb_en-us', 'product')
                subject = _("IDS_EVFS_DOC_DOWNLOAD_LIMIT_CONFIG_EMAIL_SUBJECT") % (product_name)
                time_string = BusinessDate.now().strftime('%Y-%m-%d %H:%M:%S')
                content = _("IDS_EVFS_DOC_DOWNLOAD_LIMIT_CONFIG_EMAIL_CONTENT") % (time_string,
                                                                                   product_name)
                email_send_html_content(toEmailList, subject, content)
