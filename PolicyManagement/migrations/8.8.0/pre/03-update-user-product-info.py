#!/usr/bin/env python3
# -*- coding: utf-8 -*-

# 支持测试许可证
import os
import json
import rdsdriver

def get_conn(user, password, host, port, database):
    try:
        conn = rdsdriver.connect(host=host,
                                 port=int(port),
                                 user=user,
                                 password=password,
                                 database=database,
                                 autocommit=True)
    except Exception as e:
        print("connect database error: %s", str(e))
        raise e
    return conn

def check_need_update(cursor):
    """检查是否需要更新所有用户信息"""
    sql = "select f_value from sharemgnt_db.t_sharemgnt_config where f_key = 'need_update_product_8_8_0';"
    cursor.execute(sql)
    result = cursor.fetchone()

    # 如果配置不存在，则插入配置， 否则比对配置值是否是0， 如果不是0 则不需要更新 ， 返回false
    # 如果配置存在，则比对配置值是否是0， 如果不是0 则不需要更新 ， 返回false
    if not result:
        sql = "select count(*) from policy_mgnt.t_product_relation;"
        cursor.execute(sql)
        result = cursor.fetchone()

        sql = "insert into sharemgnt_db.t_sharemgnt_config (f_key, f_value) values ('need_update_product_8_8_0', %s);"
        cursor.execute(sql, (str(result[0]),))
        return int(result[0]) == 0

    return result[0] == "0"

def check_table_exists(cursor):
    """检查表是否存在"""
    sql = """
        SELECT COUNT(*) 
        FROM INFORMATION_SCHEMA.TABLES 
        WHERE TABLE_SCHEMA = 'license' 
          AND TABLE_NAME = 'license'
    """
    cursor.execute(sql)
    result = cursor.fetchone()
    print("check table exists result: %s" , str(result[0]))
    return result[0] > 0

def get_as_actived_licenses(cursor):
    """获取 AS 已激活的许可证"""
    if not check_table_exists(cursor):
        print("license table not exists")
        return False
    
    sql = "select * from license.license;"
    cursor.execute(sql)
    result = cursor.fetchall()
    data = len(result) != 0
    print("get as actived licenses success: %s" , str(data))
    return data

def delete_all_user_product_info(cursor):
    """删除所有用户产品信息"""
    # 获取当前产品数量
    sql1 = "select count(*) from policy_mgnt.t_product_relation;"
    cursor.execute(sql1)
    result1 = cursor.fetchone()
    
    # 分批删除每次删除50000条
    for i in range(0, result1[0], 50000):
        sql = "delete from policy_mgnt.t_product_relation limit 50000;"
        cursor.execute(sql)
        print("delete user product info success: %s" , str(i))
    
    print("delete all user product info success")
    return

def update_user_product_info(cursor):
    """更新用户产品信息"""
    sql = """select f_user_id from sharemgnt_db.t_user where 
            f_user_id != '4bb41612-a040-11e6-887d-005056920bea' and 
            f_user_id != '94752844-BDD0-4B9E-8927-1CA8D427E699' and
            f_user_id != '234562BE-88FF-4440-9BFF-447F139871A2' and
            f_user_id != '266c6a42-6131-4d62-8f39-853e7093701c' """
    cursor.execute(sql)
    result = cursor.fetchall()

    # 每10000个user 插入一次
    # 将用户id数组每10000个分一次
    user_ids = [result[i:i+10000] for i in range(0, len(result), 10000)]
    index = 0
    for user_id in user_ids:
        sql = "insert into policy_mgnt.t_product_relation (f_account_id, f_product, f_account_type) values "
        for id in user_id:
            sql += "('" + id[0] + "', 'anyshare', 1),"
        sql = sql[:-1]
        cursor.execute(sql)
        index += 1
        print("insert user product info success: %s" , str(index))
    print("insert all user product info success")
    return

if __name__ == "__main__":
    try:
        # 建立数据库连接
        db_name = os.environ.get("SYSTEM_ID", "") + "policy_mgnt"
        conn = get_conn(
            os.environ["DB_USER"],
            os.environ["DB_PASSWD"],
            os.environ["DB_HOST"],
            os.environ["DB_PORT"],
            db_name
        )
        cursor = conn.cursor()

        # 检查是否需要更新所有用户信息
        result = check_need_update(cursor)
        print("check need update result: %s" , str(result))

        # 如果有as的许可，则增加用户信息
        if result and get_as_actived_licenses(cursor):
            delete_all_user_product_info(cursor)
            update_user_product_info(cursor)
    except Exception:
        raise Exception()


