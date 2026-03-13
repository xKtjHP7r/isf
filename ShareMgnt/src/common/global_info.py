#!/usr/bin/python3
# -*- coding:utf-8 -*-
# pylint: disable=W0603

import os
import threading
from src.common.lib import (check_service_node, get_server_port)
"""
本文件用于保存全局变量
"""
SYSTEM_ID = os.getenv("SYSTEM_ID", "")

DB_WRITE_IP = ''
DB_READ_IP = ''
DB_PORT = 0
DB_NAME = f"{SYSTEM_ID}sharemgnt_db"
DB_USER = ''
DB_PWD = ''
DB_BACK_COUNT = 2
DB_TYPE = 'mysql'

DB_WRITE = 'localhost'
DB_READ = 'localhost'
SHAREMGNT_DB_PORT = 0
SHAREMGNT_DB_BACK_COUNT = 2

IMPORT_TOTAL_NUM = 0
IMPORT_SUCCESS_NUM = 0
IMPORT_FAIL_NUM = 0
IMPORT_FAIL_INFO = []
IMPORT_IS_STOP = False

IS_SINGLE = check_service_node()
PORT = get_server_port()

des_key = "@npM7$2W"

UT_DB_CLEAR = False

DEFAULT_DEPART_PRIORITY = 999999

# 导出任务的缓存：{filepath: task_id}
TASK_DICT = {}
EXPORT_FILE_THREADLOCK = threading.Lock()
DELETE_FILE_PATHS= []

def init_import_variable():
    """
    初始化导入变量
    """
    global IMPORT_TOTAL_NUM
    global IMPORT_SUCCESS_NUM
    global IMPORT_FAIL_NUM
    global IMPORT_FAIL_INFO
    global IMPORT_IS_STOP
    global IMPORT_DISABLE_USER_NUM

    IMPORT_TOTAL_NUM = 0
    IMPORT_SUCCESS_NUM = 0
    IMPORT_FAIL_NUM = 0
    IMPORT_FAIL_INFO = []
    IMPORT_IS_STOP = False
    IMPORT_DISABLE_USER_NUM = 0

init_import_variable()

ADMIN_NAME = 'admin'
SECURIT_NAME = 'security'
AUDIT_NAME = 'audit'
SYSTEM_NAME = 'system'
# 内网数据交换ID
NC_EVFS_NAME_IOC_DATAEXCHANGE_ID = "da5bfdc4-cb4b-4b28-90c2-9eca46c3e500"
SYSTEM_ROLE_NAMES_ZH_CN = ["超级管理员", "系统管理员", "安全管理员", "审计管理员", "组织管理员",
                           "组织审计员", "文档审核员", "共享审核员", "定密审核员"]
SYSTEM_ROLE_NAMES_EN_US = ["super admin", "admin", "security", "audit", "general admin",
                           "general audit", "file approver", "sharing approver",
                           "security level approver"]
SYSTEM_ROLE_NAMES_ZH_TW = ["超級管理員", "系統管理員", "安全管理員", "審計管理員", "組織管理員",
                           "組織審計員", "文件核准者", "共用核准者", "定密核准者"]
SYSTEM_ROLE_NAMES = []
SYSTEM_ROLE_NAMES.extend(SYSTEM_ROLE_NAMES_ZH_CN)
SYSTEM_ROLE_NAMES.extend(SYSTEM_ROLE_NAMES_EN_US)
SYSTEM_ROLE_NAMES.extend(SYSTEM_ROLE_NAMES_ZH_TW)

# 
THIRD_SYNC_THREAD = None
DOMAIN_SYNC_THREAD = {}
THIRD_DB_SYNC_THREAD = {}
RETRY_SYNC_THREAD = {}
RETRY_SYNC_THREAD["THIRD"] = ""
RETRY_SYNC_THREAD["DOMAIN"] = set()
RETRY_SYNC_THREAD["THIRD_DB"] = set()

# 日志类型
LOG_TYPE_LOGIN = 1
LOG_TYPE_MANAGE = 2
LOG_TYPE_OPERATION = 3

# 用户类型
USER_TYPE_AUTH = 1
USER_TYPE_ANONY = 2
USER_TYPE_APP = 3
USER_TYPE_INTER = 4

# 日志级别
LOG_LEVEL_INFO = 1
LOG_LEVEL_WARN = 2

# 管理日志操作类型
LOG_OP_TYPE_CREATE = 1
LOG_OP_TYPE_ADD = 2
LOG_OP_TYPE_SET = 3
LOG_OP_TYPE_DELETE = 4
LOG_OP_TYPE_MOVE = 6
LOG_OP_TYPE_REMOVE = 7
LOG_OP_TYPE_IMPORT = 8
