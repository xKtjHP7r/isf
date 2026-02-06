#!/usr/bin/python3
# -*- coding:utf-8 -*-
# pylint: disable=C0103
"""
"""
from collections import deque
from src.modules.domain_manage import DomainManage
from src.modules.ldap_manage import (LdapManage, name2dn)
from ShareMgnt.ttypes import (ncTUsrmUserType,
                              ncTUsrmDepartType,
                              ncTUsrmDomainSyncMode)
from src.third_party_auth.ou_manage.base_ou_manage import *
from src.third_party_auth.ou_manage.as_ou_manage import ASOuManage


class DomainOuManage(BaseOuManage):
    """
    组织用户管理基类
    """
    def __init__(self, b_eacplog=False):
        """
        初始化函数
        """
        super(DomainOuManage, self).__init__()
        self.domain_manage = DomainManage()
        self.domain_dn = None
        self.domain_name = None
        self.key_config = None
        self.sync_config = None
        self.dn_dict = {}
        self.third_id_dict = {}
        self.user_total = 0
        self.ou_total = 0
        self.ldap_manage = None
        self.as_ou_manage = ASOuManage()

    def init_server_info(self, server_info):
        """
        初始化服务器信息
        """
        self.user_total = 0
        self.ou_total = 0
        self.ldap_manage, self.server_info = self.domain_manage.connect_domain(server_info.id)
        self.domain_name = server_info.name
        self.domain_dn = name2dn(server_info.name)
        self.key_config = server_info.key_config
        self.sync_config = server_info.config
        self.generate_dn_dict()

    def get_third_id_by_dn(self, dn):
        """
        根据ou dn获取第三方id
        """
        third_id = None
        try:
            third_id = self.dn_dict[dn]
        except Exception:
            pass
        return third_id

    def __generate_ou(self, dn):

        # 处理根组织下用户
        if dn == self.domain_dn:
            self.__generate_user(dn)

        for ou_info in self.ldap_manage.get_all_sub_ous_iter(dn, self.key_config):
            self.ou_total += 1
            self.dn_dict[ou_info.dn] = ou_info.third_id
            self.third_id_dict[ou_info.third_id] = ou_info.dn

            self.__generate_user(ou_info.dn)

    def __generate_user(self, dn):
        for user_info in self.ldap_manage.get_sub_users_iter(dn, self.key_config):
            self.user_total += 1
            self.dn_dict[user_info.dn] = user_info.third_id
            self.third_id_dict[user_info.third_id] = user_info.dn

    def generate_dn_dict(self):
        """
        生成域组织和用户的dn路径和第三方id字典
        """
        base_dn = []
        if not self.sync_config.ouPath:
            base_dn = [self.domain_dn]
        else:
            for path in self.sync_config.ouPath:
                base_dn.append(path)

        for dn in base_dn:
            self.dn_dict[dn] = dn
            self.third_id_dict[dn] = dn

            self.__generate_ou(dn)

        self.dn_dict[self.domain_dn] = self.domain_dn
        self.third_id_dict[self.domain_dn] = self.domain_dn

    def add_dn_dict(self, dn):
        """
        添加dn路径
        """
        for ou_info in self.ldap_manage.get_all_sub_ous_iter(dn, self.key_config):
            self.ou_total += 1
            self.dn_dict[ou_info.dn] = ou_info.third_id
            self.third_id_dict[ou_info.third_id] = ou_info.dn
        return self.get_third_id_by_dn(dn)

    def generate_upper_ou_relation(self, config):
        """
        生成上层组织关系
        """
        ou_dict = {}
        src_ou_dns = config.ouPath

        ou_ids = []
        for ou_dn in src_ou_dns:
            # 获取上层组织的ou_ids
            while not str(ou_dn).lower() == str(self.domain_dn).lower():
                ou_id = self.get_third_id_by_dn(ou_dn)
                if ou_id is None:
                    ou_id = self.add_dn_dict(ou_dn)
                ou_ids.append(ou_id)
                ou_dn = ",".join(ou_dn.split(",")[1:])

            # 生成组织关系
            parent_ou_id = '-1'
            while ou_ids:
                ou_id = ou_ids.pop()
                if parent_ou_id not in ou_dict:
                    ou_dict[parent_ou_id] = [ou_id]
                elif ou_id not in ou_dict[parent_ou_id]:
                    ou_dict[parent_ou_id].append(ou_id)
                parent_ou_id = ou_id
        return ou_dict

    def generate_ou_user_relation(self, parent_depart_id, src_ou_ids, app_id):
        """
        生成组织用户关系
        """
        # 获取域配置信息

        config = self.domain_manage.get_domain_sync_config(app_id)

        ou_dict = {}
        user_dict = {}

        root_id = src_ou_ids[0]

        if config.syncMode == ncTUsrmDomainSyncMode.NCT_SYNC_USERS_ONLY:
            # 仅同步用户
            src_user_infos = []
            ou_queue = deque(src_ou_ids)

            while len(ou_queue):
                ou_id = ou_queue.popleft()
                src_user_infos += self.get_sub_users(ou_id)
                sub_ou_ids = self.get_sub_ou_ids(ou_id)
                ou_queue.extend(sub_ou_ids)
            user_dict["-1"] = src_user_infos

            return ou_dict, user_dict

        elif root_id == self.domain_dn or config.ouPath[0] == self.domain_dn:
            # 同步域根组织
            ou_info = self.get_ou(root_id)
            dept_info = self.as_ou_manage.get_depart_info_by_id(parent_depart_id)

            if dept_info and ou_info and str(dept_info.third_id) == str(ou_info.third_id):
                # 域根组织和目的部门相同的情况
                src_ou_ids = self.get_sub_ou_ids(root_id)
                src_user_infos = self.get_sub_users(root_id)
                ou_dict = {"-1": src_ou_ids}
                user_dict = {"-1": src_user_infos}

            else:
                # 域根组织和目的部门不同
                ou_dict = {"-1": src_ou_ids}

        elif config.syncMode == ncTUsrmDomainSyncMode.NCT_SYNC_UPPER_OU:
            # 同步上层组织
            ou_dict = self.generate_upper_ou_relation(config)

        else:
            # 不同步上层组织
            ou_dict["-1"] = src_ou_ids

        ou_queue = deque(src_ou_ids)
        while len(ou_queue):
            ou_id = ou_queue.popleft()
            sub_ou_ids = self.get_sub_ou_ids(ou_id)
            ou_queue.extend(sub_ou_ids)
            ou_dict[ou_id] = sub_ou_ids
            src_user_infos = self.get_sub_users(ou_id)
            user_dict[ou_id] = src_user_infos

        return ou_dict, user_dict

    def get_ou_infos_by_ou_id(self, src_ou_ids):
        """
        通过域组织IDS获取组织信息
        """
        if not src_ou_ids:
            return []

        src_ou_infos = []
        for ou_id in src_ou_ids:
            ou_info = self.get_ou(ou_id)
            if ou_info is not None:
                ou_info.ou_name = ou_info.ou_name
                ou_info.third_id = ou_info.third_id
                src_ou_infos.append(ou_info)
        return src_ou_infos

    def convert_domain_ou_info(self, domain_ou_info):
        """
        转换DomainOuInfo为OuInfo
        """
        ou_info = OuInfo()
        ou_info.ou_name = self.domain_manage.replace_invalid_characters(domain_ou_info.ou_name)
        ou_info.third_id = domain_ou_info.third_id
        ou_info.dn = domain_ou_info.dn
        ou_info.server_type = domain_ou_info.server_type
        ou_info.type = ncTUsrmDepartType.NCT_DEPART_TYPE_DOMAIN
        return ou_info

    def convert_domain_user_info(self, domain_user_info):
        """
        转换DomainUserInfo为UserInfo
        """
        user_info = UserInfo()
        user_info.login_name = domain_user_info.login_name
        user_info.display_name = self.domain_manage.replace_invalid_characters(domain_user_info.display_name)
        user_info.email =domain_user_info.email
        user_info.idcard_number = domain_user_info.idcard_number
        user_info.tel_number = domain_user_info.tel_number
        user_info.third_id = domain_user_info.third_id
        user_info.status = domain_user_info.status
        user_info.server_type = domain_user_info.server_type
        user_info.dn = self.domain_name if user_info.server_type == 1 else domain_user_info.dn
        user_info.type = ncTUsrmUserType.NCT_USER_TYPE_DOMAIN
        user_info.csf_level = self.sync_config.csfLevel

        return user_info

    def get_ou(self, third_ou_id):
        """
        获取组织部门信息
        """
        ou_info = None

        if third_ou_id == self.domain_dn:
            ou_info = OuInfo()
            ou_info.third_id = self.domain_dn
            ou_info.ou_name = self.domain_name
        else:
            ou_dn = self.third_id_dict[third_ou_id]
            domain_ou_info = self.ldap_manage.get_domain_ou(ou_dn, self.key_config)
            if domain_ou_info:
                ou_info = self.convert_domain_ou_info(domain_ou_info)

        return ou_info

    def get_sub_ous(self, third_ou_id):
        """
        获取子组织或部门
        """
        ou_dn = None
        sub_ous = []
        ou_dn = self.third_id_dict[third_ou_id]
        sub_domain_ou_infos = self.ldap_manage.get_onelevel_sub_ous(ou_dn, self.key_config)

        for sub_domain_ou in sub_domain_ou_infos:
            ou_info = self.convert_domain_ou_info(sub_domain_ou)
            if ou_info:
                sub_ous.append(ou_info)

        return sub_ous

    def get_sub_ou_ids(self, third_ou_id):
        """
        获取子部门的id
        """
        ou_dn = None
        sub_ou_ids = []
        ou_dn = self.third_id_dict[third_ou_id]
        sub_domain_ou_infos = self.ldap_manage.get_onelevel_sub_ous(ou_dn, self.key_config)

        for sub_domain_ou in sub_domain_ou_infos:
            ou_info = self.convert_domain_ou_info(sub_domain_ou)
            if ou_info:
                sub_ou_ids.append(ou_info.third_id)

        return sub_ou_ids

    def get_sub_users(self, third_ou_id):
        """
        获取子用户
        """
        sub_users = []

        ou_dn = self.third_id_dict[third_ou_id]
        sub_domain_user_infos = self.ldap_manage.get_onelevel_sub_users(ou_dn, self.key_config)

        for sub_domain_user in sub_domain_user_infos:
            user_info = self.convert_domain_user_info(sub_domain_user)

            if user_info:
                sub_users.append(user_info)

        return sub_users

    def get_ous_num_info(self):
        """
        获取部门和用户总数信息
        """
        return self.ou_total, self.user_total
