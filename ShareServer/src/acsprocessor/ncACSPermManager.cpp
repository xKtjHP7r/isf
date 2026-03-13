#include <abprec.h>
#include <ncutil/ncBusinessDate.h>
#include <dataapi/ncGNSUtil.h>
#include <ethriftutil/ncThriftClient.h>

#include "acsprocessor.h"
#include "ncACSPermManager.h"
#include "ncACSProcessorUtil.h"

#include <ncutil/ncPerformanceProfilerPrec.h>
#include <acsprocessor/public/ncIACSCommon.h>

const String EISOO_SEPARATE = _T("/**eisoo**/");

/* Implementation file */
NS_IMPL_QUERY_INTERFACE1 (ncACSPermManager, ncIACSPermManager)

// protected
NS_IMETHODIMP_(nsrefcnt) ncACSPermManager::AddRef (void)
{
    return 1;
}

// protected
NS_IMETHODIMP_(nsrefcnt) ncACSPermManager::Release (void)
{
    return 1;
}

AB_DEFINE_THREADSAFE_SINGLETON_NO_POOL (ncACSPermManager)


ncACSPermManager::ncACSPermManager()
{
    NC_ACS_PROCESSOR_TRACE (_T("this: %p"), this);

    nsresult ret;
    _dbPermManager = do_CreateInstance (NC_DB_PERM_MANAGER_CONTRACTID, &ret);
    if (NS_FAILED (ret)) {
        THROW_E (ACS_PROCESSOR, FAILED_TO_CREATE_DB_PERM_MANANGER,
            _T("Failed to create db perm manager: 0x%x"), ret);
    }

    _dbOwnerManager = do_CreateInstance (NC_DB_OWNER_MANAGER_CONTRACTID, &ret);
    if (NS_FAILED (ret)) {
        THROW_E (ACS_PROCESSOR, FAILED_TO_CREATE_DB_OWNER_MANANGER,
            _T("Failed to create db owner manager: 0x%x"), ret);
    }

    _acsShareMgnt = do_CreateInstance (NC_ACS_SHARE_MGNT_CONTRACTID, &ret);
    if (NS_FAILED (ret)) {
        THROW_E (ACS_PROCESSOR, FAILED_TO_CREATE_ACS_SHAREMGNT,
            _T("Failed to create acs sharemgnt: 0x%x"), ret);
    }

    _acsConfManager = do_CreateInstance (NC_ACS_CONF_MANAGER_CONTRACTID, &ret);
    if (NS_FAILED (ret)) {
        THROW_E (ACS_PROCESSOR, FAILED_TO_CREATE_ACS_CONF_MANANGER,
            _T("Failed to create acs conf manager: 0x%x"), ret);
    }

    _userManager = do_CreateInstance (USER_MANAGEMENT_CONTRACTID, &ret);
    if (NS_FAILED (ret)) {
        THROW_E (ACS_PROCESSOR, FAILED_TO_CREATE_DRIVENADAPTER_MANANGER,
            _T("Failed to create usermanagement instance: 0x%x"), ret);
    }

    _acsProcessorUtil = ncACSProcessorUtil::getInstance();

}

ncACSPermManager::ncACSPermManager (ncIACSProcessorUtil* acsProcessorUtil,
                  ncIDBPermManager* dbPermManager,
                  ncIDBOwnerManager* dbOwnerManager,
                  ncIACSShareMgnt* sharemgnt,
                  ncIACSConfManager* acsConfManager,
                  userManagementInterface* userManagement)
{
    _acsProcessorUtil = acsProcessorUtil;
    _dbPermManager = dbPermManager;
    _dbOwnerManager = dbOwnerManager;
    _acsShareMgnt = sharemgnt;
    _acsConfManager = acsConfManager;
    _userManager = userManagement;

}

ncACSPermManager::~ncACSPermManager()
{
    NC_ACS_PROCESSOR_TRACE (_T("this: %p"), this);
}

/* [notxpcom] void GetAllCustomPermOwnerInfos (in ncACSOwnerPermVecRef vInfos); */
NS_IMETHODIMP_(void) ncACSPermManager::GetAllCustomPermOwnerInfos(vector<ncOwnerPermInfo>& vInfos)
{
    NC_ACS_PROCESSOR_TRACE (_T("this: %p, begin"), this);

    vInfos.clear();

    vector<dbCustomPermInfo> tmpCustomPermInfos;
    _dbPermManager->GetAllCustomPerm (tmpCustomPermInfos);

    for (size_t i = 0; i <tmpCustomPermInfos.size(); ++i){
        // 获取所有owner信息，包括继承下来的
        vector<dbOwnerInfo> dbOwnerInfos;
        _dbOwnerManager->GetInheritOwnerInfosByDocId (tmpCustomPermInfos[i].docId, dbOwnerInfos, true);

        ncOwnerPermInfo permInfo;
        permInfo.docId = tmpCustomPermInfos[i].docId;
        permInfo.accessorId = tmpCustomPermInfos[i].accessorId;
        permInfo.accessorType = ncAccessorType(tmpCustomPermInfos[i].accessorType);

        vector<String> ownerIds;
        for (size_t j =0; j < dbOwnerInfos.size(); ++j)
            ownerIds.push_back(dbOwnerInfos[j].ownerId);
        permInfo.ownerIds = ownerIds;
        vInfos.push_back(permInfo);
    }

    NC_ACS_PROCESSOR_TRACE (_T("this: %p,ret vinfos size: %d"),
        this, (int)vInfos.size ());
}

/* [notxpcom] void AddCustomPermConfigsWithoutOwner ([const] in StringRef gnsPath, [const] in StringRef userId, [const] in ncCustomPermConfigVectorRef cpConfigs, in ncCustomPermConfigVectorRef addedConfigs); */
NS_IMETHODIMP_(void) ncACSPermManager::AddCustomPermConfigsWithoutOwner(const String & gnsPath, const vector<ncCustomPermConfig> & cpConfigs, vector<ncCustomPermConfig> & addedConfigs)
{
    String userId;
    addCustomPermConfigsHelper(gnsPath, userId, cpConfigs, false, addedConfigs);
}


/* [notxpcom] void DeleteCustomPermByFileId ([const] in StringRef fileId); */
NS_IMETHODIMP_(void) ncACSPermManager::DeleteCustomPermByFileId(const String & fileId)
{
    NC_ACS_PROCESSOR_TRACE (_T("this: %p, fileId: %s begin"),
        this, fileId.getCStr ());

    _dbPermManager->DeleteCustomPermByFileId (fileId);

    NC_ACS_PROCESSOR_TRACE (_T("this: %p, fileId: %s end"),
        this, fileId.getCStr ());
}

/* [notxpcom] void DeleteCustomPermByDirId ([const] in StringRef dirId); */
NS_IMETHODIMP_(void) ncACSPermManager::DeleteCustomPermByDirId(const String & dirId)
{
    NC_ACS_PROCESSOR_TRACE (_T("this: %p, dirId: %s begin"),
        this, dirId.getCStr ());

    _dbPermManager->DeleteCustomPermByDirId (dirId);

    NC_ACS_PROCESSOR_TRACE (_T("this: %p, dirId: %s end"),
        this, dirId.getCStr ());
}

/* [notxpcom] void DeleteCustomPermByUserId ([const] in StringRef userId); */
NS_IMETHODIMP_(void) ncACSPermManager::DeleteCustomPermByUserId(const String & userId)
{
    NC_ACS_PROCESSOR_TRACE (_T("this: %p, userId: %s begin"),
        this, userId.getCStr ());

    _dbPermManager->DeleteCustomPermByUserId (userId);

    NC_ACS_PROCESSOR_TRACE (_T("this: %p, userId: %s end"),
        this, userId.getCStr ());
}

/* [notxpcom] void DeleteCustomPermByDocUserId ([const] in StringRef docId, [const] in StringRef userId); */
NS_IMETHODIMP_(void) ncACSPermManager::DeleteCustomPermByDocUserId(const String & docId, const String & userId)
{
    NC_ACS_PROCESSOR_TRACE (_T("this: %p, docId: %s, userId: %s begin"),
        this, docId.getCStr (), userId.getCStr ());

    _dbPermManager->DeleteCustomPermByDocUserId (docId, userId);

    NC_ACS_PROCESSOR_TRACE (_T("this: %p, docId: %s, userId: %s end"),
        this, docId.getCStr (), userId.getCStr ());
}

/* [notxpcom] ncCheckPermCode CheckPermission ([const] in ncSubjectAtrrRef subjectAttr, [const] in ncObjectAtrrRef objAttr, [const] in ncOpsAtrrRef optAttr, [const] in ncCheckFlagsRef checkFlags); */
NS_IMETHODIMP_(ncCheckPermCode) ncACSPermManager::CheckPermission(const ncSubjectAttr &subjectAttr, const ncObjectAttr &objAttr, const ncOpsAttr &optAttr, int checkFlags)
{
    NC_ACS_PROCESSOR_TRACE (_T("this: %p, userId: %s, gnsPath: %s, permValue: %d begin"),
        this, subjectAttr.userId.getCStr (), objAttr.gnsPath.getCStr (), optAttr.permValue);

    String userId = subjectAttr.userId;
    String accessIp = subjectAttr.accessIp;
    bool isAnonymous = ncVisitorType::ANONYMOUS == subjectAttr.visitorType;
    String docId = objAttr.gnsPath;
    int reqPerm = optAttr.permValue;
    String cid = ncGNSUtil::GetCIDPath (docId);

    // 检查网段文档库限制
    if (!isAnonymous && _acsConfManager->GetNetDocsLimitStatus() && !_acsShareMgnt->CheckNetDocLimit(cid, accessIp)) {
        return CHECK_DENY;
    }

    // 所有者，具有所有权限
    if (!isAnonymous && _dbOwnerManager->IsOwner (docId, userId)) {
        NC_ACS_PROCESSOR_TRACE (_T("this: %p, userId: %s, gnsPath: %s, permValue: %d end, owner"),
            this, userId.getCStr (), docId.getCStr (), reqPerm);
        return CHECK_OK;
    }

    // 获取访问者的访问令牌
    set<String> accessorIds;
    getAccessorIdsByUserId (userId, isAnonymous, accessorIds);

    // 根据访问令牌获取目录上的权限配置
    vector<dbPermConfig> permConfigs;
    _dbPermManager->GetPermConfigsByAccessToken (docId, accessorIds, permConfigs, true, isAnonymous);
    // 检查该文件夹自身的权限
    ncCheckPermCode checkCode = checkPerm(permConfigs, reqPerm);
    // 匿名用户检查权限 或 父目录权限检查通过 或 不是需要递归检查的权限：显示、下载，直接返回
    if ( isAnonymous || checkCode == CHECK_OK ||
        (reqPerm != ACS_AP_DISPLAY && reqPerm != (ACS_AP_DISPLAY|ACS_AP_READ)) ) {
        return checkCode;
    }

    // 对于权限：显示、下载，如果子对象上有权限，则父目录有权限

    // 获取子对象的所有者配置，目录深度由浅至深
    vector<String> subObjsAsOwner;
    _dbOwnerManager->GetSubObjsByUserId (docId, userId, subObjsAsOwner);
    // 是子对象的所有者
    if (subObjsAsOwner.size () != 0) {
        return CHECK_OK;
    }

    // 获取子对象的访问权限，目录深度由浅至深
    vector<dbAccessPerm> subObjPerms;
    _dbPermManager->GetAccessPermsOfSubObjs (docId, accessorIds, isAnonymous, subObjPerms);
    // 对子对象有权限
    for (auto subObjPermIter = subObjPerms.begin (); subObjPermIter != subObjPerms.end (); ++subObjPermIter) {
        dbAccessPerm tmpAccessPerm = *subObjPermIter;
        if ((reqPerm & (~tmpAccessPerm.allowValue)) == 0) {
            return CHECK_OK;
        }
    }

    return checkCode;
}

/* [notxpcom] ncAccessPerm GetPermission ([const] in StringRef userId, [const] in StringRef gnsPath); */
NS_IMETHODIMP_(ncAccessPerm) ncACSPermManager::GetPermission(const String & userId, const String & gnsPath)
{
    NC_ACS_PROCESSOR_TRACE (_T("this: %p, userId: %s, gnsPath: %s"),
        this, userId.getCStr (), gnsPath.getCStr ());

    ncAccessPerm result;
    result.allowValue = 0;
    result.denyValue = 0;

    // 如果是所有者，具有所有权限
    if (_dbOwnerManager->IsOwner (gnsPath, userId)) {
        result.allowValue = ACS_CP_MAX;
        result.denyValue = 0;

        return result;
    }

    // 获取与用户相关的权限配置
    vector<dbPermConfig> infos;
    getPermConfigsByUserId(gnsPath, userId, infos, true);

    return calcPerm (infos);
}

/* [notxpcom] void ListEntryDocsWithLongPath ([const] in ncSubjectAttrRef subjectAttr, [const] in ncObjectAttrRef objectAttr, in StringNcAccessPermMapRef displayIdsMap); */
NS_IMETHODIMP_(void) ncACSPermManager::ListEntryDocsWithLongPath(const ncSubjectAttr & subjectAttr, const ncObjectAttr & objectAttr, map<String, ncAccessPerm> & displayIdsMap)
{
    NC_ACS_PROCESSOR_TRACE (_T("this: %p, userId: %s, gnsPath: %s begin"),
        this, subjectAttr.userId.getCStr (), objectAttr.gnsPath.getCStr ());

    displayIdsMap.clear();

    String userId = subjectAttr.userId;
    String accessIp = subjectAttr.accessIp;
    bool isAnonymous = ncVisitorType::ANONYMOUS == subjectAttr.visitorType;
    String docId = "gns://";

    const int reqPerm = ACS_AP_DISPLAY;
    // 获取访问者的访问令牌
    set<String> accessorIds;
    getAccessorIdsByUserId (userId, isAnonymous, accessorIds);

    // 所有能访问的docId
    set<String> docIdSet;

    // 根据访问令牌获取目录上的权限配置
    map<String, dbAccessPerm*> subObjPermMap;
    vector<dbAccessPerm> subObjPerms;
    _dbPermManager->GetAccessPermsOfSubObjs (docId, accessorIds, isAnonymous, subObjPerms);
    for (size_t i = 0; i < subObjPerms.size (); ++i) {
        if (subObjPerms[i].allowValue & reqPerm) {
            docIdSet.insert (subObjPerms[i].docId);
        }
        subObjPermMap[subObjPerms[i].docId] = &subObjPerms[i];
    }

    // 获取子对象的所有者配置
    set<String> ownerSet;
    vector<String> subObjsAsOwner;
    _dbOwnerManager->GetSubObjsByUserId (docId, userId, subObjsAsOwner);
    for (size_t i = 0; i < subObjsAsOwner.size (); ++i) {
        docIdSet.insert(subObjsAsOwner[i]);
        ownerSet.insert(subObjsAsOwner[i]);
    }

    // 去除子孙文件夹，只保留父文件夹
    // 找出所有的parent目录
    if (docIdSet.empty ()) {
        return;
    }
    auto iter = docIdSet.begin ();
    String lastFilterStr = *iter;
    ++iter;
    for (; iter != docIdSet.end ();) {
        String curStr = *iter;
        if (curStr.find (lastFilterStr) == String::NO_POSITION) {
            lastFilterStr = *iter;
            ++iter;
        }
        else {
            iter = docIdSet.erase (iter);
        }
    }

    // 过滤受到网段限制的文档库
    if (!accessIp.isEmpty() && _acsConfManager->GetNetDocsLimitStatus()) {
        _acsShareMgnt->FilterByNetDocLimit(docIdSet, accessIp);
    }
    if (docIdSet.empty ()) {
        return;
    }

    // 计算权限
    for (auto iter = docIdSet.begin (); iter != docIdSet.end(); ++iter) {
        String docId = *iter;
        // 所有者拥有修改权限
        if(ownerSet.count(docId)) {
            displayIdsMap[docId] = ncAccessPerm(ACS_CP_MAX, 0);
        }
        else {
            // 向上遍历获取父路径权限,计算最终权限
            int finalDenyValue = 0;
            int finalAllowValue = 0;
            for (int i = ncGNSUtil::GetPathDepth (docId); i >= 1; --i) {
                String tmpDocId = ncGNSUtil::GetPathByDepth (docId, i);
                auto iter = subObjPermMap.find (tmpDocId);
                if (iter != subObjPermMap.end ()) {
                    dbAccessPerm &accessPerm = *(iter->second);
                    int tmpDenyValue = accessPerm.denyValue;
                    int tmpAllowValue = accessPerm.allowValue;
                    // 不同层级上，就近原则
                    // 下层已允许，上层拒绝无效
                    tmpDenyValue &= (~finalAllowValue);
                    // 下层已拒绝，上层允许无效
                    tmpAllowValue &= (~finalDenyValue);

                    // 汇总权限
                    finalDenyValue |= tmpDenyValue;
                    finalAllowValue |= tmpAllowValue;

                    // 若禁用继承, 则不再往上获取父路径权限信息
                    if (false == accessPerm.inherit) {
                        break;
                    }
                }
            }
            displayIdsMap[docId] = ncAccessPerm(finalAllowValue, finalDenyValue);
        }
    }
    NC_ACS_PROCESSOR_TRACE (_T("end"));
}

void ncACSPermManager::getAccessorIdsByUserId (const String& userId, bool isAnonymous, set<String>& accessorIds)
{
    accessorIds.clear ();

    // 匿名用户的访问令牌只有自己
    if (isAnonymous) {
        accessorIds.insert (userId);
        return;
    }

    // 获取用户访问令牌
    _userManager->GetAccessorIDsByUserID(userId, accessorIds);
}

void ncACSPermManager::getPermConfigsByUserId(const String & docId, const String & userId, vector<dbPermConfig>& permConfigs, bool withInheritedPerm)
{
    NC_ADD_PPN_LIFE_CYCLE_EE ();

    bool isAnonymous = false;

    // 获取用户相关的accessorId
    set<String> accessorIds;
    getAccessorIdsByUserId (userId, isAnonymous, accessorIds);

    // 根据docId和accessorId查询权限配置
    _dbPermManager->GetPermConfigsByAccessToken (docId, accessorIds, permConfigs, withInheritedPerm, isAnonymous);
}

ncCheckPermCode ncACSPermManager::checkPerm(const vector<dbPermConfig>& permInfos, int permValue)
{
    int tmpDenyValue = 0;
    int tmpAllowValue = 0;
    int tmpDepth = 0;
    // 目录层级由深至浅遍历
    for (size_t i = 0; i < permInfos.size (); ++i) {
        int curDepth = ncGNSUtil::GetPathDepth (permInfos[i].docId);
        if (curDepth == tmpDepth) {
            tmpDenyValue |= permInfos[i].denyValue;
            tmpAllowValue |= permInfos[i].allowValue;
        }
        else {
            // 同一对象上，拒绝优先
            tmpAllowValue &= (~tmpDenyValue);

            // 检查权限
            // 请求的权限中，任意一个被拒绝，则拒绝
            if (tmpDenyValue & permValue) {
                return CHECK_DENY;
            }

            // 已被允许的权限，不再做检查
            permValue &= (~tmpAllowValue);

            // 请求的权限都被允许，则允许
            if (permValue == 0) {
                return CHECK_OK;
            }

            // 向上一层遍历
            tmpDepth = curDepth;
            tmpDenyValue = permInfos[i].denyValue;
            tmpAllowValue = permInfos[i].allowValue;
        }
    }
    // 同一对象上，拒绝优先
    tmpAllowValue &= (~tmpDenyValue);

    // 检查权限
    // 请求的权限中，任意一个被拒绝，则拒绝
    if (tmpDenyValue & permValue) {
        return CHECK_DENY;
    }

    // 已被允许的权限，不再做检查
    permValue &= (~tmpAllowValue);

    // 请求的权限都被允许，则允许
    if (permValue == 0) {
        return CHECK_OK;
    }
    return CHECK_NOT_ALLOW;
}

void ncACSPermManager::removeDuplicateStrs (vector<String>& strs)
{
    // 先进行排序
    sort (strs.begin(), strs.end());

    // 在删除掉相邻重复的
    vector<String>::iterator pos = unique (strs.begin(), strs.end());

    // 删除掉最后无效的条目
    strs.erase (pos, strs.end());
}

ncAccessPerm ncACSPermManager::calcPermByAccessToken(const vector<dbPermConfig> permInfos, const set<String> accessorIds)
{
    int finalDenyValue = 0;
    int finalAllowValue = 0;
    int tmpDenyValue = 0;
    int tmpAllowValue = 0;
    int tmpDepth = 0;
    // 目录层级由深至浅遍历
    for (size_t i = 0; i < permInfos.size (); ++i) {
        if(accessorIds.count(permInfos[i].accessorId) == 0)
            continue;

        int curDepth = ncGNSUtil::GetPathDepth (permInfos[i].docId);
        if (curDepth == tmpDepth) {
            tmpDenyValue |= permInfos[i].denyValue;
            tmpAllowValue |= permInfos[i].allowValue;
        }
        else {
            // 同一对象上，拒绝优先
            tmpAllowValue &= (~tmpDenyValue);

            // 不同层级上，就近原则
            // 下层已允许，上层拒绝无效
            tmpDenyValue &= (~finalAllowValue);
            // 下层已拒绝，上层允许无效
            tmpAllowValue &= (~finalDenyValue);

            // 汇总权限
            finalDenyValue |= tmpDenyValue;
            finalAllowValue |= tmpAllowValue;

            // 向上一层遍历
            tmpDepth = curDepth;
            tmpDenyValue = permInfos[i].denyValue;
            tmpAllowValue = permInfos[i].allowValue;
        }
    }
    // 同一对象上，拒绝优先
    tmpAllowValue &= (~tmpDenyValue);

    // 不同层级上，就近原则
    // 下层已允许，上层拒绝无效
    tmpDenyValue &= (~finalAllowValue);
    // 下层已拒绝，上层允许无效
    tmpAllowValue &= (~finalDenyValue);

    // 汇总权限
    finalDenyValue |= tmpDenyValue;
    finalAllowValue |= tmpAllowValue;

    return ncAccessPerm(finalAllowValue, finalDenyValue);
}

ncAccessPerm ncACSPermManager::calcPerm(const vector<dbPermConfig> permInfos)
{
    int finalDenyValue = 0;
    int finalAllowValue = 0;
    int tmpDenyValue = 0;
    int tmpAllowValue = 0;
    int tmpDepth = 0;
    // 目录层级由深至浅遍历
    for (size_t i = 0; i < permInfos.size (); ++i) {
        int curDepth = ncGNSUtil::GetPathDepth (permInfos[i].docId);
        if (curDepth == tmpDepth) {
            tmpDenyValue |= permInfos[i].denyValue;
            tmpAllowValue |= permInfos[i].allowValue;
        }
        else {
            // 同一对象上，拒绝优先
            tmpAllowValue &= (~tmpDenyValue);

            // 不同层级上，就近原则
            // 下层已允许，上层拒绝无效
            tmpDenyValue &= (~finalAllowValue);
            // 下层已拒绝，上层允许无效
            tmpAllowValue &= (~finalDenyValue);

            // 汇总权限
            finalDenyValue |= tmpDenyValue;
            finalAllowValue |= tmpAllowValue;

            // 向上一层遍历
            tmpDepth = curDepth;
            tmpDenyValue = permInfos[i].denyValue;
            tmpAllowValue = permInfos[i].allowValue;
        }
    }
    // 同一对象上，拒绝优先
    tmpAllowValue &= (~tmpDenyValue);

    // 不同层级上，就近原则
    // 下层已允许，上层拒绝无效
    tmpDenyValue &= (~finalAllowValue);
    // 下层已拒绝，上层允许无效
    tmpAllowValue &= (~finalDenyValue);

    // 汇总权限
    finalDenyValue |= tmpDenyValue;
    finalAllowValue |= tmpAllowValue;

    return ncAccessPerm(finalAllowValue, finalDenyValue);
}

void ncACSPermManager::addCustomPermConfigsHelper(const String & gnsPath, const String& userId, const vector<ncCustomPermConfig> & cpConfigs, bool checkOwner, vector<ncCustomPermConfig> & addedConfigs)
{
    NC_ACS_PROCESSOR_TRACE (_T("this: %p, gnsPath: %s end, cpConfigs size: %d begin"),
        this, gnsPath.getCStr (), (int)cpConfigs.size ());


    addedConfigs.clear();

    if(cpConfigs.size() >= 128) {
        THROW_E (ACS_PROCESSOR, -1, "IDS_EXCEED_NUM_OF_MAX_CONFIGS");
    }

    // 检查是否是所有者
    if(checkOwner) {
        if(!_dbOwnerManager->IsOwner(gnsPath, userId)) {
            THROW_E (ACS_PROCESSOR, NEED_TO_BE_OWNER, LOAD_STRING (_T("IDS_NEED_TO_BE_OWER_TO_SET_PERM")));
        }
    }

    // 获取所有权限信息的访问者名信息
    ncOrgIDInfo orgIDInfo;
    for (int i = 0; i < cpConfigs.size (); ++i){
        switch (cpConfigs[i].accessorType) {
            case ACS_USER:
                orgIDInfo.vecUserIDs.push_back(cpConfigs[i].accessorId);
                break;
            case ACS_DEPARTMENT:
                orgIDInfo.vecDepartIDs.push_back(cpConfigs[i].accessorId);
                break;
            case ACS_CONTACTOR:
                orgIDInfo.vecContactorIDs.push_back(cpConfigs[i].accessorId);
                break;
            case ACS_GROUP:
                orgIDInfo.vecGroupIDs.push_back(cpConfigs[i].accessorId);
                break;
            default:
                break;
        }
    }
    ncOrgNameIDInfo oNameIDInfo;
    _userManager->GetOrgNameIDInfo (orgIDInfo, oNameIDInfo);

    // 配置组织
    bool isReadPermChanged = false;
    dbCustomPermInfo oldInfo;
    dbCustomPermInfo newInfo;

    for (size_t i = 0; i < cpConfigs.size (); ++i) {

        // 找到时间相同的配置项
        bool ret = _dbPermManager->GetCustomPermByEndTime(gnsPath, cpConfigs[i].accessorId,
            cpConfigs[i].accessorType, cpConfigs[i].isAllowed, cpConfigs[i].endTime, oldInfo);

        if(ret) {
            // 权限值已存在，则直接返回
            // 这里注意 == 的优先级高于 |，所以 | 要加括号
            if((oldInfo.permValue | cpConfigs[i].permValue) == oldInfo.permValue) {
                continue;
            }

            newInfo.id = oldInfo.id;
            newInfo.source = (int)permSourceType::permSourceUser;
            newInfo.isAllowed = oldInfo.isAllowed;
            newInfo.permValue = oldInfo.permValue | cpConfigs[i].permValue;
            newInfo.accessorType = oldInfo.accessorType;
            newInfo.accessorId = oldInfo.accessorId;
            newInfo.docId = oldInfo.docId;
            newInfo.endTime = oldInfo.endTime;

            // 进行编辑权限
            _dbPermManager->UpdateCustomPerm(newInfo);
            addedConfigs.push_back(cpConfigs[i]);

            // otag变更
            if(((oldInfo.permValue & ACS_AP_DISPLAY) == 0) &&
                (newInfo.permValue & ACS_AP_DISPLAY)) {
                    isReadPermChanged = true;
            }
            if(((oldInfo.permValue & ACS_AP_PREVIEW) == 0) &&
                (newInfo.permValue & ACS_AP_PREVIEW)) {
                    isReadPermChanged = true;
            }
            if(((oldInfo.permValue & ACS_AP_READ) == 0) &&
                (newInfo.permValue & ACS_AP_READ)) {
                    isReadPermChanged = true;
            }
            if(((oldInfo.permValue & ACS_AP_EDIT) == 0) &&
                (newInfo.permValue & ACS_AP_EDIT)) {
                    isReadPermChanged = true;
            }
            if(((oldInfo.permValue & ACS_AP_CACHE) == 0) &&
                (newInfo.permValue & ACS_AP_CACHE)) {
                    isReadPermChanged = true;
            }
            if(((oldInfo.permValue & ACS_AP_PRINT) == 0) &&
                (newInfo.permValue & ACS_AP_PRINT)) {
                    isReadPermChanged = true;
            }
        }
        else {
            newInfo.id = -1;
            newInfo.docId = gnsPath;
            newInfo.source = (int)permSourceType::permSourceUser;
            newInfo.isAllowed = cpConfigs[i].isAllowed;
            newInfo.permValue = cpConfigs[i].permValue;
            newInfo.accessorType = cpConfigs[i].accessorType;
            newInfo.accessorId = cpConfigs[i].accessorId;
            newInfo.accessorName = getNameFromOrgNameIDInfo (cpConfigs[i].accessorId, cpConfigs[i].accessorType, oNameIDInfo);
            newInfo.endTime = cpConfigs[i].endTime;

            _dbPermManager->AddCustomPerm (newInfo);
            addedConfigs.push_back(cpConfigs[i]);

            // 添加的配置包含显示或者修改
            if ((newInfo.permValue & ACS_AP_DISPLAY) ||
                (newInfo.permValue & ACS_AP_PREVIEW) ||
                (newInfo.permValue & ACS_AP_READ) ||
                (newInfo.permValue & ACS_AP_EDIT) ||
                (newInfo.permValue & ACS_AP_CACHE) ||
                (newInfo.permValue & ACS_AP_PRINT)) {
                    isReadPermChanged = true;
            }
        }
    }

    // handler
    if (isReadPermChanged) {
        // 发送权限变更NSQ
        _acsProcessorUtil->SendPermChangeNSQ (gnsPath);
    }

    NC_ACS_PROCESSOR_TRACE (_T("this: %p, gnsPath: %s end, cpConfigs size: %d end"),
        this, gnsPath.getCStr (), (int)cpConfigs.size ());
}

String ncACSPermManager::getNameFromOrgNameIDInfo(const String& id, const ncAccessorType& type, ncOrgNameIDInfo& info)
{
    String strName;

    map<String, String> *pData = nullptr;
    switch (type) {
        case ACS_USER:
            pData = &(info.mapUserInfo);
            break;
        case ACS_DEPARTMENT:
            pData = &(info.mapDepartInfo);
            break;
        case ACS_CONTACTOR:
            pData = &(info.mapContactorInfo);
            break;
        case ACS_GROUP:
            pData = &(info.mapGroupInfo);
            break;
        default:
            break;
    }

    if (pData) {
        auto iter = pData->find(id);
        if (iter != pData->end()){
            strName = iter->second;
        }
    }
    return strName;
}

/*
======================================================================================
    1. 采用accessorId和docId来唯一标识一条权限
    2. allowValue和denyValue同时提供
    3. 允许权限和拒绝权限的截至时间保持一致
======================================================================================

/*
======================================================================================
 */

/* [notxpcom] int64 GetLatestShareTime ([const] in StringRef userId, [const] in StringRef docId, [const] in ncVisitorType visitorType, in SetStringRef subShareTimeSet); */
NS_IMETHODIMP_(int64) ncACSPermManager::GetLatestShareTime(const String & userId, const String & docId, const ncVisitorType visitorType, set<String> & subShareTimeSet)
{
    bool isAnonymous = ncVisitorType::ANONYMOUS == visitorType;

    // 获取与用户相关的所有者配置
    if (!isAnonymous && _dbOwnerManager->IsOwner (docId, userId)) {
        return LLONG_MAX;
    }

    subShareTimeSet.clear ();

    // 获取用户相关的accessorId
    set<String> accessorIds;
    getAccessorIdsByUserId (userId, isAnonymous, accessorIds);
    // 获取与用户相关的权限配置
    vector<dbPermConfig> permInfos;
    _dbPermManager->GetPermConfigsByAccessToken (docId, accessorIds, permInfos, true, isAnonymous);

    int tmpDepth = 0;
    int64 tmpLatestTime = 0;
    int64 latestTime = 0;
    // 目录层级由深至浅遍历
    for (size_t i = 0; i < permInfos.size (); ++i) {
        int curDepth = ncGNSUtil::GetPathDepth (permInfos[i].docId);
        if (curDepth == tmpDepth) {
            if (permInfos[i].denyValue & ACS_AP_DISPLAY) {
                tmpLatestTime = 0;
                break;
            }
            if (permInfos[i].allowValue & ACS_AP_DISPLAY && permInfos[i].modifyTime > tmpLatestTime) {
                tmpLatestTime = permInfos[i].modifyTime;
            }
        }
        else {
            if (tmpLatestTime > latestTime) {
                latestTime = tmpLatestTime;
            }

            // 向上一层遍历
            tmpDepth = curDepth;
            tmpLatestTime = 0;
            if (permInfos[i].denyValue & ACS_AP_DISPLAY) {
                break;
            }
            if (permInfos[i].allowValue & ACS_AP_DISPLAY) {
                tmpLatestTime = permInfos[i].modifyTime;
            }
        }
    }

    if (tmpLatestTime > latestTime) {
        latestTime = tmpLatestTime;
    }

    if (!isAnonymous) {
        int depth = ncGNSUtil::GetPathDepth (docId) + 1;
        // 获取子对象的权限配置
        vector<dbAccessPerm> accessPerms;
        _dbPermManager->GetAccessPermsOfSubObjs (docId, accessorIds, isAnonymous, accessPerms);
        for (auto iter = accessPerms.begin (); iter != accessPerms.end (); ++iter) {
            if (iter->allowValue & ACS_AP_DISPLAY) {
                subShareTimeSet.insert (ncGNSUtil::GetPathByDepth (iter->docId, depth));
            }
        }
        // 获取为所有者的子对象
        vector<String> subObjsAsOwner;
        _dbOwnerManager->GetSubObjsByUserId (docId, userId, subObjsAsOwner);
        for (auto iter = subObjsAsOwner.begin (); iter != subObjsAsOwner.end (); ++iter) {
            subShareTimeSet.insert (ncGNSUtil::GetPathByDepth (*iter, depth));
        }
    }

    return latestTime;
}
