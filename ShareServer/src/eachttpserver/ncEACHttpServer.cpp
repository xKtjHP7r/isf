#include "eachttpserver.h"

#include <acsprocessor/public/ncIACSTokenManager.h>
#include <acssharemgnt/public/ncIACSShareMgnt.h>
#include <acsprocessor/public/ncIACSLockManager.h>
#include <acsprocessor/public/ncIACSDeviceManager.h>
#include <acsprocessor/public/ncIACSConfManager.h>
#include <acsprocessor/public/ncIACSMessageManager.h>
#include <acsprocessor/public/ncIACSPolicyManager.h>
#include "drivenadapter/public/policyEngineInterface.h"
#include <acsprocessor/public/ncIACSOutboxManager.h>

#include "ncEACHttpServer.h"
#include "./auth/ncEACPKIHandler.h"
#include "./auth/ncEACCAuthHandler.h"
#include "./auth/ncEACAuthHandler.h"
#include "./auth/ncEACDeviceHandler.h"

#include "./share/ncEACUserHandler.h"
#include "./share/ncEACDepartmentHandler.h"
#include "./share/ncEACContactorHandler.h"

#include "./message/ncEACMessageHandler.h"
#include "./sysconfig/ncEACConfigHandler.h"
#include "./third/ncEACThirdHandler.h"

#include <ethriftutil/ncThriftClient.h>
#include "gen-cpp/ncTEACP.h"
#include "gen-cpp/EACP_constants.h"
#include <boost/property_tree/ptree.hpp>
#include <boost/property_tree/ini_parser.hpp>

#define APP_DEFAULT_FILE               _T("/sysvol/conf/service_conf/app_default.conf")
#define EACP_ENV_HOST _T("EACP_ENV_HOST")
#define EACP_HTTPSERVER_NAME _T("eacphttpserver")

ncEACHttpServer::ncEACHttpServer(int32_t maxConcurrency)
    : _server (NULL)
    , _mutex ()
    , _maxConcurrency (maxConcurrency)
    , _acsToken (NULL)
    , _acsShareMgnt (NULL)
    , _acsPermManager (NULL)
    , _acsDeviceManager (NULL)
    , _acsConfManager (NULL)
    , _acsMessageManager (NULL)
    , _acsPolicyManager (NULL)
    , _policyEngine (NULL)
    , _acsOutboxManager (NULL)
    , _authHandler (NULL)
    , _auth1Handler (NULL)
    , _auth2Handler (NULL)
    , _depHandler (NULL)
    , _userHandler (NULL)
    , _contactorHandler (NULL)
    , _CAHandler (NULL)
    , _pkiHandler (NULL)
    , _deviceHandler(NULL)
    , _messageHandler(NULL)
    , _configHandler(NULL)
    ,_thirdHandler(NULL)
    ,_httpPort(0)
    ,_brpcServerPort(0)
{}

ncEACHttpServer::~ncEACHttpServer()
{
    if (_server) {
        delete _server;
        _server = NULL;
    }

    if (_authHandler) {
        delete _authHandler;
        _authHandler = NULL;
    }

    if (_auth1Handler) {
        delete _auth1Handler;
        _auth1Handler = NULL;
    }

    if (_auth2Handler) {
        delete _auth2Handler;
        _auth2Handler = NULL;
    }


    if (_depHandler) {
        delete _depHandler;
        _depHandler = NULL;
    }

    if (_userHandler) {
        delete _userHandler;
        _userHandler = NULL;
    }

    if (_contactorHandler) {
        delete _contactorHandler;
        _contactorHandler = NULL;
    }

    if (_CAHandler) {
        delete _CAHandler;
        _CAHandler = NULL;
    }

    if (_pkiHandler) {
        delete _pkiHandler;
        _pkiHandler = NULL;
    }

    if (_deviceHandler) {
        delete _deviceHandler;
        _deviceHandler = NULL;
    }

    if (_messageHandler) {
        delete _messageHandler;
        _messageHandler = NULL;
    }

    if (_configHandler) {
        delete _configHandler;
        _configHandler = NULL;
    }
    if (_thirdHandler) {
        delete _thirdHandler;
        _thirdHandler = NULL;
    }
}

void ncEACHttpServer::Start ()
{
    _server = new brpc::Server ();
    boost::property_tree::ptree pt;
    boost::property_tree::ini_parser::read_ini (APP_DEFAULT_FILE, pt);
    _httpPort = pt.get<int> ("ShareServer.public_port");
    _brpcServerPort = pt.get<int> ("ShareServer.brpcserver_port");
    _eacpThriftInnerPort = pt.get<int> ("ShareServer.eacp_port");
    brpc::ServerOptions options;
    options.internal_port = _brpcServerPort;
    options.idle_timeout_sec = 60;
    options.max_concurrency = _maxConcurrency;
    options.has_builtin_services = pt.get<bool> ("ShareServer.has_builtin_services", false);
    options.server_info_name = EACP_HTTPSERVER_NAME;
    brpc::fLS::FLAGS_rpcz_database_dir = BRPC_ANALYSE_DATA_PATH;
    brpc::fLI::FLAGS_http_verbose_max_body_length = 5120;
    logging::fLI::FLAGS_minloglevel = 3;

    // brpc内置服务默认关闭
    string ready = "";
    string alive = "";
    if (!options.has_builtin_services) {
        // 只在内置服务关闭时才增加健康检查
        ready = "/health/ready                             =>      Health,";
        alive = "/health/alive                             =>      Health,";
    }

    if (_server->AddService (this, brpc::SERVER_DOESNT_OWN_SERVICE,
        // 健康检查
        ready+
        alive+

        // Ping
        "/api/eacp/v1/ping                                 =>      Ping,"

        // 身份认证 V1
        "/api/eacp/v1/auth1/getconfig                      =>      Auth1_GetConfig,"
        "/api/eacp/v1/auth1/getbyntlmv1                    =>      Auth1_GetByNTLMV1,"
        "/api/eacp/v1/auth1/getbyticket                    =>      Auth1_GetByTicket,"
        "/api/eacp/v1/auth1/getbyadsession                 =>      Auth1_GetByADSession,"
        "/api/eacp/v1/auth1/modifypassword                 =>      Auth1_ModifyPassword,"
        "/api/eacp/v1/auth1/validatesecuritydevice         =>      Auth1_ValidateSecurityDevice,"
        "/api/eacp/v1/auth1/checkuninstallpwd              =>      Auth1_CheckUninstallPwd,"
        "/api/eacp/v1/auth1/checkexitpwd                   =>      Auth1_CheckExitPwd,"
        "/api/eacp/v1/auth1/getvcode                       =>      Auth1_GetVcode,"
        "/api/eacp/v1/auth1/sendsms                        =>      Auth1_SendSms,"
        "/api/eacp/v1/auth1/sendvcode                      =>      Auth1_SendVcode,"
        "/api/eacp/v1/auth1/pwd-retrieval-vcode           =>       Auth1_SendPwdRetrievalVCode,"
        "/api/eacp/v1/auth1/smsactivate                    =>      Auth1_SmsActivate,"
        "/api/eacp/v1/auth1/servertime                     =>      Auth1_ServerTime,"
        "/api/eacp/v1/auth1/sendauthvcode                  =>      Auth1_SendAuthVcode,"

        // 身份认证 V2
        "/api/eacp/v1/auth2/getconfig                      =>      Auth2_GetConfig,"
        "/api/eacp/v1/auth2/login                          =>      Auth2_Login,"
        "/api/eacp/v1/auth2/modifypassword                 =>      Auth2_ModifyPassword,"
        "/api/eacp/v1/auth2/validatesecuritydevice         =>      Auth2_ValidateSecurityDevice,"
        "/api/eacp/v1/auth2/checkuninstallpwd              =>      Auth2_CheckUninstallPwd,"
        "/api/eacp/v1/auth2/checkexitpwd                   =>      Auth2_CheckExitPwd,"

        // 配置获取
        "/api/eacp/v1/auth1/configs                        =>      Auth1_Configs,"
        "/api/eacp/v1/auth1/login-configs                  =>      Auth1_LoginConfigs,"

        // 部门管理
        "/api/eacp/v1/department/getbasicinfo              =>      Department_GetBasicInfo,"
        "/api/eacp/v1/department/getroots                  =>      Department_GetRoots,"
        "/api/eacp/v1/department/getsubdeps                =>      Department_GetSubDeps,"
        "/api/eacp/v1/department/getsubusers               =>      Department_GetSubUsers,"
        "/api/eacp/v1/department/search                    =>      Department_Search,"
        "/api/eacp/v1/department/searchcount               =>      Department_SearchCount,"

        // 用户管理
        "/api/eacp/v1/user/get                             =>      User_Get,"
        "/api/eacp/v1/user/getbasicinfo                    =>      User_GetBasicInfo,"
        "/api/eacp/v1/user/agreedtotermsofuse              =>      User_AgreedToTermsOfUse,"
        "/api/eacp/v1/user/edit                            =>      User_Edit,"

        // 联系人管理
        "/api/eacp/v1/contactor/get                        =>      Contactor_GetContactors,"
        "/api/eacp/v1/contactor/search                     =>      Contactor_Search,"
        "/api/eacp/v1/contactor/searchcount                =>      Contactor_SearchCount,"
        "/api/eacp/v1/contactor/addgroup                   =>      Contactor_AddGroup,"
        "/api/eacp/v1/contactor/editgroup                  =>      Contactor_EditGroup,"
        "/api/eacp/v1/contactor/getgroups                  =>      Contactor_GetGroup,"
        "/api/eacp/v1/contactor/addpersons                 =>      Contactor_AddPersons,"
        "/api/eacp/v1/contactor/searchpersons              =>      Contactor_SearchPersons,"
        "/api/eacp/v1/contactor/deletepersons              =>      Contactor_DeletePersons,"
        "/api/eacp/v1/contactor/getpersons                 =>      Contactor_GetPersons,"

        // CA认证
        "/api/eacp/v1/ca/get                               =>      Ca_Get,"

        // PKI认证
        "/api/eacp/v1/pki/original                         =>      Pki_onOriginal,"
        "/api/eacp/v1/pki/authen                           =>      Pki_onAuthen,"

        // 登录设备管理
        "/api/eacp/v1/device/list                          =>      Device_onList,"
        "/api/eacp/v1/device/disable                       =>      Device_onDisable,"
        "/api/eacp/v1/device/enable                        =>      Device_onEnable,"
        "/api/eacp/v1/device/erase                         =>      Device_onErase,"
        "/api/eacp/v1/device/getstatus                     =>      Device_onGetStatus,"
        "/api/eacp/v1/device/onerasesuc                    =>      Device_onEraseSuc,"

        // 消息通知
        "/api/eacp/v1/message/get                          =>      Message_Get,"
        "/api/eacp/v1/message/read                         =>      Message_Read,"
        "/api/eacp/v1/message/read2                        =>      Message_Read2,"
        "/api/eacp/v1/message/sendmail                     =>      Message_SendMail,"

        // 配置管理
        "/api/eacp/v1/config/get                           =>      Config_Get,"
        "/api/eacp/v1/config/getoemconfigbysection         =>      Config_GetOEMConfigBySection,"
        "/api/eacp/v1/config/getdocwatermarkconfig         =>      Config_GetDocWatermarkConfig,"
        "/api/eacp/v1/config/getfilecrawlconfig            =>      Config_GetFileCrawlConfig,"
        "/api/eacp/v1/config/setquickstartstatus           =>      Config_SetQuickStartStatus,"

        // 第三方接入接口 用户管理
        "/api/eacp/v1/organization/createuser                     =>      Third_CreateUser,"
        "/api/eacp/v1/organization/edituser                       =>      Third_EditUser,"
        "/api/eacp/v1/organization/deleteuser                     =>      Third_DeleteUser,"
        "/api/eacp/v1/organization/getuserbyid                    =>      Third_GetUserById,"
        "/api/eacp/v1/organization/getuserbythirdid               =>      Third_GetUserByThirdId,"
        "/api/eacp/v1/organization/getuserbyname                  =>      Third_GetUserByName,"
        "/api/eacp/v1/organization/getalluser                     =>      Third_GetAllUser,"
        "/api/eacp/v1/organization/getallusercount                =>      Third_GetAllUserCount,"

        // 第三方接入接口 组织管理
        "/api/eacp/v1/organization/createorg                      =>      Third_CreateOrg,"
        "/api/eacp/v1/organization/editorg                        =>      Third_EditOrg,"
        "/api/eacp/v1/organization/deleteorg                      =>      Third_DeleteOrg,"
        "/api/eacp/v1/organization/getallorg                      =>      Third_GetAllOrg,"
        "/api/eacp/v1/organization/getorgbyid                     =>      Third_GetOrgById,"
        "/api/eacp/v1/organization/getorgbyname                   =>      Third_GetOrgByName,"
        "/api/eacp/v1/organization/getsubdepsbyorgid              =>      Third_GetSubDepByOrgId,"
        "/api/eacp/v1/organization/getsubusersbyorgid             =>      Third_GetSubUserByOrgId,"

        // 第三方接入接口 部门管理
        "/api/eacp/v1/organization/createdep                      =>      Third_CreateDep,"
        "/api/eacp/v1/organization/editdep                        =>      Third_EditDep,"
        "/api/eacp/v1/organization/deletedep                      =>      Third_DeleteDep,"
        "/api/eacp/v1/organization/getdepbyid                     =>      Third_GetDepById,"
        "/api/eacp/v1/organization/getdepbythirdid                =>      Third_GetDepByThirdId,"
        "/api/eacp/v1/organization/getdepbyname                   =>      Third_GetDepByName,"
        "/api/eacp/v1/organization/movedep                        =>      Third_MoveDep,"
        "/api/eacp/v1/organization/adduserstodep                  =>      Third_AddUsersToDep,"
        "/api/eacp/v1/organization/moveuserstodep                 =>      Third_MoveUsersToDep,"
        "/api/eacp/v1/organization/removeusersfromdep             =>      Third_RemoveUsersFromDep,"
        "/api/eacp/v1/organization/getsubdepsbydepid              =>      Third_GetSubDepsByDepId,"
        "/api/eacp/v1/organization/getsubusersbydepid             =>      Third_GetSubUsersByDepId,"
        "/api/eacp/v1/organization/setmanager                     =>      Third_SetManager,"
        "/api/eacp/v1/organization/cancelmanager                  =>      Third_CancelManager,") != 0) {

        THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR,
                 _T("Failed to add services."));
    }

    nsresult result;
    AutoLock<ThreadMutexLock> lock (&_mutex);
    if (_acsToken == 0 ) {
        _acsToken = do_CreateInstance (NC_ACS_TOKEN_MANAGER_CONTRACTID, &result);
        if (NS_FAILED (result))
            THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_INIT_ACS_TOKEN_MANAGER_ERR,
                     LOAD_STRING (_T("IDS_EACHTTP_ACSTOKEN_INIT_ERROR")), result);
    }

    if (_acsShareMgnt == 0) {
        _acsShareMgnt = do_CreateInstance (NC_ACS_SHARE_MGNT_CONTRACTID, &result);
        if (NS_FAILED (result)) {
            THROW_E(EAC_HTTP_SERVER, EACHTTP_SERVER_INIT_ACS_SHAREMGNT_ERR,
                LOAD_STRING (_T("IDS_EACHTTP_ACS_SHAREMGNT_INIT_ERROR")), result);
        }
    }

    if (_acsPermManager == 0) {
        _acsPermManager = do_CreateInstance (NC_ACS_PERM_MANAGER_CONTRACTID, &result);
        if (NS_FAILED (result)) {
            THROW_E(EAC_HTTP_SERVER, EACHTTP_SERVER_INIT_ACS_PERM_MANAGER_ERR,
                LOAD_STRING (_T("IDS_EACHTTP_ACS_PERM_INIT_ERROR")), result);
        }
    }

    if (_acsDeviceManager == 0) {
        _acsDeviceManager = do_CreateInstance (NC_ACS_DEVICE_MANAGER_CONTRACTID, &result);
        if (NS_FAILED (result)) {
            THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_INIT_ACS_DEVICE_MANAGER_ERR,
                _T("Failed to create acs device manager: 0x%x"), result);
        }
    }

    if (_acsConfManager == 0) {
        _acsConfManager = do_CreateInstance (NC_ACS_CONF_MANAGER_CONTRACTID, &result);
        if (NS_FAILED (result)) {
            THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_INIT_ACS_CONF_MANAGER_ERR,
                _T("Failed to create acs conf manager: 0x%x"), result);
        }
    }

    if (_acsMessageManager == 0) {
        _acsMessageManager = do_CreateInstance (NC_ACS_MESSAGE_MANAGER_CONTRACTID, &result);
        if (NS_FAILED (result)) {
            THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_INIT_ACS_MESSAGE_MANAGER_ERR,
                _T("Failed to create acs message manager: 0x%x"), result);
        }

        //
        // 启动消息处理线程。
        //
        _acsMessageManager->StartMessageThread2 ();
    }

    if (_acsPolicyManager == 0) {
        _acsPolicyManager = do_CreateInstance (NC_ACS_POLICY_MANAGER_CONTRACTID, &result);
        if (NS_FAILED (result)) {
            THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_INIT_ACS_POLICY_MANAGER_ERR,
                _T("Failed to create acs policy manager: 0x%x"), result);
        }
    }

    if (_policyEngine == 0) {
        _policyEngine = do_CreateInstance (POLICY_ENGINE_CONTRACTID, &result);
        if (NS_FAILED (result)) {
            THROW_E (EAC_HTTP_SERVER, EACHTTP_XPCOM_CREATE_INSTANCE_FAILED,
                _T("Failed to create acs policy engine: 0x%x"), result);
        }
    }

    if (_acsOutboxManager == 0) {
        _acsOutboxManager = do_CreateInstance (NC_ACS_OUTBOX_MANAGER_CONTRACTID, &result);
        if (NS_FAILED (result)) {
            THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_INIT_ACS_ENTRY_DOC_IOC_ERR,
                _T("Failed to create acs outbox manager: 0x%x"), result);
        }
        //
        // 启动 outbox 推送线程并通知线程进行消息推送。
        //
        _acsOutboxManager->StartPushOutboxThread ();
        _acsOutboxManager->NotifyPushOutboxThread ();
    }

    _authHandler = new ncEACAuthHandler(_acsToken, _acsShareMgnt, _acsDeviceManager, _acsConfManager, _acsMessageManager, _acsPolicyManager, _policyEngine);
    _auth1Handler = new ncEACAuthHandler(_acsToken, _acsShareMgnt, _acsDeviceManager, _acsConfManager, _acsMessageManager, _acsPolicyManager, _policyEngine);
    _auth2Handler = new ncEACAuthHandler(_acsToken, _acsShareMgnt, _acsDeviceManager, _acsConfManager, _acsMessageManager, _acsPolicyManager, _policyEngine);
    _depHandler = new ncEACDepartmentHandler (_acsShareMgnt);
    _userHandler = new ncEACUserHandler (_acsShareMgnt);
    _contactorHandler = new ncEACContactorHandler (_acsShareMgnt);
    _CAHandler = new ncEACCAuthHandler ();
    _pkiHandler = new ncEACPKIHandler (_acsShareMgnt);
    _deviceHandler = new ncEACDeviceHandler (_acsDeviceManager);
    _messageHandler = new ncEACMessageHandler (_acsMessageManager, _acsPermManager, _acsShareMgnt);
    _configHandler = new ncEACConfigHandler (_acsShareMgnt);
    _thirdHandler = new ncEACThirdHandler ();
    //
    // 加入token过期机制以后：
    // /api/eacp/v1/auth 协议兼容旧应用默认产生 token 有效期设置为 3个月
    // /api/eacp/v1/auth1 协议默认产生 token 有效期设置为 1小时
    //

    // 单位秒
    const int64 TOKEN_SHORT_EXPIRES = 3600;         // 1小时
    const int64 TOKEN_LONG_EXPIRES = 7884000;       // 3个月

    _authHandler->setExpires (TOKEN_LONG_EXPIRES);
    _auth1Handler->setExpires (TOKEN_SHORT_EXPIRES);
    _auth2Handler->setExpires (TOKEN_SHORT_EXPIRES);

    string envHost = toSTLString (Environment::getEnvVariable (EACP_ENV_HOST));
    if (envHost.find(':') == string::npos) {
        if (_server->Start (_httpPort, &options) != 0) {
            THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR,
                    _T("Failed to start acs http service."));
        }
    }else {
        if (_server->Start (("[::0]:" + String::toString(_httpPort)).getCStr (), &options) != 0) {
            THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR,
                    (_T("Failed to start acs http service.")));
        }
    }
}

void
ncEACHttpServer::OnHealthRequest (brpc::Controller* cntl)
{
    NC_EAC_HTTP_SERVER_TRY
        // 检测EACP Thrift Sever是否正常
        try {
            ncThriftClient<ncTEACPClient> eacpClient ("localhost", _eacpThriftInnerPort, 3000, 3000, 3000);
            eacpClient->EACP_ThriftServerPing();
        }
        catch (ncTException & e) {
            SystemLog::getInstance ()->log (__FILE__, __LINE__, ERROR_LOG_TYPE,
                                        _T("Ping thrift server error: %s"), e.expMsg.c_str ());
            THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.expMsg.c_str ());
        }
        catch (TException & e) {
            SystemLog::getInstance ()->log (__FILE__, __LINE__, ERROR_LOG_TYPE,
                                        _T("Ping thrift server error: %s"), e.what ());
            THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, e.what ());
        }
        catch (...) {
            SystemLog::getInstance ()->log (__FILE__, __LINE__, ERROR_LOG_TYPE,
                                        _T("Ping thrift server error: Unkown Excepiton."));
            THROW_E (EAC_HTTP_SERVER, EACHTTP_SERVER_UNKNOWN_ERR, _T("Eacp Thrift API Server Error"));
        }

        cntl->http_response().set_content_type("application/json; charset=UTF-8");
        ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", "ok");

    NC_EAC_HTTP_SERVER_CATCH (cntl)
}

void
ncEACHttpServer::OnPingRequest (brpc::Controller* cntl)
{
    NC_EAC_HTTP_SERVER_TRY

        cntl->http_response().set_content_type("application/json; charset=UTF-8");
        ncHttpSendReply (cntl, brpc::HTTP_STATUS_OK, "ok", "ok");

    NC_EAC_HTTP_SERVER_CATCH (cntl)
}

void
ncEACHttpServer::OnAuth1Request (brpc::Controller* cntl)
{
    NC_EAC_HTTP_SERVER_TRY

        cntl->http_response().set_content_type("application/json; charset=UTF-8");
        _auth1Handler->doAuthRequestHandler(cntl);

    NC_EAC_HTTP_SERVER_CATCH (cntl)
}

void
ncEACHttpServer::OnAuth2Request (brpc::Controller* cntl)
{
    NC_EAC_HTTP_SERVER_TRY

        cntl->http_response().set_content_type("application/json; charset=UTF-8");
        _auth2Handler->doAuth2RequestHandler(cntl);

    NC_EAC_HTTP_SERVER_CATCH (cntl)
}

void
ncEACHttpServer::OnDepartmentRequest (brpc::Controller* cntl)
{
    NC_EAC_HTTP_SERVER_TRY

        cntl->http_response().set_content_type("application/json; charset=UTF-8");
        _depHandler->doDepRequestHandler(cntl);

    NC_EAC_HTTP_SERVER_CATCH (cntl)
}

void
ncEACHttpServer::OnUserRequest (brpc::Controller* cntl)
{
    NC_EAC_HTTP_SERVER_TRY

        cntl->http_response().set_content_type("application/json; charset=UTF-8");
        _userHandler->doUserRequestHandler(cntl);

    NC_EAC_HTTP_SERVER_CATCH (cntl)
}

void
ncEACHttpServer::OnContactorRequest (brpc::Controller* cntl)
{
    NC_EAC_HTTP_SERVER_TRY

        cntl->http_response().set_content_type("application/json; charset=UTF-8");
        _contactorHandler->doContactorRequestHandler(cntl);

    NC_EAC_HTTP_SERVER_CATCH (cntl)
}

void
ncEACHttpServer::OnCARequest (brpc::Controller* cntl)
{
    NC_EAC_HTTP_SERVER_TRY

        cntl->http_response().set_content_type("application/json; charset=UTF-8");
        _CAHandler->doCARequestHandler(cntl);

    NC_EAC_HTTP_SERVER_CATCH (cntl)
}

void
ncEACHttpServer::OnDeviceRequest (brpc::Controller* cntl)
{
    NC_EAC_HTTP_SERVER_TRY

        cntl->http_response().set_content_type("application/json; charset=UTF-8");
        _deviceHandler->doDeviceRequestHandler(cntl);

    NC_EAC_HTTP_SERVER_CATCH (cntl)
}

void
ncEACHttpServer::OnPKIRequest (brpc::Controller* cntl)
{
    NC_EAC_HTTP_SERVER_TRY

        cntl->http_response().set_content_type("application/json; charset=UTF-8");
        _pkiHandler->doPKIRequestHandler(cntl);

    NC_EAC_HTTP_SERVER_CATCH (cntl)
}

void
ncEACHttpServer::OnMessageRequest (brpc::Controller* cntl)
{
    NC_EAC_HTTP_SERVER_TRY

        cntl->http_response().set_content_type("application/json; charset=UTF-8");
        _messageHandler->doMessageRequestHandler(cntl);

    NC_EAC_HTTP_SERVER_CATCH (cntl)
}

void
ncEACHttpServer::OnConfigRequest (brpc::Controller* cntl)
{
    NC_EAC_HTTP_SERVER_TRY

        cntl->http_response().set_content_type("application/json; charset=UTF-8");
        _configHandler->doConfigRequestHandler(cntl);

    NC_EAC_HTTP_SERVER_CATCH (cntl)
}

void
ncEACHttpServer::OnThirdRequest (brpc::Controller* cntl)
{
    NC_EAC_HTTP_SERVER_TRY

        cntl->http_response().set_content_type("application/json; charset=UTF-8");
        _thirdHandler->doThirdRequestHandler(cntl);

    NC_EAC_HTTP_SERVER_CATCH (cntl)
}
