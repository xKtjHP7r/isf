#!/usr/bin/python3
# -*- coding:utf-8 -*-
"""通知中心运行线程"""
import threading
from src.common.sharemgnt_logger import ShareMgnt_Log
import queue


class CallableTask(object):
    """
    可调用的任务类, 在任务线程中使用
    """
    def __init__(self, module_name=None, function_name=None, function_args=None):
        self.module_name = module_name
        self.function_name = function_name
        self.function_args = function_args


class HandleTaskThread(threading.Thread):

    queue = queue.Queue()

    def __init__(self):
        """
        初始化
        """
        super(HandleTaskThread, self).__init__()
        self.queue = HandleTaskThread.queue

    def close(self):
        """
        关闭
        """
        self.queue.put("close")

    def add(self, task):
        """
        """
        self.queue.put(task)

    def handle_task(self, task):
        """
        task:元组：(module,data)
            data:元组：(function_name, args)
                function_name: 所调用模块中的方法
                args: 调用参数
        """
        if isinstance(task, CallableTask):
            # 调用类模块中的方法处理任务
            module_cls = None
            if task.module_name == "login_access_control_manage":
                from src.modules.login_access_control_manage import LoginAccessControlManage
                module_cls = LoginAccessControlManage()

            if task.module_name == "department_manage":
                from src.modules.department_manage import DepartmentManage
                module_cls = DepartmentManage()

            if module_cls is not None:
                getattr(module_cls, task.function_name)(task.function_args)

    def run(self):
        """
        执行
        """
        ShareMgnt_Log("**************** 更新token flag线程启动 *****************")

        while True:
            try:
                task = self.queue.get()
                if isinstance(task, str):
                    if task == "close":
                        break
                self.handle_task(task)

            except Exception as e:
                ShareMgnt_Log("通知中心运行异常：%s", str(e))

        ShareMgnt_Log("**************** 更新token flag线程结束 *****************")
