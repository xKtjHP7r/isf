#include <abprec.h>
#include <ncutil/ncPerformanceProfilerPrec.h>
#include <ncutil/ncBusinessDate.h>
#include <acssharemgnt/public/ncIACSShareMgnt.h>
#include "acsprocessor.h"
#include "ncACSProcessorUtil.h"

#include <ethriftutil/ncThriftClient.h>

#define int64 int64_t

#include <gen-cpp/ncTEVFS.h>
#include <gen-cpp/EVFS_constants.h>

#include "gen-cpp/ncTShareMgnt.h"
#include "gen-cpp/ShareMgnt_constants.h"
#undef int64

#include <acssharemgnt/public/ncIACSShareMgnt.h>

#include <boost/property_tree/ptree.hpp>
#include <boost/property_tree/ini_parser.hpp>

#include <dataapi/ncGNSUtil.h>
#include <ehttpclient/public/ncIEHTTPClient.h>
#include "acsServiceAccessConfig.h"
#include <acsprocessor/public/ncIACSOutboxManager.h>
#include <acsdb/public/ncIDBOutboxManager.h>
#include <drivenadapter/public/authenticationInterface.h>

#define PREFIXSTR                   _T("AnyShare://")
#define GNS_PREFIX                  _T("gns://")
#define GNS_PREFIX_LENGTH           6
#define EOFS_QUERY_ERR_GNS_OBJECT_NOT_EXIST 0x01000005L


AB_DEFINE_THREADSAFE_SINGLETON_NO_POOL (ncACSProcessorUtil)



string ncACSProcessorUtil::GetConfValue (const string& path, const string& key)
{
    string value;
    try {
        boost::property_tree::ptree pt;
        boost::property_tree::ini_parser::read_ini (path, pt);
        value = pt.get<string>(key);
    }
    catch (...) {
        SystemLog::getInstance ()->log (__FILE__, __LINE__, ERROR_LOG_TYPE,
                                        _T("Error : read configuration information from file failed."));
    }

    return value;
}

void ncACSProcessorUtil::Log (const String& userId, ncTokenVisitorType typ, ncTLogType logType,
                     ncTLogLevel level, int opType, const String& msg, const String& exmsg, const String& ip, bool logForwardedIp)
{
#ifndef __UT__
    nsresult ret;
   // 初始化nsq
    nsCOMPtr<authenticationInterface> authentication = do_CreateInstance (AUTHENTICATION_CONTRACTID, &ret);
    if (NS_FAILED (ret)) {
        THROW_E (ACS_PROCESSOR, FAILED_TO_CREATE_DRIVENADAPTER_MANANGER,
            _T("Failed to create authentication instance: 0x%x"), ret);
    }

    // 构建msg thrift接口无mac和User-Agent
    String macAddress;
    String userAgent;

    authentication->AuditLog(userId, typ, logType, level, opType, msg, exmsg, ip, macAddress, userAgent);

#endif
}

bool ncACSProcessorUtil::Usrm_GetTriSystemStatus()
{
#ifndef __UT__
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient(AcsServiceAccessConfig::getInstance()->sharemgntHost, AcsServiceAccessConfig::getInstance()->sharemgntPort);
        return shareMgntClient->Usrm_GetTriSystemStatus();
    }
    catch (ncTException & e) {
        THROW_E (ACS_PROCESSOR, GET_TRI_SYSTEM_STATUS_ERROR, _T("%s"), e.expMsg.c_str ());
    }
    catch (TException & e) {
        THROW_E (ACS_PROCESSOR, GET_TRI_SYSTEM_STATUS_ERROR, _T("%s"), e.what ());
    }
#else
    return true;
#endif
}

bool ncACSProcessorUtil::GetSyslogStatus ()
{
    // try {
    //     ncThriftClient<ncTECMSManagerClient> ecmsManagerClient (_thriftHost, g_ECMSManager_constants.NCT_ECMSMANAGER_PORT);
    //     bool result = ecmsManagerClient->get_upload_log_status ();

    //     // 当syslog开关开启时设置首次推送时间
    //     if (result) {
    //         int timestamp = ecmsManagerClient->get_upload_log_time ();
    //         ncThriftClient<ncTEACPLogClient> logClient (_thriftHost, g_EACPLog_constants.NC_T_EACP_LOG_PORT);
    //         logClient->SetSyslogFirstPushTime (int64 (timestamp) * 1000 * 1000);
    //     }

    //     return result;
    // }
    // catch (ncTException & e) {
    //     THROW_E (ACS_PROCESSOR, GET_SYSLOG_UPLOAD_STATUS_ERROR, _T("%s"), e.expMsg.c_str ());
    // }
    // catch (TException & e) {
    //     THROW_E (ACS_PROCESSOR, GET_SYSLOG_UPLOAD_STATUS_ERROR, _T("%s"), e.what ());
    // }
    return false;
}

bool ncACSProcessorUtil::GetShareDocStatus(int docType, int linkType)
{
#ifndef __UT__
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient(AcsServiceAccessConfig::getInstance()->sharemgntHost, AcsServiceAccessConfig::getInstance()->sharemgntPort);
        return shareMgntClient->GetShareDocStatus(static_cast<ncTDocType::type>(docType), static_cast<ncTTemplateType::type>(linkType));
    }
    catch (ncTException & e) {
        THROW_E (ACS_PROCESSOR, GET_SHARE_DOC_STATUS_ERROR, _T("%s"), e.expMsg.c_str ());
    }
    catch (TException & e) {
        THROW_E (ACS_PROCESSOR, GET_SHARE_DOC_STATUS_ERROR, _T("%s"), e.what ());
    }
#else
    return true;
#endif
}

void ncACSProcessorUtil::SendMail(vector<string>& mailto, const string& subject, const string& content)
{
    try {
        ncThriftClient<ncTShareMgntClient> shareMgntClient (AcsServiceAccessConfig::getInstance()->sharemgntHost, AcsServiceAccessConfig::getInstance()->sharemgntPort);
        shareMgntClient->SMTP_SendEmail(mailto, subject, content);
    }
    catch (ncTException & e) {
        THROW_E (ACS_PROCESSOR, SEND_SHARE_MAIL_ERROR, _T("%s"), e.expMsg.c_str ());
    }
    catch (TException & e) {
        THROW_E (ACS_PROCESSOR, SEND_SHARE_MAIL_ERROR, _T("%s"), e.what ());
    }
    catch (Exception& e){
        THROW_E (ACS_PROCESSOR, SEND_SHARE_MAIL_ERROR, _T("%s"), e.toString ().getCStr ());
    }
}

bool ncACSProcessorUtil::CheckSwitchingNetwork (const String& oldIp, const String& newIp)
{
    NC_ACS_PROCESSOR_TRACE (_T("[BEGIN]this: %p"), this);
    bool old_ret, new_ret;
    old_ret = isLAN(oldIp.getCStr());
    new_ret = isLAN(newIp.getCStr());
    /* 切换逻辑
    内 --> 内  (不需要切换)
    内 --> 外  (需要切换)
    外 --> 内  (需要切换)
    外 --> 外  (需要切换)
    */
    if (old_ret && new_ret) {
        return false;
    } else {
        return true;
    }

    NC_ACS_PROCESSOR_TRACE (_T("[END]this: %p"), this);
}

bool ncACSProcessorUtil::isLAN (const string& realip)
{

    /*
    以下IP范围为内网:
    A类 10.0.0.0     --  10.255.255.255
    B类 172.16.0.0   --  172.31.255.255
    C类 192.168.0.0  --  192.168.255.255
    */

    istringstream ipstream(realip);
    int ip[2];
    for(int i = 0; i < 2; i++) {
        string temp;
        getline(ipstream,temp,'.');
        istringstream t(temp);
        t >> ip[i];
    }
    if ((ip[0] == 10) || (ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31) || (ip[0] == 192 && ip[1] == 168)) {
        return true;
    }
    return false;
}

String ncACSProcessorUtil::UrlEncode3986 (const String & input)
{
    String escaped;

    int max = input.getLength();
    for(int i = 0; i < max; ++ i) {
        if ((48 <= input[i] && input[i] <= 57) ||    //0-9
            (65 <= input[i] && input[i] <= 90) ||    //abc...xyz
            (97 <= input[i] && input[i] <= 122) ||   //ABC...XYZ
            (input[i] == '_' || input[i] == '-' || input[i] == '~' || input[i] == '.')
            ) {
                escaped.append (input[i], 1);
        }
        else {
            escaped.append ("%");

            char dig1 = (input[i]&0xF0)>>4;
            char dig2 = (input[i]&0x0F);
            if ( 0 <= dig1 && dig1 <= 9) dig1 += 48;    //0,48inascii
            if (10 <= dig1 && dig1 <=15) dig1 += 65 - 10; //A,97inascii
            if ( 0 <= dig2 && dig2 <= 9) dig2 += 48;
            if (10 <= dig2 && dig2 <=15) dig2 += 65 - 10;

            String r;
            r.append (&dig1, 1);
            r.append (&dig2, 1);

            escaped.append (r);//converts char 255 to string "FF"
        }
    }

    return escaped;
}

String ncACSProcessorUtil::TimeStampToRFC3339 (int64 usTimeStamp)
{
    usTimeStamp = -1 == usTimeStamp ? 0 : usTimeStamp;

    long ts = usTimeStamp / 1000000;
    struct tm *p;
    p = gmtime (&ts);
    char time[80];
    strftime (time, 80, "%Y-%m-%dT%H:%M:%SZ", p);
    return toCFLString (time);
}

void ncACSProcessorUtil::SendPermChangeNSQ (const String& docID)
{
    nsresult ret;
    nsCOMPtr<ncIDBOutboxManager> dbOutboxManager = do_CreateInstance (NC_DB_OUTBOX_MANAGER_CONTRACTID, &ret);
    if (NS_FAILED (ret)) {
        THROW_E (ACS_PROCESSOR, FAILED_TO_CREATE_OSS_CLIENT,
            _T("Failed to create db outbox manager: 0x%x"), ret);
    }

    nsCOMPtr<ncIACSOutboxManager> acsOutboxManager = do_CreateInstance (NC_ACS_OUTBOX_MANAGER_CONTRACTID, &ret);
    if (NS_FAILED (ret)) {
        THROW_E (ACS_PROCESSOR, FAILED_TO_CREATE_OSS_CLIENT,
            _T("Failed to create acs outbox manager: 0x%x"), ret);
    }

    // 封装outbox信息（nsq）
    JSON::Value outboxMsgJson;
    JSON::Object content;
    content["doc_id"] = docID.getCStr ();
    outboxMsgJson["type"] = (int)ncOutboxType::OUTBOX_PERM_CHANGE;
    outboxMsgJson["content"] = content;
    std::string outboxMsg;
    JSON::Writer::write (outboxMsgJson.o (), outboxMsg);

    // SystemLog::getInstance ()->log (__FILE__, __LINE__, ERROR_LOG_TYPE, "SendPermChangeNSQ docid: %s", docID.getCStr ());

    // SystemLog::getInstance ()->log (__FILE__, __LINE__, ERROR_LOG_TYPE, "SendPermChangeNSQ message: %s", outboxMsg.c_str ());

    // 插入outbox信息
    dbOutboxManager->AddOutboxInfo (outboxMsg.c_str ());
    // 触发outbox消息推送线程
    acsOutboxManager->StartPushOutboxThread ();
    acsOutboxManager->NotifyPushOutboxThread ();
}
