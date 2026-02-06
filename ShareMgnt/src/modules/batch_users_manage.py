#!/usr/bin/python3
# -*- coding:utf-8 -*-
import os
import threading
import time
import xlrd
import xlsxwriter
import copy
import json
import re
from datetime import date
from datetime import datetime
from collections import namedtuple

from eisoo import langlib
from eisoo.tclients import TClient
from EThriftException.ttypes import ncTException
from ShareMgnt.constants import (ncTReportInfo,
                                 NCT_ALL_USER_GROUP,
                                 NCT_UNDISTRIBUTE_USER_GROUP,
                                 NCT_SYSTEM_ROLE_SUPPER,
                                 NCT_SYSTEM_ROLE_ADMIN,
                                 NCT_SYSTEM_ROLE_ORG_MANAGER)
from ShareMgnt.ttypes import (ncTAddDepartParam, ncTAddOrgParam,
                              ncTBatchUsersFile, ncTImportFailInfo,
                              ncTShareMgntError, ncTUsrmAddUserInfo,
                              ncTUsrmOSSInfo, ncTUsrmUserInfo,
                              ncTUsrmUserStatus,ncTUsrmImportResult)
from src.common import global_info
from src.common.global_info import IS_SINGLE
from src.common.db.connector import DBConnector
from src.common.lib import (sha2_encrypt, raise_exception,
                            remove_duplicate_item_from_list, ntlm_md4)
from src.common.eacp_log import eacp_log
from src.common.sharemgnt_logger import ShareMgnt_Log
from src.common.http import pub_nsq_msg
from src.common.encrypt.simple import des_decrypt_with_padzero
from src.common.business_date import BusinessDate
from src.modules.user_manage import UserManage
from src.modules.department_manage import DepartmentManage
from src.modules.config_manage import ConfigManage
from src.modules.role_manage import RoleManage
from src.modules.export_file_task_manage import (TaskFinishedStatus,
                                                BaseTaskInfo,
                                                GenFileThread)
from src.driven.service_access.ossgateway_config import OssgatewayDriven

TOPIC_ORG_NAME_MODIFY = "core.org.name.modify"
TOPIC_USER_MODIFIED = "user_management.user.modified"
MAXROW = 1000000 # 规定最大写1000000行（xlsx最大支持1,048,576 rows, 16,384 columns）
START_ROW = 3
DOWNLOAD_PATH = '/tmp/sysvol/cache/sharemgnt/downloadbatchusers'
IMPORT_BATCH_USERS_PATH = '/sysvol/cache/sharemgnt/importbatchusers'
IMPOTR_FAILED_USERS_PATH = '/tmp/sysvol/cache/sharemgnt/downloadfailed'
# 可以进行此操作的角色
AVAILABLE_ROLE_TUPLE = (NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_ADMIN, NCT_SYSTEM_ROLE_ORG_MANAGER)

class TaskInfo(BaseTaskInfo):
    def __init__(self, create_time=BusinessDate.time(), file_path="", finished_status=0,
                 name="", inquire_date=""):
        super(TaskInfo, self).__init__(create_time, file_path, finished_status, name)
        self.inquire_date = inquire_date

class BatchUsersManage(DBConnector):
    """
    批量导入、导出用户
    """
    def __init__(self):
        super(BatchUsersManage, self).__init__()
        self.user_manage = UserManage()
        self.depart_manage = DepartmentManage()
        self.role_manage = RoleManage()
        self.config_manage = ConfigManage()
        self.import_mutex = threading.Lock()
        self.ossgateway_driven = OssgatewayDriven()
        # evfs的对象存储信息，每次导入、导出前更新一次
        self.evfs_oss_infos = None

        # 缓存数据
        self.path_cache = {}
        self.id_to_departname_cache = {}
        self.available_path_list = []

        # 向清理线程填入需要清理的路径
        global_info.DELETE_FILE_PATHS.extend([DOWNLOAD_PATH, IMPORT_BATCH_USERS_PATH, IMPOTR_FAILED_USERS_PATH])

    @property
    def admin_list(self):
        return list(self.user_manage.get_all_admin_account().values())

    @property
    def csf_level_dict(self):
        # 获取用户可选的密级
        csf_levels = self.config_manage.get_csf_levels()
        return {k:v for k, v in list(csf_levels.items())}

    @property
    def csf_level2_dict(self):
        # 获取用户可选的密级2
        csf_levels2 = self.config_manage.get_csf_levels2()
        return {k:v for k, v in list(csf_levels2.items())}

    @property
    def IMPORT_PATH(self):
        if not os.path.exists(IMPORT_BATCH_USERS_PATH):
            os.makedirs(IMPORT_BATCH_USERS_PATH)
        return os.path.join(IMPORT_BATCH_USERS_PATH, _("IDS_USER_INFO") + '.xlsx')

    @property
    def DOWNLOAD_FAILED_USERS_PATH(self):
        if not os.path.exists(IMPOTR_FAILED_USERS_PATH):
            os.makedirs(IMPOTR_FAILED_USERS_PATH)
        return os.path.join(IMPOTR_FAILED_USERS_PATH, _("IDS_IMPORT_FAILED_USERS_DATA") + '.xlsx')

    @property
    def necessary_user_info_list(self):
        return ['login_name', 'display_name', 'depart']

    @property
    def optional_user_info_list(self):
        optional_user_info_list = ['remark',
                                'default_password',
                                'mail_address',
                                'tel_number',
                                'idcard_number',
                                'expire_time',
                                'storage_name',
                                'status',
                                'csf_level',
                                'csf_level2']
        return optional_user_info_list

    @property
    def all_user_info_list(self):
        return self.necessary_user_info_list + self.optional_user_info_list

    @property
    def user_necessary_items(self):
        UserInfoNecessaryItems = namedtuple('UserInfoNecessaryItems', self.necessary_user_info_list)
        return  UserInfoNecessaryItems(login_name = _("IDS_LOGIN_NAME"),
                                            display_name = _("IDS_DISPLAY_NAME"),
                                            depart = _("IDS_DEPART_NAMES"))

    @property
    def user_optional_items(self):
        UserInfoOptionalItems = namedtuple('UserInfoOptionalItems', self.optional_user_info_list)
        return UserInfoOptionalItems(remark = _("IDS_REMARK"),
                                default_password = _("IDS_DEFAULT_PASSWORD"),
                                mail_address = _("IDS_MAIL_ADDRESS"),
                                tel_number = _("IDS_TEL_NUMBER"),
                                idcard_number = _("IDS_IDCARD_NUMBER"),
                                expire_time = _("IDS_EXPIRE_TIME"),
                                storage_name = _("IDS_STORAGE_LOCATION"),
                                status = _("IDS_STATUS"),
                                csf_level = _("IDS_CSF_LEVEL"),
                                csf_level2 = _("IDS_CSF_LEVEL2")
                                )

    # default cell format
    def default_format(self, workbook):
        # user info cell format
        default_cell_format = workbook.add_format({'font_size': 11, 'num_format': '@'})
        # Set the alignment for data in the optional cell
        default_cell_format.set_align('center')
        default_cell_format.set_align('vcenter')
        return default_cell_format

    def fill_user_info_line(self, workbook, worksheet, row_num, start_column_num, user_info_items,
                        cell_format = None):
        if cell_format is None:
            cell_format = self.default_format(workbook)
        for value in user_info_items:
            worksheet.write(row_num, start_column_num, value, cell_format)
            start_column_num += 1

    def get_key_by_value(self, _dict, value):
        for k, v in list(_dict.items()):
            if v == value:
                return k

    def data_validateion(self, all_user_info_list, start_row, worksheet):
        """
        部分输入参数合法性检查和下拉选项设置
        """
        # 限制身份证号的长度为8到18位
        idcard_number_column = all_user_info_list.index('idcard_number')
        worksheet.data_validation(start_row, idcard_number_column, MAXROW, idcard_number_column,
                                    {'validate': 'length',
                                    'criteria': 'between',
                                    'minimum': 8,
                                    'maximum': 18,
                                    'error_message':_('IDS_LENGTH_OF_ID_NUMBER')})

        # 有效期限设置为大于等于下载模板时的日期
        expire_time_column = all_user_info_list.index('expire_time')
        today = BusinessDate.today()
        worksheet.data_validation(start_row, expire_time_column, MAXROW, expire_time_column,
                                    {'validate': 'date',
                                    'criteria': '>=',
                                    'value': today,
                                    'input_message': _('IDS_TIME_FORMAT'),
                                    'error_message':_('IDS_INVALID_DATE')})

        # 存储位置
        storage_location_list = self.get_storage_location_list()
        storage_info_column = all_user_info_list.index('storage_name')
        worksheet.data_validation(start_row, storage_info_column, MAXROW, storage_info_column,
                                        {'validate': 'list',
                                        'source': storage_location_list})

        # 状态
        user_status_column = all_user_info_list.index('status')
        worksheet.data_validation(start_row, user_status_column, MAXROW, user_status_column,
                                        {'validate': 'list',
                                        'source': [_('IDS_ENABLED'), _('IDS_DISABLED')]})

        #用户密级
        csf_level_column = all_user_info_list.index('csf_level')
        sorted_csf = sorted(list(self.csf_level_dict.items()), key = lambda item: item[1])
        worksheet.data_validation(start_row, csf_level_column, MAXROW, csf_level_column,
                                        {'validate': 'list',
                                        'source': [item[0] for item in sorted_csf]})

        #用户密级2
        csf_level2_column = all_user_info_list.index('csf_level2')
        sorted_csf2 = sorted(list(self.csf_level2_dict.items()), key = lambda item: item[1])
        worksheet.data_validation(start_row, csf_level2_column, MAXROW, csf_level2_column,
                                        {'validate': 'list',
                                        'source': [item[0] for item in sorted_csf2]})

    def __create_common_info(self, workbook, worksheet):
        """
        生成xlsx前三行信息
        """
        csf_message = _("IDS_FILE_DESCRIPTION_ADDITON")
        introduction_info = _("IDS_FILE_DESCRIPTION") % csf_message

        user_infos_length = len(self.all_user_info_list)

        # first row(merge columns to one)
        introduction_format = workbook.add_format({'bold': True,'font_size': 18, 'text_wrap': True})
        worksheet.merge_range(0, 0, 0, user_infos_length - 1, introduction_info, introduction_format)

        # Set the columns height and width
        worksheet.set_row(0, 300)
        worksheet.set_row(2, 30)
        # 除了日期，所有列设置为文本,13为column的宽度
        expire_time_colunm = self.all_user_info_list.index('expire_time')
        worksheet.set_column(0,user_infos_length, 13, self.default_format(workbook))
        worksheet.set_column(expire_time_colunm, expire_time_colunm , 13,
                                workbook.add_format({'num_format': 'yyyy/mm/dd', 'valign': 'vcenter', 'align': 'center'}))

        # fill second row(user infos、user roles and customize cell)
        second_row_format = self.default_format(workbook)
        second_row_format.set_font_size(13)
        worksheet.merge_range(1, 0, 1, user_infos_length - 1, _("IDS_USER_INFO"), second_row_format)

        # fill third row(user infos cell)
        red_cell_format = self.default_format(workbook)
        red_cell_format.set_font_color('red')
        red_cell_format.set_bold()

        row_num = 2
        column_num = 0
        self.fill_user_info_line(workbook, worksheet, row_num, column_num, self.user_necessary_items, red_cell_format)
        self.fill_user_info_line(workbook, worksheet, row_num, column_num + len(self.user_necessary_items), self.user_optional_items)

        # data validate
        self.data_validateion(self.all_user_info_list, START_ROW, worksheet)

        # 部门、身份证号field宽度需更大，重新设置
        depart_column = self.all_user_info_list.index('depart')
        idcard_column = self.all_user_info_list.index('idcard_number')
        worksheet.set_column(depart_column, depart_column, 20, self.default_format(workbook))
        worksheet.set_column(idcard_column, idcard_column, 20, self.default_format(workbook))

    def get_depart_directory(self, user_info):
        """
        根据用户信息获取所在所有部门的路径
        """
        directory = []
        for id, name in zip(user_info.departmentIds, user_info.departmentNames):
            if id in self.id_to_departname_cache:
                directory.append(self.id_to_departname_cache[id])
            else:
                parent_path = self.depart_manage.get_parent_path(id)
                full_path = parent_path + ('/' if parent_path else '') + name
                directory.append(full_path)
                self.id_to_departname_cache[id] = full_path
        return ','.join(directory)

    def __get_error_message_id(self, ex):
        if isinstance(ex, ncTException):
            return ex.expMsg, ex.errID
        else:
            return str(ex), 0

    def __fill_user_to_file(self, workbook, worksheet, all_users, start_row, start_column):
        """
        把用户信息写入xlsx文件
        """
        # 过滤重复用户
        all_users = list(remove_duplicate_item_from_list(all_users, lambda user: user.id))
        global_info.IMPORT_TOTAL_NUM = len(all_users)
        for user in all_users:
            try:
                worksheet.write(start_row, start_column, user.user.loginName, self.default_format(workbook))
                worksheet.write(start_row, start_column + 1, user.user.displayName, self.default_format(workbook))
                worksheet.write(start_row, start_column + 2, self.get_depart_directory(user.user))
                worksheet.write(start_row, start_column + 3, user.user.remark, self.default_format(workbook))
                worksheet.write(start_row, start_column + 4, '', self.default_format(workbook))
                worksheet.write(start_row, start_column + 5, user.user.email, self.default_format(workbook))
                worksheet.write(start_row, start_column + 6, user.user.telNumber, self.default_format(workbook))
                worksheet.write(start_row, start_column + 7, user.user.idcardNumber, self.default_format(workbook))
                worksheet.write(start_row, start_column + 8, self.convert_expire_time(user.user.expireTime, False),
                                workbook.add_format({'num_format': 'yyyy/mm/dd', 'valign': 'vcenter', 'align': 'center'}))
                worksheet.write(start_row, start_column + 9, self.convert_ossinfo_to_exel(user.user.ossInfo),
                                self.default_format(workbook))
                worksheet.write(start_row, start_column + 10, self.convert_status(user.user.status, False), self.default_format(workbook))
                worksheet.write(start_row, start_column + 11, self.convert_csf_level(user.user.csfLevel, False), self.default_format(workbook))
                worksheet.write(start_row, start_column + 12, self.convert_csf_level2(user.user.csfLevel2, False), self.default_format(workbook))
                start_row += 1
                global_info.IMPORT_SUCCESS_NUM += 1

                ShareMgnt_Log("导出用户成功: %s_%s",
                            user.user.loginName,
                            user.user.displayName)

            except Exception as ex:
                ex_msg, error_id = self.__get_error_message_id(ex)
                msg = "导出用户%s（%s）出现异常，异常原因：%s。"%(user.user.loginName, user.user.displayName, ex_msg)
                global_info.IMPORT_FAIL_NUM += 1
                self.add_fail_info(user.user, ex_msg, error_id)
                ShareMgnt_Log(msg)

    def make_store_file_dir(self, taskId, fileName):
        """
        新建目录
        """
        fileDir = os.path.join(DOWNLOAD_PATH, taskId)

        # 新建存放文件的目录 /sysvol/cache/sharemgnt/batchusers/taskId/
        if not os.path.exists(fileDir):
            os.makedirs(fileDir)

        return os.path.join(fileDir, fileName)

    def gen_batch_user_file(self, task_id, task_info):
        """
        生成xlsx文件
        """
        # 初始化
        global_info.init_import_variable()
        self.id_to_departname_cache = {}
        self.path_cache = {}
        self.available_path_list = []
        self.__init_evfs_oss_infos()

        # 新建文件目录
        file_dir = self.make_store_file_dir(task_id, task_info.name)

        workbook = xlsxwriter.Workbook(file_dir)
        worksheet = workbook.add_worksheet(_('IDS_USER_INFO'))

        # 获取所有用户
        all_users = []
        for each in set(task_info.inquire_date):
            all_users.extend(self.depart_manage.get_all_users_of_depart(depart_id=each, only_user_id=False))
        #导出用户方法加锁
        with self.import_mutex:
            ShareMgnt_Log('-------------------开始批量导出用户-------------------')
            self.__create_common_info(workbook, worksheet)
            self.__fill_user_to_file(workbook, worksheet, all_users, START_ROW, 0)
            ShareMgnt_Log('-------------------批量导出用户结束-------------------')

        workbook.close()

        # 更新任务状态
        task_info.set_finished_status(TaskFinishedStatus.TASK_FINISHED)

        return file_dir

    def export_batch_users(self, department_ids, responsible_person_id):
        """
        导出用户信息
        """
        global IS_SINGLE
        if not IS_SINGLE:
            with TClient('ShareMgntSingle') as client:
                return client.Usrm_ExportBatchUsers(department_ids, responsible_person_id)

        # 检查用户是否有权限可以导出
        if not self.role_manage.check_user_rights(responsible_person_id, AVAILABLE_ROLE_TUPLE):
            raise_exception(exp_msg=_("IDS_INVALID_MANAGER_ID"),
                            exp_num=ncTShareMgntError.NCT_INVALID_MANAGER_ID)

        # 检查部门
        errDetail = {}
        errDetail['unexist_depart_ids'] = []
        for each in set(department_ids):
            # 不能选择未分配组和所有用户组
            if each in [NCT_UNDISTRIBUTE_USER_GROUP, NCT_ALL_USER_GROUP]:
                raise_exception(exp_msg=_('IDS_CANNOT_SELECT_UNDISTRIBUTE_AND_ALL_GROUP'),
                                exp_num=ncTShareMgntError.NCT_CANNOT_SELECT_UNDISTRIBUTE_AND_ALL_GROUP)

            # 检查部门是否存在
            result = self.depart_manage.check_depart_exists(each, True, False)
            if not result:
                errDetail['unexist_depart_ids'].append(each)
        if errDetail['unexist_depart_ids']:
            raise_exception(exp_msg=_("src department not exist."),
                            exp_num=ncTShareMgntError.NCT_SRC_DEPARTMENT_NOT_EXIST,
                            exp_detail=json.dumps(errDetail, ensure_ascii=False))

        # 导入、导入任务同时只能执行一个
        BaseTaskInfo.check_task_exist()
        if self.import_mutex.locked():
            raise_exception(exp_msg=_("IDS_BATCH_USERS_IMPORTING"),
                            exp_num=ncTShareMgntError.NCT_BATCH_USERS_IMPORTING)

        # 添加任务
        task_info = TaskInfo()
        task_info.name = _("IDS_USER_INFO") + '.xlsx'
        task_info.inquire_date = department_ids
        # 创建生成任务
        task_id = BaseTaskInfo.add_gen_file_task(task_info)

        # 启动任务处理线程
        gen_file_thread = GenFileThread(task_id, task_info, self.gen_file_handler, DOWNLOAD_PATH)
        gen_file_thread.daemon = True
        gen_file_thread.start()

        return task_id

    def gen_file_handler(self, taskId, taskInfo):
        """
        生成文件
        """
        if taskId is None or taskInfo is None:
            return

        # 生成文件
        return self.gen_batch_user_file(taskId, taskInfo)

    def download_batch_users_file(self, taskId):
        """
        批量下载用户
        """
        global IS_SINGLE
        if not IS_SINGLE:
            with TClient('ShareMgntSingle') as client:
                return client.Usrm_DownloadBatchUsers(taskId)

        taskInfo = BaseTaskInfo.get_gen_task_info(taskId, _("IDS_DOWNLOAD_BATCH_USERS_TASK_NOT_EXIST"),
                                    ncTShareMgntError.NCT_DOWNLOAD_BATCH_USERS_TASK_NOT_EXIST)
        finished_status = taskInfo.get_finished_status()

        if finished_status == TaskFinishedStatus.TASK_FINISHED:
            reportInfo = ncTReportInfo()
            with open(taskInfo.file_path, 'rb') as fd:
                reportInfo.reportData = fd.read()
            reportInfo.reportName = taskInfo.name
            BaseTaskInfo.del_gen_file_task(taskId, _("IDS_DOWNLOAD_BATCH_USERS_TASK_NOT_EXIST"),
                                    ncTShareMgntError.NCT_DOWNLOAD_BATCH_USERS_TASK_NOT_EXIST)
            return reportInfo

        if finished_status == TaskFinishedStatus.TASK_IN_PROCESS:
            raise_exception(exp_msg=_("IDS_BATCH_USERS_EXPORTING"),
                            exp_num=ncTShareMgntError.NCT_BATCH_USERS_EXPORTING)

        if finished_status == TaskFinishedStatus.TASK_ERROR:
            BaseTaskInfo.del_gen_file_task(taskId, _("IDS_DOWNLOAD_BATCH_USERS_TASK_NOT_EXIST"),
                                    ncTShareMgntError.NCT_DOWNLOAD_BATCH_USERS_TASK_NOT_EXIST)
            raise_exception(exp_msg=_("IDS_DOWNLOAD_BATCH_USERS_FAILED"),
                            exp_num=ncTShareMgntError.NCT_DOWNLOAD_BATCH_USERS_FAILED)

    def __check_xlsx_file(self, user_infos_table, responsible_person_id):
        """
        检查文件
        """
        # 检查格式
        user_info_list = user_infos_table.row_values(START_ROW - 1)
        original_user_info_list = [each for each in self.user_necessary_items] + [each for each in self.user_optional_items]
        if len(user_info_list) != len(original_user_info_list):
            raise_exception(exp_msg=_('IDS_WRONG_FILE_FORMAT'),
                            exp_num=ncTShareMgntError.NCT_WRONG_FILE_FORMAT)
        for i in range(len(user_info_list)):
            if user_info_list[i] != original_user_info_list[i]:
                raise_exception(exp_msg=_('IDS_WRONG_FILE_FORMAT'),
                                exp_num=ncTShareMgntError.NCT_WRONG_FILE_FORMAT)

    def __parse_xlsx_file(self, user_infos_table):
        """
        解析用户信息
        """
        all_rows_num = user_infos_table.nrows
        global_info.IMPORT_TOTAL_NUM = all_rows_num - START_ROW

        for i in range(START_ROW, all_rows_num):
            user_info_list = user_infos_table.row_values(i)
            yield dict(list(zip(self.all_user_info_list, user_info_list)))

    def convert_expire_time(self, expire_time, is_import = True):
        if is_import:
            try:
                # 不填为不限制时间
                if not expire_time:
                    return -1
                # 输入的格式为unicode, 比如'2020/01/01, 00:00:00',转换为'2020/01/01,23:59:59'
                if isinstance(expire_time, str) and '/' in expire_time:
                    return int(time.mktime(time.strptime(expire_time, '%Y/%m/%d'))) + 86399
                else:
                    # 输入的格式为天数,比如43697.0(0代表从1990.01.01开始计算)
                    return int(xlrd.xldate.xldate_as_datetime(expire_time, 0).strftime('%s')) + 86399
            except:
                raise_exception(exp_msg=_("IDS_INVALID_DATE"),
                                exp_num=ncTShareMgntError.NCT_INVALID_DATETIME)
        else:
            return '' if expire_time == -1 else datetime.fromtimestamp(expire_time).strftime('%Y/%m/%d')

    def convert_status(self, status, is_import = True):
        if is_import:
            if status ==_('IDS_DISABLED'):
                return ncTUsrmUserStatus.NCT_STATUS_DISABLE
            elif status in [_('IDS_ENABLED'), '', '']:
                return ncTUsrmUserStatus.NCT_STATUS_ENABLE
            else:
                raise_exception(exp_msg=_('IDS_INVALID_USER_STATUS'),
                                exp_num=ncTShareMgntError.NCT_INVALID_USER_STATUS)
        else:
            return _('IDS_DISABLED') if status else _('IDS_ENABLED')

    def __init_evfs_oss_infos(self):
        # 更新所有对象存储信息
        # 每次导入、导出前更新一次
        code, self.evfs_oss_infos = self.ossgateway_driven.get_as_storage_info()
        if code != 200:
            ShareMgnt_Log(
                f'get_as_storage_info failed: {code},{self.evfs_oss_infos}')
            raise_exception(exp_msg=_("IDS_GET_OSS_INFO_FAILD"),
                            exp_num=ncTShareMgntError.NCT_NO_AVAILABLE_OSS)

    def get_oss_infos(self):
        # 获取所有对象存储信息
        evfs_oss_infos = self.evfs_oss_infos
        oss_infos = []
        for evfs_oss_info in evfs_oss_infos:
            oss_info = ncTUsrmOSSInfo()
            oss_info.ossId = evfs_oss_info["id"]
            oss_info.ossName = evfs_oss_info["name"]
            oss_info.enabled = evfs_oss_info["enabled"]
            oss_infos.append(oss_info)
        return oss_infos

    def get_storage_location_list(self):
        # 获取展示给用户的对象存储信息列表
        oss_infos = self.get_oss_infos()
        # 首项增加空信息
        oss_infos.insert(0, ncTUsrmOSSInfo())
        return [self.convert_ossinfo_to_exel(i) for i in oss_infos]

    def convert_exel_ossinfo(self, exel_oss_name):
        """
        把exel中存储信息转化为ncTUsrmOSSInfo
        exel_oss_name: exel中存储信息
        oss_infos：所有可用存储信息
        """
        # 如果导入，根据字符串返回oss_name
        exel_oss_name = self.convert_decoding(exel_oss_name)
        if exel_oss_name in [_('IDS_UNSPECIFIED_STORAGE'), '', '']:
            return ncTUsrmOSSInfo()
        else:
            # 获取对象存储信息
            for evfs_oss_info in self.evfs_oss_infos:
                # 先用对象存储名和站点名匹配对象存储
                if evfs_oss_info["name"] == exel_oss_name:
                    oss_info = ncTUsrmOSSInfo()
                    oss_info.ossId = evfs_oss_info["id"]
                    oss_info.ossName = evfs_oss_info["name"]
                    oss_info.enabled = evfs_oss_info["enabled"]
                    return oss_info

            # 都匹配不上，报错
            raise_exception(exp_msg=_('IDS_OSS_NOT_EXIST') % exel_oss_name,
                            exp_num=ncTShareMgntError.NCT_OSS_NOT_EXIST)

    def convert_ossinfo_to_exel(self, oss_info):
        """
        把ncTUsrmOSSInfo转化为exel中存储信息
        """
        # 如果ossId为空，显示“未指定（使用默认存储）”
        if not oss_info or not oss_info.ossId:
            return _("IDS_UNSPECIFIED_STORAGE")
        for evfs_oss_info in self.evfs_oss_infos:
            if evfs_oss_info["id"] == oss_info.ossId:
                return evfs_oss_info["name"]

        # 如果没有匹配到evfs中的信息，直接返回
        return _("IDS_UNSPECIFIED_STORAGE")

    def convert_user_space(self, space, is_import = True):
        if is_import:
            try:
                if space:
                    if type(space) not in [str, bytes]:
                        space = str(space)
                    return float(self.convert_decoding(space)) * (1024 ** 3)
                else:
                    # 不填默认为系统设置的
                    return ConfigManage().get_default_space_size()
            except Exception as ex:
                ShareMgnt_Log(str(ex))
                raise_exception(exp_msg=_("IDS_INVALID_USER_SPACE"),
                                exp_num=ncTShareMgntError.NCT_INVALID_USER_SPACE)
        else:
            return format(space / (1024 ** 3), '0.2f')

    def convert_csf_level(self, csf_level, is_import = True):
        if is_import:
            try:
                if csf_level:
                    return self.csf_level_dict[self.convert_decoding(csf_level)]
                else:
                    # 不填默认用户密级最小值
                    return self.config_manage.get_min_csf_level()
            except:
                # 密级不存在或者不合法，报错
                csf_level = self.config_manage.get_csf_levels()
                csf_level_list = list(csf_level.values())
                raise_exception(exp_msg=(_("IDS_INVALID_CSF_LEVEL") % csf_level_list),
                                exp_num=ncTShareMgntError.NCT_INVALID_CSF_LEVEL)
        else:
            return self.get_key_by_value(self.csf_level_dict, csf_level)


    def convert_csf_level2(self, csf_level2, is_import = True):
        if is_import:
            try:
                if csf_level2:
                    return self.csf_level2_dict[self.convert_decoding(csf_level2)]
                else:
                    return self.config_manage.get_min_csf_level2()
            except:
                csf_level2 = self.config_manage.get_csf_levels2()
                csf_level2_list = list(csf_level2.values())
                raise_exception(exp_msg=(_("IDS_INVALID_CSF_LEVEL2") % csf_level2_list),
                                exp_num=ncTShareMgntError.NCT_INVALID_CSF_LEVEL2)
        else:
            return self.get_key_by_value(self.csf_level2_dict, csf_level2)

    def convert_decoding(self, x):
        if x:
            return x.strip() if isinstance(x, str) else bytes.decode(x).strip()
        else:
            return ''

    def __parse_user_info(self, user_info):
        """
        把exel中的用户数据转换为ncTUsrmAddUserInfo
        """
        try:
            user = ncTUsrmAddUserInfo()
            user.password = self.convert_decoding(user_info.get('default_password'))
            user.user = ncTUsrmUserInfo()
            user.user.loginName = self.convert_decoding(user_info.get('login_name'))
            user.user.displayName = self.convert_decoding(user_info.get('display_name'))
            user.user.departmentNames = self.convert_decoding(user_info.get('depart')).split(',')
            user.user.remark = self.convert_decoding(user_info.get('remark'))
            user.user.email = self.convert_decoding(user_info.get('mail_address'))
            user.user.telNumber = self.convert_decoding(user_info.get('tel_number'))
            user.user.idcardNumber = self.convert_decoding(user_info.get('idcard_number'))
            user.user.expireTime = self.convert_expire_time(user_info.get('expire_time'))
            user.user.ossInfo = ncTUsrmOSSInfo()
            user.user.ossInfo = self.convert_exel_ossinfo(user_info.get('storage_name'))
            user.user.status = self.convert_status(user_info.get('status'))
            user.user.csfLevel = self.convert_csf_level(user_info.get('csf_level'))
            user.user.csfLevel2 = self.convert_csf_level2(user_info.get('csf_level2'))
            return user
        except Exception as ex:
            ex_msg, error_id = self.__get_error_message_id(ex)
            msg = '导入用户失败: %s_%s, %s' % (user.user.loginName, user.user.displayName, ex_msg)
            ShareMgnt_Log(msg)

            global_info.IMPORT_FAIL_NUM += 1
            self.add_fail_info(user.user, ex_msg, error_id)

    def get_restrict_depart(self, responsible_person_id):
        """
        获取管理员所管辖的部门
        """
        if self.role_manage.check_user_rights(responsible_person_id, (NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_ADMIN)):
            root_infos = []
        else:
            root_infos = self.depart_manage.get_supervisory_root_org(responsible_person_id)

        restrict_depart_list = []
        for each_root in root_infos:
            if each_root.isOrganization:
                restrict_depart_list.append(each_root.name)
            else:
                path_list = self.depart_manage.get_dept_path_name_to_root(each_root.id)
                full_path = '/'.join(path_list)
                restrict_depart_list.append(full_path)
        return restrict_depart_list

    def __check_available_one(self, import_departs, restrict_departs):
        """
        import_departs:待导入的部门名，如['A/B', 'A/B/C']
        restrict_departs:允许的部门名，如['A' ,'C/dd']
        只要import_departs的一个部门不在restrict_departs，报错
        """
        for each_import in import_departs:
            for each_restrict in restrict_departs:
                if each_import.startswith(each_restrict):
                    continue
                else:
                    raise_exception(exp_msg=_("IDS_CANNOT_DELETE_DEPART_NO_PERM"),
                                    exp_num=ncTShareMgntError.NCT_CANNOT_DELETE_DEPART_NO_PERM)

    def __check_available_all(self, import_departs, restrict_departs):
        """
        import_departs:待导入的部门名，如['A/B', 'A/B/C']
        restrict_departs:允许的部门名，如['A' ,'C/dd']
        import_departs的所有部门不在restrict_departs，报错；有一个部门在，则不报错
        """
        flag = False
        for each_import in import_departs:
            if flag:
                break
            for each_restrict in restrict_departs:
                if each_import.startswith(each_restrict):
                    flag = True
                    break

        if not flag:
            raise_exception(exp_msg=_("IDS_CANNOT_DELETE_DEPART_NO_PERM"),
                            exp_num=ncTShareMgntError.NCT_CANNOT_DELETE_DEPART_NO_PERM)

    def check_avaible(self, depart_paths, restrict_list, user_info = None):
        """
        检查组织管理员是否能新建用户、部门
        user_id:要导入的用户
        depart_paths:要导入用户的部门
        restrict_list：负责人所管辖的部门
        """
        # 管辖部门为空，说明为超级管理员，不用检查
        if not restrict_list:
            return

        if user_info:
            # user_info不为空，说明是修改用户
            # 检查要导入用户的部门
            # 待修改用户的所有部门都不在管辖部门内，报错
            user_exist_paths =  self.get_depart_directory(user_info).split(',')
            self.__check_available_all(user_exist_paths, restrict_list)

            # 去掉之前用户所在的所有部门
            # 剩下的部门只要有一个部门不在负责人的管辖范围内，报错
            depart_paths = set(depart_paths) - set(user_exist_paths)
            # 没有新增部门，跳出检查；新增了部门，在下面检查新增部门的合法性
            if not depart_paths:
                return

        # 只要有一个部门不在负责人的管辖范围内，报错
        self.__check_available_all(depart_paths, restrict_list)

    def get_department_id(self, depart_path):
        """
        根据传入的depart_path得到部门id，如果不存在，创建部门，并把映射关系写入到path_cache
        depart_path格式:'AS/web/xxx'
        返回部门xxx的id
        """
        latest_id = ''
        not_found = []
        depth = depart_path.count('/')
        while depth > 0:
            # 查根据部门深度，从内层部门至外层部门循环查找不在缓存中的部门
            if depart_path in self.path_cache:
                latest_id = self.path_cache[depart_path]
                break
            else:
                not_found.append(depart_path)
                depth -= 1
                depart_path = depart_path[:depart_path.rfind('/')]

        if not latest_id:
            current_path = depart_path
            # 缓存中无对应的根组织路径
            # 添加组织
            try:
                organization = ncTAddOrgParam()
                organization.orgName = current_path
                depart_id = self.depart_manage.create_organization(organization)
                ShareMgnt_Log("新建组织成功，%s_%s", depart_id, current_path)
            except ncTException as ex:
                if ex.errID == ncTShareMgntError.NCT_ORGNIZATION_HAS_EXIST:
                    organization = self.depart_manage.get_organization_by_Name(current_path)
                    depart_id = organization.organizationId
                else:
                    raise ex

            latest_id = depart_id
            self.path_cache[current_path] = latest_id

        while len(not_found) > 0:
            # 缓存中存在根组织
            # 根据not_found表中的数据依次创建部门
            current_path = not_found.pop()
            name = current_path.rsplit('/')[-1]
            # 添加部门
            try:
                tmp_depart = ncTAddDepartParam()
                tmp_depart.departName = name
                tmp_depart.parentId = latest_id
                depart_id = self.depart_manage.add_department(tmp_depart)
                ShareMgnt_Log("新建部门成功，%s_%s", depart_id, name)
            except ncTException as ex:
                if ex.errID == ncTShareMgntError.NCT_DEPARTMENT_HAS_EXIST:
                    depart_id = self.depart_manage.get_depart_id_by_name(tmp_depart.departName, tmp_depart.parentId)
                else:
                    raise ex

            latest_id = depart_id
            self.path_cache[current_path] = latest_id

        return latest_id

    def import_user(self, user, user_cover, responsible_person_id, restrict_depart_names):
        """
        导入用户
        """
        if not user:
            return

        try:
            origin_user_info = copy.deepcopy(user)
            # 如果身份号有'*'，认为没有修改
            if '*' * 11 in user.user.idcardNumber:
                user.user.idcardNumber = ''

            # 账号不能和保留管理员账号重复
            if user.user.loginName.lower() in self.admin_list:
                raise_exception(exp_msg=_("account confict with admin"),
                                exp_num=ncTShareMgntError.NCT_ACCOUNT_CONFICT_WITH_ADMIN)

            # 显示名不能为空
            if not user.user.displayName:
                raise_exception(exp_msg=_("IDS_INVALID_DISPLAY_NAME"),
                                exp_num=ncTShareMgntError.NCT_INVALID_DISPLAY_NAME)

            # 检查用户参数
            self.user_manage.check_user(user.user, True)

            #检查重复邮箱
            self.user_manage.check_duplicate_email_by_login_name(user.user.email, user.user.loginName)

            # 检查是否存在同账号用户
            as_user = self.user_manage.get_user_by_loginname(user.user.loginName)

            if as_user:
                # 覆盖用户
                if user_cover:
                    # 只有新增用户时可以填写初始密码
                    if user.password:
                        raise_exception(exp_msg=_('IDS_CANNOT_FILL_IN_DEFAUL_PASSWORD'),
                                        exp_num=ncTShareMgntError.NCT_CANNOT_FILL_IN_DEFAUL_PASSWORD)
                    # 组织管理员不能修改自身信息(安全性考虑)
                    if as_user.id == responsible_person_id and self.user_manage.check_is_responsible_person(responsible_person_id):
                        raise_exception(exp_msg=_("IDS_ORG_MANAGER_CANNOT_EDIT_OWN_AUTH_INFO"),
                                        exp_num=ncTShareMgntError.NCT_ORG_MANAGER_CANNOT_EDIT_OWN_AUTH_INFO)
                    # 使用唯一显示名
                    user.user.displayName = self.user_manage.get_unique_displayname(user.user.displayName, as_user.id)
                    # 设置用户状态
                    user_status = True if user.user.status== ncTUsrmUserStatus.NCT_STATUS_ENABLE else False
                    self.user_manage.set_user_status(as_user.id, user_status)

                    # 获取部门id，若部门不存在，新建部门
                    self.check_avaible(user.user.departmentNames, restrict_depart_names, as_user.user)
                    user.user.departmentIds = [self.get_department_id(each) for each in user.user.departmentNames]

                    sql = """
                    UPDATE `t_user` SET
                        `f_display_name` = %s,
                        `f_remark` = %s,
                        `f_mail_address` = %s,
                        `f_tel_number` = %s,
                        `f_idcard_number` = %s,
                        `f_expire_time` = %s,
                        `f_oss_id` = %s,
                        `f_csf_level` = %s,
                        `f_csf_level2` = %s
                    WHERE `f_user_id` = %s
                    """
                    self.w_db.query(sql,
                                    user.user.displayName,
                                    user.user.remark,
                                    user.user.email,
                                    user.user.telNumber,
                                    user.user.idcardNumber,
                                    user.user.expireTime,
                                    user.user.ossInfo.ossId,
                                    user.user.csfLevel,
                                    user.user.csfLevel2,
                                    as_user.id)

                    # 补充部门关联关系
                    for depart_id in user.user.departmentIds:
                        self.depart_manage.add_user_to_department([as_user.id], depart_id)

                    if as_user.user.displayName != user.user.displayName:
                        # 发送用户显示名更新nsq消息
                        pub_nsq_msg(TOPIC_ORG_NAME_MODIFY, {"id": as_user.id, "new_name": user.user.displayName, "type": "user"})

                    user_modify_info = {}
                    if as_user.user.telNumber != user.user.telNumber:
                        user_modify_info["new_telephone"] = user.user.telNumber
                    if as_user.user.email != user.user.email:
                        user_modify_info["new_email"] = user.user.email

                    if len(user_modify_info) > 0:
                        # 发送用户信息更新nsq消息
                        user_modify_info["user_id"] = as_user.id
                        pub_nsq_msg(TOPIC_USER_MODIFIED, user_modify_info)

                    # 记录日志
                    ShareMgnt_Log("编辑用户成功: %s_%s",
                                user.user.loginName,
                                user.user.displayName)

                    eacp_log(responsible_person_id,
                            global_info.LOG_TYPE_MANAGE,
                            global_info.USER_TYPE_AUTH,
                            global_info.LOG_LEVEL_INFO,
                            global_info.LOG_OP_TYPE_SET,
                            _('IDS_EDIT_USER_SUCCEEDED_MSG') %
                            (origin_user_info.user.displayName, origin_user_info.user.loginName),
                            self.get_ex_msg(origin_user_info, 0))
            # 新增用户
            else:
                user.user.displayName = self.user_manage.get_unique_displayname(user.user.displayName)
                # 权重设为默认值
                user.user.priority = 999
                # 不设置密码管控
                user.user.pwdControl = 0

                # 设置密码，不填为默认密码
                if user.password:
                    user.password = user.password.strip()
                    self.user_manage.check_password_valid(user.password)
                    user.sha2Password = sha2_encrypt(user.password)
                    user.ntlmPassword = ntlm_md4(user.password)
                else:
                    user.sha2Password = self.user_manage.user_default_password.sha2_pwd
                    user.ntlmPassword = self.user_manage.user_default_password.ntlm_pwd

                # 获取部门id，若部门不存在，新建部门
                self.check_avaible(user.user.departmentNames, restrict_depart_names)
                user.user.departmentIds = [self.get_department_id(each) for each in user.user.departmentNames]

                user_id = self.user_manage.add_user_to_db(user)

                ShareMgnt_Log("导入用户成功: %s_%s_%s",
                            user_id,
                            user.user.loginName,
                            user.user.displayName)

                eacp_log(responsible_person_id,
                        global_info.LOG_TYPE_MANAGE,
                        global_info.USER_TYPE_AUTH,
                        global_info.LOG_LEVEL_INFO,
                        global_info.LOG_OP_TYPE_CREATE,
                        _('IDS_ADD_USER_SUCCEEDED_MSG') %
                        (origin_user_info.user.displayName, origin_user_info.user.loginName),
                        self.get_ex_msg(origin_user_info, 1))

            global_info.IMPORT_SUCCESS_NUM += 1

        except Exception as ex:
            ex_msg, error_id = self.__get_error_message_id(ex)
            change_space_to_null = lambda  x: x if x else 'NULL'
            msg = '导入用户失败: %s_%s, %s' % (change_space_to_null(origin_user_info.user.loginName),
                                        change_space_to_null(origin_user_info.user.displayName), ex_msg)

            ShareMgnt_Log(msg)

            global_info.IMPORT_FAIL_NUM += 1
            self.add_fail_info(origin_user_info.user, ex_msg, error_id)

            if isinstance(ex, ncTException) and ex.expMsg == _("user num overflow"):
                global_info.IMPORT_IS_STOP = True

    def get_ex_msg(self, user, is_add_user = 1):
        """
        附加信息为：用户名 “<用户名>”；原显示名 “<原显示名>”；邮箱地址 “<邮箱地址>”；手机号码“<手机号码>”；
        身份证号“<身份证号>”；存储位置 “<存储位置>”；用户密级 “<用户密级>; 用户密级2 “<用户密级2>”
        """
        if is_add_user:
            display_name = _("IDS_DISPLAY_NAME")
        else:
            display_name = _("IDS_ORIGINAL_DISPLAY_NAME")
        convert_idcard = lambda num:num[:3] + len(num[3:-4]) * '*' + num[-4:] if num and len(num) == 18 else ''
        csf_level = self.convert_csf_level(user.user.csfLevel, False)
        csf_level2 = self.convert_csf_level2(user.user.csfLevel2, False)

        user_info_value =[user.user.loginName,
                        user.user.displayName,
                        user.user.email,
                        user.user.telNumber,
                        convert_idcard(user.user.idcardNumber),
                        csf_level,
                        csf_level2,
                        self.convert_ossinfo_to_exel(user.user.ossInfo)]
        user_info_key = [_("IDS_LOGIN_NAME"),
                        display_name,
                        _("IDS_MAIL_ADDRESS"),
                        _("IDS_TEL_NUMBER"),
                        _("IDS_IDCARD_NUMBER"),
                        _("IDS_CSF_LEVEL"),
                        _("IDS_CSF_LEVEL2"),
                        _("IDS_STORAGE_LOCATION")]

        ex_msg = ''
        for k, v in zip(user_info_key, user_info_value):
            ex_msg = ex_msg + k + '“%s”；'%v
        return ex_msg.rstrip('；')

    def convert_scope(self, num_type, num):
        """
        转换数字的大小，防止数值超过定义文件的范围接口调用失败
        num_type = 0, 为i32
        num_type = 1, 为i64
        """
        if not num:
            return
        if num_type:
            if num < -2147483648 or num > 2147483647:
                return 0
        else:
            if num < -9223372036854775808 or num > 9223372036854775807:
                return 0

    def convert_user_info_scope(self, user):
        user.createTime = self.convert_scope(1,user.createTime)
        user.priority = self.convert_scope(0, user.priority)
        user.csfLevel = self.convert_scope(0, user.csfLevel)
        user.expireTime = self.convert_scope(0, user.expireTime)

    def add_fail_info(self, user, msg, error_id = 0):
        fail_info = ncTImportFailInfo()
        fail_info.index = global_info.IMPORT_SUCCESS_NUM + global_info.IMPORT_FAIL_NUM + START_ROW
        fail_info.userInfo = user
        fail_info.errorMessage = msg
        fail_info.errorID = error_id
        self.convert_user_info_scope(user)
        global_info.IMPORT_FAIL_INFO.append(fail_info)

    def import_batch_users(self, userinfo_file, user_cover, responsible_person_id):
        """
        导入用户
        """
        global IS_SINGLE
        if not IS_SINGLE:
            with TClient('ShareMgntSingle',0) as client:
                return client.Usrm_ImportBatchUsers(userinfo_file, user_cover, responsible_person_id)

        # 检查传入的文件名
        if not userinfo_file.fileName.endswith('.xlsx'):
            raise_exception(exp_msg=_("IDS_INVALID_FILENAME"),
                            exp_num=ncTShareMgntError.NCT_INVALID_FILENAME)

        # 检查用户是否可以导入
        if not self.role_manage.check_user_rights(responsible_person_id, AVAILABLE_ROLE_TUPLE):
            raise_exception(exp_msg=_("IDS_INVALID_MANAGER_ID"),
                            exp_num=ncTShareMgntError.NCT_INVALID_MANAGER_ID)

        with open(self.IMPORT_PATH, 'wb') as f:
            f.write(userinfo_file.data)

        # 导入、导入任务同时只能执行一个
        BaseTaskInfo.check_task_exist()
        if self.import_mutex.locked():
            raise_exception(exp_msg=_("IDS_BATCH_USERS_IMPORTING"),
                            exp_num=ncTShareMgntError.NCT_BATCH_USERS_IMPORTING)

        # 导入用户方法加锁
        with self.import_mutex:
            ShareMgnt_Log('-------------------开始批量导入用户-------------------')

            # 初始化
            global_info.init_import_variable()
            # 导入大量数据缓冲
            global_info.IMPORT_TOTAL_NUM = -1
            try:
                data = xlrd.open_workbook(self.IMPORT_PATH)
            except Exception:
                raise_exception(exp_msg=_('IDS_WRONG_FILE_FORMAT'),
                                exp_num=ncTShareMgntError.NCT_WRONG_FILE_FORMAT)
            user_infos_table = data.sheets()[0]
            self.__check_xlsx_file(user_infos_table, responsible_person_id)

            # 初始化
            self.id_to_departname_cache = {}
            self.path_cache = {}
            self.available_path_list = []
            self.__init_evfs_oss_infos()
            # 获取管辖部门
            restrict_depart_names = self.get_restrict_depart(responsible_person_id)

            for user_info in self.__parse_xlsx_file(user_infos_table):
                user = self.__parse_user_info(user_info)
                self.import_user(user, user_cover, responsible_person_id, restrict_depart_names)

            # 关闭exel文件
            data.release_resources()
            del data

            ShareMgnt_Log('-------------------批量导入用户结束-------------------')

    def down_load_import_failed_users(self):
        """
        下载导入失败的用户信息exel表
        """
        global IS_SINGLE
        if not IS_SINGLE:
            with TClient('ShareMgntSingle') as client:
                return client.Usrm_DownloadImportFailedUsers()

        self.create_import_failed_users_file()
        reportInfo = ncTReportInfo()
        with open(self.DOWNLOAD_FAILED_USERS_PATH, 'rb') as fd:
            reportInfo.reportData = fd.read()
        reportInfo.reportName = _("IDS_IMPORT_FAILED_USERS_DATA") + '.xlsx'
        return reportInfo

    def create_import_failed_users_file(self):
        """
        生成导入失败的用户信息exel表
        """
        try:
            error_row_list = [i.index for i in global_info.IMPORT_FAIL_INFO]

            data = xlrd.open_workbook(self.IMPORT_PATH)
            user_infos_table = data.sheets()[0]

            workbook = xlsxwriter.Workbook(self.DOWNLOAD_FAILED_USERS_PATH)
            worksheet = workbook.add_worksheet(_('IDS_IMPORT_FAILED_USERS_DATA'))
            self.__create_common_info(workbook, worksheet)

            start = START_ROW
            for i in error_row_list:
                # 读取数据时为(exel序号数 - 1)
                user_info_list = user_infos_table.row_values(i - 1)
                for each in range(len(user_info_list)):
                    worksheet.write(start, each, user_info_list[each], self.default_format(workbook))
                start += 1

            # 关闭exel
            workbook.close()
            data.release_resources()
            del data
        except Exception as ex:
            ex_msg, error_id = self.__get_error_message_id(ex)
            ShareMgnt_Log(ex_msg)
            raise_exception(exp_msg=ex_msg, exp_num= error_id)

    def get_export_batch_users_task_status(self, taskId):
        """
        获取文件状态
        """
        global IS_SINGLE
        if not IS_SINGLE:
            with TClient('ShareMgntSingle',0) as client:
                return client.Usrm_GetExportBatchUsersTaskStatus(taskId)

        taskInfo = BaseTaskInfo.get_gen_task_info(taskId, _("IDS_DOWNLOAD_BATCH_USERS_TASK_NOT_EXIST"),
                                    ncTShareMgntError.NCT_DOWNLOAD_BATCH_USERS_TASK_NOT_EXIST)
        if taskInfo.get_finished_status() == TaskFinishedStatus.TASK_IN_PROCESS:
            return False

        return True

    def get_import_user_progress(self):
        """
        导入用户的进度
        """
        global IS_SINGLE
        if not IS_SINGLE:
            with TClient('ShareMgntSingle') as client:
                return client.Usrm_GetProgress()

        import_result = ncTUsrmImportResult()
        import_result.totalNum = global_info.IMPORT_TOTAL_NUM
        import_result.successNum = global_info.IMPORT_SUCCESS_NUM
        import_result.failNum = global_info.IMPORT_FAIL_NUM

        if global_info.IMPORT_IS_STOP:
            import_result.totalNum = import_result.successNum + import_result.failNum
        return import_result

    def get_import_user_errinfo(self , start, limit):
        """
        获取错误信息
        """
        global IS_SINGLE
        if not IS_SINGLE:
            with TClient('ShareMgntSingle') as client:
                return client.Usrm_GetErrorInfos(start, limit)

        count = len(global_info.IMPORT_FAIL_INFO)
        if start < 0:
            raise_exception(exp_msg=_("IDS_START_LESS_THAN_ZERO"),
                            exp_num=ncTShareMgntError.NCT_START_LESS_THAN_ZERO)

        if start > count or (start == count and count > 0):
            raise_exception(exp_msg=_("IDS_START_MORE_THAN_TOTAL"),
                            exp_num=ncTShareMgntError.NCT_START_MORE_THAN_TOTAL)

        if limit < -1:
            raise_exception(exp_msg=_("IDS_LIMIT_LESS_THAN_MINUS_ONE"),
                            exp_num=ncTShareMgntError.NCT_LIMIT_LESS_THAN_MINUS_ONE)
        if limit == -1:
            end = count
        else:
            end = start + limit

        return global_info.IMPORT_FAIL_INFO[start:end]
