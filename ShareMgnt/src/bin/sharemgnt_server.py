#!/usr/bin/python3
# -*- coding:utf-8 -*-
"""
sharemgnt server
"""
import os
import sys
import cx_Oracle
import suds
import threading

cur_path = os.path.dirname(os.path.abspath(sys.argv[0]))
# 如果是源码方式启动
if cur_path.find("/src/") != -1:
    project_path = os.path.realpath(os.path.join(cur_path, "../../"))
    sys.path.append(project_path)

from Crypto.Cipher import AES

from thrift.transport import TSocket
from thrift.transport import TTransport
from thrift.protocol import TBinaryProtocol
from thrift.server import TServer

from ShareMgnt import ncTShareMgnt

from eisoo import sighandler
from eisoo import langlib

from src.common.db.db_manager import SharemgntDBManager
from src.common.global_info import IS_SINGLE, PORT
from src.common.sharemgnt_logger import ShareMgnt_Log
from src.handler.sharemgnt_handler import ShareMgntHandler
from src.modules.online_manage import ThreadGetOnlineInfo
from src.modules.nc_thread import NotifiCenterThread
from src.third_party_auth.third_party_manage import ThirdPartyManage
from src.third_party_auth.third_sync_manage import SyncRetryThread
from src.modules.handle_task_thread import HandleTaskThread
from src.modules.vcode_auto_delete_thread import VcodeAutoDeleteThread
from src.modules.limit_rate_manage import LimitRateManage
from src.modules.active_user_manage import (ActiveUserCountThread,
                                            ActiveReportTaskAutoDeleteThread)
from src.modules.consistency_recovery_thread import ConsistencyRecoveryThread
from src.modules.space_report_manage import SpaceReportTaskAutoDeleteThread
from src.modules.config_manage import ConfigManage
from src.modules.scan_virus_manage import ScanVirusManage
from src.modules.domain_manage import InitAvailableDomainPoolThread
from src.modules.export_file_task_manage import DeleteFileThread


def main():
    """
    ShareMgnt服务启动主函数
    """
    ShareMgnt_Log("ShareMgnt beginning...")

    # 设置打印堆栈信号量
    sighandler.handle_signals()

    # 加载语言
    langstr = langlib.get_lang()
    ShareMgnt_Log("Language: %s", langstr)

    # 获取语言资源路径， 主执行文件路径的上级路径
    local_path = os.path.realpath(os.path.join(cur_path, "../"))
    ShareMgnt_Log("local_path: %s", local_path)
    langlib.init_language("ShareMgnt", local_path=local_path)

    # 检查是否是主节点
    global IS_SINGLE
    service_node = IS_SINGLE

    # 检查数据库信息
    try:
        ShareMgnt_Log("check and init sharemgnt db ...")
        db_manager = SharemgntDBManager()
        db_manager.check_db_service()
        db_manager.init_db()
    except Exception as ex:
        ShareMgnt_Log("check and init sharemgnt db failed:%s", str(ex))
        import traceback
        traceback.print_exc()

    # 在线统计线程
    ShareMgnt_Log("Starting online user collecting thread...")
    thread_get_online_info = ThreadGetOnlineInfo()
    thread_get_online_info.setDaemon(True)
    thread_get_online_info.start()

    # 开启初始化可用域池线程
    if service_node:
        ShareMgnt_Log("Starting initialize available domain pool thread...")
        init_domain_thread = InitAvailableDomainPoolThread()
        init_domain_thread.daemon = True
        init_domain_thread.start()

    # 开启域正向同步线程
    if service_node:
        try:
            ShareMgnt_Log("Starting domain sync thread...")
            ThirdPartyManage().start_all_domain_sync()
        except Exception as ex:
            ShareMgnt_Log("start domain sync failed:%s", str(ex))

     # 同步第三方插件
    if service_node:
        try:
            ShareMgnt_Log("Starting sync third party plugin...")
            threading.Thread(target=ThirdPartyManage().sync_third_party_plugin, name="Retry").start()
        except Exception as ex:
            ShareMgnt_Log("sync third party plugin failed:%s", str(ex))

    # 开启第三方数据同步
    if service_node:
        try:
            ShareMgnt_Log("Starting third db sync thread...")
            ThirdPartyManage().start_all_third_db_sync()
        except Exception as ex:
            ShareMgnt_Log("start third db sync failed:%s", str(ex))

    # 启用第三方插件特有服务
    if service_node:
        try:
            ShareMgnt_Log("Starting third plugin service...")
            ThirdPartyManage().start_third_plugin_service()
        except Exception as ex:
            ShareMgnt_Log("Starting third plugin service failed:%s", str(ex))

    # 加载消息中心线程
    if service_node:
        nc = NotifiCenterThread.instance()
        nc.daemon = True
        nc.start()

    # 加载任务处理线程
    if service_node:
        handle_task_thread = HandleTaskThread()
        handle_task_thread.daemon = True
        handle_task_thread.start()

    # 开启验证码自动删除线程
    # 无需服务节点，验证码自动删除线程可以在每个节点上运行
    vcode_auto_delete_thread = VcodeAutoDeleteThread()
    vcode_auto_delete_thread.daemon = True
    vcode_auto_delete_thread.start()

    # 开启活跃报表过期下载任务清理线程
    active_report_task_auto_delete_thread = ActiveReportTaskAutoDeleteThread()
    active_report_task_auto_delete_thread.daemon = True
    active_report_task_auto_delete_thread.start()

    # 开启活跃用户统计线程
    active_user_count_thread = ActiveUserCountThread()
    active_user_count_thread.daemon = True
    active_user_count_thread.start()

    # 开启用户限速值线程
    LIMIT_USER_GROUP = 1
    limit_rate_config = LimitRateManage().get_limit_rate_config()
    if limit_rate_config.isEnabled and limit_rate_config.limitType == LIMIT_USER_GROUP:
        LimitRateManage().start_update_user_limit_rate_thread()

    # 启动数据一致性恢复线程
    # 无需服务节点，数据一致性恢复线程可以在每个节点上运行
    consistency_recovery_thread = ConsistencyRecoveryThread()
    consistency_recovery_thread.daemon = True
    consistency_recovery_thread.start()

    # 启动用户空间使用情况报表任务清理线程
    space_report_task_auto_delete_thread = SpaceReportTaskAutoDeleteThread()
    space_report_task_auto_delete_thread.daemon = True
    space_report_task_auto_delete_thread.start()

    # 启动清理过期文件线程
    delete_file_thread = DeleteFileThread()
    delete_file_thread.daemon = True
    delete_file_thread.start()

    # 重启服务获取开关设置自动更新病毒线程
    if service_node:
        enable_update_virus_db = ConfigManage().get_custom_config_of_bool("enable_update_virus_db")
        ScanVirusManage().set_update_virusdb_thread_running(enable_update_virus_db)

    # 开启同步重试线程
    sync_retry_thread = SyncRetryThread()
    sync_retry_thread.daemon = True
    sync_retry_thread.start()


    # ShareMgnt.thrift Server 10 Threads
    global PORT
    handler = ShareMgntHandler()
    processor = ncTShareMgnt.Processor(handler)
    transport = TSocket.TServerSocket(port=PORT)
    tfactory = TTransport.TBufferedTransportFactory()
    pfactory = TBinaryProtocol.TBinaryProtocolFactory()
    server = TServer.TThreadPoolServer(processor,
                                       transport,
                                       tfactory,
                                       pfactory,
                                       daemon=True)
    server.setNumThreads(10)

    ShareMgnt_Log("Starting Thrift Server...")
    server.serve()


if __name__ == "__main__":
    try:
        main()
    except Exception as ex:
        ShareMgnt_Log("ShareMgnt start failed, ex: %s", str(ex))
        raise
