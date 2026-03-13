from src.common.db.connector import ConnectorManager
from EThriftException.ttypes import ncTException
from src.common import global_info

# 此方法模拟redis的setnx方法
# 支持只能释放自己创建的锁，不能释放别人的锁，但是支持替代别人的过期锁
# 此方法可以支持简单的分布式锁，可能出现因为业务卡死造成的强锁问题,多次执行
def acquire_lock(lock_name, holder_id, duration_sec=10): 
    conn = ConnectorManager.get_db_conn()
    cursor = conn.cursor()


    # 尝试直接插入（对应 Redis 的 NX）
    try:
        if global_info.DB_TYPE == 'kdb9':
            cursor.execute(
                "INSERT INTO t_distributed_lock (f_lock_key, f_hoder_id, f_expire_time) VALUES ('%s', '%s', NOW() + INTERVAL '%d SECOND')" % (lock_name, holder_id, duration_sec)
            )
        elif global_info.DB_TYPE == 'dm8':
            cursor.execute(
                "INSERT INTO t_distributed_lock (f_lock_key, f_hoder_id, f_expire_time) VALUES (%s, %s, SYSDATE + %s/86400)",
                (lock_name, holder_id, duration_sec)
            )
        else:
            cursor.execute(
                "INSERT INTO t_distributed_lock (f_lock_key, f_hoder_id, f_expire_time) VALUES (%s, %s, NOW() + INTERVAL %s SECOND)",
                (lock_name, holder_id, duration_sec)
            )
        return True
    except:
        # 插入失败，说明锁已存在，检查是否过期
        # 如果已过期，抢占它（对应 Redis 的过期覆盖）
        if global_info.DB_TYPE == 'kdb9':
            cursor.execute(
                "UPDATE t_distributed_lock SET f_hoder_id = '%s', f_expire_time = NOW() + INTERVAL '%d SECOND' WHERE f_lock_key = '%s' AND f_expire_time < NOW()" % (holder_id, duration_sec, lock_name),
            )
        elif global_info.DB_TYPE == 'dm8':
            cursor.execute(
                "UPDATE t_distributed_lock SET f_hoder_id = %s, f_expire_time = SYSDATE + %s/86400 WHERE f_lock_key = %s AND f_expire_time < SYSDATE",
                (holder_id, duration_sec, lock_name)
            )
        else:
            cursor.execute(
                "UPDATE t_distributed_lock SET f_hoder_id = %s, f_expire_time = NOW() + INTERVAL %s SECOND WHERE f_lock_key = %s AND f_expire_time < NOW()",
                (holder_id, duration_sec, lock_name)
            )
        return cursor.rowcount > 0 # 如果更新了行，说明抢锁成功
    
def release_lock(lock_name, holder_id):
    # 只能释放自己创建的锁，不能释放别人的锁
    conn = ConnectorManager.get_db_conn()
    cursor = conn.cursor()
    cursor.execute(
        "DELETE FROM t_distributed_lock WHERE f_lock_key = %s AND f_hoder_id = %s",
        (lock_name, holder_id)
    )
    return cursor.rowcount > 0