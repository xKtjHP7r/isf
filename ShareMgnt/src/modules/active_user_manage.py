#!/usr/bin/python3
# -*- coding:utf-8 -*-
"""
活跃用户统计
"""
import os
import csv
import math
import time
import uuid
import shutil
import calendar
import datetime
import threading
import traceback
from hashlib import md5
from eisoo.tclients import TClient
from src.common.db.connector import DBConnector
from src.common.db.db_manager import get_db_name
from src.common.lib import (escape_format_percent, raise_exception)
from src.common.nc_senders import email_send_html_content
from src.common.sharemgnt_logger import ShareMgnt_Log
from src.common.business_date import BusinessDate
from src.common.global_info import IS_SINGLE
from src.modules.config_manage import ConfigManage
from src.modules.smtp_manage import MailRecipient
from src.modules.user_manage import UserManage
from ShareMgnt.constants import (ncTReportInfo, NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT)
from ShareMgnt.ttypes import (ncTActiveReportInfo,
                              ncTActiveUserInfo,
                              ncTShareMgntError)

from src.modules.distributed_lock import acquire_lock

_file_path = '/tmp/sysvol/cache/sharemgnt/activeuser/'
task_dict = {}
threadLock = threading.Lock()
WAIT_TIME_ONE_HOUR = 3600
CPU_THRESHOLD = 0.8
IS_SWIFT = False

class ProblemType:
    """
    异常类型
    """
    SERVICE_PROBLEM = 1
    DATABASE_PROBLEM = 2
    CPU_PROBLEM = 3
    SYSVOL_PROBLEM = 4
    REPLICAS_PROBLEM = 5
    BALANCE_PROBLEM = 6

class TaskFinishedStatus:
    """
    任务完成状态
    """
    TASK_ERROR = -1
    TASK_IN_PROCESS = 0
    TASK_FINISHED = 1

class TaskType:
    """
    任务类型
    """
    TASK_MONTH = 1
    TASK_YEAR = 2


class TaskInfo:
    """
    @ todo: 活跃报表生成任务结构体
    @ create_time: 任务创建时间
    @ file_path: 生成文件路径
    @ finished_status: 任务处理状态
    @ name: 文件名
    """
    def __init__(self, create_time=BusinessDate.time(), file_path="", finished_status=0,
                 name="", inquire_date="", task_type=TaskType.TASK_MONTH):
        self.create_time = create_time
        self.file_path = file_path
        self.finished_status = finished_status
        self.name = name
        self.inquire_date = inquire_date
        self.task_type = task_type
        self.lock = threading.Lock()

    def get_finished_status(self):
        with self.lock:
            return self.finished_status

    def set_finished_status(self, status):
        with self.lock:
            self.finished_status = status


class ActiveUserManage(DBConnector):
    """
    """
    def __init__(self):
        self.config_manage = ConfigManage()
        self.user_manage = UserManage()
        self.mail_recipient = MailRecipient("eisoo_recipient_config")

    def __check_file_name(self, name):
        """
        检查文件名
        """
        if name and name.endswith(".csv"):
            return

        raise_exception(exp_msg=_("IDS_INVALID_FILE_NAME"),
                        exp_num=ncTShareMgntError.NCT_INVALID_FILE_NAME)

    def add_gen_file_task(self, taskInfo):
        """
        添加任务
        返回任务 id
        """
        global task_dict
        taskId = str(uuid.uuid1())
        with threadLock:
            task_dict[taskId] = taskInfo
        return taskId

    def get_gen_task_info(self, taskId):
        """
        获取任务信息
        """
        global task_dict

        with threadLock:
            # 任务不存在
            if taskId not in task_dict:
                raise_exception(exp_msg=_("IDS_DOWNLOAD_ACTIVE_REPORT_NOT_EXIST"),
                                exp_num=ncTShareMgntError.NCT_DOWNLOAD_ACTIVE_REPORT_NOT_EXIST)
            else:
                return task_dict[taskId]

    def get_gen_active_report_status(self, taskId):
        """
        获取生成活跃报表文件状态
        """
        global IS_SINGLE
        if not IS_SINGLE:
            with TClient('ShareMgntSingle') as client:
                return client.GetGenActiveReportStatus(taskId)

        taskInfo = self.get_gen_task_info(taskId)
        finished_status = taskInfo.get_finished_status()

        if taskInfo.get_finished_status() == TaskFinishedStatus.TASK_IN_PROCESS:
            return False

        return True

    def del_gen_file_task(self, taskId):
        """
        删除任务
        """
        global task_dict

        with threadLock:
            # 任务不存在
            if taskId not in task_dict:
                raise_exception(exp_msg=_("IDS_DOWNLOAD_ACTIVE_REPORT_NOT_EXIST"),
                                exp_num=ncTShareMgntError.NCT_DOWNLOAD_ACTIVE_REPORT_NOT_EXIST)
            else:
                del(task_dict[taskId])

    def get_active_report_file_info(self, taskId):
        """
        获取生成的活跃报表文件地址
        """
        global IS_SINGLE
        if not IS_SINGLE:
            with TClient('ShareMgntSingle') as client:
                return client.GetActiveReportFileInfo(taskId)

        taskInfo = self.get_gen_task_info(taskId)
        finished_status = taskInfo.get_finished_status()

        if finished_status == TaskFinishedStatus.TASK_FINISHED:
            reportInfo = ncTReportInfo()
            with open(taskInfo.file_path, 'rb') as fd:
                reportInfo.reportData = fd.read()
            reportInfo.reportName = taskInfo.name
            self.del_gen_file_task(taskId)
            return reportInfo

        if finished_status == TaskFinishedStatus.TASK_IN_PROCESS:
            raise_exception(exp_msg=_("IDS_DOWNLOAD_ACTIVE_REPORT_IN_PROGRESS"),
                            exp_num=ncTShareMgntError.NCT_DOWNLOAD_ACTIVE_REPORT_IN_PROGRESS)

        if finished_status == TaskFinishedStatus.TASK_ERROR:
            self.del_gen_file_task(taskId)
            raise_exception(exp_msg=_("IDS_DOWNLOAD_ACTIVE_REPORT_FAILED"),
                            exp_num=ncTShareMgntError.NCT_DOWNLOAD_ACTIVE_REPORT_FAILED)

    def set_active_report_notify_status(self, status):
        """
        设置活跃报表通知开关状态
        """
        status = 1 if status else 0
        self.config_manage.set_config("enable_active_report_notify", status)

    def get_active_report_notify_status(self):
        """
        获取活跃报表通知开关状态
        """
        return bool(int(self.config_manage.get_config("enable_active_report_notify")))

    def set_eisoo_recipient_email(self, emailList):
        """
        设置通知到爱数的邮件接收地址
        """
        self.mail_recipient.set_config(emailList)

    def get_eisoo_recipient_email(self):
        """
        获取通知到爱数的邮件接收地址
        """
        return self.mail_recipient.get_config()

    def get_total_count_month(self, inquireDate):
        """
        获取月用户总数、激活用户数
        """
        strDate = "%s-%s" % (inquireDate.year, str(inquireDate.month).zfill(2))
        sql = """
        SELECT `f_total_count`, `f_activate_count`
        FROM `t_active_user_month`
        WHERE `f_time` = %s
        """
        result = self.r_db.one(sql, strDate)
        user_count, activate_count = 0, 0
        if result:
            user_count = int(result["f_total_count"])
            activate_count = int(result["f_activate_count"])

        cur_time = BusinessDate.now()
        if inquireDate.month == cur_time.month and inquireDate.year == cur_time.year:
            user_count += self.user_manage.get_all_user_count()
            activate_count += self.get_activate_user_count()
        return user_count, activate_count

    def get_total_count_year(self, inquireDate):
        """
        获取年用户总数、激活用户数
        """
        sql = """
        SELECT `f_total_count`, `f_activate_count`
        FROM `t_active_user_year`
        WHERE `f_time` = %s
        """
        result = self.r_db.one(sql, inquireDate.year)
        user_count, activate_count = 0, 0
        if result:
            user_count = int(result["f_total_count"])
            activate_count = int(result["f_activate_count"])

        cur_time = BusinessDate.now()
        if inquireDate.year == cur_time.year:
            user_count += self.user_manage.get_all_user_count()
            activate_count += self.get_activate_user_count()
        return user_count, activate_count

    def export_active_report_month(self, name, inquireDate):
        """
        创建月度活跃报表导出任务
        """
        global IS_SINGLE
        if not IS_SINGLE:
            with TClient('ShareMgntSingle') as client:
                return client.ExportActiveReportMonth(name, inquireDate)

        # 检查文件名
        self.__check_file_name(name)

        # 检查时间格式
        try:
            datetime.datetime.strptime(inquireDate, "%Y-%m")
        except Exception:
            raise_exception(_("date time illegal"))

        # 添加任务
        taskInfo = TaskInfo()
        taskInfo.name = name
        taskInfo.inquire_date = inquireDate
        taskInfo.task_type = TaskType.TASK_MONTH

        # 创建生成任务
        taskId = self.add_gen_file_task(taskInfo)

        # 启动任务处理线程
        gen_file_thread = GenFileThread(taskId, taskInfo)
        gen_file_thread.daemon = True
        gen_file_thread.start()

        return taskId

    def export_active_report_year(self, name, inquireDate):
        """
        创建年度活跃报表导出任务
        """
        global IS_SINGLE
        if not IS_SINGLE:
            with TClient('ShareMgntSingle') as client:
                return client.ExportActiveReportYear(name, inquireDate)

        # 检查文件名
        self.__check_file_name(name)

        # 检查时间格式
        try:
            datetime.datetime.strptime(inquireDate, "%Y")
        except Exception:
            raise_exception(_("date time illegal"))

        # 添加任务
        taskInfo = TaskInfo()
        taskInfo.name = name
        taskInfo.inquire_date = inquireDate
        taskInfo.task_type = TaskType.TASK_YEAR

        # 创建生成任务
        taskId = self.add_gen_file_task(taskInfo)

        # 启动任务处理线程
        gen_file_thread = GenFileThread(taskId, taskInfo)
        gen_file_thread.daemon = True
        gen_file_thread.start()

        return taskId

    def make_store_file_dir(self, taskId):
        """
        新建保存日志文件的目录
        """
        fileDir = os.path.join(_file_path, taskId)

        # 新建存放文件的目录 /sysvol/cache/sharemgnt/activeuser/taskId/
        if not os.path.exists(fileDir):
            os.makedirs(fileDir)

        return fileDir

    def delete_history_active_user(self, date_month):
        """
        统计上月活跃用户之后清空上月数据
        """
        str_date = "%s-%%" % str(date_month)
        sql = f"""
        DELETE FROM `{get_db_name("anyshare")}`.`t_active_user_info`
        WHERE `f_time` like %s
        """
        self.w_db.query(sql, str_date)

    def get_db_active_user_day(self, inquireDate):
        """
        获取当月每天的活跃数信息
        """
        year = inquireDate.year
        month = inquireDate.month
        days = calendar.monthrange(year, month)[1]
        month = str(month).zfill(2)

        startTime = "%s-%s-01" % (year, month)
        endTime = "%s-%s-%s" % (year, month, days)

        sql = """
        SELECT `f_active_count`, `f_activate_count`, `f_time`
        FROM `t_active_user_day`
        WHERE `f_time` >= %s and `f_time` <= %s
        ORDER BY `f_time` ASC
        """
        results = self.r_db.all(sql, startTime, endTime)
        return results if results else []

    def get_db_active_user_month(self, inquireDate):
        """
        获取年中每月的活跃数信息
        """
        year = inquireDate.year
        startTime = "%s-01" % year
        endTime = "%s-12" % year

        sql = """
        SELECT `f_active_count`, `f_activate_count`, `f_time`
        FROM `t_active_user_month`
        WHERE `f_time` >= %s and `f_time` <= %s
        ORDER BY `f_time` ASC
        """
        results = self.r_db.all(sql, startTime, endTime)
        return results if results else []

    def get_active_report_info(self, inquireDate, taskType):
        """
        根据月份或年份获取活跃报表信息
        """
        try:
            if taskType == TaskType.TASK_MONTH:
                inquireDate = datetime.datetime.strptime(inquireDate, "%Y-%m")
                db_active_users = self.get_db_active_user_day(inquireDate)
            else:
                inquireDate = datetime.datetime.strptime(inquireDate, "%Y")
                db_active_users = self.get_db_active_user_month(inquireDate)
        except Exception:
            raise_exception(_("date time illegal"))

        reportInfo = ncTActiveReportInfo()
        reportInfo.avgCount = 0
        reportInfo.avgActivity = 0.0
        reportInfo.userInfos = []

        maxActiveCnt, minActiveCnt = -1, float("inf")
        maxActivity, minActivity = 0.0, float("inf")
        minActCntDate = "%s-%s-01" % (inquireDate.year, str(inquireDate.month).zfill(2))
        maxActCntDate = "%s-%s-01" % (inquireDate.year, str(inquireDate.month).zfill(2))

        for user in db_active_users:
            activeUser = ncTActiveUserInfo()
            activeUser.userActivity = 0.0
            activeUser.time = user["f_time"]
            activeUser.activeCount = user["f_active_count"]

            if activeUser.activeCount > maxActiveCnt:
                maxActiveCnt = activeUser.activeCount
                maxActCntDate = activeUser.time

            if activeUser.activeCount < minActiveCnt:
                minActiveCnt = activeUser.activeCount
                minActCntDate = activeUser.time

            activateCount = int(user["f_activate_count"])
            if activateCount:
                activeUser.userActivity = round(float(activeUser.activeCount) / activateCount, 4)
                if activeUser.userActivity > 1.0:
                    activeUser.userActivity = 1.0

                if activeUser.userActivity > maxActivity:
                    maxActivity = activeUser.userActivity

                if activeUser.userActivity < minActivity:
                    minActivity = activeUser.userActivity

            reportInfo.avgCount += activeUser.activeCount
            reportInfo.avgActivity += activeUser.userActivity
            reportInfo.userInfos.append(activeUser)

        totalCount = 0
        totalActivateCount = 0
        if db_active_users:
            if taskType == TaskType.TASK_MONTH:
                totalCount, totalActivateCount = self.get_total_count_month(inquireDate)
            else:
                totalCount, totalActivateCount = self.get_total_count_year(inquireDate)

            reportInfo.avgActivity = round(float(reportInfo.avgActivity) / len(db_active_users), 4)
            reportInfo.avgCount = math.ceil(float(reportInfo.avgCount) / len(db_active_users))

        if maxActiveCnt == -1:
            maxActiveCnt = 0
        if minActiveCnt == float("inf"):
            minActiveCnt = 0
        if minActivity == float("inf"):
            minActivity = 0.0

        return reportInfo, maxActiveCnt, minActiveCnt, maxActivity, minActivity, \
            totalCount, totalActivateCount, maxActCntDate, minActCntDate

    def get_active_report_month(self, inquireDate):
        """
        获取月度活跃报表信息
        """
        reportInfo = self.get_active_report_info(inquireDate, TaskType.TASK_MONTH)
        return reportInfo[0]

    def get_active_report_year(self, inquireDate):
        """
        获取年度活跃报表信息
        """
        reportInfo = self.get_active_report_info(inquireDate, TaskType.TASK_YEAR)
        return reportInfo[0]

    def gen_active_report_file(self, taskId, taskInfo):
        """
        生成活跃报表文件
        """
        # 新建文件目录
        fileDir = self.make_store_file_dir(taskId)

        reportInfos = self.get_active_report_info(taskInfo.inquire_date, taskInfo.task_type)
        reportInfo = reportInfos[0]
        maxActiveCnt = reportInfos[1]
        minActiveCnt = reportInfos[2]
        maxActivity = reportInfos[3]
        minActivity = reportInfos[4]
        totalCount = reportInfos[5]
        activateCount = reportInfos[6]

        # 生成文件
        csvFileName = os.path.join(fileDir, taskInfo.name)
        with open(csvFileName, "w") as fd:
            # Excel BOM头
            fd.write(bytes.decode(b'\xef\xbb\xbf'))
            csv_write = csv.writer(fd)
            if taskInfo.task_type == TaskType.TASK_MONTH:
                csv_write.writerow([_("IDS_MONTHLY_OVERALL_INDEX")])
            else:
                csv_write.writerow([_("IDS_YEARLY_OVERALL_INDEX")])
            csv_write.writerow([_("IDS_AUM_INDEX"), _("IDS_AUM_VALUE")])
            csv_write.writerow([_("IDS_AUM_TOTAL_USER_COUNT"), totalCount])
            csv_write.writerow([_("IDS_AUM_ACTIVATE_COUNT"), activateCount])
            csv_write.writerow([_("IDS_AUM_AVERAGE_ACTIVE_USER"), reportInfo.avgCount])
            csv_write.writerow([_("IDS_AUM_AVERAGE_ACTIVE_DEGREE"), reportInfo.avgActivity])
            csv_write.writerow([_("IDS_AUM_LOWEST_ACTIVE_USER"), minActiveCnt])
            csv_write.writerow([_("IDS_AUM_LOWEST_ACTIVE_DEGREE"), minActivity])
            csv_write.writerow([_("IDS_AUM_HIGHEST_ACTIVE_USER"), maxActiveCnt])
            csv_write.writerow([_("IDS_AUM_HIGHEST_ACTIVE_DEGREE"), maxActivity])
            csv_write.writerow([])

            if taskInfo.task_type == TaskType.TASK_MONTH:
                csv_write.writerow([_("IDS_MONTHLY_DETAILED_INDEX")])
                csv_write.writerow([_("IDS_AUM_DATE"), _("IDS_AUM_DAILY_ACTIVE_USER"), _("IDS_AUM_DAILY_ACTIVE_DEGREE")])
            else:
                csv_write.writerow([_("IDS_YEARLY_DETAILED_INDEX")])
                csv_write.writerow([_("IDS_AUM_MONTH"), _("IDS_AUM_MONTHLY_ACTIVE_USER"), _("IDS_AUM_MONTHLY_ACTIVE_DEGREE")])
            for user in reportInfo.userInfos:
                csv_write.writerow([user.time.replace("-", "/"), user.activeCount, user.userActivity])

        # 更新任务状态
        taskInfo.set_finished_status(TaskFinishedStatus.TASK_FINISHED)

        return csvFileName

    def gen_file_handler(self, taskId, taskInfo):
        """
        生成活跃报表文件
        """
        if taskId is None or taskInfo is None:
            return

        # 生成文件
        return self.gen_active_report_file(taskId, taskInfo)

    def get_activate_user_count(self):
        """
        获取激活用户数
        """
        sql = """
        SELECT COUNT(f_user_id) AS cnt FROM `t_user`
        WHERE `f_activate_status` = %s
        AND `f_user_id` != %s
        AND `f_user_id` != %s
        AND `f_user_id` != %s
        AND `f_user_id` != %s
        """
        result = self.r_db.one(sql, 1, NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT)
        return int(result["cnt"]) if result else 0

    def update_active_user_day(self):
        """
        更新日活跃用户数
        """
        # 获取上一天的日期
        today = BusinessDate.today()
        yesterday = today - datetime.timedelta(days=1)
        strDate = "%s-%s-%s" % (yesterday.year, str(yesterday.month).zfill(2), str(yesterday.day).zfill(2))

        # 获取日活跃用户数
        sql = f"""
        SELECT COUNT(DISTINCT(`active_user`.`f_user_id`)) AS `active_count`
        FROM `{get_db_name("anyshare")}`.`t_active_user_info` AS `active_user`
        WHERE `active_user`.`f_time` = %s
        """
        result = self.r_db.one(sql, strDate)
        active_count = result["active_count"] if result else 0

        # 获取日激活用户数
        activate_count = self.get_activate_user_count()

        sql = """
        SELECT `f_activate_count` FROM `t_active_user_day`
        WHERE `f_time` = %s
        """
        result = self.r_db.one(sql, strDate)
        if result:
            activate_count += result["f_activate_count"]
            sql = """
            UPDATE `t_active_user_day` SET `f_active_count` = %s, `f_activate_count` = %s
            WHERE `f_time` = %s
            """
            self.w_db.query(sql, active_count, activate_count, strDate)
        else:
            sql = """
            INSERT INTO `t_active_user_day` (`f_active_count`, `f_activate_count`, `f_time`)
            VALUES(%s, %s, %s)
            """
            self.w_db.query(sql, active_count, activate_count, strDate)

    def update_active_user_month(self):
        """
        更新月活跃用户数
        """
        # 获取上一天的日期
        today = BusinessDate.today()
        yesterday = today - datetime.timedelta(days=1)
        startDate = "%s-%s-01" % (yesterday.year, str(yesterday.month).zfill(2))
        endDate = "%s-%s-%s" % (yesterday.year, str(yesterday.month).zfill(2), str(yesterday.day).zfill(2))

        # 获取月活跃用户数
        sql = f"""
        SELECT `active_user`.`f_user_id` AS `active_count`
        FROM `{get_db_name("anyshare")}`.`t_active_user_info` AS `active_user`
        WHERE `active_user`.`f_time` >= %s and `active_user`.`f_time` <= %s
        GROUP BY `active_user`.`f_user_id`
        HAVING COUNT(`active_user`.`f_user_id`) >= 4
        """
        results = self.r_db.all(sql, startDate, endDate)
        active_count = len(results)

        # 获取月用户总数
        sql = """
        SELECT `f_total_count`, `f_activate_count`
        FROM `t_active_user_month`
        WHERE `f_time` = %s
        """
        strDate = "%s-%s" % (yesterday.year, str(yesterday.month).zfill(2))
        result = self.r_db.one(sql, strDate)
        user_count = self.user_manage.get_all_user_count()

        # 获取月激活用户数
        activate_count = self.get_activate_user_count()

        if result:
            user_count += result["f_total_count"]
            activate_count += result["f_activate_count"]
            sql = """
            UPDATE `t_active_user_month`
            SET `f_active_count` = %s, `f_total_count` = %s, `f_activate_count` = %s
            WHERE `f_time` = %s
            """
            self.w_db.query(sql, active_count, user_count, activate_count, strDate)
        else:
            sql = """
            INSERT INTO `t_active_user_month`
            (`f_active_count`, `f_total_count`, `f_activate_count`, `f_time`)
            VALUES(%s, %s, %s, %s)
            """
            self.w_db.query(sql, active_count, user_count, activate_count, strDate)

        self.delete_history_active_user(strDate)

    def update_total_user_year(self):
        """
        更新年用户总数
        """
        # 获取上一天的日期
        today = BusinessDate.today()
        yesterday = today - datetime.timedelta(days=1)

        # 获取年用户总数
        sql = """
        SELECT `f_total_count`, `f_activate_count`
        FROM `t_active_user_year`
        WHERE `f_time` = %s
        """
        result = self.r_db.one(sql, yesterday.year)

        # 用户总数 = 当前用户数 + 已记录的删除用户数
        user_count = self.user_manage.get_all_user_count()

        # 获取月激活用户数
        activate_count = self.get_activate_user_count()

        if result:
            user_count += result["f_total_count"]
            activate_count += result["f_activate_count"]
            sql = """
            UPDATE `t_active_user_year` SET `f_total_count` = %s, `f_activate_count` = %s
            WHERE `f_time` = %s
            """
            self.w_db.query(sql, user_count, activate_count, yesterday.year)
        else:
            sql = """
            INSERT INTO `t_active_user_year` (`f_total_count`, `f_activate_count`, `f_time`)
            VALUES(%s, %s, %s)
            """
            self.w_db.query(sql, user_count, activate_count, yesterday.year)

    def active_user_count(self):
        """
        统计活跃用户
        """
        curTime = BusinessDate.now()
        self.update_active_user_day()

        if curTime.day == 1:
            self.update_active_user_month()

            if curTime.month == 1:
                self.update_total_user_year()


class ActiveUserCountThread(threading.Thread):
    """
    活跃用户数统计线程
    """
    def __init__(self):
        super(ActiveUserCountThread, self).__init__()
        self.active_user_manage = ActiveUserManage()

    def get_wait_time(self):
        """
        获取到下次执行sleep的时间
        """
        now = BusinessDate.now()
        tmp_time = now

        if tmp_time.hour > 0 or tmp_time.minute > 10:
            tmp_time = tmp_time + datetime.timedelta(days=1)

        tmp_time = tmp_time.replace(hour=0, minute=10, second=59)
        wait_time = time.mktime(tmp_time.timetuple()) - time.mktime(now.timetuple())

        return wait_time

    def run(self):
        holder_id = str(uuid.uuid1())
        ShareMgnt_Log("ActiveUserCountThread start, holder_id: %s", holder_id)
        while True:
            wait_time = self.get_wait_time()
            time.sleep(wait_time)
            try:
                ShareMgnt_Log("Updating active user count...")
                # 此进程24小时执行一次
                # 此方法不解锁，保证执行间隔略大于1小时，只用防止多副本重复执行
                if acquire_lock("active_user_count_lock", holder_id, 60*60):
                    ShareMgnt_Log("ActiveUserCountThread get lock success, update active user count, holder_id: %s", holder_id)
                    self.active_user_manage.active_user_count()
                else:
                    ShareMgnt_Log("ActiveUserCountThread get lock failed, skip update active user count")
            except Exception as e:
                ShareMgnt_Log(traceback.format_exc())
                ShareMgnt_Log("ActiveUserCountThread error: %s", str(e))
            time.sleep(120)


class GenFileThread(threading.Thread):
    """
    生成活跃报表文件线程
    """
    def __init__(self, taskId, taskInfo):
        super(GenFileThread, self).__init__()
        self.taskId = taskId
        self.taskInfo = taskInfo
        self.active_user_manage = ActiveUserManage()

    def run(self):
        """
        执行
        """
        ShareMgnt_Log("**************** generate file thread start *************** *")

        try:
            self.taskInfo.file_path = self.active_user_manage.gen_file_handler(self.taskId, self.taskInfo)
        except Exception as ex:
            # 任务处理异常，更新任务状态
            fileDir = os.path.join(_file_path, self.taskId)
            shutil.rmtree(fileDir)
            self.taskInfo.set_finished_status(TaskFinishedStatus.TASK_ERROR)
            ShareMgnt_Log("generate file thread run error: %s", str(ex))

        ShareMgnt_Log("**************** generate file thread end *******************")


class ActiveReportTaskAutoDeleteThread(threading.Thread):

    def __init__(self):
        """
        初始化
        """
        super(ActiveReportTaskAutoDeleteThread, self).__init__()

    def delete_overtime_task(self):
        """
        删除创建时间超过 1h 的任务
        """
        global task_dict

        with threadLock:
            items = list(task_dict.items())

        for (taskId, taskInfo) in items:
            if taskInfo.create_time < (BusinessDate.time() - WAIT_TIME_ONE_HOUR):
                with threadLock:
                    if taskId in task_dict:
                        del(task_dict[taskId])

    def run(self):
        """
        执行
        """
        ShareMgnt_Log("**************** active report download task auto delete thread start *****************")

        while True:
            try:
                self.delete_overtime_task()
            except Exception as e:
                print(("active report download task auto delete thread run error: %s", str(e)))
            time.sleep(WAIT_TIME_ONE_HOUR)

        ShareMgnt_Log("**************** active report download task auto delete thread end *****************")
