#!/usr/bin/python3
#-*- coding:utf-8 -*-

"""生成并安装Thrift API的python代码到python系统路径"""

from setuptools import setup
import os
import inspect
import shutil
import platform
import subprocess
import time
import sys
import thrift

lockfilepath = os.path.realpath(__file__) + ".lock"


def script_path():
    """
    function: get absolute directory of the file
        in which this function defines
    usage: path = script_path()
    note: please copy the function to target script,
        don't use it like <module>.script_path()
    """
    this_file = inspect.getfile(inspect.currentframe())
    return os.path.abspath(os.path.dirname(this_file))


def thrift_generate_path():
    """thrift生成文件路径"""
    return os.path.join(script_path(), "gen-py-tmp")


def thrift_tool_path():
    """获取thrift工具路径"""
    # thrift_file_name = "thrift"
    # if platform.machine() == 'aarch64':
    #     thrift_file_name = 'thrift-arm'

    # thrift_file_path = os.path.join(script_path(), thrift_file_name)
    # os.chmod(thrift_file_path, 0o777)
    # return thrift_file_path

    return "thrift"

def thrift_file_path():
    """获取thrift文件路径"""

    # return os.path.join(os.path.dirname(os.path.abspath(__file__)), "../API/ThriftAPI")
    return "/sysvol/apphome/app/API/ThriftAPI"

def generate_thrift_files():
    """
    执行Thrift命令生成Thrift python接口文件
    """
    scr_path = script_path()
    gen_py_path = thrift_generate_path()
    if os.path.exists(gen_py_path):
        shutil.rmtree(gen_py_path)
    os.mkdir(gen_py_path)

    tf_files = os.listdir(thrift_file_path())
    thrift_bin_path = thrift_tool_path()
    tf_file_path = thrift_file_path()
    for tf_file in tf_files:
        if os.path.splitext(tf_file)[1] == ".thrift":
            args = [thrift_bin_path, "-r", "--gen", "py", "-out"]
            args.append(gen_py_path)
            args.append(os.path.join(tf_file_path, tf_file))
            proc = subprocess.Popen(args, shell=False,
                                    stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            (outdata, errdata) = proc.communicate()
            if proc.returncode != 0:
                raise Exception(str(errdata))


def get_thrift_package_dir():
    """
    搜索gen-py目录下的所有Thrift接口包
    """
    gen_py_path = thrift_generate_path()
    if not os.path.exists(gen_py_path):
        raise Exception("Directory gen-py not exists.")
    sub_dirs = os.listdir(gen_py_path)
    package_dir = {}
    for sub_dir in sub_dirs:
        if os.path.isdir(os.path.join(gen_py_path, sub_dir)):
            package_dir[sub_dir] = os.path.join(gen_py_path, sub_dir)
    return package_dir


if __name__ == '__main__':
    # 尝试10次后退出
    cnt = 10
    while cnt > 0:
        try:
            os.mknod(lockfilepath)
            break
        except OSError:
            time.sleep(5)
            cnt = cnt - 1
            if cnt <= 0:
                sys.exit(1)
        except AttributeError:
            break
    try:
        generate_thrift_files()
        thrift_package_dir = get_thrift_package_dir()

        os.chdir(script_path())
        setup(
            name='ThriftAPI',
            version="1.0.0",
            package_dir=thrift_package_dir,
            packages=[k for k in thrift_package_dir.keys()],
            package_data={
                # If any package contains *-remote files, include them:
                '': ['*-remote'],
            },
        )
    finally:
        if os.path.exists('build'):
            shutil.rmtree('build')
        if os.path.exists('dist'):
            shutil.rmtree('dist')
        if os.path.exists('ThriftAPI.egg-info'):
            shutil.rmtree('ThriftAPI.egg-info')
        if os.path.exists(lockfilepath):
            # 删除锁文件
            os.remove(lockfilepath)
