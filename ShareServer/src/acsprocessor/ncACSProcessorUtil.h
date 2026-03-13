#ifndef __NC_ACS_PROCESSOR_UTIL_H
#define __NC_ACS_PROCESSOR_UTIL_H

#include <dataapi/ncJson.h>

#include <acsprocessor/public/ncIACSCommon.h>

#include "gen-cpp/ncTShareMgnt.h"
#include "gen-cpp/ShareMgnt_constants.h"

#include "gen-cpp/ncTEACP.h"
#include "gen-cpp/EACP_constants.h"

#include <drivenadapter/public/authenticationInterface.h>

struct ncACSPathInfo {
    String path;
    String name;
    bool isFile;
};

class ncIACSProcessorUtil {
public:
    virtual string GetConfValue (const string& path, const string& key) = 0;

    // eacplog.thrift
    virtual void Log (const String& userId, ncTokenVisitorType typ, ncTLogType logType,
                     ncTLogLevel level, int opType, const String& msg, const String& exmsg, const String& ip, bool logForwardedIp = false) = 0;

    // sharemgnt.thrift
    virtual bool Usrm_GetTriSystemStatus() = 0;
    virtual bool GetShareDocStatus(int docType, int linkType) = 0;

    //ECMSManager.thrift
    virtual bool GetSyslogStatus () = 0;

    virtual void SendPermChangeNSQ (const String& docID) = 0;
};

class ncACSProcessorUtil: public ncIACSProcessorUtil
{
    AB_DECLARE_THREADSAFE_SINGLETON (ncACSProcessorUtil)
public:
    virtual string GetConfValue (const string& path, const string& key);
    virtual bool CheckSwitchingNetwork (const String& oldIp, const String& newIp);

    // eacplog.thrift
    virtual void Log (const String& userId, ncTokenVisitorType typ, ncTLogType logType,
                     ncTLogLevel level, int opType, const String& msg, const String& exmsg, const String& ip, bool logForwardedIp = false);

    // sharemgnt.thrift
    virtual bool Usrm_GetTriSystemStatus();
    virtual bool GetShareDocStatus(int docType, int linkType);
    virtual void SendMail(vector<string>& mailto, const string& subject, const string& content);

    //ECMSManager.thrift
    virtual bool GetSyslogStatus ();

    virtual String UrlEncode3986 (const String &in);

    // 微秒时间戳转RFC3339
    virtual String TimeStampToRFC3339 (int64 usTimeStamp);

    // 发送权限变更NSQ
    virtual void SendPermChangeNSQ (const String& docID);

private:
    /***
     * 判断是否是内网
     */
    bool isLAN (const string& realip);

};

#endif  // __NC_ACS_PROCESSOR_UTIL_H
