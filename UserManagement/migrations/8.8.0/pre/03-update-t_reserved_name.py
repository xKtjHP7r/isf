#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import os
import rdsdriver
import logging

# 配置日志
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)


def get_conn(user, password, host, port, database):
    try:
        conn = rdsdriver.connect(host=host,
                                 port=int(port),
                                 user=user,
                                 password=password,
                                 database=database,
                                 autocommit=True)
    except Exception as e:
        logger.error("connect database error: %s", str(e))
        raise
    return conn


class TableMigrator:
    def __init__(self):
        self.table_old = "t_reserved_name"
        self.table_new = "t_reserved_name_new"
        self.table_backup = "t_reserved_name_backup"
        self.batch_size = 2000
        self.conn = get_conn(os.environ["DB_USER"], os.environ["DB_PASSWD"],
                    os.environ["DB_HOST"], os.environ["DB_PORT"], os.environ.get("SYSTEM_ID", "") + "sharemgnt_db")
        self.db_type = os.environ.get("DB_TYPE", "unknown").lower()

    def table_exists(self, table_name):
        """检查表是否存在"""
        with self.conn.cursor() as cursor:
            cursor.execute(f"SHOW TABLES LIKE '{table_name}'")
            return cursor.fetchone() is not None

    def check_idempotency(self):
        """
        幂等性检查：
        1. 如果备份表存在
            a. 如果新表存在 -> 原表已重命名为备份表，新表未重命名为原表，执行重命名操作。
            b. 如果新表不存在 -> 认为迁移已完成，停止。
        2. 如果新表存在 -> 认为上次失败，删除新表以便重试。
        """
        logger.info("environmental inspection is underway....")

        # 检查是否已经迁移完成
        if self.table_exists(self.table_backup):
            if self.table_exists(self.table_new):
                with self.conn.cursor() as cursor:
                    cursor.execute(f"RENAME TABLE {self.table_new} TO {self.table_old}")
                logger.info("Residual table has been renamed.")
                return False

            logger.warning(f"Backup table `{self.table_backup}` already exists.")
            logger.warning("The migration task has been completed, the script will stop execution.")
            return False # 停止执行

        # 检查是否存在残留的中间表
        if self.table_exists(self.table_new):
            logger.info(f"Detected residual intermediate table `{self.table_new}`, cleaning up...")
            with self.conn.cursor() as cursor:
                cursor.execute(f"DROP TABLE {self.table_new}")
            logger.info("Residual table has been cleaned up.")

        return True # 继续执行

    def create_new_table(self):
        logger.info(f"Creating new table `{self.table_new}`...")

        ddl = f"""
        CREATE TABLE IF NOT EXISTS `{self.table_new}` (
            `f_id` char(40) NOT NULL COMMENT 'id',
            `f_name` char(150) NOT NULL COMMENT '名称',
            `f_create_time` bigint(20) NOT NULL COMMENT '创建时间',
            `f_update_time` bigint(20) NOT NULL COMMENT '修改时间',
            PRIMARY KEY (`f_id`),
            KEY `idx_name` (`f_name`)
        ) ENGINE=InnoDB COMMENT='保留名称表';
        """

        with self.conn.cursor() as cursor:
            cursor.execute(ddl)
        logger.info("New table created successfully.")

    def migrate_data(self):
        """
        分批迁移数据。
        """
        logger.info("Data migration is underway...")

        insert_sql = f"INSERT INTO `{self.table_new}` (f_id, f_name, f_create_time, f_update_time) VALUES (%s, %s, %s, %s)"

        total_migrated = 0
        cursor = None

        try:
            cursor = self.conn.cursor()
            last_id = ""
            while True:
                if last_id == "":
                    read_sql = f"SELECT f_id, f_name, f_create_time, f_update_time FROM `{self.table_old}` ORDER BY f_id LIMIT {self.batch_size}"
                    cursor.execute(read_sql)
                else:
                    read_sql = f"SELECT f_id, f_name, f_create_time, f_update_time FROM `{self.table_old}` WHERE f_id > %s ORDER BY f_id LIMIT {self.batch_size}"
                    cursor.execute(read_sql, (last_id,))
                rows = cursor.fetchall()
                if len(rows) == 0:
                    break
                last_id = rows[-1][0]
                insert_records = []
                for row in rows:
                    insert_records.append((row[0], row[1], row[2], row[3]))
                cursor.executemany(insert_sql, insert_records)
                total_migrated += len(rows)
                print(f"\rNumber of migrated data rows: {total_migrated}", end='')

            print("")
            logger.info(f"Data migration completed, migrated {total_migrated} rows.")

        except Exception as e:
            self.conn.rollback()
            logger.error("Error occurred during data migration: %s", e)
            raise
        finally:
            if cursor:
                cursor.close()

    def verify_data(self):
        """简单验证数据量是否一致"""
        logger.info("Verifying data consistency...")
        with self.conn.cursor() as cursor:
            cursor.execute(f"SELECT COUNT(*) FROM `{self.table_old}`")
            count_old = cursor.fetchone()[0]

            cursor.execute(f"SELECT COUNT(*) FROM `{self.table_new}`")
            count_new = cursor.fetchone()[0]

        if count_old != count_new:
            raise Exception(f"Data quantity is inconsistent! Original table: {count_old}, new table: {count_new}")

        logger.info("Data quantity verification passed.")

    def switch_tables(self):
        logger.info("Switching table names...")
        sql = f"""
        RENAME TABLE `{self.table_old}` TO `{self.table_backup}`,
                     `{self.table_new}` TO `{self.table_old}`
        """
        with self.conn.cursor() as cursor:
            cursor.execute(sql)
        logger.info("Table name switching completed, migration completed.")

    def run(self):
        try:
            logger.info("Database type: %s", self.db_type)
            if self.db_type == "unknown" or self.db_type == "dm8" or self.db_type == "kdb9":
                logger.warning("No migration required.")
                return

            if not self.check_idempotency():
                return

            self.create_new_table()
            self.migrate_data()
            self.verify_data()
            self.switch_tables()

        except Exception as e:
            logger.error("Migration script execution failed, transaction has been rolled back (if any).error: %s", e)
            raise

if __name__ == "__main__":
    try:
        migrator = TableMigrator()
        migrator.run()
    except Exception as e:
        logger.error("Migration script execution failed, error: %s", e)
        raise