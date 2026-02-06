#!/usr/bin/python3
# -*- coding:utf-8 -*-
import os
import csv
import time
import threading
import uuid
import shutil
import json
from decimal import Decimal
from eisoo.tclients import TClient
from src.common.lib import (raise_exception,
                            check_name)
from src.common.global_info import IS_SINGLE
from src.common.db.connector import DBConnector
from src.common.db.db_manager import get_db_name
from ShareMgnt.ttypes import ncTDocType
from src.common.sharemgnt_logger import ShareMgnt_Log
from src.common.business_date import BusinessDate
from src.modules.user_manage import UserManage
from src.modules.department_manage import DepartmentManage
from src.driven.service_access.ossgateway_config import OssgatewayDriven
from src.modules.role_manage import RoleManage
from ShareMgnt.ttypes import ncTShareMgntError
from ShareMgnt.constants import (ncTReportInfo,
                                 NCT_USER_ADMIN,
                                 NCT_USER_AUDIT,
                                 NCT_USER_SYSTEM,
                                 NCT_USER_SECURIT,
                                 NCT_SYSTEM_ROLE_SUPPER,
                                 NCT_SYSTEM_ROLE_ADMIN,
                                 NCT_SYSTEM_ROLE_SECURIT,
                                 NCT_SYSTEM_ROLE_AUDIT,
                                 NCT_SYSTEM_ROLE_ORG_MANAGER,
                                 NCT_SYSTEM_ROLE_ORG_AUDIT)

_file_path = '/tmp/sysvol/cache/sharemgnt/spacereport/'
threadLock = threading.Lock()
task_dict = {}
WAIT_TIME_ONE_HOUR = 3600


class TaskStatus:
    """
    任务状态
    """
    TASK_ERROR = -1
    TASK_CREATED = 0
    TASK_IN_PROCESS = 1
    TASK_SUCCESS = 2


class CustomSpaceInfo:
    """
    文档库空间使用情况信息
    """

    def __init__(self):
        self.docid = ""
        self.docName = ""
        self.typeName = ""
        self.ownerNames = ""
        self.createrName = ""
        self.relateDepartName = "--"
        self.ossID = ""
        self.ossName = ""
        self.userdsize = 0
        self.totalsize = 0
        self.usedSpaceRate = 0


class UserSpaceInfo:
    """
    用户空间使用情况信息
    """

    def __init__(self):
        self.displayName = ""
        self.loginName = ""
        self.departNames = ""
        self.roleNames = ""
        self.authType = ""


class TaskInfo:
    """
    保存报表任务信息
    Args:
        taskType: 1->个人文档库;3->文档库; 5->归档库
        fileName: 该任务生成文件的名称
        filePath: 生成的文件所在路径
        taskStatus: 任务的执行状态
        createTime: 任务的创建时间
        lock:
    """

    def __init__(self, taskType, fileName="", filePath="", taskStatus=TaskStatus.TASK_CREATED, createTime=BusinessDate.time()):
        self.taskType = taskType
        self.fileName = fileName
        self.filePath = filePath
        self.taskStatus = taskStatus
        self.createTime = createTime
        self.lock = threading.Lock()
        self.operator_id = ""

    def get_task_status(self):
        with self.lock:
            return self.taskStatus

    def set_task_status(self, taskStatus):
        with self.lock:
            self.taskStatus = taskStatus


class SpaceReportManage(DBConnector):

    def __init__(self):
        self.user_manage = UserManage()
        self.department_manage = DepartmentManage()
        self.role_manage = RoleManage()

        # 初始化添加系统中固定的角色
        self.role_dict = {}
        self.role_dict[NCT_SYSTEM_ROLE_ORG_AUDIT] = _("organization audit")
        self.role_dict[NCT_SYSTEM_ROLE_ORG_MANAGER] = _("organization manager")
        self.role_dict[NCT_SYSTEM_ROLE_SUPPER] = _("supper")
        self.role_dict[NCT_SYSTEM_ROLE_SECURIT] = _("security")
        self.role_dict[NCT_SYSTEM_ROLE_ADMIN] = _("admin")
        self.role_dict[NCT_SYSTEM_ROLE_AUDIT] = _("audit")
        self.role_dict["normal"] = _("normal")
        # 初始化认证类型
        self.auth_type = []
        self.auth_type.append(_("IDS_USER_TYPE_LOCAL"))
        self.auth_type.append(_("IDS_USER_TYPE_DOMAIN"))
        self.auth_type.append(_("IDS_USER_TYPE_THIRD"))
        self.ossgateway_driven = OssgatewayDriven()

    def add_gen_file_task(self, taskInfo):
        """
        将任务添加进任务队列
        """
        global task_dict
        taskId = str(uuid.uuid1())
        with threadLock:
            task_dict[taskId] = taskInfo
        return taskId

    def gen_store_file_dir(self, taskId):
        """
        创建文件保存目录
        """
        fileDir = os.path.join(_file_path, taskId)

        # 文件目录为:/sysvol/cache/sharemgnt/spacereport/taskId/
        if not os.path.exists(fileDir):
            os.makedirs(fileDir)

        return fileDir

    def _gen_custom_space_report_file(self, csv_write):
        """
        文档库空间使用情况
        """
        # 获取需要导出的文档库的所有信息
        custom_space_infos = []

        # 获取对象存储信息
        oss_dict = {}
        code, data = self.ossgateway_driven.get_as_storage_info()

        if code != 200:
            ShareMgnt_Log(
                f'get_as_storage_info failed: {code},{data.get("message")},{data.get("cause")}')
            raise_exception(exp_msg=_("IDS_GET_OSS_INFO_FAILD"),
                            exp_num=ncTShareMgntError.NCT_NO_AVAILABLE_OSS)

        # 将对象存储的id和name做映射
        for item in data:
            oss_dict[item["id"]] = item["name"]

        # 写入表头
        csv_write.writerow([_("IDS_CUSTOM_TITLE")])
        csv_write.writerow([_("IDS_CUSDOC_NAME"),
                            _("IDS_DOC_TYPE"),
                            _("IDS_DOC_OWNER"),
                            _("IDS_CREATER_NAME"),
                            _("IDS_RELATE_DEPART"),
                            _("IDS_OSS_NAME")])
        # 逐行写入文件
        for space_info in custom_space_infos:
            csv_write.writerow(["\t%s" % space_info.docName,
                                "\t%s" % space_info.typeName,
                                "\t%s" % space_info.ownerNames,
                                "\t%s" % space_info.createrName,
                                "\t%s" % space_info.relateDepartName,
                                "\t%s" % space_info.ossName])

    def _gen_arch_space_report_file(self, csv_write):
        """
        归档库空间使用情况
        """
        # 获取需要导出的归档库的所有信息
        acvhive_space_info = []

        # 获取对象存储信息
        oss_dict = {}
        code, data = self.ossgateway_driven.get_as_storage_info()

        if code != 200:
            ShareMgnt_Log(
                f'get_as_storage_info failed: {code},{data.get("message")},{data.get("cause")}')
            raise_exception(exp_msg=_("IDS_GET_OSS_INFO_FAILD"),
                            exp_num=ncTShareMgntError.NCT_NO_AVAILABLE_OSS)

        # 将对象存储的id和name做映射
        for item in data:
            oss_dict[item["id"]] = item["name"]

        # 写入表头
        csv_write.writerow([_("IDS_ARCDOC_TITLE")])
        csv_write.writerow([_("IDS_ARCDOC_NAME"),
                            _("IDS_DOC_OWNER"),
                            _("IDS_CREATER_NAME"),
                            _("IDS_OSS_NAME")])
        # 逐行写入文件
        for space_info in acvhive_space_info:
            csv_write.writerow(["\t%s" % space_info.docName,
                                "\t%s" % space_info.ownerNames,
                                "\t%s" % space_info.createrName,
                                "\t%s" % space_info.ossName])

    def __get_users_in_scope(self, operator_id):
        """
        获取操作员所在范围内的所有用户
        如果是超级管理员/系统管理员，则可以获取所有用户信息
        如果是组织管理员，则可以获取管理的用户信息
        如果是其他管理员或者普通用户，则不可以获取用户信息

        参数:
        operator_id (string): 操作者id

        返回值:
        show_all_users (bool): 是否包含所有用户
        user_ids_list (list): show_all_usersw为false时有效，范围内所有用户id列表
        """
        # 获取操作者的角色
        role_ids = self.role_manage.get_user_role_id(operator_id)

        # 如果是超级管理员或者安全管理员或者系统管理员，则返回标识True, 获取所有用户信息
        # 如果是普通用户，则返回标识False, 获取返回空用户信息
        # 如果是组织管理员，则返回标识False, 获取管理的用户信息
        if NCT_SYSTEM_ROLE_SUPPER in role_ids or NCT_SYSTEM_ROLE_ADMIN in role_ids or NCT_SYSTEM_ROLE_SECURIT in role_ids:
            return True, []
        elif NCT_SYSTEM_ROLE_ORG_MANAGER in role_ids:
            user_ids = self.department_manage.get_user_ids_by_admin_id(operator_id)
            return False, user_ids
        else:
            return False, []

    def __get_user_space_infos(self, operator_id):
        """
        获取个人文档的所有导出信息
        """
        # 判断用户是否存在
        self.user_manage.check_user_exists(operator_id)

        # 获取范围内的用户
        show_all_users, user_ids_in_scpoe = self.__get_users_in_scope(operator_id)

        user_space_infos = []
        sql = """
        SELECT u.f_user_id, u.f_display_name AS f_display_name, u.f_login_name AS f_login_name, u.f_auth_type AS f_auth_type
        FROM `t_user` u
        WHERE u.`f_user_id` NOT IN (%s, %s, %s, %s)
        """
        results = self.r_db.all(sql,
                                NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT)

        depart_path_dict = {}
        auth_type_dict = {}
        # 保存用户自定义角色的role id和role name
        role_name_dict = {}
        for result in results:
            space_info = UserSpaceInfo()
            user_id = result['f_user_id']

            # 判断用户是否在范围内,如果不在，则不导出该用户的信息
            if not show_all_users and user_id not in user_ids_in_scpoe:
                continue

            space_info.displayName = result['f_display_name']
            space_info.loginName = result['f_login_name']
            # 处理部门路径
            depart_paths = []
            depart_ids_list = self.user_manage.get_belong_depart_id(user_id)
            if depart_ids_list[0] == "-1":
                space_info.departNames = _("undistributed user")
            else:
                for depart_id in depart_ids_list:
                    if depart_id not in depart_path_dict:
                        depart_info = self.department_manage.get_department_info(
                            depart_id, b_include_org=True, include_parent_path=True)
                        depart_path_dict[depart_id] = "%s/%s" % (
                            depart_info.parentPath, depart_info.departmentName) if depart_info.parentPath != '' else depart_info.departmentName
                    depart_paths.append(depart_path_dict[depart_id])
                space_info.departNames = " ".join(depart_paths)
            # 处理角色名称
            role_names = []
            role_ids_list = self.role_manage.get_user_role_ids_order_by(
                user_id)
            role_ids_list.append("normal")
            for role_id in role_ids_list:
                if role_id in self.role_dict:
                    role_names.append(self.role_dict[role_id])
                else:
                    if role_id not in role_name_dict:
                        # 根据role_id获取role name
                        role_name_dict[role_id] = self.role_manage.get_role_name_by_id(
                            role_id)
                    role_names.append(role_name_dict[role_id])
            space_info.roleNames = "/".join(role_names)
            # 处理认证类型
            space_info.authType = self.auth_type[result['f_auth_type'] - 1]

            user_space_infos.append(space_info)

        return user_space_infos

    def _gen_user_space_report_file(self, csv_write, operator_id):
        """
        用户空间使用情况
        """
        # 获取需要导出的个人文档的所有信息
        user_space_infos = self.__get_user_space_infos(operator_id)
        # 写入表头
        csv_write.writerow([_("IDS_USERDOC_TITLE")])
        csv_write.writerow([_("IDS_DISPLAY_NAME"),
                            _("IDS_LOGIN_NAME"),
                            _("IDS_DEPART_NAMES"),
                            _("IDS_ROLE_NAMES"),
                            _("IDS_AUTH_TYPE")])
        # 逐行写入文件
        for space_info in user_space_infos:
            csv_write.writerow(["\t%s" % space_info.displayName,
                                "\t%s" % space_info.loginName,
                                "\t%s" % space_info.departNames,
                                "\t%s" % space_info.roleNames,
                                space_info.authType])

    def gen_space_report_file(self, taskId, taskInfo):
        # 创建文件保存目录
        fileDir = self.gen_store_file_dir(taskId)
        # 生成文件
        csvFileName = os.path.join(fileDir, taskInfo.fileName)
        with open(csvFileName, "w") as fd:
            # Excel BOM头
            fd.write(bytes.decode(b'\xef\xbb\xbf'))
            csv_write = csv.writer(fd)
            if ncTDocType.NCT_USER_DOC == taskInfo.taskType:
                # 个人文档空间使用情况
                # 部门，用户名，已用空间，总空间
                self._gen_user_space_report_file(csv_write, taskInfo.operator_id)
            elif ncTDocType.NCT_CUSTOM_DOC == taskInfo.taskType:
                # 文档库空间使用情况
                self._gen_custom_space_report_file(csv_write)
            elif ncTDocType.NCT_ARCHIVE_DOC == taskInfo.taskType:
                # 归档库空间使用情况
                self._gen_arch_space_report_file(csv_write)

        taskInfo.set_task_status(TaskStatus.TASK_SUCCESS)
        return csvFileName

    def gen_file_handler(self, taskId, taskInfo):
        if taskId is None or taskInfo is None:
            return

        taskInfo.set_task_status(TaskStatus.TASK_IN_PROCESS)
        return self.gen_space_report_file(taskId, taskInfo)

    def __check_export_doctype(self, objType):
        """
        检查文档库类型合法性
        """
        if objType in (ncTDocType.NCT_USER_DOC, ncTDocType.NCT_CUSTOM_DOC, ncTDocType.NCT_ARCHIVE_DOC):
            return

        raise_exception(exp_msg=_("IDS_DOC_TYPE_NOT_SUPPORT_EXPORT"),
                        exp_num=ncTShareMgntError.NCT_DOC_TYPE_NOT_SUPPORT_EXPORT)

    def export_space_report(self, name, objType, operator_id):
        """
        创建文件生成任务
        """
        global IS_SINGLE

        # 检查要导出的文档类型合法性
        self.__check_export_doctype(objType)
        # 检查文件名的合法性
        if not check_name(name):
            raise_exception(exp_msg=_("IDS_INVALID_FILE_NAME"),
                            exp_num=ncTShareMgntError.NCT_INVALID_FILE_NAME)
        # 如果是个人文档库，检查操作者是否有效
        if objType == ncTDocType.NCT_USER_DOC:
            self.user_manage.check_user_exists(operator_id)

        # 限制同时执行的任务数
        for k, v in list(task_dict.items()):
            if v.taskType == objType and (v.taskStatus == TaskStatus.TASK_IN_PROCESS or v.taskStatus == TaskStatus.TASK_CREATED):
                errDetail = {}
                errDetail["taskId"] = k
                raise_exception(exp_msg=_("IDS_HAVE_OTHER_SAME_TYPE_SPACE_REPORT_TASK_IN_PROGRESS"),
                                exp_num=ncTShareMgntError.NCT_HAVE_OTHER_SAME_TYPE_SPACE_REPORT_TASK_IN_PROGRESS,
                                exp_detail=json.dumps(errDetail, ensure_ascii=False))

        taskInfo = TaskInfo(objType)
        # 构造文件名: test.2019-03-26.csv
        taskInfo.fileName = "%s.csv" % name

        taskId = self.add_gen_file_task(taskInfo)
        taskInfo.operator_id = operator_id

        # 启动生成文件的线程
        gen_file_thread = GenFileThread(taskId, taskInfo)
        gen_file_thread.daemon = True
        gen_file_thread.start()

        return taskId

    def __gen_space_report_task_info(self, taskId):

        global task_dict

        with threadLock:
            # 任务不存在
            if taskId not in task_dict:
                raise_exception(exp_msg=_("IDS_EXPORT_SPACE_REPORT_NOT_EXIST"),
                                exp_num=ncTShareMgntError.NCT_EXPORT_SPACE_REPORT_NOT_EXIST)
            else:
                return task_dict[taskId]

    def get_gen_space_report_status(self, taskId):
        """
        获取生成报表任务状态
        """
        global IS_SINGLE

        taskInfo = self.__gen_space_report_task_info(taskId)

        task_status = taskInfo.get_task_status()

        if task_status == TaskStatus.TASK_IN_PROCESS or task_status == TaskStatus.TASK_CREATED:
            return False
        elif task_status == TaskStatus.TASK_SUCCESS:
            return True
        else:
            raise_exception(exp_msg=_("IDS_EXPORT_SPACE_REPORT_FAILED"),
                            exp_num=ncTShareMgntError.NCT_EXPORT_SPACE_REPORT_FAILED)

    def del_gen_file_task(self, taskId):
        """
        删除任务
        """
        global task_dict

        with threadLock:
            # 任务不存在
            if taskId not in task_dict:
                raise_exception(exp_msg=_("IDS_EXPORT_SPACE_REPORT_NOT_EXIST"),
                                exp_num=ncTShareMgntError.NCT_EXPORT_SPACE_REPORT_NOT_EXIST)
            else:
                del(task_dict[taskId])

    def get_space_report_file_info(self, taskId):
        """
        获取生成的活跃报表信息
        """
        global IS_SINGLE

        taskInfo = self.__gen_space_report_task_info(taskId)

        if taskInfo.get_task_status() == TaskStatus.TASK_SUCCESS:
            reportInfo = ncTReportInfo()
            with open(taskInfo.filePath, 'rb') as fd:
                reportInfo.reportData = fd.read()
            reportInfo.reportName = taskInfo.fileName
            return reportInfo

        if taskInfo.get_task_status() == TaskStatus.TASK_IN_PROCESS or taskInfo.get_task_status() == TaskStatus.TASK_CREATED:
            raise_exception(exp_msg=_("IDS_EXPORT_SPACE_REPORT_IN_PROGRESS"),
                            exp_num=ncTShareMgntError.NCT_EXPORT_SPACE_REPORT_IN_PROGRESS)

        if taskInfo.get_task_status() == TaskStatus.TASK_ERROR:
            self.del_gen_file_task(taskId)
            raise_exception(exp_msg=_("IDS_EXPORT_SPACE_REPORT_FAILED"),
                            exp_num=ncTShareMgntError.NCT_EXPORT_SPACE_REPORT_FAILED)


class GenFileThread(threading.Thread):

    def __init__(self, taskId, taskInfo):
        super(GenFileThread, self).__init__()
        self.taskId = taskId
        self.taskInfo = taskInfo
        self.sapce_report_manage = SpaceReportManage()

    def run(self):
        ShareMgnt_Log(
            "**************** generate space report file thread start *************** *")

        try:
            self.taskInfo.filePath = self.sapce_report_manage.gen_file_handler(
                self.taskId, self.taskInfo)
        except Exception as ex:
            # 任务处理异常，更新任务状态
            fileDir = os.path.join(_file_path, self.taskId)
            # 清理掉创建的目录
            if os.path.exists(fileDir):
                shutil.rmtree(fileDir)
            self.taskInfo.set_task_status(TaskStatus.TASK_ERROR)
            ShareMgnt_Log(
                "generate space report file thread run error: %s", str(ex))

        ShareMgnt_Log(
            "**************** generate space report file thread end *************** *")


class SpaceReportTaskAutoDeleteThread(threading.Thread):

    def __init__(self):
        """
        初始化
        """
        super(SpaceReportTaskAutoDeleteThread, self).__init__()
        # 重新启动服务时，删除目录下的所有文件
        if os.path.exists(_file_path):
            shutil.rmtree(_file_path)

    def delete_overtime_task(self):
        """
        删除创建时间超过 1h 的任务
        """
        global task_dict

        with threadLock:
            items = list(task_dict.items())

        for (taskId, taskInfo) in items:
            if taskInfo.createTime < (BusinessDate.time() - WAIT_TIME_ONE_HOUR):
                with threadLock:
                    if taskId in task_dict:
                        # 检查文件是否已经删除
                        # 出现情况：
                        #   用户点击导出任务后，关闭前端页面，此时前端将不会获取到对应的任务Id，而此时，后端任务仍在进行中
                        fileDir = os.path.join(_file_path, taskId)
                        if os.path.exists(fileDir):
                            shutil.rmtree(fileDir)
                        # 删除任务
                        del(task_dict[taskId])
                        ShareMgnt_Log("delete task: %s success.", taskId)

    def run(self):
        """
        执行
        """
        ShareMgnt_Log(
            "**************** space report task auto delete thread start *****************")

        while True:
            try:
                self.delete_overtime_task()
            except Exception as e:
                print(("space report task auto delete thread run error: %s", str(e)))
            time.sleep(WAIT_TIME_ONE_HOUR)

        ShareMgnt_Log(
            "**************** space report task auto delete thread end *****************")
