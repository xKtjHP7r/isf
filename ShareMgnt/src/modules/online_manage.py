#!/usr/bin/python3
# -*- coding:utf-8 -*-
"""This is spcae manage class"""
import uuid
import calendar
import time
from threading import Thread
from src.common.db.connector import DBConnector
from src.common.lib import (raise_exception)
from src.common.sharemgnt_logger import ShareMgnt_Log
from src.common.business_date import BusinessDate
from ShareMgnt.ttypes import ncTOpermOnlineUserInfo
from datetime import datetime
from src.modules.user_manage import UserManage
from src.modules.distributed_lock import acquire_lock, release_lock

class OnlineManage(DBConnector):
    """
    OnlineManage
    """
    def __init__(self):
        self.user_manage = UserManage()

    def get_current_online_user(self):
        """
        获取当前在线用户总数
        """
        sql = """
        SELECT `f_time`, `f_count` FROM `t_online_user_real_time`
        WHERE `f_time` <= %s
        ORDER BY `f_time` DESC LIMIT 0, 120
        """
        now = BusinessDate.now().strftime("%Y-%m-%d %H:%M:%S")
        db_result = self.r_db.all(sql, now)
        results = []
        for row in db_result:
            result = ncTOpermOnlineUserInfo()
            result.time = row['f_time']
            result.count = row['f_count']
            results.append(result)

        return results

    def get_max_online_user_day(self, date_month):
        """
        获取一个月内每天的最大在线用户数
        """
        try:
            date_month = datetime.strptime(date_month, "%Y-%m")
        except Exception:
            raise_exception(_("date time illegal"))

        year = date_month.year
        month = date_month.month
        days = calendar.monthrange(year, month)[1]
        month = str(month).zfill(2)

        start_time = "%s-%s-01" % (year, month)
        end_time = "%s-%s-%s" % (year, month, days)

        sql = """
        SELECT `f_time`, `f_count` FROM `t_max_online_user_day`
        WHERE `f_time` >= %s AND `f_time` <= %s
        ORDER BY `f_time`
        """
        db_result = self.r_db.all(sql, start_time, end_time)

        results = []
        for row in db_result:
            result = ncTOpermOnlineUserInfo()
            result.time = row['f_time']
            result.count = row['f_count']
            results.append(result)

        return results

    def get_max_online_user_month(self, start_month, end_month):
        """
        获取指定月份范围内的每月的最大在线用户数
        """
        try:
            start_date_month = datetime.strptime(start_month, "%Y-%m")
            end_date_month = datetime.strptime(end_month, "%Y-%m")
        except ValueError:
            raise_exception(_("date time illegal"))

        if (start_date_month.year > end_date_month.year or
            (start_date_month.year == end_date_month.year and
                start_date_month.month > end_date_month.month)):
            raise_exception(_("date time illegal"))

        sql = """
        SELECT `f_time`, `f_count` FROM `t_max_online_user_month`
        WHERE `f_time` >= %s AND `f_time` <= %s
        ORDER BY `f_time` ASC
        """
        db_result = self.r_db.all(sql, start_month, end_month)

        results = []
        for row in db_result:
            result = ncTOpermOnlineUserInfo()
            result.time = row['f_time']
            result.count = row['f_count']
            results.append(result)

        return results

    def get_earliest_time(self):
        """
        获取有记录的最早时间
        """
        sql = """
        SELECT f_time FROM t_max_online_user_month
        ORDER BY f_time ASC
        LIMIT 1
        """
        db_result = self.r_db.one(sql)

        earliest_time = ''
        if db_result:
            earliest_time = db_result['f_time']

        return earliest_time

    def update_max_online_user_month(self, current_month, online_user_count, current_uuid):
        """
        更新每月最大在线用户数
        """
        sql = """
        SELECT `f_count` FROM `t_max_online_user_month`
        WHERE `f_time` = %s LIMIT 1
        """
        db_result = self.r_db.one(sql, current_month)

        if not db_result:
            self.w_db.insert('t_max_online_user_month',
                             [current_month, online_user_count, current_uuid])
        else:
            if online_user_count > db_result['f_count']:
                sql = """
                UPDATE `t_max_online_user_month`
                SET `f_count` = %s
                WHERE `f_time` = %s
                """
                self.r_db.query(sql, online_user_count, current_month)

        # 删除未来数据，避免往前修改系统时间引起数据异常
        now = BusinessDate.now().strftime("%Y-%m")
        sql = """
        DELETE FROM `t_max_online_user_month` WHERE `f_time` > %s
        """
        self.w_db.query(sql, now)

    def update_max_online_user_day(self, current_day, online_user_count, current_uuid):
        """
        更新每日最大在线用户数
        """
        sql = """
        SELECT `f_count` FROM `t_max_online_user_day`
        WHERE `f_time` = %s LIMIT 1
        """
        db_result = self.r_db.one(sql, current_day)

        if not db_result:
            self.w_db.insert('t_max_online_user_day',
                             [current_day, online_user_count, current_uuid])
        else:
            if online_user_count > db_result['f_count']:
                sql = """
                UPDATE `t_max_online_user_day`
                SET `f_count` = %s
                WHERE `f_time` = %s
                """
                self.w_db.query(sql, online_user_count, current_day)

        # 删除未来数据，避免往前修改系统时间引起数据异常
        now = BusinessDate.now().strftime("%Y-%m-%d")
        sql = """
        DELETE FROM `t_max_online_user_day` WHERE `f_time` > %s
        """
        self.w_db.query(sql, now)

    def update_online_user(self):
        """
        更新当前客户端在线用户数
        """
        now = BusinessDate.now()

        # 2012-03-09 11:23:09
        # 2012-03-09
        # 2012-03
        current_uuid = str(uuid.uuid1())
        current_time = now.strftime("%Y-%m-%d %H:%M:%S")
        current_day = now.strftime("%Y-%m-%d")
        current_month = now.strftime("%Y-%m")

        online_user_count = None
        online_user_count = self.user_manage.get_online_user_count()

        # 删除前一天的数据
        sql = """
        DELETE FROM `t_online_user_real_time` WHERE `f_time` < %s
        """
        self.w_db.query(sql, current_day)

        # 删除未来数据，避免往前修改系统时间引起数据异常
        sql = """
        DELETE FROM t_online_user_real_time WHERE f_time > now()
        """
        self.w_db.query(sql)

        # 记录当前时间点在线用户数
        self.w_db.insert('t_online_user_real_time', [current_time, online_user_count, current_uuid])

        # 更新每日最大在线用户数
        self.update_max_online_user_day(current_day, online_user_count, current_uuid)

        # 更新每月最大在线用户数
        self.update_max_online_user_month(current_month, online_user_count, current_uuid)


class ThreadGetOnlineInfo(Thread):
    """
    定时统计客户端在线用户线程
    """
    def __init__(self):
        """
        初始化
        """
        super(ThreadGetOnlineInfo, self).__init__()
        self.online_manage = OnlineManage()

    def run(self):
        holder_id = str(uuid.uuid1())
        ShareMgnt_Log("ThreadGetOnlineInfo start, holder_id: %s", holder_id)
        b_get_lock = False
        while True:
            try:
                # 保证锁有效时间为10s
                # 每5s尝试获取一次，如果获取到锁，则更新在线用户数，如果获取不到锁，则等待5s后继续尝试
                time.sleep(5)
                if b_get_lock:
                    release_lock("online_user_count_lock", holder_id)
                    b_get_lock = False
                
                b_get_lock = acquire_lock("online_user_count_lock", holder_id, 10)
                if b_get_lock:
                    #ShareMgnt_Log("ThreadGetOnlineInfo get lock success, update online user count, holder_id: %s", holder_id)
                    self.online_manage.update_online_user()
                else:
                    #ShareMgnt_Log("ThreadGetOnlineInfo get lock failed, skip update online user count, holder_id: %s", holder_id)
                    pass
            except Exception as e:
                ShareMgnt_Log("ThreadGetOnlineInfo error: %s", str(e))
                import traceback
                traceback.print_exc()