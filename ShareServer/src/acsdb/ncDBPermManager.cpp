#include <abprec.h>
#include <ncutil/ncBusinessDate.h>

#include "acsdb.h"
#include "ncDBPermManager.h"
#include <dataapi/ncGNSUtil.h>
#include "common/util.h"

/* Implementation file */
NS_IMPL_QUERY_INTERFACE1 (ncDBPermManager, ncIDBPermManager)

// protected
NS_IMETHODIMP_(nsrefcnt) ncDBPermManager::AddRef (void)
{
    return 1;
}

// protected
NS_IMETHODIMP_(nsrefcnt) ncDBPermManager::Release (void)
{
    return 1;
}

AB_DEFINE_THREADSAFE_SINGLETON_NO_POOL (ncDBPermManager)

ncDBPermManager::ncDBPermManager()
{
    NC_ACS_DB_TRACE (_T(""), this);
}

ncDBPermManager::~ncDBPermManager()
{
    NC_ACS_DB_TRACE (_T(""), this);
}

/* [notxpcom] void GetCustomPermByDocIds ([const] in StringVecRef docIds, in dbCustomPermInfoVectorRef infos); */
NS_IMETHODIMP_(void) ncDBPermManager::GetCustomPermByDocIds(const vector<String> & docIds, vector<dbCustomPermInfo> & infos)
{
    NC_ACS_DB_TRACE (_T("docIds size: %d begin"),  (int)docIds.size ());

    infos.clear ();

    if (docIds.size () == 0) {
        return;
    }

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    String groupStr;
    for (size_t i = 0; i < docIds.size (); ++i) {
        groupStr.append ("\'", 1);
        groupStr.append (dbOper->EscapeEx(docIds[i]));
        groupStr.append ("\'", 1);

        if (i != (docIds.size () -1)) {
            groupStr.append (",", 1);
        }
    }

    String dbName = Util::getDBName("anyshare");
    String strSql;
    // 构造select in 查询语句
    strSql.format (_T("select f_primary_id, f_type, f_perm_value, f_doc_id, f_accessor_id, f_accessor_name, f_accessor_type, "
                      "f_end_time, f_create_time, f_modify_time from %s.t_acs_custom_perm "
                      "where f_accessor_type != 4 and f_type in (1,2) and f_doc_id in(%s) order by f_doc_id"),
                    dbName.getCStr(), groupStr.getCStr ());

    ncDBRecords results;

    dbOper->Select (strSql, results);

    dbCustomPermInfo tmpInfo;
    for (size_t i = 0; i < results.size (); ++i) {
        tmpInfo.id = Int64::getValue (results[i][0]);
        tmpInfo.isAllowed = Int::getValue (results[i][1]) == 2;
        tmpInfo.permValue = Int::getValue (results[i][2]);
        tmpInfo.docId = results[i][3];
        tmpInfo.accessorId = results[i][4];
        tmpInfo.accessorName = results[i][5];
        tmpInfo.accessorType = Int::getValue (results[i][6]);
        tmpInfo.endTime = Int64::getValue (results[i][7]);
        tmpInfo.createTime = Int64::getValue (results[i][8]);
        tmpInfo.modifyTime = Int64::getValue (results[i][9]);

        infos.push_back (tmpInfo);
    }

    NC_ACS_DB_TRACE (_T("docIds size: %d end, ret infos size: %d"),  (int)docIds.size (), (int)infos.size ());
}

/* [notxpcom] void AddCustomPerm ([const] in dbCustomPermInfoRef info); */
NS_IMETHODIMP_(void) ncDBPermManager::AddCustomPerm(const dbCustomPermInfo & info)
{
    NC_ACS_DB_TRACE (_T("isAllowed: %s, permValue: %d, accessorId: %s, accessorType: %d, docId: %s, endTime: %lld begin"),
         String::toString (info.isAllowed).getCStr (),
        info.permValue,
        info.accessorId.getCStr (),
        info.accessorType,
        info.docId.getCStr (),
        info.endTime);

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    String dbName = Util::getDBName("anyshare");
    String strSql;
    strSql.format (_T("insert into %s.t_acs_custom_perm(f_type, f_perm_value, f_source, f_doc_id, f_accessor_id, f_accessor_name, f_accessor_type, f_end_time, f_modify_time, f_create_time) ")
                    _T("values (%d, %d, %d, '%s', '%s', '%s', %d, %lld, %lld, %lld)"),
        dbName.getCStr(),
        info.isAllowed ? 2 : 1,
        info.permValue,
        info.source,
        dbOper->EscapeEx(info.docId).getCStr (),
        dbOper->EscapeEx(info.accessorId).getCStr (),
        dbOper->EscapeEx(info.accessorName).getCStr (),
        info.accessorType,
        info.endTime,
        BusinessDate::getCurrentTime (),
        BusinessDate::getCurrentTime ());

    dbOper->Execute (strSql);

    NC_ACS_DB_TRACE (_T("isAllowed: %s, permValue: %d, accessorId: %s, accessorType: %d, docId: %s, endTime: %lld end"),
         String::toString (info.isAllowed).getCStr (),
        info.permValue,
        info.accessorId.getCStr (),
        info.accessorType,
        info.docId.getCStr (),
        info.endTime);
}

/* [notxpcom] void UpdateCustomPerm ([const] in dbCustomPermInfoRef info); */
NS_IMETHODIMP_(void) ncDBPermManager::UpdateCustomPerm(const dbCustomPermInfo& info)
{
    NC_ACS_DB_TRACE (_T("id: %lld, isAllowed: %s, permValue: %d, accessorId: %s, accessorType: %d, docId: %s, endTime: %lld begin"),
        info.id,
        String::toString (info.isAllowed).getCStr (),
        info.permValue,
        info.accessorId.getCStr (),
        info.accessorType,
        info.docId.getCStr (),
        info.endTime);

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    String dbName = Util::getDBName("anyshare");
    String strSql;
    strSql.format (_T("update %s.t_acs_custom_perm set f_doc_id = '%s', f_accessor_id = '%s', f_accessor_type = %d, ")
                    _T("f_type = %d, f_perm_value = %d, f_source = %d, f_end_time = %lld, f_modify_time = %lld where f_primary_id = %lld"),
        dbName.getCStr(),
        dbOper->EscapeEx(info.docId).getCStr (),
        dbOper->EscapeEx(info.accessorId).getCStr (),
        info.accessorType,
        info.isAllowed ? 2 : 1,
        info.permValue,
        info.source,
        info.endTime,
        BusinessDate::getCurrentTime (),
        info.id);

    dbOper->Execute (strSql);

    NC_ACS_DB_TRACE (_T("id: %lld, isAllowed: %s, permValue: %d, accessorId: %s, accessorType: %d, docId: %s, endTime: %lld end"),
        info.id,
        String::toString (info.isAllowed).getCStr (),
        info.permValue,
        info.accessorId.getCStr (),
        info.accessorType,
        info.docId.getCStr (),
        info.endTime);
}

/* [notxpcom] void DeleteCustomPerm (in int64 id); */
NS_IMETHODIMP_(void) ncDBPermManager::DeleteCustomPerm(int64 id)
{
    NC_ACS_DB_TRACE (_T("id: %lld begin"), id);

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    String dbName = Util::getDBName("anyshare");
    String strSql;
    strSql.format (_T("delete from %s.t_acs_custom_perm where f_primary_id = %lld"), dbName.getCStr(), id);

    dbOper->Execute (strSql);

    NC_ACS_DB_TRACE (_T("id: %lld end"), id);
}

/* [notxpcom] bool GetCustomPermById (in int64 id, in dbCustomPermInfoRef resultInfo); */
NS_IMETHODIMP_(bool) ncDBPermManager::GetCustomPermById(int64 id, dbCustomPermInfo & resultInfo)
{
    NC_ACS_DB_TRACE (_T("id: %lld begin"), id);

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    resultInfo.isAllowed = true;
    resultInfo.permValue = 0;
    resultInfo.accessorId = String::EMPTY;
    resultInfo.docId = String::EMPTY;

    String dbName = Util::getDBName("anyshare");
    String strSql;
    strSql.format (_T("select f_type, f_perm_value, f_doc_id, f_accessor_id, f_accessor_type, f_end_time from %s.t_acs_custom_perm where f_primary_id = %lld"),
        dbName.getCStr(), id);

    ncDBRecords results;
    dbOper->Select (strSql, results);

    bool ret = false;
    if (results.size () >= 1) {
        resultInfo.id = id;
        resultInfo.isAllowed = Int::getValue (results[0][0]) == 2;
        resultInfo.permValue = Int::getValue (results[0][1]);
        resultInfo.docId = results[0][2];
        resultInfo.accessorId = results[0][3];
        resultInfo.accessorType = Int::getValue (results[0][4]);
        resultInfo.endTime = Int64::getValue (results[0][5]);

        ret = true;
    }
    else {
        resultInfo.id = -1;
        resultInfo.isAllowed = false;
        resultInfo.permValue = 0;
        resultInfo.docId = "";
        resultInfo.accessorId = "";
        resultInfo.accessorType = -1;
        resultInfo.endTime = -1;
    }

    NC_ACS_DB_TRACE (_T("id: %lld end, ret = %s, f_is_allowed = %s, f_perm_value = %d, f_doc_id = %s, f_accessor_id = %s, f_accessor_type = %d, f_end_time = %lld"),
        resultInfo.id,
        String::toString(ret).getCStr(),
        String::toString(resultInfo.isAllowed).getCStr(),
        resultInfo.permValue,
        resultInfo.docId.getCStr(),
        resultInfo.accessorId.getCStr(),
        resultInfo.accessorType,
        resultInfo.endTime);

    return ret;
}

/* [notxpcom] bool GetCustomPermByEndTime ([const] in StringRef docId, [const] in StringRef accessorId, in int accessorType, in bool isAllowed, in int64 endTime, in dbCustomPermInfoRef resultInfo); */
NS_IMETHODIMP_(bool) ncDBPermManager::GetCustomPermByEndTime(const String & docId, const String & accessorId, int accessorType, bool isAllowed, int64 endTime, dbCustomPermInfo & resultInfo)
{
    NC_ACS_DB_TRACE (_T("docId = %s, accessorId = %s, accessorType = %d, isAllowed = %s, endTime = %lld begin"),
        docId.getCStr(),
        accessorId.getCStr(),
        accessorType,
        String::toString(isAllowed).getCStr(),
        endTime);

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    resultInfo.isAllowed = true;
    resultInfo.permValue = 0;
    resultInfo.accessorId = String::EMPTY;
    resultInfo.docId = String::EMPTY;

    String dbName = Util::getDBName("anyshare");
    String strSql;
    strSql.format (_T("select f_primary_id, f_perm_value from %s.t_acs_custom_perm where f_doc_id = '%s' and f_accessor_id = '%s' and f_accessor_type = %d and f_type = %d and f_end_time = %lld"),
        dbName.getCStr(),
        dbOper->EscapeEx(docId).getCStr(),
        dbOper->EscapeEx(accessorId).getCStr(),
        accessorType,
        isAllowed ? 2: 1,
        endTime);

    ncDBRecords results;
    dbOper->Select (strSql, results);

    bool ret = false;
    if (results.size () >= 1) {
        resultInfo.id = Int64::getValue(results[0][0]);
        resultInfo.permValue = Int::getValue (results[0][1]);

        resultInfo.docId = docId;
        resultInfo.accessorId = accessorId;
        resultInfo.accessorType = accessorType;
        resultInfo.isAllowed = isAllowed;
        resultInfo.endTime = endTime;

        ret = true;
    }
    else {
        resultInfo.id = -1;
        resultInfo.isAllowed = false;
        resultInfo.permValue = 0;
        resultInfo.docId = "";
        resultInfo.accessorId = "";
        resultInfo.accessorType = -1;
        resultInfo.endTime = -1;
    }

    NC_ACS_DB_TRACE (_T("id: %lld end, ret = %s, f_is_allowed = %s, f_perm_value = %d, f_doc_id = %s, f_accessor_id = %s, f_accessor_type = %d, f_end_time = %lld"),
        resultInfo.id,
        String::toString(ret).getCStr(),
        String::toString(resultInfo.isAllowed).getCStr(),
        resultInfo.permValue,
        resultInfo.docId.getCStr(),
        resultInfo.accessorId.getCStr(),
        resultInfo.accessorType,
        resultInfo.endTime);

    return ret;
}

/* [notxpcom] void DeleteCustomPermByFileId ([const] in StringRef fileId); */
NS_IMETHODIMP_(void) ncDBPermManager::DeleteCustomPermByFileId(const String& fileId)
{
    NC_ACS_DB_TRACE (_T("fileId: %s begin"),  fileId.getCStr ());

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    String dbName = Util::getDBName("anyshare");
    String strSql;
    strSql.format (_T("delete from %s.t_acs_custom_perm where f_doc_id = '%s'"),
        dbName.getCStr(), dbOper->EscapeEx(fileId).getCStr ());

    dbOper->Execute (strSql);

    NC_ACS_DB_TRACE (_T("fileId: %s end"),  fileId.getCStr ());
}

/* [notxpcom] void DeleteCustomPermByDirId ([const] in StringRef dirId); */
NS_IMETHODIMP_(void) ncDBPermManager::DeleteCustomPermByDirId(const String& dirId)
{
    NC_ACS_DB_TRACE (_T("dirId: %s begin"),  dirId.getCStr ());

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    String dbName = Util::getDBName("anyshare");
    String escDirId = dbOper->EscapeEx(dirId);
    String strSql;
    strSql.format (_T("delete from %s.t_acs_custom_perm where f_doc_id = '%s' or f_doc_id like '%s/%%'"),
        dbName.getCStr(), escDirId.getCStr (), escDirId.getCStr ());

    dbOper->Execute (strSql);

    NC_ACS_DB_TRACE (_T("dirId: %s end"),  dirId.getCStr ());
}

/* [notxpcom] void DeleteCustomPermByUserId ([const] in StringRef userId); */
NS_IMETHODIMP_(void) ncDBPermManager::DeleteCustomPermByUserId(const String& userId)
{
    NC_ACS_DB_TRACE (_T("userId: %s begin"),  userId.getCStr ());

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    String dbName = Util::getDBName("anyshare");
    String strSql;
    strSql.format (_T("delete from %s.t_acs_custom_perm where f_accessor_id = '%s'"),
        dbName.getCStr(), dbOper->EscapeEx(userId).getCStr ());

    dbOper->Execute (strSql);

    NC_ACS_DB_TRACE (_T("userId: %s end"),  userId.getCStr ());
}

/* [notxpcom] void DeleteCustomPermByDocUserId ([const] in StringRef docId, [const] in StringRef userId); */
NS_IMETHODIMP_(void) ncDBPermManager::DeleteCustomPermByDocUserId(const String& docId, const String& userId)
{
    NC_ACS_DB_TRACE (_T("docId: %s, userId: %s begin"), docId.getCStr (), userId.getCStr ());

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    String dbName = Util::getDBName("anyshare");
    String strSql;
    strSql.format (_T("delete from %s.t_acs_custom_perm where f_accessor_id = '%s' and f_doc_id = '%s'"),
                    dbName.getCStr(), dbOper->EscapeEx(userId).getCStr (), dbOper->EscapeEx(docId).getCStr ());

    dbOper->Execute (strSql);

    NC_ACS_DB_TRACE (_T("docId: %s ,userId: %s end"), docId.getCStr (), userId.getCStr ());
}

/* [notxpcom] void GetAllCustomPerm (in dbCustomPermInfoVectorRef infos); */
NS_IMETHODIMP_(void) ncDBPermManager::GetAllCustomPerm(vector<dbCustomPermInfo> & infos)
{
    NC_ACS_DB_TRACE (_T("begin this: %p"), this);

    infos.clear ();

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    String dbName = Util::getDBName("anyshare");
    String strSql;
    // 构造select in 查询语句
    strSql.format (_T("select f_primary_id, f_type, f_perm_value, f_doc_id, f_accessor_id,f_accessor_type, f_end_time from %s.t_acs_custom_perm where f_type in (1,2) and f_accessor_type != 4 order by f_doc_id"),
                    dbName.getCStr());

    ncDBRecords results;

    dbOper->Select (strSql, results);

    dbCustomPermInfo tmpInfo;
    for (size_t i = 0; i < results.size (); ++i) {
        tmpInfo.id = Int64::getValue (results[i][0]);
        tmpInfo.isAllowed = Int::getValue (results[i][1]) == 2;
        tmpInfo.permValue = Int::getValue (results[i][2]);
        tmpInfo.docId = results[i][3];
        tmpInfo.accessorId = results[i][4];
        tmpInfo.accessorType = Int::getValue (results[i][5]);
        tmpInfo.endTime = Int64::getValue (results[i][6]);

        infos.push_back (tmpInfo);
    }

    NC_ACS_DB_TRACE (_T(" ret infos size: %d"),  (int)infos.size ());
}

/*
======================================================================================
    1. 采用accessorId和docId来唯一标识一条权限
    2. allowValue和denyValue同时提供
    3. 允许权限和拒绝权限的截至时间保持一致
======================================================================================
 */
/* [notxpcom] void GetPermConfig ([const] in StringRef docId, [const] in StringRef accessorId, in dbPermConfigRef permConfig); */
NS_IMETHODIMP_(void) ncDBPermManager::GetPermConfig(const String & docId, const String & accessorId, dbPermConfig & permConfig)
{
    NC_ACS_DB_TRACE (_T("docId: %s, accessorId:%s begin"), docId.getCStr (), accessorId.getCStr ());

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());
    String escDocId = dbOper->EscapeEx(docId);
    String escAccessorId = dbOper->EscapeEx(accessorId);

    String dbName = Util::getDBName("anyshare");
    String strSql;
    // 构造select in 查询语句
    strSql.format (_T("select f_doc_id, f_accessor_id, max(f_accessor_type), "
                      "bit_or(case f_type when 1 then f_perm_value else 0 end) denyvalue, "
                      "bit_or(case f_type when 2 then f_perm_value else 0 end) allowvalue, "
                      "max(f_end_time), min(f_create_time), max(f_modify_time) "
                      "from %s.t_acs_custom_perm "
                      "where f_doc_id = '%s' and f_accessor_id = '%s' "
                      "group by f_doc_id, f_accessor_id"),
                    dbName.getCStr(), escDocId.getCStr (), escAccessorId.getCStr ());

    ncDBRecords results;
    dbOper->Select (strSql, results);

    if (results.size() != 0) {
        permConfig.docId = results[0][0];
        permConfig.accessorId = results[0][1];
        permConfig.accessorType = Int::getValue (results[0][2]);
        permConfig.denyValue = Int::getValue (results[0][3]);
        permConfig.allowValue = Int::getValue (results[0][4]);
        permConfig.endTime = Int64::getValue (results[0][5]);
        permConfig.createTime = Int64::getValue (results[0][6]);
        permConfig.modifyTime = Int64::getValue (results[0][7]);
    }

    NC_ACS_DB_TRACE (_T("docId: %s, accessorId:%s end"), docId.getCStr (), accessorId.getCStr ());
}

/* [notxpcom] void AddPermConfig ([const] in dbPermConfigRef config); */
NS_IMETHODIMP_(void) ncDBPermManager::AddPermConfig(const dbPermConfig & config)
{
    NC_ACS_DB_TRACE (_T("begin"));

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    String escDocId = dbOper->EscapeEx(config.docId);
    String escAccessorId = dbOper->EscapeEx(config.accessorId);
    String escAccessorName = dbOper->EscapeEx(config.accessorName);

    String dbName = Util::getDBName("anyshare");
    String strSql;
    strSql.format("select f_create_time from %s.t_acs_custom_perm where f_accessor_id = '%s' and f_doc_id = '%s'",
                    dbName.getCStr(), escAccessorId.getCStr(), escDocId.getCStr());
    ncDBRecords results;
    dbOper->Select (strSql, results);

    int64 currentTime = BusinessDate::getCurrentTime ();
    int64 createTime = currentTime;
    if (results.size() > 0) {
        createTime = Int64::getValue (results[0][0]);
    }

    try {
        dbOper->StartTransaction ();

        // 先删除访问者所有的配置
        strSql.format("delete from %s.t_acs_custom_perm where f_accessor_id = '%s' and f_doc_id = '%s'",
                        dbName.getCStr(), escAccessorId.getCStr(), escDocId.getCStr());
        dbOper->Execute(strSql);

        // 允许权限值
        if(config.allowValue != 0) {
            strSql.format("insert into %s.t_acs_custom_perm(f_doc_id,f_accessor_id,f_accessor_name,f_accessor_type,f_type,f_perm_value,f_end_time,f_modify_time,f_create_time)"
                            "values('%s','%s', '%s', %d, %d, %d, %lld, %lld, %lld)",
                            dbName.getCStr(),
                            escDocId.getCStr(), escAccessorId.getCStr(), escAccessorName.getCStr(), config.accessorType,
                            2, config.allowValue, config.endTime, currentTime, createTime);
            dbOper->Execute(strSql);
        }
        // 拒绝权限值
        if(config.denyValue != 0) {
            strSql.format("insert into %s.t_acs_custom_perm(f_doc_id,f_accessor_id,f_accessor_name,f_accessor_type,f_type,f_perm_value,f_end_time,f_modify_time,f_create_time)"
                            "values('%s','%s', '%s', %d, %d, %d, %lld, %lld, %lld)",
                            dbName.getCStr(),
                            escDocId.getCStr(), escAccessorId.getCStr(), escAccessorName.getCStr(), config.accessorType,
                            1, config.denyValue, config.endTime, currentTime, createTime);
            dbOper->Execute(strSql);
        }

        dbOper->Commit ();
    }
    catch(Exception&) {
        dbOper->Rollback ();
        throw;
    }

    NC_ACS_DB_TRACE (_T("end"));
}

/* [notxpcom] void DelPermConfig ([const] in StringRef docId, [const] in StringRef accessorId); */
NS_IMETHODIMP_(void) ncDBPermManager::DelPermConfig(const String & docId, const String & accessorId)
{
    NC_ACS_DB_TRACE (_T("begin"));

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    String dbName = Util::getDBName("anyshare");
    // 删除访问者所有的配置
    String strSql;
    strSql.format("delete from %s.t_acs_custom_perm where f_accessor_id = '%s' and f_doc_id = '%s'",
                   dbName.getCStr(), dbOper->EscapeEx(accessorId).getCStr(), dbOper->EscapeEx(docId).getCStr());
    dbOper->Execute(strSql);

    NC_ACS_DB_TRACE (_T("end"));
}

/* [notxpcom] void GetPermConfigsByAccessToken ([const] in StringRef docId, [const] in StringSetRef accessToken, in dbPermConfigVectorRef permConfigs, in bool withInheritedPerm, in bool isAnonymous); */
NS_IMETHODIMP_(void) ncDBPermManager::GetPermConfigsByAccessToken(const String & docId, const set<String> & accessToken, vector<dbPermConfig> & permConfigs, bool withInheritedPerm, bool isAnonymous)
{
    NC_ACS_DB_TRACE (_T("docId: %d, accessToken size: %d begin"),  docId.getCStr (), (int)accessToken.size ());

    permConfigs.clear ();
    int depth = ncGNSUtil::GetPathDepth (docId);
    if (depth == 0 || accessToken.size () == 0) {
        return;
    }

    vector<String> docIds;
    docIds.push_back (docId);
    if (withInheritedPerm) {
        // 向上遍历获取父路径
        for (int i = depth-1; i >= 1; --i) {
            String curGNSPath = ncGNSUtil::GetPathByDepth (docId, i);
            docIds.push_back (curGNSPath);
        }
    }

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    String docIdGroup;
    for (size_t i = 0; i < docIds.size (); ++i) {
        docIdGroup.append ("\'", 1);
        docIdGroup.append (dbOper->EscapeEx(docIds[i]));
        docIdGroup.append ("\'", 1);

        if (i != (docIds.size () -1)) {
            docIdGroup.append (",", 1);
        }
    }

    String accessorIdGroup;
    for (auto iter = accessToken.begin (); iter != accessToken.end ();) {
        accessorIdGroup.append ("\'", 1);
        accessorIdGroup.append (dbOper->EscapeEx(*iter));
        accessorIdGroup.append ("\'", 1);

        if (++iter != accessToken.end ()) {
            accessorIdGroup.append (",", 1);
        }
    }

    // 实名用户才会获取禁用继承权限
    String whereClause;
    if (!isAnonymous) {
        whereClause = _T("f_type=3 or");
    }

    String dbName = Util::getDBName("anyshare");
    String strSql;
    // 构造select in 查询语句
    strSql.format (_T("select f_doc_id, f_accessor_id, max(f_accessor_type), "
                        "bit_or(case f_type when 1 then f_perm_value else 0 end) denyvalue, "
                        "bit_or(case f_type when 2 then f_perm_value else 0 end) allowvalue, "
                        "bit_or(case f_type when 3 then 1 else 0 end) disable_inherit, "
                        "max(f_end_time), min(f_create_time), max(f_modify_time) "
                        "from %s.t_acs_custom_perm "
                        "where "
                        "( %s f_accessor_id in (%s) ) "
                        "and "
                        "f_doc_id in(%s) "
                        "group by f_doc_id, f_accessor_id "
                        "order by length(f_doc_id) desc, disable_inherit asc"),
                        dbName.getCStr(), whereClause.getCStr (), accessorIdGroup.getCStr (), docIdGroup.getCStr ());

    ncDBRecords results;
    dbOper->Select (strSql, results);

    dbPermConfig tmpInfo;
    for (size_t i = 0; i < results.size (); ++i) {
        if (Int::getValue (results[i][5]) == 1) {
            break;
        }

        tmpInfo.docId = results[i][0];
        tmpInfo.accessorId = results[i][1];
        tmpInfo.accessorType = Int::getValue (results[i][2]);
        tmpInfo.denyValue = Int::getValue (results[i][3]);
        tmpInfo.allowValue = Int::getValue (results[i][4]);
        tmpInfo.endTime = Int64::getValue (results[i][6]);
        tmpInfo.createTime = Int64::getValue (results[i][7]);
        tmpInfo.modifyTime = Int64::getValue (results[i][8]);

        permConfigs.push_back (tmpInfo);
    }

    NC_ACS_DB_TRACE (_T("docId: %s, accessToken size: %d end, ret permConfigs size: %d"),  docId.getCStr (), (int)accessToken.size (), (int)permConfigs.size ());
}

/* [notxpcom] void GetAccessPermsOfSubObjs ([const] in StringRef docId, [const] in StringSetRef accessToken, in bool isAnonymous, in dbAccessPermVectorRef perms); */
NS_IMETHODIMP_(void) ncDBPermManager::GetAccessPermsOfSubObjs(const String & docId, const set<String> & accessToken, bool isAnonymous, vector<dbAccessPerm> & perms)
{
    NC_ACS_DB_TRACE (_T("docId: %d, accessToken size: %d begin"),  docId.getCStr (), (int)accessToken.size ());

    if (accessToken.size () == 0) {
        return;
    }

    perms.clear ();

    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (GetDBOperator ());

    String accessorIdGroup;
    for (auto iter = accessToken.begin (); iter != accessToken.end ();) {
        accessorIdGroup.append ("\'", 1);
        accessorIdGroup.append (dbOper->EscapeEx(*iter));
        accessorIdGroup.append ("\'", 1);

        if (++iter != accessToken.end ()) {
            accessorIdGroup.append (",", 1);
        }
    }

    String docIdCond;
    if (ncGNSUtil::GetPathDepth (docId) != 0) {
        docIdCond.format ("and f_doc_id like '%s/%%'", dbOper->EscapeEx(docId).getCStr ());
    }

    // 实名用户才会获取禁用继承权限
    String whereClause;
    if (!isAnonymous) {
        whereClause = _T("f_type=3 or");
    }

    String dbName = Util::getDBName("anyshare");
    String strSql;
    // 构造select in 查询语句
    strSql.format (_T("select f_doc_id, "
                        "bit_or(case f_type when 1 then f_perm_value else 0 end) denyvalue, "
                        "bit_or(case f_type when 2 then f_perm_value else 0 end) allowvalue, "
                        "bit_or(case f_type when 3 then 1 else 0 end) disable_inherit "
                        "from %s.t_acs_custom_perm "
                        "where "
                        "( %s f_accessor_id in (%s) ) %s "
                        "group by f_doc_id "
                        "order by length(f_doc_id) asc"),
                        dbName.getCStr(), whereClause.getCStr (), accessorIdGroup.getCStr (), docIdCond.getCStr ());

    ncDBRecords results;
    dbOper->Select (strSql, results);

    dbAccessPerm tmpInfo;
    for (size_t i = 0; i < results.size (); ++i) {
        tmpInfo.docId = results[i][0];
        tmpInfo.denyValue = Int::getValue (results[i][1]);
        tmpInfo.allowValue = Int::getValue (results[i][2]);
        tmpInfo.inherit = Int::getValue (results[i][3])?false:true;
        tmpInfo.allowValue &= (~tmpInfo.denyValue);

        perms.push_back (tmpInfo);
    }

    NC_ACS_DB_TRACE (_T("docId: %s, accessToken size: %d end, ret perms size: %d"),  docId.getCStr (), (int)accessToken.size (), (int)perms.size ());
}

/*
======================================================================================
 */

ncIDBOperator* ncDBPermManager::GetDBOperator ()
{
    nsCOMPtr<ncIDBOperator> dbOper = getter_AddRefs (ncACSDBGetDBOperator ());
    NS_ADDREF (dbOper.get ());
    return dbOper;
}
