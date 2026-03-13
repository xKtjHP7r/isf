#ifndef __NC_ACS_PERM_MANAGER_MOCK_H
#define __NC_ACS_PERM_MANAGER_MOCK_H

#include <gmock/gmock.h>
#include <acsprocessor/public/ncIACSPermManager.h>

class ncACSPermManagerMock: public ncIACSPermManager
{
    XPCOM_OBJECT_MOCK (ncACSPermManagerMock)

public:
    MOCK_METHOD1(GetAllCustomPermOwnerInfos, void(vector<ncOwnerPermInfo>&));
    MOCK_METHOD3(AddCustomPermConfigsWithoutOwner, void(const String&, const vector<ncCustomPermConfig>&, vector<ncCustomPermConfig>&));
    MOCK_METHOD1(DeleteCustomPermByFileId, void(const String&));
    MOCK_METHOD1(DeleteCustomPermByDirId, void(const String&));
    MOCK_METHOD1(DeleteCustomPermByUserId, void(const String&));
    MOCK_METHOD4(CheckPermission, ncCheckPermCode(const ncSubjectAttr&, const ncObjectAttr&, const ncOpsAttr&, int));
    MOCK_METHOD2(GetPermission, ncAccessPerm(const String&, const String&));
    MOCK_METHOD3(ListEntryDocsWithLongPath, void(const ncSubjectAttr&, const ncObjectAttr&, map<String, ncAccessPerm>&));
    MOCK_METHOD2(DeleteCustomPermByDocUserId, void(const String&, const String&));
    MOCK_METHOD3(GetPermConfig, void(const String&, const String&, ncPermConfig&));
    MOCK_METHOD2(AddPermConfig, void(const ncPermConfig&, bool));
    MOCK_METHOD3(DelPermConfig, void(const String&, const String&, bool));
    MOCK_METHOD4(GetPermConfigs, void(const String&, vector<ncPermConfig>&, bool&, bool));
    MOCK_METHOD2(AddPermConfigs, void(const String&, const vector<ncPermConfig>&));
    MOCK_METHOD4(GetLatestShareTime, int64(const String&, const String&, const ncVisitorType, set<String>&));
    MOCK_METHOD10(CheckPermConfigs, void(const String&, const String&, bool, bool&, vector<ncPermConfig>&, vector<ncPermConfig>&, vector<ncPermConfig>&, vector<ncPermConfig>& , vector<String>& , map<String, ncPermPair>&));
    MOCK_METHOD1(IsDocCreatorFreeze, bool (const String&));
    MOCK_METHOD1(IsDocCreatorRealNameAuth, bool (const String&));
    MOCK_METHOD2(DeleteContactPermByUserID, void(const String&, const String &));
};

#endif // End __NC_ACS_PERM_MANAGER_MOCK_H
