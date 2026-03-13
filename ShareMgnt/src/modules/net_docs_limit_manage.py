#!/usr/bin/python3
# -*- coding:utf-8 -*-
"""This is net docs limit class"""
import uuid
from src.common.db.connector import DBConnector
from src.common.lib import (escape_key,
                            check_net_ip,
                            check_net_mask,
                            raise_exception)
from src.common.db.connector import ConnectorManager
from ShareMgnt.ttypes import (ncTShareMgntError,
                              ncTDocInfo)
from eisoo.tclients import TClient


class NetDocsLimitManage(DBConnector):

    """
    Net docs limit manage
    """

    def __init__(self):
        """
        init
        """

    def check_net_info(self, net_info, b_check_net_Id=False):
        """
        检查网段文档库绑定配置信息
        """
        # 检查ip合法性
        if (not net_info.ip or
                not check_net_ip(net_info.ip)):
            raise_exception(exp_msg=_("IDS_INVALID_NET_IP_PARAM"),
                            exp_num=ncTShareMgntError.NCT_INVALID_NET_IP_PARAM)

        # 检查掩码合法性
        if (not net_info.subNetMask or
                not check_net_mask(net_info.subNetMask)):
            raise_exception(exp_msg=_("IDS_INVALID_NET_MASK_PARAM"),
                            exp_num=ncTShareMgntError.NCT_INVALID_NET_MASK_PARAM)

        if b_check_net_Id:
            query_sql = """
            SELECT *
            FROM `t_net_docs_limit_info`
            WHERE `f_id` = %s
            """
            result = self.r_db.one(query_sql, net_info.id)
            if not result:
                raise_exception(exp_msg=_("IDS_NET_DOCS_LIMIT_ID_NOT_EXIST"),
                                exp_num=ncTShareMgntError.NCT_NET_DOCS_LIMIT_ID_NOT_EXIST)

    def check_net_exist(self, ip, sub_net_mask):
        """
        检查网段信息是否已设置
        """
        query_sql = """
        SELECT *
        FROM `t_net_docs_limit_info`
        WHERE `f_ip` = %s and `f_sub_net_mask` = %s
        """
        result = self.r_db.one(query_sql, ip, sub_net_mask)
        return True if result else False

    def add_net(self, net_info):
        """
        添加网段设置
        """
        # 检查用户网段配置信息参数
        self.check_net_info(net_info)

        # 检查网段信息是否已设置
        if self.check_net_exist(net_info.ip, net_info.subNetMask):
                raise_exception(exp_msg=_("IDS_NET_DOCS_LIMIT_EXIST"),
                                exp_num=ncTShareMgntError.NCT_NET_DOCS_LIMIT_EXIST)

        # 生成用户网段配置信息id
        net_info.id = str(uuid.uuid1())

        # 添加net_docs_info 至数据库
        self.add_net_info_to_db(net_info)

    def add_net_info_to_db(self, net_info):
        """
        添加net_info 至数据库
        """
        insert_sql = """
        INSERT INTO `t_net_docs_limit_info`
        (`f_id`, `f_ip`, `f_sub_net_mask`, `f_doc_id`)
        VALUES(%s, %s, %s, '')
        """

        self.w_db.query(insert_sql,
                        net_info.id,
                        net_info.ip,
                        net_info.subNetMask)

    def edit_net(self, net_info):
        """
        编辑网段设置
        """
        self.check_net_info(net_info, True)

        # 获取旧网段配置信息
        query_sql = """
        SELECT DISTINCT `f_ip`, `f_sub_net_mask`
        FROM `t_net_docs_limit_info`
        WHERE `f_id` = %s
        """
        result = self.r_db.all(query_sql, net_info.id)
        old_ip = result[0]['f_ip']
        old_sub_net_mask = result[0]['f_sub_net_mask']

        # 网段发生变化则更新网段设置
        if (old_ip != net_info.ip or old_sub_net_mask != net_info.subNetMask):
            # 需要判断是否与已有网段设置冲突
            if self.check_net_exist(net_info.ip, net_info.subNetMask):
                    raise_exception(exp_msg=_("IDS_NET_DOCS_LIMIT_EXIST"),
                                    exp_num=ncTShareMgntError.NCT_NET_DOCS_LIMIT_EXIST)

            self.update_net_info_to_db(net_info)

    def update_net_info_to_db(self, net_info):
        """
        更新网段文档库数据库中网段设置
        """
        update_sql = """
        UPDATE `t_net_docs_limit_info`
        SET `f_ip` = %s, `f_sub_net_mask` = %s
        WHERE `f_id` = %s
        """
        self.w_db.all(update_sql,
                      net_info.ip,
                      net_info.subNetMask,
                      net_info.id)

    def delete_net(self, net_id):
        """
        删除一条用户网段配置信息
        """
        delete_sql = """
        DELETE
        FROM `t_net_docs_limit_info`
        WHERE `f_id` = %s
        """
        affect_row = self.w_db.query(delete_sql, net_id)
        if not affect_row:
            raise_exception(exp_msg=_("IDS_NET_DOCS_LIMIT_ID_NOT_EXIST"),
                            exp_num=ncTShareMgntError.NCT_NET_DOCS_LIMIT_ID_NOT_EXIST)

    def add_docs(self, net_id, docId_list):
        """
        添加网段绑定的文档库信息
        """
        query_sql = """
        SELECT f_doc_id, f_ip, f_sub_net_mask
        FROM `t_net_docs_limit_info`
        WHERE `f_id` = %s
        """
        results = self.r_db.all(query_sql, net_id)
        if not results:
            raise_exception(exp_msg=_("IDS_NET_DOCS_LIMIT_ID_NOT_EXIST"),
                            exp_num=ncTShareMgntError.NCT_NET_DOCS_LIMIT_ID_NOT_EXIST)

        doc_ids = []
        for ret in results:
            doc_ids.append(ret['f_doc_id'])

        # 过滤需要添加的文档id列表
        need_add_doc_ids = set()
        for doc_id in docId_list:
            if doc_id not in doc_ids:
                need_add_doc_ids.add(doc_id)

        if need_add_doc_ids:
            # 使用事务更新数据
            conn = ConnectorManager.get_db_conn()
            cursor = conn.cursor()

            insert_sql = """
            INSERT INTO `t_net_docs_limit_info`
            (`f_id`, `f_ip`, `f_sub_net_mask`, `f_doc_id`)
            VALUES(%s, %s, %s, %s)
            """
            try:
                for doc_id in need_add_doc_ids:
                    cursor.execute(insert_sql, (net_id,
                                                results[0]['f_ip'],
                                                results[0]['f_sub_net_mask'],
                                                doc_id))
                conn.commit()
            except Exception as e:
                conn.rollback()
                raise_exception(exp_msg=str(e),
                                exp_num=ncTShareMgntError.NCT_DB_OPERATE_FAILED)

    def delete_docs(self, net_id, doc_id):
        """
        删除绑定文档库设置
        """
        if not doc_id:
            raise_exception(exp_msg=_("IDS_DOC_ID_NOT_SET"),
                            exp_num=ncTShareMgntError.NCT_DOC_ID_NOT_SET)

        delete_sql = """
        DELETE
        FROM `t_net_docs_limit_info`
        WHERE `f_id` = %s and `f_doc_id` = %s
        """
        self.w_db.query(delete_sql, net_id, doc_id)

    def get_docs_info_by_id(self, net_id):
        """
        在指定网段设置中获取所有绑定的文档库信息
        """
        query_sql = """
        SELECT f_doc_id
        FROM `t_net_docs_limit_info`
        WHERE `f_id` = %s
        """
        results = self.r_db.all(query_sql, net_id)

        if not results:
            raise_exception(exp_msg=_("IDS_NET_DOCS_LIMIT_ID_NOT_EXIST"),
                            exp_num=ncTShareMgntError.NCT_NET_DOCS_LIMIT_ID_NOT_EXIST)
        docs = []
        need_delete_docids = []
        for ret in results:
            if ret['f_doc_id']:
                tmp_doc = ncTDocInfo()
                tmp_doc.id = ret['f_doc_id']
                doc_info = self.get_doc_info_by_doc_id(ret['f_doc_id'])
                tmp_doc.name = doc_info.name
                tmp_doc.typeName = doc_info.typeName
                if tmp_doc.name:
                    docs.append(tmp_doc)
                else:
                    need_delete_docids.append(tmp_doc.id)
        if need_delete_docids:
            self.delete_docids_from_db(net_id, need_delete_docids)

        return docs

    def get_doc_info_by_doc_id(self, doc_id):
        """
        转换数据库记录中的文档信息
        """
        with TClient("EFAST") as client:
            doc_info = client.EFAST_GetCustomDocByDocId(doc_id)

        return doc_info

    def get_docs(self, net_id):
        """
        在指定网段设置中获取所有绑定的文档库信息
        """
        return self.get_docs_info_by_id(net_id)

    def search_docs(self, net_id, name):
        """
        在指定网段设置中搜索某个文档库
        """
        # 1. 先获取指定网段中的所有访问者
        docs_list = self.get_docs_info_by_id(net_id)

        # 2. 根据name进行过滤
        filter_docs_list = []
        for docs in docs_list:
            if name in docs.name:
                filter_docs_list.append(docs)

        return filter_docs_list

    def delete_docids_from_db(self, net_id, docid_list):
        """
        删除已被移除的文档库id
        """
        # 使用事务批量删除数据
        conn = ConnectorManager.get_db_conn()
        cursor = conn.cursor()

        delete_sql = """
        DELETE
        FROM `t_net_docs_limit_info`
        WHERE `f_id` = %s and `f_doc_id` = %s
        """
        try:
            # 单条插入审核范围信息
            for docid in docid_list:
                cursor.execute(delete_sql, (net_id,
                                            docid))

            conn.commit()

        except Exception as e:
            conn.rollback()
            raise_exception(exp_msg=str(e),
                            exp_num=ncTShareMgntError.NCT_DB_OPERATE_FAILED)
