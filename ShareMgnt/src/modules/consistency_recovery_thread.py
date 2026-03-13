#!/usr/bin/python3
# -*- coding:utf-8 -*-
"""数据一致性恢复线程"""
import threading
import time
from src.modules.user_manage import UserManage
from src.common.sharemgnt_logger import ShareMgnt_Log
from src.common.db.connector import DBConnector

from ShareMgnt.constants import (NCT_USER_ADMIN,
                                 NCT_SYSTEM_ROLE_SUPPER,
                                 NCT_SYSTEM_ROLE_ADMIN,
                                 NCT_SYSTEM_ROLE_SECURIT,
                                 NCT_SYSTEM_ROLE_AUDIT,
                                 NCT_SYSTEM_ROLE_ORG_MANAGER,
                                 NCT_SYSTEM_ROLE_ORG_AUDIT)
WAIT_TIME = 86400   # 24  * 3600


def wrap_recovery(func):
    """
    获取类当前调用的函数名
    """
    def _deco(*args, **kwargs):
        ShareMgnt_Log("**************** %s invoked *****************", func.__name__)
        return func(*args, **kwargs)

    return _deco


class ConsistencyRecoveryThread(threading.Thread, DBConnector):

    def __init__(self):
        """
        初始化
        """
        super(ConsistencyRecoveryThread, self).__init__()
        self.user_manage = UserManage()

    @wrap_recovery
    def recovery_supper_role(self):
        """
        开始进行超级管理员数据一致性恢复
        """
        # 保证权责集中模式下不能有系统管理员、安全管理员、审计管理员，权责分离模式下不能有超级管理员
        tris_status = self.user_manage.get_trisystem_status()
        if tris_status:
            sql = """
            DELETE FROM t_role WHERE f_role_id = %s
            """
            self.w_db.query(sql, NCT_SYSTEM_ROLE_SUPPER)
            sql = """
            DELETE FROM t_user_role_relation WHERE f_role_id = %s
            """
            self.w_db.query(sql, NCT_SYSTEM_ROLE_SUPPER)
        else:
            sql = """
            DELETE FROM t_role WHERE f_role_id in (%s, %s, %s)
            """
            self.w_db.query(sql, NCT_SYSTEM_ROLE_ADMIN,
                            NCT_SYSTEM_ROLE_SECURIT, NCT_SYSTEM_ROLE_AUDIT)
            sql = """
            DELETE FROM t_user_role_relation WHERE f_role_id in (%s, %s, %s)
            """
            self.w_db.query(sql, NCT_SYSTEM_ROLE_ADMIN,
                            NCT_SYSTEM_ROLE_SECURIT, NCT_SYSTEM_ROLE_AUDIT)

    @wrap_recovery
    def recovery_org_manage(self):
        """
        开始进行组织管理员数据一致性恢复
        """
        # 如果角色已被删除，则需要清空相应策略配置
        sql = """
        DELETE FROM t_department_responsible_person
        WHERE f_user_id not in (
            SELECT f_user_id
            FROM t_user_role_relation
            WHERE f_role_id = %s)
        """
        self.w_db.query(sql, NCT_SYSTEM_ROLE_ORG_MANAGER)

    @wrap_recovery
    def recovery_org_audit(self):
        """
        开始进行组织审计员数据一致性恢复
        """
        # 如果角色已被删除，则需要清空相应策略配置
        sql = """
        DELETE FROM t_department_audit_person
        WHERE f_user_id not in (
            SELECT f_user_id
            FROM t_user_role_relation
            WHERE f_role_id = %s)
        """
        self.w_db.query(sql, NCT_SYSTEM_ROLE_ORG_AUDIT)

    @wrap_recovery
    def recovery_role_attribute(self):
        """
        开始进行角色属性数据一致性恢复
        """
        # 如果角色已被删除，则需要清空相应策略配置
        sql = """
        DELETE FROM t_user_role_attribute
        WHERE f_user_id not in (
            SELECT f_user_id
            FROM t_user_role_relation
            WHERE f_role_id in (%s, %s, %s, %s))
        """
        self.w_db.query(sql, NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_ADMIN,
                        NCT_SYSTEM_ROLE_SECURIT, NCT_SYSTEM_ROLE_AUDIT)

    def start_role_recovery(self):
        """
        开始进行角色模块相关数据恢复
        """
        self.recovery_supper_role()
        self.recovery_org_manage()
        self.recovery_org_audit()
        self.recovery_role_attribute()

    def run(self):
        """
        执行
        """
        ShareMgnt_Log("**************** consistency recovery thread start *****************")

        while True:
            try:
                self.start_role_recovery()

            except Exception as e:
                print(("consistency recovery thread error: %s", str(e)))
            time.sleep(WAIT_TIME)

        ShareMgnt_Log("**************** consistency recovery thread end *****************")
