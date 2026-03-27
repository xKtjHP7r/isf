#!/usr/bin/python3
# -*- coding: utf-8 -*-

import datetime
import uuid
import time
import json
from eisoo.tclients import TClient
from src.common.business_date import BusinessDate
from src.common.db.connector import DBConnector, ConnectorManager
from src.common.lib import (escape_key,
                            raise_exception,
                            check_start_limit,
                            generate_group_str,
                            generate_search_order_sql)
from src.modules.config_manage import ConfigManage
from src.modules.user_manage import UserManage
from src.modules.department_manage import DepartmentManage
from ShareMgnt.constants import NCT_ALL_USER_GROUP
from ShareMgnt.ttypes import (ncTGlobalRecycleRetentionConfig,
                              ncTAutoCleanConfig,
                              ncTAutoCleanObjInfo,
                              ncTAuditObjectType,
                              ncTShareMgntError)
from ShareMgnt.constants import (NCT_USER_ADMIN,
                                 NCT_USER_AUDIT,
                                 NCT_USER_SYSTEM,
                                 NCT_USER_SECURIT)

USER_OBJ = 1
DEPART_OBJ = 2

class DocAutoCleanManage(DBConnector):
    """
    个人文档自动清理策略管理
    """
    def __init__(self):
        """
        初始化
        """
        self.config_manage = ConfigManage()
        self.user_manage = UserManage()
        self.depart_manage = DepartmentManage()

    def __check_strategy_exist(self, strategyId, b_raise=True):
        """
        检查策略存在
        """
        sql = """
        SELECT `f_strategy_id`
        FROM `t_doc_auto_clean_strategy`
        WHERE `f_strategy_id` = %s
        """
        result = self.r_db.one(sql, strategyId)

        if not result:
            if b_raise:
                raise_exception(exp_msg=_("IDS_AUTO_CLEAN_CONFIG_NOT_EXIST"),
                                exp_num=ncTShareMgntError.NCT_AUTO_CLEAN_CONFIG_NOT_EXIST)
            else:
                return False
        else:
            return True

    def __check_not_exist_obj(self, users, departs):
        """
        获取不存在的对象列表抛错提示
        """
        userNameDict = {}
        departNameDict = {}
        userIdList = []
        departIdList = []
        notExistNames = []

        for user in users:
            userNameDict[user.id] = user.name
        for depart in departs:
            departNameDict[depart.id] = depart.name

        userIdList = list(userNameDict.keys())
        departIdList = list(departNameDict.keys())

        # 用户存在检查
        if userNameDict:
            userGroupStr = generate_group_str(list(userNameDict.keys()))
            existUserIds = []
            sql = """
            SELECT f_user_id FROM t_user WHERE f_user_id IN ({0})
            """.format(userGroupStr)
            results = self.r_db.all(sql)

            for r in results:
                existUserIds.append(r['f_user_id'])
            notExistUserIds = list(set(userIdList) - set(existUserIds))

            for notExistUserId in notExistUserIds:
                notExistNames.append(userNameDict[notExistUserId])

        # 部门存在检查
        if departNameDict:
            departGroupStr = generate_group_str(list(departNameDict.keys()))
            existDepartIds = []
            sql = """
            SELECT f_department_id FROM t_department WHERE f_department_id IN ({0})
            """.format(departGroupStr)
            results = self.r_db.all(sql)

            for r in results:
                existDepartIds.append(r['f_department_id'])
            notExistDepartIds = list(set(departIdList) - set(existDepartIds))

            for notExistDepartId in notExistDepartIds:
                notExistNames.append(departNameDict[notExistDepartId])

        # 不存在的对象抛错提示
        if notExistNames:
            errMsg = ",".join(notExistNames)
            errDetail = {}
            errDetail["names"] = notExistNames
            raise_exception(exp_msg=(_("IDS_NOT_EXIST") % errMsg),
                            exp_num=ncTShareMgntError.NCT_OBJ_NOT_EXIST,
                            exp_detail=json.dumps(errDetail, ensure_ascii=False))

    def __get_next_clean_time(self, cleanCycle, cleanCycleModifyTime):
        """
        获取下次清理时间
        """
        # 当天的0点
        currentTimeMin = datetime.datetime.combine(BusinessDate.now(), datetime.time.min)
        # 配置清理周期时间
        cleanCycleMT = datetime.datetime.fromtimestamp(cleanCycleModifyTime / 1000000)
        # 配置清理周期时的0点
        cleanCycleMTMin =  datetime.datetime.combine(cleanCycleMT, datetime.time.min)

        # 第一个清理周期是配置归档周期的第二天
        nextTimeMin = cleanCycleMTMin + datetime.timedelta(days=1)

        deltaDays = (currentTimeMin - nextTimeMin).days
        if deltaDays > 0: # 超过第一个清理周期
            nextDelta = cleanCycle - (deltaDays % cleanCycle)
            if nextDelta != cleanCycle: # 不为清理周期的整数倍
                nextTimeMin = currentTimeMin + datetime.timedelta(days=nextDelta)
            else:
                nextTimeMin = currentTimeMin
        return int(time.mktime(nextTimeMin.timetuple()) * 1000000)

    def __check_exist_in_other_strategy_obj(self, users, departs, strategyId):
        """
        获取已经配置策略的对象抛错提示
        """
        objDict = {}
        for user in users:
            objDict[user.id] = user.name
        for depart in departs:
            objDict[depart.id] = depart.name

        if objDict:
            objGroupStr = generate_group_str(list(objDict.keys()))
            sql = """
            SELECT `f_obj_id` FROM `t_doc_auto_clean_strategy` WHERE `f_strategy_id` != %s AND `f_obj_id` IN ({0})
            """.format(objGroupStr)
            results = self.r_db.all(sql, strategyId)

            existObjNames = []
            for r in results:
                existObjNames.append(objDict[r['f_obj_id']])

            if existObjNames:
                errMsg = ",".join(existObjNames)
                errDetail = {}
                errDetail["names"] = existObjNames
                raise_exception(exp_msg=(_("IDS_AUTO_CLEAN_CONFIG_EXIST") % errMsg),
                                exp_num=ncTShareMgntError.NCT_AUTO_CLEAN_CONFIG_EXIST,
                                exp_detail=json.dumps(errDetail, ensure_ascii=False))

    def set_doc_auto_clean_status(self, status):
        """
        开启/禁用自动清理策略
        """
        # 开启自动清理策略时，重置已有策略的清理时间
        if int(status):
            date = int(BusinessDate.time() * 1000000)
            update_sql = """
            UPDATE `t_doc_auto_clean_strategy` SET `f_clean_cycle_modify_time` = %s
            """
            self.w_db.query(update_sql, (date))

        self.config_manage.set_config('doc_auto_clean_status', int(status))

    def get_doc_auto_clean_status(self):
        """
        获取自动清理策略启用/禁用状态
        """
        return bool(int(self.config_manage.get_config('doc_auto_clean_status')))

    def get_global_recycle_retention_config(self):
        """
        获取管理员级别的回收站中数据保留时间
        """
        config_dict = json.loads(self.config_manage.get_config("global_recycle_retention_config"))
        config = ncTGlobalRecycleRetentionConfig()
        config.isEnable = config_dict['isEnable']
        config.days = config_dict['days']
        return config

    def set_global_recycle_retention_config(self, config):
        """
        设置管理员级别的回收站中数据保留时间
        """
        # 检查参数是否为空
        if config is None or config.isEnable is None or config.days is None:
            raise_exception(exp_msg=_("parameter is none"),
                        exp_num=ncTShareMgntError.NCT_PARAMETER_IS_NULL)

        # 检查参数合法性
        if config.days < 1:
            raise_exception(exp_msg=(_("INVALID_PARAM") % (config.days)),
                        exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

        # set接口只对 isEnable 和 days 做更改
        # 该配置后续需要迁移到EFAST中
        config_dict = json.loads(self.config_manage.get_config("global_recycle_retention_config"))
        old_config = ncTGlobalRecycleRetentionConfig()
        old_config.isEnable = config_dict['isEnable']
        old_config.days = config_dict['days']
        if (old_config.isEnable == config.isEnable == False) or   \
            ((old_config.isEnable == config.isEnable == True) and old_config.days == config.days):
                return

        config_dict["isEnable"] = config.isEnable
        config_dict["days"] = config.days

        self.config_manage.set_config("global_recycle_retention_config", json.dumps(config_dict))

    def add_auto_clean_config(self, config):
        """
        增加一条自动清理策略配置
        """
        if config is None:
            raise_exception(exp_msg=(_("INVALID_PARAM") % \
                                    ("ncTAutoCleanConfig is None")),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)
        if not config.users and not config.departs:
            raise_exception(exp_msg=(_("INVALID_PARAM") % \
                                    ("ncTAutoCleanConfig.users and departs both empty")),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)
        if not config.users:
            config.users = []
        if not config.departs:
            config.departs = []

        strategyId = str(uuid.uuid1())

        # 检查对象存在
        self.__check_not_exist_obj(config.users, config.departs)

        # 检查策略存在
        self.__check_exist_in_other_strategy_obj(config.users, config.departs, strategyId)

        # 检查数据清理周期的合法性
        if config.cleanCycleDays is None or config.cleanCycleDays < 1:
            raise_exception(exp_msg=(_("INVALID_PARAM") % config.cleanCycleDays),
                                exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

        # 检查数据保留时间是否小于数据清理周期
        clean_cycle_hours = config.cleanCycleDays * 24
        if config.enableRemainHours and (config.remainHours is None or config.remainHours <= 0 or config.remainHours >= clean_cycle_hours):
            raise_exception(exp_msg=(_("INVALID_PARAM") % str(config.remainHours)),
                                exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)
        # 新增策略时，若没有开启数据保留时间，则为默认策略的保留时间
        if not config.enableRemainHours:
            defaultConfig = self.get_auto_clean_config_by_objId(NCT_ALL_USER_GROUP)
            config.remainHours = defaultConfig.remainHours

        # 使用事务插入数据
        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()

        insert_sql = """
        INSERT INTO `t_doc_auto_clean_strategy`
        (`f_strategy_id`, `f_obj_id`, `f_obj_type`, `f_enable_remain_hours`, `f_remain_hours`, `f_clean_cycle_days`, `f_clean_cycle_modify_time`, `f_create_time`, `f_status`)
        VALUES(%s, %s, %s, %s, %s, %s, %s, %s, %s)
        """

        date = int(BusinessDate.time() * 1000000)
        try:
            # 插入用户配置
            for user in config.users:
                cursor.execute(insert_sql, (strategyId, user.id, USER_OBJ, config.enableRemainHours, config.remainHours, config.cleanCycleDays, date, date, 1))

            # 插入部门配置
            for depart in config.departs:
                cursor.execute(insert_sql, (strategyId, depart.id, DEPART_OBJ, config.enableRemainHours, config.remainHours, config.cleanCycleDays, date, date, 1))

            conn.commit()
        except Exception as e:
            conn.rollback()
            raise_exception(exp_msg=str(e),
                            exp_num=ncTShareMgntError.NCT_DB_OPERATE_FAILED)
        finally:
            cursor.close()
            conn.close()

        returnConfig = ncTAutoCleanConfig()
        returnConfig.strategyId = strategyId
        returnConfig.cleanNextTime = self.__get_next_clean_time(config.cleanCycleDays, date)

        return returnConfig

    def edit_auto_clean_config(self, config):
        """
        编辑一条自动清理策略配置
        """
        if not config:
            raise_exception(exp_msg=(_("INVALID_PARAM") % \
                                    ("ncTAutoCleanConfig is none or empty value")),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

        # 检查策略存在
        self.__check_strategy_exist(config.strategyId)

        # 检查数据清理周期的合法性
        if not config.cleanCycleDays or config.cleanCycleDays < 1:
            raise_exception(exp_msg=(_("INVALID_PARAM") % str(config.cleanCycleDays)),
                                exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

        # 获取已有策略配置
        oldStrategyItem, createTime = self.get_strategy_item_byId(config.strategyId)

        # 检查数据保留时间是否小于数据清理周期
        clean_cycle_hours = config.cleanCycleDays * 24
        if ((config.enableRemainHours or config.enableRemainHours != oldStrategyItem.enableRemainHours )
            and (config.remainHours >= clean_cycle_hours or config.remainHours is None)):
            raise_exception(exp_msg=_("INVALID_PARAM"),
                                exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

        # 默认策略直接更新并返回
        defaultConfig = self.get_auto_clean_config_by_objId(NCT_ALL_USER_GROUP)
        if config.strategyId == defaultConfig.strategyId:
            # 默认策略禁止变更配置对象
            if config.departs or len(config.users) != 1 or config.users[0].id != NCT_ALL_USER_GROUP:
                raise_exception(exp_msg=(_("INVALID_PARAM") % (str(config.users) + str(config.departs))),
                                exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)

            if defaultConfig.cleanCycleDays != config.cleanCycleDays or defaultConfig.status != config.status:
                config.cleanCycleModifyTime = int(BusinessDate.time() * 1000000)
            else:
                config.cleanCycleModifyTime = defaultConfig.cleanCycleModifyTime

            sql = """
            UPDATE `t_doc_auto_clean_strategy`
            SET `f_enable_remain_hours` = %s, `f_remain_hours` = %s, `f_clean_cycle_days` = %s, `f_clean_cycle_modify_time` = %s, `f_status` = %s
            WHERE `f_strategy_id` = %s
            """
            self.w_db.query(sql, config.enableRemainHours, config.remainHours, config.cleanCycleDays, config.cleanCycleModifyTime, config.status, config.strategyId)

            config.cleanNextTime = self.__get_next_clean_time(config.cleanCycleDays, config.cleanCycleModifyTime)

            return config

        if not config.users and not config.departs:
            raise_exception(exp_msg=(_("INVALID_PARAM") % \
                                    ("ncTAutoCleanConfig.users and departs both empty")),
                            exp_num=ncTShareMgntError.NCT_INVALID_PARAMTER)
        if not config.users:
            config.users = []
        if not config.departs:
            config.departs = []

        # 检查对象存在
        self.__check_not_exist_obj(config.users, config.departs)

        # 检查策略存在
        self.__check_exist_in_other_strategy_obj(config.users, config.departs, config.strategyId)

        # 自动清理周期有变化，更新清理周期修改时间
        if oldStrategyItem.cleanCycleDays != config.cleanCycleDays:
            config.cleanCycleModifyTime = int(BusinessDate.time() * 1000000)
        else:
            config.cleanCycleModifyTime = oldStrategyItem.cleanCycleModifyTime

        # 先删除旧配置，再全部新增
        self.delete_auto_clean_config(config.strategyId)

        # 使用事务插入数据
        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()

        insert_sql = """
        INSERT INTO `t_doc_auto_clean_strategy`
        (`f_strategy_id`, `f_obj_id`, `f_obj_type`, `f_enable_remain_hours`, `f_remain_hours`, `f_clean_cycle_days`, `f_clean_cycle_modify_time`, `f_create_time`, `f_status`)
        VALUES(%s, %s, %s, %s, %s, %s, %s, %s, %s)
        """

        try:
            # 插入用户配置
            for user in config.users:
                cursor.execute(insert_sql, (config.strategyId, user.id, USER_OBJ, config.enableRemainHours, config.remainHours, config.cleanCycleDays, config.cleanCycleModifyTime, createTime, 1))

            # 插入部门配置
            for depart in config.departs:
                cursor.execute(insert_sql, (config.strategyId, depart.id, DEPART_OBJ, config.enableRemainHours, config.remainHours, config.cleanCycleDays, config.cleanCycleModifyTime, createTime, 1))

            conn.commit()
        except Exception as e:
            conn.rollback()
            raise_exception(exp_msg=str(e),
                            exp_num=ncTShareMgntError.NCT_DB_OPERATE_FAILED)
        finally:
            cursor.close()
            conn.close()

        config.cleanNextTime = self.__get_next_clean_time(config.cleanCycleDays, config.cleanCycleModifyTime)

        return config

    def get_strategy_item_byId(self, strategyId):
        """
        根据策略Id获取策略创建时间
        """
        sql = """
        SELECT DISTINCT `f_strategy_id`, `f_enable_remain_hours`, `f_clean_cycle_days`, `f_clean_cycle_modify_time`, `f_create_time`
        FROM `t_doc_auto_clean_strategy`
        WHERE `f_strategy_id` = %s
        """
        result = self.r_db.one(sql, strategyId)
        config = ncTAutoCleanConfig()
        config.enableRemainHours = result['f_enable_remain_hours']
        config.cleanCycleDays = result['f_clean_cycle_days']
        config.cleanCycleModifyTime = result['f_clean_cycle_modify_time']
        return config, result['f_create_time']

    def delete_auto_clean_config(self, strategyId):
        """
        删除一条自动清理策略配置
        """
        # 禁止删除默认策略
        defaultConfig = self.get_auto_clean_config_by_objId(NCT_ALL_USER_GROUP)
        if strategyId == defaultConfig.strategyId:
            raise_exception(exp_msg=_("IDS_CANNOT_DELETE_DEFAULT_AUTO_CLEAN_CONFIG"),
                            exp_num=ncTShareMgntError.NCT_CAN_NOT_DELETE_DEFAULT_AUTO_CLEAN_CONFIG)

        sql = """
        DELETE FROM `t_doc_auto_clean_strategy` WHERE `f_strategy_id` = %s
        """
        self.w_db.query(sql, strategyId)

    def get_auto_clean_config_count(self, searchKey):
        """
        获取自动清理策略配置总数
        """
        # searchKey 为空表示获取所有
        result = -1
        if searchKey:
            esckey = "%%%s%%" % escape_key(searchKey)
            sql = """
            SELECT COUNT(DISTINCT `t_doc_auto_clean_strategy`.`f_strategy_id`) AS cnt
            FROM `t_doc_auto_clean_strategy`
            LEFT JOIN `t_user`
            ON `t_doc_auto_clean_strategy`.`f_obj_id` = `t_user`.`f_user_id`
            LEFT JOIN `t_department`
            ON `t_doc_auto_clean_strategy`.`f_obj_id` = `t_department`.`f_department_id`
            WHERE `t_user`.`f_display_name` LIKE %s OR `t_department`.`f_name` LIKE %s
            """
            result = self.r_db.one(sql, esckey, esckey)
        else:
            sql = """
            SELECT COUNT(DISTINCT f_strategy_id) AS cnt
            FROM t_doc_auto_clean_strategy
            """
            result = self.r_db.one(sql)

        return result['cnt']

    def search_auto_clean_config_by_page(self, start, limit, searchKey):
        """
        根据关键字(匹配用户名、显示名、部门名、组织名)搜索自动清理策略
        """
        # 检查参数
        limit_statement = check_start_limit(start, limit)

        # 增加搜索排序子句
        order_by_str = generate_search_order_sql(['t_doc_auto_clean_strategy.f_create_time'])

        # searchKey 为空表示获取所有
        results = []
        if searchKey:
            esckey = "%%%s%%" % escape_key(searchKey)
            search_sql = """
            SELECT `t_doc_auto_clean_strategy`.`f_strategy_id` AS `f_strategy_id`,
            MIN(`t_doc_auto_clean_strategy`.`f_create_time`) AS `min_create_time`
            FROM `t_doc_auto_clean_strategy`
            LEFT JOIN `t_user`
            ON `t_doc_auto_clean_strategy`.`f_obj_id` = `t_user`.`f_user_id`
            LEFT JOIN `t_department`
            ON `t_doc_auto_clean_strategy`.`f_obj_id` = `t_department`.`f_department_id`
            WHERE `t_user`.`f_display_name` LIKE %s OR `t_department`.`f_name` LIKE %s
            GROUP BY `f_strategy_id`
            ORDER BY `min_create_time` DESC
            {0}
            """.format(limit_statement)
            results = self.r_db.all(search_sql, esckey, esckey)
        else:
            search_sql = """
            SELECT `f_strategy_id`,
            MIN(`f_create_time`) AS `min_create_time`
            FROM `t_doc_auto_clean_strategy`
            WHERE 1=%s
            GROUP BY `f_strategy_id`
            ORDER BY `min_create_time` DESC
            {0}
            """.format(limit_statement)
            results = self.r_db.all(search_sql, 1)

        config_list = []
        for result in results:
            config = self.get_auto_clean_config_by_strategyId(result['f_strategy_id'])
            if config:
                config_list.append(config)

        return config_list

    def get_auto_clean_config_by_strategyId(self, strategyId):
        """
        获取策略Id相同的一组自动清理策略
        """
        sql = """
        SELECT `f_obj_id`, `f_obj_type`, `f_enable_remain_hours`, `f_remain_hours`, `f_clean_cycle_days`, `f_clean_cycle_modify_time`, `f_create_time`,`f_status`
        FROM `t_doc_auto_clean_strategy`
        WHERE `f_strategy_id` = %s
        """
        results = self.r_db.all(sql, strategyId)

        if not results:
            return None

        config = ncTAutoCleanConfig()
        config.strategyId = strategyId
        config.cleanCycleDays = results[0]['f_clean_cycle_days']
        config.cleanCycleModifyTime = results[0]['f_clean_cycle_modify_time']
        config.enableRemainHours = results[0]['f_enable_remain_hours']
        config.remainHours = results[0]['f_remain_hours']
        config.cleanNextTime = self.__get_next_clean_time(config.cleanCycleDays, config.cleanCycleModifyTime)
        config.createTime = results[0]['f_create_time']
        config.status = results[0]['f_status']

        users = []
        departs = []
        for result in results:
            cleanObject = ncTAutoCleanObjInfo()
            cleanObject.id = result['f_obj_id']

            # 用户
            if result['f_obj_type'] == ncTAuditObjectType.NCT_AUDIT_OBJECT_USER:
                if result['f_obj_id'] == NCT_ALL_USER_GROUP:
                    cleanObject.name = _("all user")
                else:
                    user_info = self.user_manage.get_user_by_id(result['f_obj_id'])
                    cleanObject.name = user_info.user.displayName
                users.append(cleanObject)
            # 部门
            elif result['f_obj_type'] == ncTAuditObjectType.NCT_AUDIT_OBJECT_DEPT:
                depart_info = self.depart_manage.get_department_info(result['f_obj_id'], b_include_org=True)
                cleanObject.name = depart_info.departmentName
                departs.append(cleanObject)

        config.users = users
        config.departs = departs
        return config

    def get_auto_clean_config_by_objId(self, objId):
        """
        根据对象Id获取单条原子策略
        """
        sql = """
        SELECT `f_strategy_id`, `f_obj_id`, `f_enable_remain_hours`, `f_remain_hours`, `f_clean_cycle_days`, `f_clean_cycle_modify_time`, `f_create_time`,`f_status`
        FROM `t_doc_auto_clean_strategy`
        WHERE `f_obj_id` = %s
        """
        result = self.r_db.one(sql, objId)

        if result:
            config = ncTAutoCleanConfig()
            config.strategyId = result['f_strategy_id']
            config.enableRemainHours = result['f_enable_remain_hours']
            config.remainHours = result['f_remain_hours']
            config.cleanCycleDays = result['f_clean_cycle_days']
            config.cleanCycleModifyTime = result['f_clean_cycle_modify_time']
            config.createTime = result['f_create_time']
            config.status = result['f_status']
            config.cleanNextTime = self.__get_next_clean_time(config.cleanCycleDays, config.cleanCycleModifyTime)
            config.users = []
            config.departs = []
            return config
        return None

    def get_auto_clean_config_by_userId(self, userId):
        """
        根据用户ID获取生效的本地策略，优先级：用户>子部门>父部门，相同层级部门取最新创建的
        """
        # 检测用户存在
        self.user_manage.check_user_exists(userId, raise_ex=True)

        # 获取默认策略
        defaultConfig = self.get_auto_clean_config_by_objId(NCT_ALL_USER_GROUP)

        # 快速判断，如果只有一条默认策略则直接返回
        sql = """
        SELECT COUNT(*) AS cnt FROM t_doc_auto_clean_strategy
        """
        cnt = self.r_db.one(sql)['cnt']
        if 1 == cnt:
            return defaultConfig

        # 1.用户策略
        userConfig = self.get_auto_clean_config_by_objId(userId)
        if userConfig:
            return userConfig

        # 2.部门策略
        # 获取该用户相关的部门树
        depart_tree = self.depart_manage.get_depart_tree_of_user(userId)

        # 遍历部门树，获取部门树所有部门的配置
        all_config_dict = {}
        for depart_id in depart_tree:
            config = self.get_auto_clean_config_by_objId(depart_id)
            if not config:
                continue

            # 所有部门的配置先置为有效
            config.valid = True
            all_config_dict[depart_id] = config

        # 每个叶子节点向上遍历
        for depart_id in depart_tree:
            if depart_tree[depart_id].subDepartIds:
                continue

            # 向上遍历
            all_path_depart_ids = [depart_id]
            parent_id = depart_tree[depart_id].parentDepartId
            while parent_id:
                all_path_depart_ids.append(parent_id)
                parent_id = depart_tree[parent_id].parentDepartId

            # 遍历路径，如果路径上有设置的配置，则剩余的路径上存在的配置必须置为失效
            for i in range(0, len(all_path_depart_ids)):
                # 不存在配置，则继续向上遍历
                if all_path_depart_ids[i] not in all_config_dict:
                    continue

                # 存在配置，则需要将剩余路径上存在的配置设置为失效
                for j in range(i + 1, len(all_path_depart_ids)):
                    remain_depart_id = all_path_depart_ids[j]
                    if remain_depart_id in all_config_dict:
                        all_config_dict[remain_depart_id].valid = False

        # 针对所有部门配置进行计算，取有效的，时间最近的
        departConfig = None
        for depart_id in all_config_dict:
            if not all_config_dict[depart_id].valid:
                continue

            # 第一次设置departConfig
            if not departConfig:
                departConfig = all_config_dict[depart_id]
                continue

            # 比result新，则更新为该配置
            if all_config_dict[depart_id].createTime > departConfig.createTime:
                departConfig = all_config_dict[depart_id]

        # 删除掉动态添加的valid属性
        if departConfig:
            del departConfig.valid
            return departConfig

        # 3.默认策略
        return defaultConfig

    def get_all_auto_clean_userId(self):
        """
        获取所有待清理用户id
        """
        userIds = set()
        defaultsql = """
        SELECT f_status, f_obj_id
        FROM t_doc_auto_clean_strategy
        WHERE f_obj_id='-2'
        """
        defaultresult = self.r_db.one(defaultsql)
        if defaultresult:
            if defaultresult['f_status']:
                sql = """
                SELECT f_user_id FROM t_user
                WHERE f_user_id != '{0}'
                AND f_user_id != '{1}'
                AND f_user_id != '{2}'
                AND f_user_id != '{3}'
                """.format(NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT)
                results = self.r_db.all(sql)
                for result in results:
                    objId = result['f_user_id']
                    userIds.add(objId)
                return list(userIds)

        sql = """
        SELECT f_obj_id, f_obj_type
        FROM t_doc_auto_clean_strategy
        WHERE f_obj_id!='-2'
        """
        results = self.r_db.all(sql)

        if not results:
            return []

        for result in results:
            objId = result['f_obj_id']

            # 用户
            if result['f_obj_type'] == ncTAuditObjectType.NCT_AUDIT_OBJECT_USER:
                userIds.add(objId)
            # 部门
            elif result['f_obj_type'] == ncTAuditObjectType.NCT_AUDIT_OBJECT_DEPT:
                userIds.update(self.depart_manage.get_all_users_of_depart(objId))

        return list(userIds)
