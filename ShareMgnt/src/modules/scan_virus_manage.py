#!/usr/bin/python3
# -*- coding:utf-8 -*-

from src.common.lib import (check_start_limit,
                            raise_exception,
                            generate_group_str,
                            check_filename,
                            exec_command,
                            check_service_node)
from src.common.db.connector import DBConnector
from src.common.db.db_manager import get_db_name
from src.common.sharemgnt_logger import ShareMgnt_Log
from src.common.nc_senders import email_send_html_content
from src.common.business_date import BusinessDate
from src.common.http import send_request
from src.modules.user_manage import UserManage
from src.modules.config_manage import ConfigManage
from src.modules.department_manage import DepartmentManage
from src.driven.service_access.ossgateway_config import OssgatewayDriven
from src.modules.role_manage import RoleManage
from src.modules.oem_manage import OEMManage
from ShareMgnt.constants import (ncTScanTaskStatus,
                                 ncTScanTaskInfo,
                                 ncTVirusInfo,
                                 ncTDocType,
                                 ncTVirusDBInfo,
                                 NCT_SYSTEM_ROLE_SUPPER,
                                 NCT_SYSTEM_ROLE_SECURIT,
                                 ncTScanType,
                                 ncTScanScope,
                                 ncTVirusUpdateMethodType)
from ShareMgnt.ttypes import ncTShareMgntError

from enum import IntEnum
from eisoo.tclients import TClient
from thrift.Thrift import TException
import json
import time
import datetime
import os
import ftplib
import socket
import threading
import shutil
import requests
import uuid
import configparser
import urllib.request, urllib.parse, urllib.error
import hashlib

ftp_update_virus_db_thread = None          # FTP更新病毒库线程
TEST_LICENSE = "test"


def _get_update_virusdb_machine_id():
    """
    生成用于临时目录隔离的机器/Pod标识。
    逻辑对齐：
    mid := os.Getenv(conf.Service.PodID)
    if mid == "" { mid = os.Hostname(); mid = utils.MD5(mid)[:8] }
    """
    # 部署模板目前注入的是 POD_IP；同时兼容常见的 POD_ID / PODID。
    mid = os.getenv("POD_ID") or os.getenv("PODID") or os.getenv("POD_IP") or ""
    if mid == "":
        host = socket.gethostname() or ""
        mid = hashlib.md5(host.encode("utf-8")).hexdigest()[:8]
    return mid

class AntivirusTaskStatus(IntEnum):
    TS_NOT_START = 0                # 未开始
    TS_RUNNING = 1                  # 进行中
    TS_NO_VIRUS = 2                 # 已完成无毒
    TS_VIRUS = 3                    # 已完成有毒
    TS_FAIL = -1                    # 已完成出错


class AntivirusTaskType(IntEnum):
    UPLOAD_SCAN_TASK = 0            # 上传任务
    FULL_SCAN_TASK = 1              # 全盘任务

class ScanVirusManage(DBConnector):
    """
    病毒扫描管理
    """
    def __init__(self):
        """
        初始化
        """
        self.user_manage = UserManage()
        self.department_manage = DepartmentManage()
        self.config_manage = ConfigManage()
        self.role_manage = RoleManage()
        self.oem_manage = OEMManage()
        self.Cid = "a1c47132fbb04d40911ebe2eda1a624f" # 已经不再使用
        self.ossgateway_driven = OssgatewayDriven()

    def continue_scan_virus_task(self):
        """
        继续扫描任务
        """
        select_sql = f"""
        SELECT f_status FROM {get_db_name('ets')}.fullscan_antivirus_schedule
        """

        update_sql = f"""
        UPDATE `{get_db_name('ets')}`.`fullscan_antivirus_schedule`
        SET `f_status`=%s, `f_resume_time`=%s
        WHERE `f_status`=%s
        """

        res = self.r_db.one(select_sql)
        # 有任务暂停时才能继续
        if res['f_status'] != ncTScanTaskStatus.NCT_STOP:
            raise_exception(exp_msg=_("IDS_UPDATE_ANTIVIRUS_TASK_STATUS_ERROR") % str(res['f_status']),
                            exp_num=ncTShareMgntError.NCT_UPDATE_ANTIVIRUS_TASK_STATUS_ERROR)

        # 设置暂停并更新断点开始时间
        nowDate = int(BusinessDate.time() * 1000000)
        affect_row = self.w_db.query(update_sql, int(ncTScanTaskStatus.NCT_RUNNING), nowDate, int(ncTScanTaskStatus.NCT_STOP))
        if not affect_row:
            raise_exception(exp_msg=_("IDS_UPDATE_ANTIVIRUS_TASK_STATUS_ERROR") % str(res['f_status']),
                            exp_num=ncTShareMgntError.NCT_UPDATE_ANTIVIRUS_TASK_STATUS_ERROR)

    def stop_scan_virus_task(self):
        """
        暂停扫描任务
        """
        select_sql = f"""
        SELECT f_status, f_resume_time FROM {get_db_name('ets')}.fullscan_antivirus_schedule
        """

        update_sql = f"""
        UPDATE `{get_db_name('ets')}`.`fullscan_antivirus_schedule`
        SET `f_status`=%s, `f_used_time`=`f_used_time`+%s
        WHERE `f_status`=%s
        """

        res = self.r_db.one(select_sql)
        # 有任务在运行才能暂停
        if res['f_status'] != ncTScanTaskStatus.NCT_RUNNING:
            raise_exception(exp_msg=_("IDS_UPDATE_ANTIVIRUS_TASK_STATUS_ERROR") % str(res['f_status']),
                            exp_num=ncTShareMgntError.NCT_UPDATE_ANTIVIRUS_TASK_STATUS_ERROR)

        # 设置暂停并累加已使用时间
        useTime = int(BusinessDate.time() * 1000000) - res['f_resume_time']
        affect_row = self.w_db.query(update_sql, int(ncTScanTaskStatus.NCT_STOP), useTime, int(ncTScanTaskStatus.NCT_RUNNING))
        if not affect_row:
            raise_exception(exp_msg=_("IDS_UPDATE_ANTIVIRUS_TASK_STATUS_ERROR") % str(res['f_status']),
                            exp_num=ncTShareMgntError.NCT_UPDATE_ANTIVIRUS_TASK_STATUS_ERROR)

    def cancel_scan_virus_task(self):
        """
        取消扫描任务
        """
        select_sql = f"""
        SELECT f_status FROM {get_db_name('ets')}.fullscan_antivirus_schedule
        """

        update_sql_part1 = (
            f"UPDATE `{get_db_name('ets')}`.`fullscan_antivirus_schedule` "
        )
        update_sql_part2 = """
        SET `f_status`=%s, `f_end_time`=0, `f_start_time`=0, `f_resume_time`=0, `f_used_time`=0, `f_ver_progress_cnt`=0,  `f_progress_cid`='',`f_scan_cids`='{}', `f_scan_scope`='{}', `f_wait_scan_total`=0
        WHERE `f_status`=%s OR `f_status`=%s
        """
        update_sql = update_sql_part1 + update_sql_part2

        res = self.r_db.one(select_sql)
        # 有任务在处理才能取消
        if res['f_status'] != ncTScanTaskStatus.NCT_RUNNING and res['f_status'] != ncTScanTaskStatus.NCT_STOP:
            raise_exception(exp_msg=_("IDS_UPDATE_ANTIVIRUS_TASK_STATUS_ERROR") % str(res['f_status']),
                            exp_num=ncTShareMgntError.NCT_UPDATE_ANTIVIRUS_TASK_STATUS_ERROR)

        affect_row = self.w_db.query(update_sql, int(ncTScanTaskStatus.NCT_NOT_START), int(ncTScanTaskStatus.NCT_RUNNING), int(ncTScanTaskStatus.NCT_STOP))
        if not affect_row:
            raise_exception(exp_msg=_("IDS_UPDATE_ANTIVIRUS_TASK_STATUS_ERROR") % str(res['f_status']),
                exp_num=ncTShareMgntError.NCT_UPDATE_ANTIVIRUS_TASK_STATUS_ERROR)

    def delete_invalid_slave_site_virus_task(self, siteOSSIds):
        """
        删除移除主分站点关系后，主站点依旧保存的分站点杀毒任务
        """
        ossIds = ""
        for index in range(len(siteOSSIds)):
            if index != 0:
                ossIds += ","
            ossIds += "'" + siteOSSIds[index] + "'"
        if ossIds == "":
            return

        delete_sql = f"""
        DELETE FROM {get_db_name('ets')}.antivirus_task WHERE f_oss_id in (%s)
        """
        self.w_db.query(delete_sql %(ossIds))

    def get_scan_virus_task_result_cnt(self):
        """
        获取本次扫描染毒文件数
        """
        # 获取开始时间
        sql = f"""
        SELECT COUNT(*) AS cnt FROM `{get_db_name('ets')}`.`antivirus_task`
        WHERE `f_task_type` = %s AND `f_status` = %s
        """
        res = self.r_db.one(sql, int(AntivirusTaskType.FULL_SCAN_TASK), int(AntivirusTaskStatus.TS_VIRUS))
        return res['cnt']

    def get_virus_info_by_page(self, start, limit):
        """
        分页获取本次扫描染毒文件信息
        """
        limit_statement = check_start_limit(start, limit)

        sql = f"""
        SELECT `f_doc_id`, `f_start_time`, `f_end_time`, `f_msg`, `f_file_name`
        FROM `{get_db_name('ets')}`.`antivirus_task`
        WHERE `f_task_type` = %s AND `f_status` = %s ORDER BY `f_start_time` DESC
        {limit_statement}
        """
        results = self.r_db.all(sql, int(AntivirusTaskType.FULL_SCAN_TASK), int(AntivirusTaskStatus.TS_VIRUS))

        virusInfoList = []
        for res in results:
            virusInfo = ncTVirusInfo()

            msgJson = res['f_msg']
            msg = json.loads(msgJson)
            virusInfo.virusName = msg['virusname']
            virusInfo.riskType = msg['risktype']
            virusInfo.processType = msg['processtype']

            virusInfo.parentPath = res['f_file_name']
            virusInfo.startTime = res['f_start_time']
            virusInfo.endTime = res['f_end_time']

            fileName = virusInfo.parentPath.rsplit('/', 1)[1]
            virusInfo.fileName = fileName
            virusInfoList.append(virusInfo)
        return virusInfoList

    def get_scan_virus_task_result(self):  # 获取扫描结果（进度）
        """
        获取扫描结果
        """
        taskInfo = ncTScanTaskInfo()
        sql1 = f"""
        SELECT f_status, f_end_time, f_start_time, f_used_time, f_ver_progress_cnt,
        f_scan_cids, f_scan_scope, f_node_ip, f_node_id, f_wait_scan_total
        FROM {get_db_name('ets')}.fullscan_antivirus_schedule
        """
        res = self.r_db.one(sql1)
        taskInfo.status = res['f_status']
        taskInfo.endTime = res['f_end_time']
        taskInfo.startTime = res['f_start_time']
        taskInfo.useTime = res['f_used_time']
        taskInfo.scanFileCount = res['f_ver_progress_cnt']
        scanCidsJson = res['f_scan_cids']
        scanScopeJson = res['f_scan_scope']
        taskInfo.nodeIP = res['f_node_ip']
        taskInfo.nodeId = res['f_node_id']
        waitScanTotal =  res['f_wait_scan_total']
        # 初始扫描范围
        scanScope = ncTScanScope()
        scanScope.userIds = []
        scanScope.departIds = []
        scanScope.cids = []
        if scanScopeJson:
            scanScopeDict = json.loads(scanScopeJson)
            scanScope.userIds = scanScopeDict['userIds'] if 'userIds' in scanScopeDict else []
            scanScope.departIds = scanScopeDict['departIds'] if 'departIds' in scanScopeDict else []
            scanScope.cids = scanScopeDict['cids'] if 'cids' in scanScopeDict else []
        taskInfo.scanScope = scanScope
        # 由于单个任务持续时间很短，这里使用近似算法，直接获取已开始的最新任务
        sql2 = f"""
        SELECT `f_doc_id`, `f_file_name` FROM `{get_db_name('ets')}`.`antivirus_task`
        WHERE `f_task_type` = %s AND `f_status` = %s ORDER BY `f_start_time` ASC LIMIT 1
        """
        scanFile = self.r_db.one(sql2, int(AntivirusTaskType.FULL_SCAN_TASK),int(AntivirusTaskStatus.TS_RUNNING))
        if not scanFile:
            taskInfo.scanFilePath = ""
        else:
            scanFileName = scanFile['f_file_name']
            taskInfo.scanFilePath = scanFileName
        # 获取扫描进度，初始化阶段时为0
        if taskInfo.status == ncTScanTaskStatus.NCT_NOT_START or taskInfo.status == ncTScanTaskStatus.NCT_INITING:
            taskInfo.progressRate = 0

        if taskInfo.status == ncTScanTaskStatus.NCT_FINISH:
            taskInfo.progressRate = 1
        # 扫描类型
        cids = []
        taskInfo.scanType = ncTScanType.NCT_SCAN_TYPE_ALL
        if scanCidsJson:
            scanCids = json.loads(scanCidsJson)
            if "cids" in scanCids and len(scanCids["cids"]) != 0:
                taskInfo.scanType = ncTScanType.MCT_SCAN_TYPE_CUSTOM
                cids = scanCids["cids"]
        if taskInfo.status == ncTScanTaskStatus.NCT_RUNNING or taskInfo.status == ncTScanTaskStatus.NCT_STOP:
            # 待扫描文件数
            countAll = waitScanTotal
            taskProgress = taskInfo.scanFileCount
            # 任务完成数
            if taskProgress is None:
                countFinish = 0
            else:
                countFinish = taskProgress

            if countFinish < 0:
                countFinish = 0
                taskInfo.scanFileCount = countFinish

            if countAll == 0:
                taskInfo.progressRate = 0
            else:
                taskInfo.progressRate = round(float(countFinish) / countAll, 4)

            if taskInfo.progressRate > 1:
                taskInfo.progressRate = 1
        return taskInfo
    def __upload_file_to_storage(self, data, ossId):
        """
        函数功能: 将文件上传到对象存储中
        """
        uploadOSSId = ossId
        uid = str(uuid.uuid1())
        objectId = ''.join(uid.split('-'))

        code, uploadinfo = self.ossgateway_driven.get_upload_info(uploadOSSId, objectId, storage_prefix=True)
        if code != 200:
            raise_exception(exp_msg=f'get upload url failed: {code},{uploadinfo.get("message")},{uploadinfo.get("cause")}',
                            exp_num=ncTShareMgntError.NCT_UPLOAD_OSS_FAILED)
        rsp = requests.request(uploadinfo.get("method"),
                               uploadinfo.get("url"),
                               headers=uploadinfo.get("headers"),
                               data=data,
                               verify=False)

        if "[2" not in str(rsp):
            raise_exception(exp_msg=rsp.text,
                            exp_num=ncTShareMgntError.NCT_UPLOAD_OSS_FAILED)
        config = self.config_manage.get_custom_config_of_string("antivirus_config")
        configJson = json.loads(config)
        if "virusobjectid" in configJson:
            return objectId, configJson["virusobjectid"]
        else:
            return objectId, ""

    def __delete_file_from_storage(self, ossId, objectId):
        """
        函数功能: 删除对象存储中的文件
        """
        code, deleteinfo = self.ossgateway_driven.get_delete_info(ossId, objectId, storage_prefix=True)
        if code != 200:
            ShareMgnt_Log(f'delete old global virus db failed: {code},{deleteinfo.get("message")},{deleteinfo.get("cause")}')
            return None
        rsp = requests.request(deleteinfo.get("method"),
                               deleteinfo.get("url"),
                               headers=deleteinfo.get("headers"),
                               verify=False)
        if "[2" not in str(rsp):
            ShareMgnt_Log("delete old global virus db failed: %s", str(rsp))

    def set_global_virus_db(self, virusDB):
        """
        上传病毒库到对象存储
        """
        # 检查授权
        self.check_update_virusdb()

        # 文件名合法性检测
        check_filename(virusDB.virusDBName)

        uploadOSSId = ""
        ossIds = []
        #获取一个可用对象存储id
        ossIds = self.ossgateway_driven.get_local_storages()
        if not ossIds:
            raise_exception(exp_msg="no available oss",
                            exp_num=ncTShareMgntError.NCT_NO_AVAILABLE_OSS)
        else:
            for ossinfo in ossIds:
                if ossinfo.get("default"):
                    uploadOSSId = ossinfo.get("id")
            if not uploadOSSId:
                uploadOSSId = ossIds[0].get("id")

        newObjectId, oldObjectId = self.__upload_file_to_storage(virusDB.virusDBData, uploadOSSId)

        config = self.config_manage.get_custom_config_of_string("antivirus_config")
        configJson = json.loads(config)
        configJson["virusdbname"] = virusDB.virusDBName
        configJson["updatetime"] = int(BusinessDate.time() * 1000000)
        configJson["virusobjectid"] = newObjectId
        configJson["ossid"] = uploadOSSId
        configJson["cid"] = self.Cid
        config = json.dumps(configJson)
        self.config_manage.set_custom_config_of_string("antivirus_config", config)

        if len(oldObjectId)!=0:
            self.__delete_file_from_storage(uploadOSSId, oldObjectId)


    def get_virus_db_download_url(self):
        """
        获取下载病毒库url
        """
        config = self.config_manage.get_custom_config_of_string("antivirus_config")
        virusDownloadUrl = ""
        configJson = json.loads(config)
        virusObjectId = configJson['virusobjectid'] if "virusobjectid" in configJson else ""

        if virusObjectId:
            ossId = configJson['ossid']
            code, downloadinfo = self.ossgateway_driven.get_download_info(ossId, virusObjectId,expires_time= 60 * 60 * 24, save_name='malware.rmd', storage_prefix=True)
            if code != 200:
                ShareMgnt_Log(f'get download url failed: {code},{downloadinfo.get("message")},{downloadinfo.get("cause")}')
                raise_exception(exp_msg=_("GET_URL_FAILED"),
                                exp_num=ncTShareMgntError.NCT_GET_URL_FAILED)
            if not downloadinfo.get("url"):
                raise_exception(exp_msg=_("GET_URL_FAILED"),
                                exp_num=ncTShareMgntError.NCT_GET_URL_FAILED)
            else:
                virusDownloadUrl = downloadinfo.get("url")

        return virusDownloadUrl

    def set_virus_db_download_url(self, virusDBDownloadUrl):
        """
        设置下载病毒库url
        """
        config = None
        config = self.config_manage.get_custom_config_of_string("antivirus_config")
        config = json.loads(config)
        config['virusDownloadUrl'] = virusDBDownloadUrl
        config = json.dumps(config)
        self.config_manage.set_custom_config_of_string("antivirus_config", config)
        return

    def get_virus_db(self):
        """
        获取病毒库信息
        """
        virusDB = ncTVirusDBInfo()
        config = self.config_manage.get_custom_config_of_string("antivirus_config")
        configJson = json.loads(config)
        virusDB.virusDBName = configJson["virusdbname"] if "virusdbname" in configJson else ""
        virusDB.updateTime = configJson["updatetime"] if "updatetime" in configJson else 0
        return virusDB

    def check_enable_antivirus(self, raise_ex=True):
        """
        开启杀毒功能时检查是否被授权过
        """
        # 有一个授权激活或者过期则通过
        return True

        if raise_ex:
            raise_exception(exp_msg=_("IDS_ENABLE_ANTIVIRUS_FAIL"),
                            exp_num=ncTShareMgntError.NCT_ENABLE_ANTIVIRUS_FAIL)
        else:
            return False

    def check_update_virusdb(self):
        """
        更新病毒库时检查授权是否可用
        """
        # 有一个授权激活则通过
        return

        raise_exception(exp_msg=_("IDS_ANTIVIRUS_OPTION_LICENSE_EXPIRE"),
                        exp_num=ncTShareMgntError.NCT_ANTIVIRUS_OPTION_LICENSE_EXPIRE)

    def notify_scan_finish(self):
        """
        杀毒完成发送邮件通知超级管理员或者安全管理员
        """
        # 开启了三权分立发送给安全管理员
        toEmailList = []
        if self.user_manage.get_trisystem_status():
            toEmailList = self.role_manage.get_role_mails(NCT_SYSTEM_ROLE_SECURIT)

        # 没有开启三权分立发送给超级管理员
        else:
            toEmailList = self.role_manage.get_role_mails(NCT_SYSTEM_ROLE_SUPPER)

        if len(toEmailList):
            product_name = self.oem_manage.get_config_by_option('shareweb_en-us', 'product')
            subject = _("IDS_ANTIVIRUS_SCAN_TASK_FINISH_EMAIL_SUBJECT") % (product_name)
            content = _("IDS_ANTIVIRUS_SCAN_TASK_FINISH_EMAIL_CONTENT") % (product_name)
            email_send_html_content(toEmailList, subject, content)

    def virus_ftp_server_test(self):
        """
        开启自动更新病毒库时测试ftp是否正常
        """
        config = self.config_manage.get_custom_config_of_string("antivirus_config")
        antivirusConfig = json.loads(config)
        ftpAddr = antivirusConfig["ftpaddr"] if "ftpaddr" in antivirusConfig else ""
        port = int(antivirusConfig["port"]) if "port" in antivirusConfig else 21
        user = antivirusConfig["user"] if "user" in antivirusConfig else ""
        password = antivirusConfig["password"] if "password" in antivirusConfig else ""
        try:
            ftp = ftplib.FTP()
            ftp.connect(ftpAddr, port)
            ftp.login(user, password)
            return True
        except socket.gaierror as e:
            # ftp地址异常
            raise_exception(exp_msg=_("IDS_ANTIVIRUS_FTP_NOT_AVAILABLE"),
                            exp_num=ncTShareMgntError.NCT_ANTIVIRUS_FTP_NOT_AVAILABLE)
        except ftplib.error_perm as e:
            # ftp账号或密码不正确
            raise_exception(exp_msg=_("IDS_ANTIVIRUS_FTP_LOGIN_FAILED"),
                            exp_num=ncTShareMgntError.NCT_ANTIVIRUS_FTP_LOGIN_FAILED)
        except socket.error as e:
            # 网络异常，连接超时
            raise_exception(exp_msg=_("IDS_ANTIVIRUS_FTP_NETWORK_ERROR"),
                            exp_num=ncTShareMgntError.NCT_ANTIVIRUS_FTP_NETWORK_ERROR)


    def set_update_virusdb_thread_running(self, enable_update_virus_db):
        """
        FTP方式更新病毒库或多站点更新病毒库
        """
        global ftp_update_virus_db_thread

        # 杀毒开关开启后判断采用FTP更新还是多站点病毒库更新
        if enable_update_virus_db == ncTVirusUpdateMethodType.FTP_UPDATE:
            if ftp_update_virus_db_thread:
                ftp_update_virus_db_thread.run_immediately()
            else:
                ftp_update_virus_db_thread = UpdateVirusDBThread()
                ftp_update_virus_db_thread.daemon = True
                ftp_update_virus_db_thread.start()

class UpdateVirusDBThread(threading.Thread):
    """
    自动更新病毒库线程
    """
    TERMINATE = False
    WAIT_TIME = 24 * 3600
    _MID = _get_update_virusdb_machine_id()
    TMPLOCALPATH = os.path.join("/sysvol/cache/antivirus/rising", f"tmplocalpath_{_MID}")
    FILELIST = []

    def __init__(self):
        """
        初始化
        """
        super(UpdateVirusDBThread, self).__init__()
        self.scan_virus_manage = ScanVirusManage()
        self.evt = threading.Event()

    def update_virus_db(self):
        """
        更新病毒库操作
        """
        updateVirusConfig = self.scan_virus_manage.config_manage.get_custom_config_of_string("antivirus_config")
        antivirusConfig = json.loads(updateVirusConfig)
        ftpAddr = antivirusConfig["ftpaddr"] if "ftpaddr" in antivirusConfig else "rspub.rising.com.cn"
        port = int(antivirusConfig["port"]
                   ) if "port" in antivirusConfig else 21
        user = antivirusConfig["user"] if "user" in antivirusConfig else "eisoo-cloud"
        password = antivirusConfig["password"] if "password" in antivirusConfig else "2Z4LxA1D44kWMuW"
        self.WAIT_TIME = int(
            antivirusConfig["waittime"]) if "waittime" in antivirusConfig else self.WAIT_TIME
        smtpMessage = ""
        try:
            ftp = ftplib.FTP()
            ftp.connect(ftpAddr, port)
            ftp.login(user, password)
            subfiles = ftp.cwd("/vlcoful")
            ftp.retrlines("LIST", self.__first_filter)
        except socket.gaierror as e:
            # ftp地址不正确
            smtpMessage = "IDS_ANTIVIRUS_FTP_NOT_AVAILABLE"
        except ftplib.error_perm as e:
            # ftp账号或密码不正确
            smtpMessage = "IDS_ANTIVIRUS_FTP_LOGIN_FAILED"
        except socket.error as e:
            # 网络异常，连接超时
            smtpMessage = "IDS_ANTIVIRUS_FTP_NETWORK_ERROR"
        if smtpMessage:
            self.sendErrorToManager(smtpMessage)
            ShareMgnt_Log(_(smtpMessage))
            return

        # 获取最新病毒库文件信息
        refileInfo = self.__get_newest_virus_db(self.FILELIST)
        fileName = refileInfo[1]
        virus_update_time = refileInfo[0]
        refileName = os.path.splitext(fileName)[0] + ".rmd"
        # 获取原病毒库信息
        virusDB = self.scan_virus_manage.get_virus_db()
        virusDBName = virusDB.virusDBName if virusDB.virusDBName else ""

        # 当前为最新病毒库返回(授权变更【有效期为7年，2019/08/01-2027/08/01】，病毒库获取时取消授权文件时间判断)
        if refileName == virusDBName:
            return

        # ftp上下载病毒库到本地
        if not os.path.exists(self.TMPLOCALPATH):
            os.makedirs(self.TMPLOCALPATH)
        localFilePath = os.path.join(self.TMPLOCALPATH, refileName)

        file_handler = open(localFilePath, 'wb')
        ftp.retrbinary("RETR %s" % (fileName), file_handler.write)
        file_handler.close()
        virusDB = ncTVirusDBInfo()
        with open(localFilePath, 'rb') as file_object:
            all_the_text = file_object.read()
        virusDB.virusDBData = all_the_text
        virusDB.virusDBName = refileName
        virusDB.updateTime = virus_update_time
        try:
            if self.__check_virus_db_info(ftp, localFilePath, fileName):
                self.scan_virus_manage.set_global_virus_db(virusDB)
        except Exception as e:
            self.sendErrorToManager(str(e.expMsg))
            ShareMgnt_Log("set global virus db: %s", str(e.expMsg))
        finally:
            # 删除临时目录
            if os.path.exists(self.TMPLOCALPATH):
                shutil.rmtree(self.TMPLOCALPATH)

    def __first_filter(self, line):
        """
        返回ftp中病毒库列表信息
        """
        fileInfo = line.split()
        self.FILELIST.append(fileInfo)
        return self.FILELIST

    def __get_newest_virus_db(self, fileInfos):
        '''
        获取最新病毒库文件
        '''
        reFileInfoList = []
        cur_year = BusinessDate.now().strftime("%Y")

        for fileInfo in fileInfos:
            tmpfileInfo = []
            if fileInfo[7].isdigit() or not fileInfo[8].endswith("bas"):
                continue
            # 获取病毒库文件的上传时间戳
            filetime = str(cur_year)+"-" + \
                fileInfo[5]+"-"+fileInfo[6]+' '+fileInfo[7]
            format_time = datetime.datetime.strptime(
                filetime, "%Y-%b-%d %H:%M")
            format_time = format_time.strftime("%Y%m%d%H%M")
            format_time = time.strptime(format_time, "%Y%m%d%H%M")
            format_time = time.mktime(format_time)

            tmpfileInfo.append(format_time)
            tmpfileInfo.append(fileInfo[8])
            reFileInfoList.append(tmpfileInfo)
        fileInfoList = sorted(reFileInfoList, key=lambda x: x[1], reverse=True)
        return fileInfoList[0]

    def __check_virus_db_info(self, ftp, localFilePath, virusDBFileName):
        """
        检查病毒库大小，比较MD5信息是否一致
        """
        virus_db_size = os.path.getsize(localFilePath)
        if virus_db_size < 50 * 1024 * 1024:
            raise_exception("virus db size error")
        md5_hash = hashlib.md5()
        with open(localFilePath, 'rb') as file:
            for chunk in iter(lambda: file.read(4096), b""):
                md5_hash.update(chunk)
        md5Str = md5_hash.hexdigest()
        listFilePath = os.path.join(self.TMPLOCALPATH, "md5.list")
        file_handler = open(listFilePath, 'wb')
        ftp.retrbinary("RETR %s" % ("md5.list"), file_handler.write)
        file_handler.close()
        with open(listFilePath, 'r') as file:
                for line in file:
                    key, value = line.strip().split()
                    if key == virusDBFileName and value == md5Str:
                        return True
        raise_exception("virus db hash not found")

    def sendErrorToManager(self, content):
        """
        发送错误信息邮件通知管理员
        """
        toEmailList = []
        if self.scan_virus_manage.user_manage.get_trisystem_status():
            toEmailList = self.scan_virus_manage.role_manage.get_role_mails(
                NCT_SYSTEM_ROLE_SECURIT)

        # 没有开启三权分立发送给超级管理员
        else:
            toEmailList = self.scan_virus_manage.role_manage.get_role_mails(
                NCT_SYSTEM_ROLE_SUPPER)

        if len(toEmailList):
            product_name = self.scan_virus_manage.oem_manage.get_config_by_option(
                'shareweb_en-us', 'product')
            subject = _("IDS_ANTIVIRUS_SCAN_TASK_FINISH_EMAIL_SUBJECT") % (
                product_name)
            email_send_html_content(toEmailList, subject, _(
                "IDS_ANTIVIRUS_ERROR_EMAIL_CONTENT") % (_(content), product_name))

    def run_immediately(self):
        self.TERMINATE = False
        if not self.evt.isSet():
            self.evt.set()

    def close(self):
        """
        关闭
        """
        self.TERMINATE = True

    def run(self):
        """
        执行
        """
        ShareMgnt_Log(
            "**************** update virusDB thread start *****************")

        while True:
            try:
                if not self.TERMINATE:
                    self.update_virus_db()
                    self.evt.clear()
                    self.evt.wait(self.WAIT_TIME)
                    self.evt.set()
                else:
                    self.evt.clear()
                    self.evt.wait(self.WAIT_TIME)
            except Exception as e:
                ShareMgnt_Log("FTP update virusDB thread run error: %s", str(e))
                self.evt.wait(self.WAIT_TIME)

        ShareMgnt_Log("****************FTP update virusDB delete thread end *****************")
