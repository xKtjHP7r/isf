#ifndef __NC_ACS_PERM_MANAGER_H
#define __NC_ACS_PERM_MANAGER_H

#include <acsdb/public/ncIDBPermManager.h>
#include <acsdb/public/ncIDBOwnerManager.h>
#include <acssharemgnt/public/ncIACSShareMgnt.h>
#include "./public/ncIACSPermManager.h"
#include "drivenadapter/public/userManagementInterface.h"

#include "ncACSProcessorUtil.h"
#include "./public/ncIACSConfManager.h"

/* Header file */
class ncACSPermManager : public ncIACSPermManager
{
    AB_DECLARE_THREADSAFE_SINGLETON (ncACSPermManager)

public:
    NS_DECL_ISUPPORTS
    NS_DECL_NCIACSPERMMANAGER

    ncACSPermManager();
    ncACSPermManager(ncIACSProcessorUtil* acsProcessorUtil, ncIDBPermManager* dbPermManager,
                        ncIDBOwnerManager* dbOwnerManager,
                        ncIACSShareMgnt* sharemgnt,
                        ncIACSConfManager* acsConfManager,
                        userManagementInterface* userManagement);

private:
    ~ncACSPermManager();

    void getAccessorIdsByUserId (const String& userId, bool isAnonymous, set<String>& accessorIds);
    void getPermConfigsByUserId(const String & docId, const String & userId, vector<dbPermConfig>& permConfigs, bool withInheritedPerm);
    ncCheckPermCode checkPerm(const vector<dbPermConfig>& permInfos, int permValue);
    void removeDuplicateStrs (vector<String>& strs);
    ncAccessPerm calcPermByAccessToken(const vector<dbPermConfig> permInfos, const set<String> accessorIds);
    ncAccessPerm calcPerm(const vector<dbPermConfig> permInfos);

    void addCustomPermConfigsHelper(const String & gnsPath, const String& userId, const vector<ncCustomPermConfig> & cpConfigs, bool checkOwner, vector<ncCustomPermConfig> & addedConfigs);

    String getNameFromOrgNameIDInfo(const String& id, const ncAccessorType& type, ncOrgNameIDInfo& info);

protected:
    nsCOMPtr<ncIDBPermManager>          _dbPermManager;
    nsCOMPtr<ncIDBOwnerManager>         _dbOwnerManager;
    nsCOMPtr<ncIACSShareMgnt>           _acsShareMgnt;
    ncIACSProcessorUtil*                _acsProcessorUtil;
    nsCOMPtr<ncIACSConfManager>         _acsConfManager;
    nsCOMPtr<userManagementInterface>   _userManager;
};

#endif // __NC_ACS_PERM_MANAGER_H
