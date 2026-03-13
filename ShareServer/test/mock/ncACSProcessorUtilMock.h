#ifndef __NC_ACS_PROCESSORUTIL_MOCK_H
#define __NC_ACS_PROCESSORUTIL_MOCK_H

#include <gmock/gmock.h>
#include <acsprocessor/ncACSProcessorUtil.h>

struct ncACSPathInfo;


class ncACSProcessorUtilMock: public ncIACSProcessorUtil
{
    XPCOM_OBJECT_MOCK (ncACSProcessorUtilMock)

public:
    MOCK_METHOD2(GetConfValue, string(const string&, const string&));
    MOCK_METHOD7(Log, void(const String&, ncTokenVisitorType typ, ncTLogType::type, ncTLogLevel::type, int,
                        const String&, const String&, const String&));
    MOCK_METHOD0(Usrm_GetTriSystemStatus, bool());
    MOCK_METHOD0(GetSyslogStatus, bool());
    MOCK_METHOD2(GetShareDocStatus, bool(int, int));
    MOCK_METHOD1(SendPermChangeNSQ, void(const String&));
};

#endif // End __NC_ACS_PROCESSORUTIL_MOCK_H
