#ifndef __NC_ACS_DB_PERM_MOCK_H
#define __NC_ACS_DB_PERM_MOCK_H

#include <gmock/gmock.h>
#include <acsdb/public/ncIDBPermManager.h>

class ncDBPermManagerMock: public ncIDBPermManager
{
    XPCOM_OBJECT_MOCK (ncDBPermManagerMock);

public:
    MOCK_METHOD2(GetCustomPermByDocIds, void(const vector<String>&, vector<dbCustomPermInfo>&));
    MOCK_METHOD1(AddCustomPerm, void(const dbCustomPermInfo&));
    MOCK_METHOD1(UpdateCustomPerm, void(const dbCustomPermInfo&));
    MOCK_METHOD1(DeleteCustomPerm, void(int64));
    MOCK_METHOD2(GetCustomPermById, bool(int64, dbCustomPermInfo&));
    MOCK_METHOD6(GetCustomPermByPermValue, bool(const String&, const String&, int, bool, int, dbCustomPermInfo&));
    MOCK_METHOD6(GetCustomPermByEndTime, bool(const String&, const String&, int, bool, int64, dbCustomPermInfo&));
    MOCK_METHOD1(DeleteCustomPermByFileId, void(const String&));
    MOCK_METHOD1(DeleteCustomPermByDirId, void(const String&));
    MOCK_METHOD1(DeleteCustomPermByUserId, void(const String&));
    MOCK_METHOD2(DeleteCustomPermByDocUserId, void(const String&, const String&));
    MOCK_METHOD2(GetCustomPerm, bool(const dbCustomPermInfo&, dbCustomPermInfo&));
    MOCK_METHOD2(GetCustomPermByAccessorId, void(const String&, vector <dbCustomPermInfo>&));
    MOCK_METHOD2(GetCustomPermByAccessorIds, void(const vector<String>&, vector <dbCustomPermInfo>&));
    MOCK_METHOD1(GetAllCustomPerm, void(vector <dbCustomPermInfo>&));
    MOCK_METHOD3(GetPermConfig, void(const String&, const String&, dbPermConfig&));
    MOCK_METHOD1(AddPermConfig, void(const dbPermConfig&));
    MOCK_METHOD2(DelPermConfig, void(const String&, const String&));
    MOCK_METHOD4(GetPermConfigs, void(const String&, vector<dbPermConfig>&, bool&, bool));
    MOCK_METHOD2(AddPermConfigs, void(const String&, const vector<dbPermConfig>&));
    MOCK_METHOD2(DelPermConfigs, void(const String&, const vector<String>&));
    MOCK_METHOD5(GetPermConfigsByAccessToken, void(const String&, const set<String>&, vector<dbPermConfig>&, bool, bool));
    MOCK_METHOD4(GetAccessPermsOfSubObjs, void(const String&, const set<String>&, bool, vector<dbAccessPerm>&));
    MOCK_METHOD2(DeleteContactPermByUserID, void(const String&, const String &));
};

#endif // End __NC_ACS_DB_MOCK_H
