#!/usr/bin/python3
# -*- coding:utf-8 -*-
"""
角色管理类
"""
import uuid
from src.common.db.connector import (DBConnector)
from src.modules.user_manage import UserManage
from src.common.db.connector import ConnectorManager
from src.modules.department_manage import DepartmentManage
from src.common.lib import (raise_exception,
                            check_email, check_name, generate_group_str)
from ShareMgnt.ttypes import (
    ncTShareMgntError, ncTRoleInfo, ncTRoleMemberInfo)
from src.modules.config_manage import ConfigManage
from src.common import global_info
from ShareMgnt.constants import (NCT_USER_ADMIN,
                                 NCT_USER_AUDIT,
                                 NCT_USER_SYSTEM,
                                 NCT_USER_SECURIT,
                                 NCT_SYSTEM_ROLE_SUPPER,
                                 NCT_SYSTEM_ROLE_ADMIN,
                                 NCT_SYSTEM_ROLE_SECURIT,
                                 NCT_SYSTEM_ROLE_AUDIT,
                                 NCT_SYSTEM_ROLE_ORG_MANAGER,
                                 NCT_SYSTEM_ROLE_ORG_AUDIT)
from EThriftException.ttypes import ncTException
from src.common.sharemgnt_logger import ShareMgnt_Log
from src.common.http import pub_nsq_msg
import json

TOPIC_ORG_MANAGER_SETTED = "sharemgnt.org_manager.setted"
TOPIC_ORG_MANAGER_DELETED = "sharemgnt.org_manager.deleted"


class RoleManage(DBConnector):
    def __init__(self):
        """
        """
        self.user_manage = UserManage()
        self.department_manage = DepartmentManage()
        self.config_manage = ConfigManage()
        self.system_role = [NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_ADMIN, NCT_SYSTEM_ROLE_SECURIT,
                            NCT_SYSTEM_ROLE_AUDIT, NCT_SYSTEM_ROLE_ORG_MANAGER, NCT_SYSTEM_ROLE_ORG_AUDIT]

    def __check_role_name_valid(self, name):
        """
        检查角色名称是否合法
        """

        # 检查角色名称是否合法
        if not check_name(name):
            raise_exception(exp_msg=_("IDS_INVALID_ROLE_NAME"),
                            exp_num=ncTShareMgntError.NCT_INVALID_ROLE_NAME)

    def __check_role_info(self, roleInfo):
        """
        检查角色信息
        """
        # 检查创建者是否存在
        self.user_manage.check_user_exists(roleInfo.creatorId, True)

        # 去掉空格
        roleInfo.name = roleInfo.name.strip()

        # 检查角色名是否存在
        if roleInfo.name and self.__check_role_name_exist(roleInfo.name):
            raise_exception(exp_msg=_("IDS_ROLE_NAME_EXIST"),
                            exp_num=ncTShareMgntError.NCT_ROLE_NAME_EXIST)

        self.__check_role_name_valid(roleInfo.name)

        # 检查职能描述是否为None
        if roleInfo.description is None:
            roleInfo.description = ""

    def __check_role_name_exist(self, name, role_id=None):
        """
        检查角色名是否存在
        """
        # 检查角色名是否和内置系统角色名冲突
        if name.lower() in global_info.SYSTEM_ROLE_NAMES:
            return True

        sql = """
        SELECT f_role_id
        FROM `t_role`
        WHERE `f_name` = %s
        """
        result = self.r_db.one(sql, name)
        if not result:
            return False
        if role_id and role_id == result['f_role_id']:
            return False
        return True

    def add(self, roleInfo):
        """
        添加角色
        """
        # 检查角色信息
        self.__check_role_info(roleInfo)

        # 检查用户是否可以创建角色
        self.__check_creator(roleInfo.creatorId)

        # 生成一条唯一id
        roleInfo.id = str(uuid.uuid1())

        # 保存数据到数据库
        self.add_role_info_to_db(roleInfo)

        return roleInfo.id

    def __check_creator(self, creatorId):
        """
        检查创建者
        """
        # 获取用户角色
        user_roles = self.get_user_role_id(creatorId)

        # 只有超级管理员、安全管理员、组织管理员可以创建角色
        user_roles = set(user_roles)
        validate_role = set([NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_SECURIT,
                            NCT_SYSTEM_ROLE_ORG_MANAGER])

        # 取交集
        if not (validate_role & user_roles):
            raise_exception(exp_msg=_("IDS_INVALID_CREATOR"),
                            exp_num=ncTShareMgntError.NCT_INVALID_CREATOR)

    def is_user_has_role(self, userId, roleIds):
        """
        roleIds: 角色集合
        判断用户是否拥有指定角色集合中的一种或多种角色
        """
        # 获取用户角色
        user_roles = self.get_user_role_id(userId)

        # 取交集
        if set(user_roles) & set(roleIds):
            return True
        return False

    def add_role_info_to_db(self, roleInfo):
        """
        保存数据到数据库
        """
        sql = """
        INSERT INTO `t_role`
        (`f_role_id`, `f_name`, `f_description`, `f_creator_id`)
        VALUES(%s, %s, %s, %s)
        """
        self.w_db.query(sql, roleInfo.id, roleInfo.name,
                        roleInfo.description, roleInfo.creatorId)

    def change_supper_role(self, status):
        """
        开启三权分立后，超级管理员角色自动变为系统管理员
        关闭则清空三权分立角色
        """
        # 使用事务修改数据
        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()
        try:
            if status:
                # 开启三权分立情况下，删除超级管理员角色，增加系统管理员、安全管理员和审计管理员
                sql = """
                DELETE
                FROM `t_role`
                WHERE f_role_id = %s
                """
                cursor.execute(sql, (NCT_SYSTEM_ROLE_SUPPER,))

                sql = """
                DELETE
                FROM `t_user_role_relation`
                WHERE f_role_id = %s
                """
                cursor.execute(sql, (NCT_SYSTEM_ROLE_SUPPER,))

                self.__replace_role(cursor, NCT_SYSTEM_ROLE_ADMIN, 2, "admin")
                self.__replace_role(
                    cursor, NCT_SYSTEM_ROLE_SECURIT, 3, "security")
                self.__replace_role(cursor, NCT_SYSTEM_ROLE_AUDIT, 4, "audit")

                # 增加admin与系统管理员关联
                self.__add_role_relation(
                    cursor, NCT_USER_ADMIN, NCT_SYSTEM_ROLE_ADMIN)

                # 增加securit与安全管理员关联
                self.__add_role_relation(
                    cursor, NCT_USER_SECURIT, NCT_SYSTEM_ROLE_SECURIT)

                # 增加audit与审计管理员关联
                self.__add_role_relation(
                    cursor, NCT_USER_AUDIT, NCT_SYSTEM_ROLE_AUDIT)
            else:
                # 关闭三权分立情况下，删除系统管理员、安全管理员和审计管理员，增加超级管理员角色
                sql = """
                DELETE
                FROM `t_role`
                WHERE f_role_id in (%s, %s, %s)
                """
                cursor.execute(sql, (NCT_SYSTEM_ROLE_ADMIN, NCT_SYSTEM_ROLE_SECURIT,
                                     NCT_SYSTEM_ROLE_AUDIT))

                # 删除三权分立角色关联
                sql = """
                DELETE
                FROM `t_user_role_relation`
                WHERE `f_role_id` in (%s, %s, %s)
                """
                cursor.execute(sql, (NCT_SYSTEM_ROLE_ADMIN, NCT_SYSTEM_ROLE_SECURIT,
                                     NCT_SYSTEM_ROLE_AUDIT))

                self.__replace_role(
                    cursor, NCT_SYSTEM_ROLE_SUPPER, 1, "supper")

                # 增加超级管理员角色关联
                self.__add_role_relation(
                    cursor, NCT_USER_ADMIN, NCT_SYSTEM_ROLE_SUPPER)
            conn.commit()
        except Exception as e:
            conn.rollback()
            raise_exception(exp_msg=str(e),
                            exp_num=ncTShareMgntError.NCT_DB_OPERATE_FAILED)
        finally:
            cursor.close()
            conn.close()

    def __replace_role(self, cursor, role_id, priority, description):
        """
        替换角色信息
        """

        check_role_sql = """
        select f_priority from t_role where f_role_id = %s
        """

        insert_role_sql = """
        insert into t_role (f_role_id, f_priority, f_description) values (%s, %s, %s);
        """

        update_role_sql = """
        update t_role set f_priority = %s, f_description = %s where f_role_id = %s
        """

        cursor.execute(check_role_sql, (role_id,))
        result = cursor.fetchall()

        if result:
            cursor.execute(update_role_sql, (priority, description, role_id))
        else:
            cursor.execute(insert_role_sql, (role_id, priority, description))

    def __add_role_relation(self, cursor, user_id, role_id):
        """
        替换角色信息
        """

        check_role_relation_sql = """
        select f_role_id from t_user_role_relation where f_user_id = %s and f_role_id = %s
        """

        insert_role_relation_sql = """
        insert into t_user_role_relation (f_user_id, f_role_id) values (%s, %s);
        """

        if cursor:
            cursor.execute(check_role_relation_sql, (user_id, role_id))
            result = cursor.fetchall()

            if not result:
                cursor.execute(insert_role_relation_sql, (user_id, role_id))

            return

        result = self.r_db.one(check_role_relation_sql, user_id, role_id)
        if not result:
            self.w_db.query(insert_role_relation_sql, user_id, role_id)

    def get_user_role_id(self, userId):
        """
        获取用户角色id
        """
        sql = """
        SELECT t.f_role_id
        FROM `t_role` as t
        INNER JOIN t_user_role_relation as r
        ON t.f_role_id = r.f_role_id
        WHERE r.f_user_id = %s
        """
        results = self.r_db.all(sql, userId)
        user_roles = [result['f_role_id'] for result in results]
        return user_roles

    def get_user_role(self, userId):
        """
        获取用户角色
        """
        sql = """
        SELECT t.f_role_id, t.f_description, t.f_name
        FROM `t_role` as t
        INNER JOIN t_user_role_relation as r
        ON t.f_role_id = r.f_role_id
        WHERE r.f_user_id = %s
        ORDER BY t.f_priority
        """
        results = self.r_db.all(sql, userId)
        role_infos = []
        for res in results:
            role_info = ncTRoleInfo()
            role_info.name = res["f_name"]
            role_info.description = res["f_description"]
            role_info.id = res["f_role_id"]
            role_infos.append(role_info)
        return role_infos

    def get_user_role_ids_order_by(self, userId):
        """
        获取用户角色信息,根据优先级和名称排序
        """
        sql = """
        SELECT t.f_role_id, t.f_description, t.f_name
        FROM `t_role` as t
        INNER JOIN t_user_role_relation as r
        ON t.f_role_id = r.f_role_id
        WHERE r.f_user_id = %s
        ORDER BY t.f_priority, t.f_name
        """
        results = self.r_db.all(sql, userId)
        role_ids = []
        for res in results:
            role_ids.append(res["f_role_id"])
        return role_ids

    def get(self, userId):
        """
        获取角色
        """
        # 检查用户是否存在
        self.user_manage.check_user_exists(userId, True)

        # 先获取用户具有的所有角色
        user_roles = self.get_user_role_id(userId)
        # 如果没有任何角色
        if not user_roles:
            return []

        sql = """
        SELECT f_name, f_description, f_role_id, f_creator_id
        FROM t_role
        """
        filter_str = ""
        black_list = []
        White_list = []
        # 如果是超级管理员可以看到所有
        if NCT_SYSTEM_ROLE_SUPPER in user_roles:
            if userId != NCT_USER_ADMIN:
                black_list.append(NCT_SYSTEM_ROLE_SUPPER)
        # 如果是安全管理员角色, 不能看到系统管理员、安全管理员、审计管理员
        elif NCT_SYSTEM_ROLE_SECURIT in user_roles:
            black_list.extend([NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_ADMIN,
                               NCT_SYSTEM_ROLE_SECURIT, NCT_SYSTEM_ROLE_AUDIT])
        # 如果是组织管理员角色, 不能看到超级管理员、系统管理员、安全管理员、审计管理员、组织审计员
        elif NCT_SYSTEM_ROLE_ORG_MANAGER in user_roles:
            black_list.extend([NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_ADMIN,
                               NCT_SYSTEM_ROLE_SECURIT, NCT_SYSTEM_ROLE_AUDIT,
                               NCT_SYSTEM_ROLE_ORG_AUDIT])
            # 如果既是组织审计员，则可以看到组织审计员角色
            if NCT_SYSTEM_ROLE_ORG_AUDIT in user_roles:
                black_list.remove(NCT_SYSTEM_ROLE_ORG_AUDIT)
        # 如果是组织审计管理员角色, 只能看到组织审计员
        elif NCT_SYSTEM_ROLE_ORG_AUDIT in user_roles:
            White_list.extend([NCT_SYSTEM_ROLE_ORG_AUDIT])

        # 其他的角色不能看到超级管理员、安全管理员、审计管理员、组织审计员
        else:
            return []

        # 白名单中有数据表明只能查看白名单中的角色
        if White_list:
            filter_str = generate_group_str(White_list)
            sql = sql + " WHERE f_role_id in (%s)" % (filter_str)
        # 黑名单中有数据表明不能查看的角色
        elif black_list:
            filter_str = generate_group_str(black_list)
            sql = sql + " WHERE f_role_id not in (%s)" % (filter_str)

        # 增加排序
        sql = sql + " ORDER BY f_priority, upper(f_name)"

        results = self.r_db.all(sql)

        role_infos = []
        batch_creatorids = set()
        for res in results:
            role_info = ncTRoleInfo()
            role_info.name = res["f_name"]
            role_info.description = res["f_description"]
            role_info.id = res["f_role_id"]
            role_info.creatorId = res["f_creator_id"]
            role_info.displayName = ''
            if res["f_creator_id"]:
                batch_creatorids.add(res["f_creator_id"])
            role_infos.append(role_info)

        # 获取创建者显示名称
        if batch_creatorids:
            batch_user_infos = self.user_manage.get_batch_user_infos_by_id(
                batch_creatorids)
            for role_info in role_infos:
                if role_info.creatorId and role_info.creatorId in batch_user_infos:
                    role_info.displayName = batch_user_infos[role_info.creatorId]["f_display_name"]
        return role_infos

    def __check_role_id_exist(self, roleId):
        """
        检查角色id是否存在
        """
        sql = """
        SELECT count(*) as cnt
        FROM `t_role`
        WHERE `f_role_id` = %s
        """
        result = self.r_db.one(sql, roleId)
        return True if result['cnt'] != 0 else False

    def __check_user_can_manage_member(self, userId, user_roles, roleId, memberId):
        """
        检查成员是否在用户管辖范围内
        """
        # 如果用户是超级管理员、安全管理员则不检查
        can_manage_all_user_roles = [
            NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_SECURIT]
        if set(user_roles) & set(can_manage_all_user_roles):
            return True

        # 如果只是组织管理员，并且所设置的角色不是组织审计员时，需按组织管理员的范围检查
        if (NCT_SYSTEM_ROLE_ORG_MANAGER in user_roles) and (NCT_SYSTEM_ROLE_ORG_AUDIT != roleId):
            # 先获取用户的所有部门，在获取管理员所能管理的所有部门取交集:
            dept_ids = self.user_manage.get_all_path_dept_id(memberId)
            manager_dept_ids = self.department_manage.get_supervisory_all_departids(
                userId, NCT_SYSTEM_ROLE_ORG_MANAGER)
            if set(dept_ids) & set(manager_dept_ids):
                return True

        # 如果只是组织审计员，并且所设置的角色是组织审计员时，需按组织审计员的范围检查
        if (NCT_SYSTEM_ROLE_ORG_AUDIT in user_roles) and (NCT_SYSTEM_ROLE_ORG_AUDIT == roleId):
            # 先获取用户的所有部门，在获取管理员所能管理的所有部门取交集:
            dept_ids = self.user_manage.get_all_path_dept_id(memberId)
            manager_dept_ids = self.department_manage.get_supervisory_all_departids(
                userId, NCT_SYSTEM_ROLE_ORG_AUDIT)
            if set(dept_ids) & set(manager_dept_ids):
                return True
        return False

    def __check_user_can_oprate_role_member(self, userId, user_roles, roleId, memberId):
        """
        检查用户是否可以操作角色成员
        """
        # 检查成员是否在用户的管辖范围内
        if not self.__check_user_can_manage_member(userId, user_roles, roleId, memberId):
            return False

        # 超级管理员只有admin才可以操作
        if roleId == NCT_SYSTEM_ROLE_SUPPER:
            if userId != NCT_USER_ADMIN:
                return False

        # 系统管理员、安全管理员、审计管理员不可以操作
        elif roleId in [NCT_SYSTEM_ROLE_ADMIN,
                        NCT_SYSTEM_ROLE_SECURIT,
                        NCT_SYSTEM_ROLE_AUDIT]:
            return False

        # 组织管理员角色，只允许超级管理员、安全管理员、组织管理员操作
        elif roleId == NCT_SYSTEM_ROLE_ORG_MANAGER:
            can_set_roles = [NCT_SYSTEM_ROLE_SUPPER,
                             NCT_SYSTEM_ROLE_SECURIT,
                             NCT_SYSTEM_ROLE_ORG_MANAGER]
            if not (set(user_roles) & set(can_set_roles)):
                return False
            # 只有超级管理员可以操作自身组织管理员角色
            if userId == memberId:
                if not (set(user_roles) & set([NCT_SYSTEM_ROLE_SUPPER])):
                    return False

        # 组织审计员角色，只允许超级管理员、安全管理员和组织审计员操作
        elif roleId == NCT_SYSTEM_ROLE_ORG_AUDIT:
            can_set_roles = [NCT_SYSTEM_ROLE_SUPPER,
                             NCT_SYSTEM_ROLE_SECURIT,
                             NCT_SYSTEM_ROLE_ORG_AUDIT]
            if not (set(user_roles) & set(can_set_roles)):
                return False
            # 只有超级管理员可以操作自身组织审计员角色
            if userId == memberId:
                if not (set(user_roles) & set([NCT_SYSTEM_ROLE_SUPPER])):
                    return False
        else:
            # 剩余所有角色只允许超级管理员、安全管理员、组织管理员操作
            can_set_roles = [NCT_SYSTEM_ROLE_SUPPER,
                             NCT_SYSTEM_ROLE_SECURIT,
                             NCT_SYSTEM_ROLE_ORG_MANAGER]
            if not (set(user_roles) & set(can_set_roles)):
                return False

        return True

    def __set_role_member(self, userId, roleId, memberInfo):
        """
        检查角色成员
        """
        # 先获取用户具有的所有角色
        user_roles = self.get_user_role_id(userId)

        # 检查用户是否可以操作角色成员
        if not self.__check_user_can_oprate_role_member(userId, user_roles, roleId,
                                                        memberInfo.userId):
            raise_exception(exp_msg=_("IDS_INVALID_OPERATOR"),
                            exp_num=ncTShareMgntError.NCT_INVALID_OPERATOR)

        # 如果是组织管理员
        try:
            if roleId == NCT_SYSTEM_ROLE_ORG_MANAGER:
                depart_ids = memberInfo.manageDeptInfo.departmentIds
                # 如果是超级管理员、安全管理员不需要根据管理员id检查范围
                if set(user_roles) & set([NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_SECURIT]):
                    self.department_manage.set_responsible_person(
                        memberInfo.userId, depart_ids)
                else:
                    self.department_manage.set_responsible_person(memberInfo.userId, depart_ids,
                                                                  userId)

                # 发送事件消息
                pub_nsq_msg(TOPIC_ORG_MANAGER_SETTED, {"user_id": memberInfo.userId})

            if roleId == NCT_SYSTEM_ROLE_ORG_AUDIT:
                depart_ids = memberInfo.manageDeptInfo.departmentIds
                # 如果是超级管理员、安全管理员不需要根据管理员id检查范围
                if set(user_roles) & set([NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_SECURIT]):
                    self.department_manage.set_audit_person(
                        memberInfo.userId, depart_ids)
                else:
                    self.department_manage.set_audit_person(memberInfo.userId, depart_ids,
                                                            userId)

            self.__add_role_relation(None, memberInfo.userId, roleId)

        except Exception as ex:
            ShareMgnt_Log(str(ex))
            raise ex

    def set_member(self, userId, roleId, memberInfo):
        """
        设置成员包含添加和编辑成员
        """
        # 检查用户是否存在
        self.user_manage.check_user_exists(userId, True)

        # 检查角色是否存在
        if not self.__check_role_id_exist(roleId):
            raise_exception(exp_msg=_("IDS_ROLE_NOT_EXIST"),
                            exp_num=ncTShareMgntError.NCT_ROLE_NOT_EXIST)

        # 检查成员是否存在
        self.user_manage.check_user_exists(memberInfo.userId, True)

        # 不支持赋予内置账号其余角色
        if memberInfo.userId in [NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT]:
            raise_exception(exp_msg=_("IDS_INVALID_MEMBER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_MEMBER)

        # 检查角色成员设置
        self.__set_role_member(userId, roleId, memberInfo)

    def get_role_creator_id(self, roleId):
        """
        获取角色创建者id
        """
        sql = """
        SELECT f_creator_id
        FROM `t_role`
        WHERE `f_role_id` = %s
        """
        result = self.r_db.one(sql, roleId)
        return result['f_creator_id'] if result else ''

    def get_role_name_by_id(self, roleId):
        """
        根据用户角色获取用户名称
        """
        sql = """
        SELECT f_name
        FROM `t_role`
        WHERE `f_role_id` = %s
        """
        result = self.r_db.one(sql, roleId)
        return result['f_name'] if result else ''

    def edit(self, userId, roleInfo):
        """
        编辑角色
        """
        # 检查用户是否存在
        self.user_manage.check_user_exists(userId, True)

        # 检查角色id是否存在
        if not self.__check_role_id_exist(roleInfo.id):
            raise_exception(exp_msg=_("IDS_ROLE_NOT_EXIST"),
                            exp_num=ncTShareMgntError.NCT_ROLE_NOT_EXIST)

        # 判断角色名是否发生变更
        role_name = self.get_role_name_by_id(roleInfo.id)

        # 检查名称是否变更，如果变更需要增加名称合法性检查
        if role_name != roleInfo.name:
            if roleInfo.name and self.__check_role_name_exist(roleInfo.name, roleInfo.id):
                raise_exception(exp_msg=_("IDS_ROLE_NAME_EXIST"),
                                exp_num=ncTShareMgntError.NCT_ROLE_NAME_EXIST)

            self.__check_role_name_valid(roleInfo.name)

        # 系统角色不允许编辑和删除
        if roleInfo.id in self.system_role:
            raise_exception(exp_msg=_("IDS_SYS_ROLE_CANNOT_SET_OR_DELETE"),
                            exp_num=ncTShareMgntError.NCT_SYS_ROLE_CANNOT_SET_OR_DELETE)

        # 检查用户是否允许编辑
        self.__check_user_can_manage_role(userId, roleInfo.id)

        # 更新角色信息
        self.update_role(roleInfo)

    def __check_user_can_manage_role(self, userId, roleId):
        """
        检查用户是否可以管理角色
        """
        # 检查用户角色是否允许编辑，允许超级管理员或安全管理员编辑所有，只允许组织管理员编辑自己创建的
        user_roles = self.get_user_role_id(userId)

        # 获取角色创建者
        creator_id = self.get_role_creator_id(roleId)
        can_manage = False
        if NCT_SYSTEM_ROLE_SUPPER in user_roles or NCT_SYSTEM_ROLE_SECURIT in user_roles:
            can_manage = True
        elif NCT_SYSTEM_ROLE_ORG_MANAGER in user_roles and userId == creator_id:
            can_manage = True

        if not can_manage:
            raise_exception(exp_msg=_("IDS_INVALID_OPERATOR"),
                            exp_num=ncTShareMgntError.NCT_INVALID_OPERATOR)

    def update_role(self, roleInfo):
        """
        更新角色信息
        """
        sql = """
        UPDATE `t_role`
        set `f_name` = %s, `f_description` = %s
        WHERE `f_role_id` = %s
        """
        self.w_db.query(sql, roleInfo.name, roleInfo.description, roleInfo.id)

    def delete(self, userId, roleId):
        """
        删除角色信息
        """
        # 检查用户是否存在
        self.user_manage.check_user_exists(userId, True)

        # 检查角色id是否存在
        if not self.__check_role_id_exist(roleId):
            raise_exception(exp_msg=_("IDS_ROLE_NOT_EXIST"),
                            exp_num=ncTShareMgntError.NCT_ROLE_NOT_EXIST)

        # 系统角色不允许编辑和删除
        if roleId in self.system_role:
            raise_exception(exp_msg=_("IDS_SYS_ROLE_CANNOT_SET_OR_DELETE"),
                            exp_num=ncTShareMgntError.NCT_SYS_ROLE_CANNOT_SET_OR_DELETE)

        # 判断用户是否可以执行删除操作
        self.__check_user_can_manage_role(userId, roleId)

        # 先删除成员，再删除记录
        sql = """
        SELECT f_user_id, f_role_id
        FROM `t_user_role_relation`
        WHERE `f_role_id` = %s
        """
        results = self.r_db.all(sql, roleId)
        for result in results:
            # 删除成员
            self.__delete_member(
                userId, result["f_role_id"], result["f_user_id"], True)

        sql = """
        DELETE
        FROM `t_role`
        WHERE `f_role_id` = %s
        """
        self.w_db.query(sql, roleId)

    def __delete_role_relation(self, roleId, memberId):
        """
        删除用户角色关系
        """
        sql = """
        DELETE
        FROM `t_user_role_relation`
        WHERE `f_role_id` = %s and `f_user_id` = %s
        """
        self.w_db.query(sql, roleId, memberId)

        # 如果角色是超级管理员、系统管理员、安全管理员、审计管理员需要将邮箱关联属性删掉
        if roleId in [NCT_SYSTEM_ROLE_SUPPER, NCT_SYSTEM_ROLE_ADMIN,
                      NCT_SYSTEM_ROLE_SECURIT, NCT_SYSTEM_ROLE_AUDIT]:
            sql = """
            DELETE
            FROM `t_user_role_attribute`
            WHERE `f_user_id` = %s
            """
            self.w_db.query(sql, memberId)

    def __delete_member(self, userId, roleId, memberId, forceDelete=False):
        """
        内部删除角色成员
        """
        try:
            has_except = False
            # 根据角色删除策略配置
            if roleId == NCT_SYSTEM_ROLE_ORG_MANAGER:
                try:
                    self.department_manage.cancel_responsible_person(
                        memberId, userId)
                except ncTException as ex:
                    if ex.errID != ncTShareMgntError.NCT_USER_NOT_ADMIN:
                        raise ex

                # 发送事件消息
                pub_nsq_msg(TOPIC_ORG_MANAGER_DELETED, {"user_id": memberId})

            if roleId == NCT_SYSTEM_ROLE_ORG_AUDIT:
                self.department_manage.cancel_audit_person(memberId, userId)

            # 删除用户角色关系
            self.__delete_role_relation(roleId, memberId)

        except Exception as ex:
            ShareMgnt_Log(str(ex))
            has_except = True
            raise ex

        finally:
            # 如果有异常则需要判断是否强制删除
            if has_except and forceDelete:
                # 删除用户角色关系
                self.__delete_role_relation(roleId, memberId)

    def __check_member_in_role(self, roleId, memberId, raise_ex=True):
        """
        检查成员在角色内
        """
        sql = """
        SELECT f_user_id
        FROM `t_user_role_relation`
        WHERE `f_role_id` = %s and `f_user_id` = %s
        """
        result = self.r_db.one(sql, roleId, memberId)
        if not result:
            if raise_ex:
                raise_exception(exp_msg=_("IDS_ROLE_MEMBER_NOT_EXIST"),
                                exp_num=ncTShareMgntError.NCT_ROLE_MEMBER_NOT_EXIST)
            else:
                return False
        return True

    def delete_member(self, userId, roleId, memberId):
        """
        删除角色成员
        """
        # 检查用户是否存在
        self.user_manage.check_user_exists(userId, True)
        self.user_manage.check_user_exists(memberId, True)
        # 检查角色id是否存在
        if not self.__check_role_id_exist(roleId):
            raise_exception(exp_msg=_("IDS_ROLE_NOT_EXIST"),
                            exp_num=ncTShareMgntError.NCT_ROLE_NOT_EXIST)

        self.__check_member_in_role(roleId, memberId)
        # 不允许删除内置账号
        if memberId in [NCT_USER_ADMIN, NCT_USER_AUDIT, NCT_USER_SYSTEM, NCT_USER_SECURIT]:
            raise_exception(exp_msg=_("IDS_INVALID_MEMBER"),
                            exp_num=ncTShareMgntError.NCT_INVALID_MEMBER)

        # 检查用户是否可以操作角色成员
        user_roles = self.get_user_role_id(userId)
        if not self.__check_user_can_oprate_role_member(userId, user_roles, roleId, memberId):
            raise_exception(exp_msg=_("IDS_INVALID_OPERATOR"),
                            exp_num=ncTShareMgntError.NCT_INVALID_OPERATOR)

        self.__delete_member(userId, roleId, memberId)

    def get_member(self, userId, roleId, memberId=None):
        """
        获取角色成员
        """
        try:
            # 检查用户是否存在
            self.user_manage.check_user_exists(userId, True)

            # 检查角色id是否存在
            if not self.__check_role_id_exist(roleId):
                raise_exception(exp_msg=_("IDS_ROLE_NOT_EXIST"),
                                exp_num=ncTShareMgntError.NCT_ROLE_NOT_EXIST)
            if memberId:
                sql = """
                SELECT f_user_id
                FROM `t_user_role_relation`
                WHERE `f_role_id` = %s and `f_user_id` = %s
                """
                results = self.r_db.all(sql, roleId, memberId)
            else:
                sql = """
                SELECT f_user_id
                FROM `t_user_role_relation`
                WHERE `f_role_id` = %s and `f_user_id` not in (%s, %s, %s)
                """
                results = self.r_db.all(
                    sql, roleId, NCT_USER_ADMIN, NCT_USER_SECURIT, NCT_USER_AUDIT)
            if not results:
                return []

            memberIds = [result["f_user_id"] for result in results]

            # 根据用户角色过滤掉自己不能看到的成员
            user_roles = self.get_user_role_id(userId)
            if not user_roles:
                return []
            memberIds = self.__filter_role_member(
                userId, user_roles, roleId, memberIds)

            # 获取成员信息
            batch_user_infos = self.user_manage.get_batch_user_infos_by_id(
                memberIds, userId)
            role_members = []
            for user_id in batch_user_infos:
                role_member = ncTRoleMemberInfo()
                role_member.userId = user_id
                role_member.displayName = batch_user_infos[user_id]["f_display_name"]
                role_members.append(role_member)

            if role_members:
                # 填充所属部门id
                self.user_manage.fill_role_member_departments(role_members)
                # 填充组织管理员所管辖部门信息
                if roleId == NCT_SYSTEM_ROLE_ORG_MANAGER:
                    self.department_manage.fill_role_manage_departments(
                        role_members)
                # 填充组织审计员所管辖部门信息
                if roleId == NCT_SYSTEM_ROLE_ORG_AUDIT:
                    self.department_manage.fill_role_audit_departments(
                        role_members)

            return role_members

        except Exception as ex:
            ShareMgnt_Log(str(ex))
            raise ex

    def get_all_user_id_by_roles(self, roleIds):
        """
        根据角色id获取所有用户id
        """
        filter_str = generate_group_str(roleIds)
        sql = """
        SELECT f_user_id
        FROM t_user_role_relation
        """
        sql = sql + " WHERE f_role_id in (%s)" % (filter_str)
        results = self.r_db.all(sql)
        return [result["f_user_id"] for result in results]

    def search_member(self, userId, roleId, name):
        """
        搜索角色成员
        """
        # 获取用户所能看到的角色后过滤
        member_infos = []
        results = self.get_member(userId, roleId)
        for result in results:
            if name in result.displayName:
                member_infos.append(result)
        return member_infos

    def __filter_role_member(self, userId, user_roles, roleId, memberIds):
        """
        根据用户角色过滤角色中的成员
        """
        # 超级管理员只有admin才可以看
        if roleId == NCT_SYSTEM_ROLE_SUPPER:
            if userId == NCT_USER_ADMIN:
                return memberIds

        # 系统管理员、安全管理员、审计管理员不可以看
        elif roleId in [NCT_SYSTEM_ROLE_ADMIN,
                        NCT_SYSTEM_ROLE_SECURIT,
                        NCT_SYSTEM_ROLE_AUDIT]:
            return []

        # 组织管理员角色，超级管理员、安全管理员可以看所有、组织管理员可以看到管辖部门下的用户
        elif roleId == NCT_SYSTEM_ROLE_ORG_MANAGER:
            can_see_roles = [NCT_SYSTEM_ROLE_SUPPER,
                             NCT_SYSTEM_ROLE_SECURIT]
            if set(user_roles) & set(can_see_roles):
                return memberIds
            if NCT_SYSTEM_ROLE_ORG_MANAGER in user_roles:
                # 获取用户所能管辖部门下的所有用户
                manage_user_ids = self.department_manage.get_supervisory_user_ids(
                    userId)
                return list(set(memberIds) & set(manage_user_ids))

        # 组织审计员角色，只允许超级管理员、安全管理员和组织审计员设置
        elif roleId == NCT_SYSTEM_ROLE_ORG_AUDIT:
            can_see_roles = [NCT_SYSTEM_ROLE_SUPPER,
                             NCT_SYSTEM_ROLE_SECURIT]
            if set(user_roles) & set(can_see_roles):
                return memberIds
            if NCT_SYSTEM_ROLE_ORG_AUDIT in user_roles:
                # 获取用户所能管辖审计部门下的所有用户
                manage_user_ids = self.department_manage.get_supervisory_user_ids(
                    userId, NCT_SYSTEM_ROLE_ORG_AUDIT)
                return list(set(memberIds) & set(manage_user_ids))

        # 剩余所有角色只允许超级管理员、安全管理员查看所有、组织管理员查看自己管辖范围内的
        else:
            can_see_roles = [NCT_SYSTEM_ROLE_SUPPER,
                             NCT_SYSTEM_ROLE_SECURIT]
            if set(user_roles) & set(can_see_roles):
                return memberIds
            if NCT_SYSTEM_ROLE_ORG_MANAGER in user_roles:
                # 获取用户所能管辖部门下的所有用户
                manage_user_ids = self.department_manage.get_supervisory_user_ids(
                    userId)
                return list(set(memberIds) & set(manage_user_ids))

        return []

    def get_member_detail(self, userId, roleId, memberId):
        """
        获取成员详细信息
        """
        if memberId:
            results = self.get_member(userId, roleId, memberId)
            if results:
                return results[0]

        raise_exception(exp_msg=_("IDS_ROLE_MEMBER_NOT_EXIST"),
                        exp_num=ncTShareMgntError.NCT_ROLE_MEMBER_NOT_EXIST)

    def __check_mail_list(self, mailList):
        """
        检查邮箱列表是否合法
        """
        for mail in mailList:
            if not check_email(mail):
                return False
        return True

    def set_user_role_mail(self, adminId, mailList):
        """
        设置用户角色邮箱列表
        """
        # 检查用户是否存在
        self.user_manage.check_user_exists(adminId)

        if not self.__check_mail_list(mailList):
            raise_exception(exp_msg=_("email illegal"),
                            exp_num=ncTShareMgntError.NCT_INVALID_EMAIL)
        jsonConf = json.dumps(mailList)

        check_sql = """
        select f_mail_address from t_user_role_attribute where f_user_id = %s
        """
        result = self.r_db.one(check_sql, adminId)
        if result:
            update_sql = """
            update t_user_role_attribute set f_mail_address = %s where f_user_id = %s
            """
            self.w_db.query(update_sql, jsonConf, adminId)
        else:
            insert_sql = """
            insert into t_user_role_attribute(f_user_id, f_mail_address) values (%s, %s)
            """
            self.w_db.query(insert_sql, adminId, jsonConf)

    def get_user_role_mail(self, adminId):
        """
        获取用户角色邮箱列表
        """
        sql = """
        SELECT `f_mail_address`
        FROM `t_user_role_attribute`
        WHERE `f_user_id` = %s
        """
        result = self.r_db.one(sql, adminId)
        if result:
            return json.loads(result["f_mail_address"])
        return []

    def get_role_mails(self, roleId):
        """
        获取指定角色下所有邮箱列表
        """
        sql = """
        SELECT `f_mail_address`
        FROM `t_user_role_attribute`
        INNER JOIN t_user_role_relation
        USING(f_user_id)
        WHERE `f_role_id` = %s
        """
        results = self.r_db.all(sql, roleId)
        mailList = []
        for result in results:
            mailList.extend(json.loads(result['f_mail_address']))
        # 使用set去重
        return set(mailList)

    def check_member_exist(self, roleId, memberId):
        """
        检查成员是否存在
        """
        return self.__check_member_in_role(roleId, memberId, False)

    def check_is_role_supper(self, user_id):
        """
        根据id检查用户是否为超级管理员
        """
        role_ids = self.get_user_role_id(user_id)
        return True if NCT_SYSTEM_ROLE_SUPPER in role_ids else False

    def check_user_rights(self, user_id, role_tuple):
        """
        检查用户是否是否包含指定的角色
        """
        roles = self.get_user_role_id(user_id)
        if set(roles) & set(role_tuple):
            return True
        else:
            return False
