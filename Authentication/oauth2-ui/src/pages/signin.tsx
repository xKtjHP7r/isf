import {
  withI18n,
  useI18n,
  GetI18nServerSideProps,
  getRequestLanguage,
  supportLanguages,
} from "../i18n";
import React, { useEffect, FunctionComponent, useRef } from "react";
import { useCookie } from "react-use";
import { useLocalStore, useObserver } from "mobx-react-lite";
import "mobx-react-lite/batchingForReactDom";
import classNames from "classnames";
import querystring from "query-string";
import { pki } from "node-forge";
import { Request, Response } from "express";
import axios from "axios";
import openapi, { Auth1SendauthvcodeReqDeviceinfo } from "../http/index";
import {
  eacpPublicApi,
  eacpPrivateApi,
  authenticationPrivateApi,
  getServiceNameFromApi,
  getErrorCodeFromService,
  isThirdRememberLogin,
  deployPublicApi,
} from "../core/config";
import { getLoginRequest, acceptLoginRequest } from "../services/hydra";
import { ErrorCode, getErrorMessage } from "../core/errorcode";
import Button from "antd/lib/button";
import Input from "antd/lib/input";
import Form from "antd/lib/form";
import { Password } from "../controls";
import Checkbox from "antd/lib/checkbox";
import { AuthBySMS, AppHead, ThirdAuthFirst } from "../components";
import AccountIcon from "@icons/account.svg";
import PasswordIcon from "@icons/password.svg";
import AvatarIcon from "@icons/avatar.svg";
import DynamicPasswordIcon from "@icons/dynamic-password.svg";
import ErrorPage from "./_error";
import {
  SignoutLogin3rdPartyStatus,
  SignoutLogin3rdPartyStatusKey,
} from "../const";
import { getApplicationbyslug } from "../api/getApplicationbyslug";
import { getServerPrefix, getUrlPrefix } from "../common/getUrlPrefix";
import {
  getIsElectronOpenExternal,
  getIsMobile,
} from "../common/getClientType";

interface ISigninProps {
  /**
   * 认证唯一标识
   */
  challenge?: string;

  /**
   * csrf token令牌
   */
  csrftoken: string;

  /**
   * 登录设备信息
   */
  device?: Auth1SendauthvcodeReqDeviceinfo;

  /**
   * ip
   */
  ip?: string;

  /**
   * 认证配置
   */
  authconfig?: { [key: string]: any };

  /**
   * oem配置
   */
  oemconfig?: { [key: string]: any };

  /**
   * 错误信息
   */
  error?: { [key: string]: any } | null;

  /**
   * 是否是IE浏览器
   */
  isIE?: boolean;

  userAgent?: string | undefined;

  hiddenPage?: boolean;
  remember_visible?: boolean;
  isThirdRememberLogin?: boolean;
  urlPrefix?: string;
}

interface ISigninState {
  /**
   * 选择语言
   */
  languageUser: string;

  /**
   * 登录状态
   */
  loginStatus: LoginStatus;

  /**
   * 账号
   */
  account: string;

  /**
   * 密码
   */
  password: string;

  /**
   * 图形验证码信息
   */
  captchaInfo: CaptchaInfo;

  /**
   * 图形验证码
   */
  captcha: string;

  /**
   * 动态密码
   */
  dynamicPassword: string;

  /**
   * 是否记住登录状态
   */
  rememberLogin: boolean;

  /**
   * 认证方式
   */
  authMethod: AuthMethod;

  /**
   * 是否需要短信验证码
   */
  needAuthBySMS: boolean;

  /**
   * 短信验证码绑定手机号
   */
  cellphone: string;

  /**
   * 发送短信验证码时间间隔
   */
  timeInterval: number;

  /**
   * 是否重复发送短信验证码
   */
  isDuplicateSend: boolean;

  /**
   * 错误状态码
   */
  errorStatus: number;

  /**
   * 错误信息
   */
  errorInfo: any;

  /**
   * 校验输入
   */
  checkInput: () => boolean;

  /**
   * 获取图形验证码信息
   */
  getCaptchaInfo: () => void;

  /**
   * 认证
   */
  auth: () => void;

  /**
   * 第三方优先显示的情况下，切换账号密码登录
   */
  displayAccountLoginUnderTheThirdAuthFirst: boolean;

  [key: string]: any;
}

/**
 * 登录操作
 */
export enum LoginStatus {
  /**
   * 登录
   */
  READY,

  /**
   * 登录中
   */
  LOADING,
}

/**
 * 登录方式
 */
export enum AuthMethod {
  /**
   * 账号密码
   */
  Account,

  /**
   * 图形密码
   */
  AccountAndImgCaptcha,

  /**
   * 短信验证码
   */
  AccountAndSmsCaptcha,

  /**
   * 动态密码
   */
  AccountAndDynamicPassword,
}

/**
 * 验证码信息
 */
export interface CaptchaInfo {
  /**
   * 验证码标识
   */
  uuid: string;

  /**
   * 验证码图片
   */
  vcode: string;
}

enum ClientTypeEnum {
  unknown = "unknown",
  ios = "ios",
  android = "android",
  windows_phone = "windows_phone",
  windows = "windows",
  mac_os = "mac_os",
  web = "web",
  mobile_web = "mobile_web",
  console_web = "console_web",
  deploy_web = "deploy_web",
  nas = "nas",
  linux = "linux",
}

const PublicKey: any = pki.publicKeyFromPem(`-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDB2fhLla9rMx+6LWTXajnK11Kd
p520s1Q+TfPfIXI/7G9+L2YC4RA3M5rgRi32s5+UFQ/CVqUFqMqVuzaZ4lw/uEdk
1qHcP0g6LB3E9wkl2FclFR0M+/HrWmxPoON+0y/tFQxxfNgsUodFzbdh0XY1rIVU
IbPLvufUBbLKXHDPpwIDAQAB
-----END PUBLIC KEY-----`);

const PublicKeyPlus: any = pki.publicKeyFromPem(`-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsyOstgbYuubBi2PUqeVj
GKlkwVUY6w1Y8d4k116dI2SkZI8fxcjHALv77kItO4jYLVplk9gO4HAtsisnNE2o
wlYIqdmyEPMwupaeFFFcg751oiTXJiYbtX7ABzU5KQYPjRSEjMq6i5qu/mL67XTk
hvKwrC83zme66qaKApmKupDODPb0RRkutK/zHfd1zL7sciBQ6psnNadh8pE24w8O
2XVy1v2bgSNkGHABgncR7seyIg81JQ3c/Axxd6GsTztjLnlvGAlmT1TphE84mi99
fUaGD2A1u1qdIuNc+XuisFeNcUW6fct0+x97eS2eEGRr/7qxWmO/P20sFVzXc2bF
1QIDAQAB
-----END PUBLIC KEY-----
`);

const OAuthIsSkipLabel = "oauth2.isSkip";
const isSkipMaxAge = 60 * 60 * 1000 * 24 * 30;
const clearOemConfigTimer = 15 * 60 * 1000;
const AccessThirdAuthOnlys: (keyof typeof ClientTypeEnum)[] = [
  ClientTypeEnum.web,
  ClientTypeEnum.mac_os,
  ClientTypeEnum.windows,
  ClientTypeEnum.linux,
  ClientTypeEnum.android,
  ClientTypeEnum.ios,
  ClientTypeEnum.mobile_web,
];

interface OEMConfigType {
  [key: string]: {
    [key: string]: {
      productOemConfig?: {
        isTransparentBoxStyle: boolean;
        theme: string;
        showBanner: boolean;
        showAgreement: boolean;
        showPolicy: boolean;
        favicon: string;
        webTemplate: string;
        desktopTemplate: string;
        loginBoxStyle: string;
        hideLogo: boolean;
      };
      config?: {
        logo: string;
        darklogo: string;
        product: string;
        portalBanner: string;
      };
    };
  };
}

let oemConfig: OEMConfigType = {};

setInterval(() => {
  //每15分钟清理一次缓存值
  oemConfig = {};
}, clearOemConfigTimer);

export const SignInForm: FunctionComponent<Omit<ISigninProps, "error">> = ({
  challenge,
  csrftoken,
  device,
  ip,
  authconfig,
  oemconfig,
  isIE,
  userAgent,
  hiddenPage,
  remember_visible,
  isThirdRememberLogin,
}) => {
  const { t, lang } = useI18n();
  const inputEl = useRef<Password>(null);
  const [loginAccount, setLoginAccount] = useCookie(
    `${device?.client_type}.login_account`
  );
  const clientType = ["console_web", "deploy_web", "web", "windows", "mac_os"];
  const isShowAccount = clientType.some((item) => item === device?.client_type);
  const setIsSkip = (rememberLogin: string) => {
    document.cookie = `${OAuthIsSkipLabel}=${rememberLogin};max-age=${isSkipMaxAge};path=/`;
  };

  const urlPrefix = getUrlPrefix();
  const isWebMobile = getIsMobile(userAgent);
  const isElectronOpenExternal =
    device?.client_type === "windows" ||
    device?.client_type === "mac_os" ||
    device?.client_type === "linux";

  const store = useLocalStore<ISigninState>(() => {
    // 语言选择
    let languageUser: "CN" | "EN" | "Hant";
    if (lang === "zh-tw" || lang === "zh-hk") {
      languageUser = "Hant";
    } else if (lang.startsWith("zh")) {
      languageUser = "CN";
    } else {
      languageUser = "EN";
    }

    // 登录方式
    let authMethod = AuthMethod.Account;

    const {
      vcode_login_config: vcodeLogin,
      dualfactor_auth_server_status: dualfactor,
      sms_login_visible,
    } = authconfig!;

    if (vcodeLogin && vcodeLogin.isenable) {
      if (vcodeLogin.passwderrcnt === 0) {
        authMethod = AuthMethod.AccountAndImgCaptcha;
      } else {
        authMethod = AuthMethod.Account;
      }
    } else if (vcodeLogin && !vcodeLogin.isenable) {
      if (sms_login_visible && dualfactor && dualfactor.auth_by_sms) {
        authMethod = AuthMethod.AccountAndSmsCaptcha;
      } else if (sms_login_visible && dualfactor && dualfactor.auth_by_OTP) {
        authMethod = AuthMethod.AccountAndDynamicPassword;
      } else {
        authMethod = AuthMethod.Account;
      }
    }

    return {
      loginStatus: LoginStatus.READY,
      account: !remember_visible && isShowAccount ? "" : loginAccount || "",
      password: "",
      captchaInfo: {
        uuid: "",
        vcode: "",
      },
      captcha: "",
      dynamicPassword: "",
      rememberLogin: false,
      authMethod: authMethod,
      languageUser: languageUser,
      needAuthBySMS: false,
      cellphone: "",
      timeInterval: 60,
      isDuplicateSend: false,
      errorStatus: ErrorCode.Normal,
      errorInfo: null,
      displayAccountLoginUnderTheThirdAuthFirst: false,
      get getError(): string {
        switch (store.errorStatus) {
          case ErrorCode.PasswordInvalidLocked:
            return getErrorMessage(store.errorStatus, t, {
              time: store.errorInfo.detail.remainlockTime,
            });
          default:
            return getErrorMessage(store.errorStatus, t);
        }
      },
      checkInput(): boolean {
        switch (true) {
          case !store.account:
            store.errorStatus = ErrorCode.NoAccount;
            return false;
          case !store.password:
            store.errorStatus = ErrorCode.NoPassword;
            return false;
          case !store.captcha &&
            store.authMethod === AuthMethod.AccountAndImgCaptcha:
            store.errorStatus = ErrorCode.NoCaptcha;
            return false;
          case !store.dynamicPassword &&
            store.authMethod === AuthMethod.AccountAndDynamicPassword:
            store.errorStatus = ErrorCode.NoDynamicPassword;
            return false;
          case store.password.length > 100:
            store.errorStatus = ErrorCode.AuthFailed;
            return false;
          default:
            store.loginStatus = LoginStatus.LOADING;
            return true;
        }
      },
      async getCaptchaInfo() {
        const uuid = store.captchaInfo.uuid;
        const { data } = await openapi.post("/eacp/v1/auth1/getvcode", {
          uuid,
        });
        store.captchaInfo = data;
        store.captcha = "";
      },
      async refreshAuthMethod() {
        try {
          const {
            data: {
              vcode_login_config: vcodeLogin,
              dualfactor_auth_server_status: dualfactor,
            },
          } = await openapi.get("/eacp/v1/auth1/login-configs", {
            headers: {
              "X-Forwarded-For": ip as string,
            },
          });
          const { sms_login_visible } = authconfig!;
          if (vcodeLogin && vcodeLogin.isenable) {
            if (vcodeLogin.passwderrcnt === 0) {
              store.authMethod !== AuthMethod.AccountAndImgCaptcha
                ? (store.authMethod = AuthMethod.AccountAndImgCaptcha)
                : null;
            } else {
              store.authMethod !== AuthMethod.Account
                ? ((store.dynamicPassword = ""),
                  (store.captcha = ""),
                  (store.captchaInfo = {
                    uuid: "",
                    vcode: "",
                  }),
                  (store.needAuthBySMS = false),
                  (store.authMethod = AuthMethod.Account))
                : null;
            }
          } else if (vcodeLogin && !vcodeLogin.isenable) {
            if (sms_login_visible && dualfactor && dualfactor.auth_by_sms) {
              store.authMethod !== AuthMethod.AccountAndSmsCaptcha
                ? ((store.dynamicPassword = ""),
                  (store.captcha = ""),
                  (store.captchaInfo = {
                    uuid: "",
                    vcode: "",
                  }),
                  (store.needAuthBySMS = false),
                  (store.authMethod = AuthMethod.AccountAndSmsCaptcha))
                : null;
            } else if (
              sms_login_visible &&
              dualfactor &&
              dualfactor.auth_by_OTP
            ) {
              store.authMethod !== AuthMethod.AccountAndDynamicPassword
                ? ((store.captcha = ""),
                  (store.captchaInfo = {
                    uuid: "",
                    vcode: "",
                  }),
                  (store.needAuthBySMS = false),
                  (store.authMethod = AuthMethod.AccountAndDynamicPassword))
                : null;
            } else {
              store.authMethod !== AuthMethod.Account
                ? ((store.dynamicPassword = ""),
                  (store.captcha = ""),
                  (store.captchaInfo = {
                    uuid: "",
                    vcode: "",
                  }),
                  (store.needAuthBySMS = false),
                  (store.authMethod = AuthMethod.Account))
                : null;
            }
          }
          store.authMethod === AuthMethod.AccountAndImgCaptcha
            ? ((store.dynamicPassword = ""),
              (store.needAuthBySMS = false),
              await store.getCaptchaInfo())
            : null;
        } catch (err) {
          return Promise.reject(err);
        }
      },
      async auth() {
        if (store.authMethod === AuthMethod.AccountAndSmsCaptcha) {
          // 账号密码 + 短信验证码
          try {
            const {
              data: { authway, isduplicatesended, sendinterval },
            } = await openapi.post("/eacp/v1/auth1/sendauthvcode", {
              account: store.account,
              password: btoa(
                PublicKey.encrypt(store.password, "RSAES-PKCS1-V1_5")
              ),
              oldtelnum: "",
              device: device as Auth1SendauthvcodeReqDeviceinfo,
            });
            store.isDuplicateSend = isduplicatesended;
            store.timeInterval = sendinterval;
            store.cellphone = authway;
            store.needAuthBySMS = true;
          } catch (e: any) {
            if (e.response) {
              const {
                response: { data: err },
              } = e;
              if (err.detail && err.detail.isShowStatus) {
                // 错误密码达指定次数，开启图形验证码登录方式
                store.dynamicPassword = "";
                store.needAuthBySMS = false;
                store.authMethod = AuthMethod.AccountAndImgCaptcha;
                await store.getCaptchaInfo();
              } else {
                // 获取最新登录方式
                await store.refreshAuthMethod();
              }
              store.errorInfo = err;
              store.errorStatus = err.code;
              store.loginStatus = LoginStatus.READY;
            } else {
              store.errorInfo = null;
              store.errorStatus = ErrorCode.NoNetwork;
              store.loginStatus = LoginStatus.READY;
            }
          }
        } else {
          // login服务中调用getNew接口（重定向时，返回json信息，前端重定向）
          try {
            setLoginAccount(store.account);
            const {
              data: { redirect },
            } = await axios.post(
              `${location.protocol}//${location.hostname}:${location.port}${urlPrefix}/oauth2/signin`,
              {
                _csrf: csrftoken,
                challenge,
                account: store.account,
                password: btoa(
                  PublicKeyPlus.encrypt(store.password, "RSAES-PKCS1-V1_5")
                ),
                vcode: { id: store.captchaInfo.uuid, content: store.captcha },
                dualfactorauthinfo: {
                  validcode: { vcode: "" }, // 短信验证码
                  OTP: { OTP: store.dynamicPassword }, // 动态密保
                },
                remember: store.rememberLogin,
                device,
              }
            );
            setIsSkip(store.rememberLogin.toString());
            window.location.replace(redirect);
          } catch (e: any) {
            if (e?.response?.data) {
              const {
                response: { data: err },
              } = e;
              store.errorInfo = err;
              store.errorStatus =
                err.code || err.status_code || e?.response?.status;
              store.loginStatus = LoginStatus.READY;
              if (
                store.authMethod === AuthMethod.AccountAndImgCaptcha &&
                err.detail &&
                err.detail?.isShowStatus
              ) {
                // 图形验证码登录方式，刷新图形验证码
                await store.getCaptchaInfo();
              } else if (
                store.authMethod !== AuthMethod.AccountAndImgCaptcha &&
                err.detail &&
                err.detail?.isShowStatus
              ) {
                // 错误密码达指定次数，开启图形验证码登录方式
                store.dynamicPassword = "";
                store.needAuthBySMS = false;
                store.authMethod = AuthMethod.AccountAndImgCaptcha;
                await store.getCaptchaInfo();
              } else {
                // 获取最新登录方式
                await store.refreshAuthMethod();
              }
            } else {
              store.errorInfo = null;
              store.errorStatus = ErrorCode.NoNetwork;
              store.loginStatus = LoginStatus.READY;
            }
          }
        }
      },
    };
  });

  const openThirdAuth = (authconfig?: { [key: string]: any }) => {
    document.cookie = `is_previous_login_3rd_party=true;path=/${
      location.protocol === "https:" ? ";secure" : ""
    }`;
    setIsSkip(isThirdRememberLogin ? "true" : "false");

    if (!top) return;
    if (isElectronOpenExternal) {
      window.parent.postMessage(
        {
          msg: "OauthUIOpenExternal",
          data: authconfig!.thirdauth.config.authServer,
        },
        "*"
      );
    } else {
      top.window.location.href = authconfig!.thirdauth.config.authServer;
    }
    document.cookie = `login_challenge=${challenge};path=/${
      location.protocol === "https:" ? ";secure" : ""
    }`;
  };

  function onChangePassword(e: React.ChangeEvent<HTMLInputElement>) {
    setPassword(e);
    store.errorStatus = ErrorCode.Normal;
  }

  function setPassword(e: React.ChangeEvent<HTMLInputElement>) {
    store.password = e.target.value;
  }

  useEffect(() => {
    if (store.authMethod === AuthMethod.AccountAndImgCaptcha) {
      store.getCaptchaInfo();
    }
  }, []);

  const autoOpenThirdAuth = () => {
    if (
      device?.client_type !== "console_web" &&
      device?.client_type !== "deploy_web" &&
      authconfig!.thirdauth &&
      authconfig!.thirdauth.config &&
      (authconfig!.sso === "true" ||
        authconfig!.thirdauth.config.autoCasRedirect)
    ) {
      openThirdAuth(authconfig);
    }
  };

  const getMarginTop = () => {
    return oemconfig?.showPortalBanner
      ? oemconfig?.webTemplate === "default"
        ? "8px"
        : "0px"
      : oemconfig?.showUserAgreement || oemconfig?.showPrivacyPolicy
      ? 0
      : "12px";
  };

  const switchLoginMode = (value: boolean) => {
    store.displayAccountLoginUnderTheThirdAuthFirst = value;
  };

  useEffect(() => {
    autoOpenThirdAuth();
    if (inputEl.current) {
      const input = inputEl.current;
      store.account ? input.focus() : null;
    }
  }, []);

  return useObserver(() => {
    return (
      <div
        className="top-signin"
        style={{
          marginTop: getMarginTop(),
          opacity: hiddenPage ? 0 : 1,
        }}
      >
        <div
          className={classNames(
            "oauth2-ui-wrapper",
            isIE ? "oauth2-ui-wrapper-ie" : null
          )}
        >
          <AppHead oemconfig={oemconfig!} />
          {!store.needAuthBySMS ? (
            <div className="signin-wrapper">
              {!oemconfig?.hideLogo && (
                <div className="logo-wrapper">
                  <img
                    className="logo-image"
                    src={`data:image/png;base64,${oemconfig!.logo}`}
                    onDragStart={(e) => {
                      e.preventDefault();
                    }}
                  />
                </div>
              )}
              {(oemconfig?.productId === "anyshare" || oemconfig?.productId === "deploy") ? (
                (device?.client_type === "console_web" &&
                  oemconfig?.productId === "anyshare") ||
                device?.client_type === "deploy_web" ? (
                  <div className="title-wrapper">
                    {t(`title-${device?.client_type}`)}
                  </div>
                ) : oemconfig?.showPortalBanner && oemconfig?.portalBanner ? (
                  <div className="title-wrapper">
                    {t(`title-${device?.client_type}`) ||
                      oemconfig?.portalBanner}
                  </div>
                ) : oemconfig?.showPortalBanner ? (
                  <div className="title-wrapper">
                    {t(`title-${device?.client_type}`) || t(`title-default`)}
                  </div>
                ) : null
              ) : oemconfig?.showPortalBanner && oemconfig?.portalBanner ? (
                <div className="title-wrapper">{oemconfig?.portalBanner}</div>
              ) : null}

              {authconfig!.third_auth_only ? (
                <div className="third-auth-only">
                  <div className="third-auth-only-avatar">
                    <span
                      style={{ fontSize: "100px" }}
                      className="anticon anticon-avatar"
                    >
                      <AvatarIcon />
                    </span>
                  </div>
                  <Button
                    className={classNames(
                      "third-auth-only-button",
                      "as-components-oem-background-color"
                    )}
                    type="primary"
                    onClick={() => {
                      openThirdAuth(authconfig);
                    }}
                  >
                    {authconfig!.thirdauth?.config?.loginButtonText ||
                      t("third")}
                  </Button>
                </div>
              ) : authconfig!.thirdPartyExists &&
                authconfig!.thirdAuthFirst &&
                !store.displayAccountLoginUnderTheThirdAuthFirst ? (
                <ThirdAuthFirst
                  className={
                    !oemconfig?.showPortalBanner
                      ? "signin-not-portalbanner"
                      : undefined
                  }
                  authconfig={authconfig}
                  onAccountLoginClick={switchLoginMode.bind(null, true)}
                  onThirdAuthClick={openThirdAuth.bind(null, authconfig)}
                  device={device}
                />
              ) : (
                <div className="signin-content">
                  <Form>
                    <Form.Item>
                      <Input
                        name="account"
                        className="input-item signin-item"
                        type="text"
                        autoComplete="off"
                        prefix={
                          <span className="icon">
                            <AccountIcon />
                          </span>
                        }
                        placeholder={t("account")}
                        value={store.account}
                        onChange={(e) => {
                          store.account = e.target.value;
                          store.errorStatus = ErrorCode.Normal;
                        }}
                        onKeyDown={(e) => {
                          if (
                            store.loginStatus === LoginStatus.READY &&
                            e.keyCode === 13 &&
                            store.checkInput()
                          ) {
                            store.auth();
                          }
                        }}
                        onDrop={(e) => {
                          e.preventDefault();
                        }}
                      />
                    </Form.Item>
                    <Form.Item>
                      <Password
                        name="password"
                        className="input-item signin-item"
                        type="password"
                        autoComplete="off"
                        ref={inputEl}
                        prefix={
                          <span className="icon">
                            <PasswordIcon />
                          </span>
                        }
                        placeholder={t("password")}
                        value={store.password}
                        onChange={onChangePassword}
                        onBlur={setPassword}
                        onKeyDown={(e) => {
                          if (
                            store.loginStatus === LoginStatus.READY &&
                            e.keyCode === 13 &&
                            store.checkInput()
                          ) {
                            store.auth();
                          }
                        }}
                        onDrop={(e) => {
                          e.preventDefault();
                        }}
                      />
                    </Form.Item>
                    {store.authMethod === AuthMethod.AccountAndImgCaptcha ? (
                      <Form.Item>
                        <Input
                          name="captcha"
                          className="input-item captcha-item"
                          type="text"
                          autoComplete="off"
                          placeholder={t("captcha")}
                          value={store.captcha}
                          onChange={(e) => {
                            store.captcha = e.target.value;
                            store.errorStatus = ErrorCode.Normal;
                          }}
                          onKeyDown={(e) => {
                            if (
                              store.loginStatus === LoginStatus.READY &&
                              e.keyCode === 13 &&
                              store.checkInput()
                            ) {
                              store.auth();
                            }
                          }}
                          onDrop={(e) => {
                            e.preventDefault();
                          }}
                        />
                        <img
                          className="captcha-item-image"
                          src={`data:image/jpeg;base64,${store.captchaInfo.vcode}`}
                          onClick={() => {
                            store.getCaptchaInfo();
                          }}
                        />
                      </Form.Item>
                    ) : null}
                    {store.authMethod ===
                    AuthMethod.AccountAndDynamicPassword ? (
                      <Form.Item>
                        <Password
                          name="dynamicPassword"
                          className="input-item signin-item"
                          type="password"
                          autoComplete="new-password"
                          prefix={
                            <span className="icon">
                              <DynamicPasswordIcon />
                            </span>
                          }
                          placeholder={t("dynamic-password")}
                          value={store.dynamicPassword}
                          onChange={(e) => {
                            store.dynamicPassword = e.target.value;
                            store.errorStatus = ErrorCode.Normal;
                          }}
                          onKeyDown={(e) => {
                            if (
                              store.loginStatus === LoginStatus.READY &&
                              e.keyCode === 13 &&
                              store.checkInput()
                            ) {
                              store.auth();
                            }
                          }}
                          onDrop={(e) => {
                            e.preventDefault();
                          }}
                        />
                      </Form.Item>
                    ) : null}
                    {(authconfig!.rememberpass ||
                      authconfig!.vcode_server_status.send_vcode_by_sms ||
                      authconfig!.vcode_server_status.send_vcode_by_email) &&
                    !authconfig!.enable_secret_mode ? (
                      <Form.Item>
                        {authconfig!.remember_password_visible &&
                        authconfig!.rememberpass &&
                        remember_visible ? (
                          <div className="remember-password">
                            <Checkbox
                              name="remember"
                              onChange={(e) => {
                                store.rememberLogin = e.target.checked;
                              }}
                              checked={store.rememberLogin}
                            >
                              <span className="remember-password-text">
                                {t("remember")}
                              </span>
                            </Checkbox>
                          </div>
                        ) : null}
                        {authconfig!.reset_password_visible &&
                        (authconfig!.vcode_server_status.send_vcode_by_sms ||
                          authconfig!.vcode_server_status
                            .send_vcode_by_email) &&
                        !authconfig!.enable_secret_mode ? (
                          <div className="forget-password">
                            <a
                              onClick={async () => {
                                const {
                                  data: {
                                    vcode_server_status,
                                    enable_secret_mode,
                                  },
                                } = await openapi.get(
                                  "/eacp/v1/auth1/login-configs"
                                );
                                if (
                                  ((vcode_server_status as any)
                                    .send_vcode_by_sms ||
                                    (vcode_server_status as any)
                                      .send_vcode_by_email) &&
                                  !enable_secret_mode
                                ) {
                                  window.location.href = `${urlPrefix}/oauth2/reset?${
                                    oemconfig?.productId
                                      ? `product=${oemconfig?.productId}&`
                                      : ""
                                  }login_challenge=${challenge}&redirect=${
                                    location.protocol
                                  }//${location.hostname}:${
                                    location.port
                                  }${urlPrefix}/oauth2/signin?login_challenge=${challenge}`;
                                } else {
                                  store.errorStatus =
                                    ErrorCode.CloseForgetPasswordResetBySend;
                                }
                              }}
                            >
                              {t("reset")}
                            </a>
                          </div>
                        ) : null}
                      </Form.Item>
                    ) : null}
                    <Form.Item>
                      <Button
                        className={classNames(
                          "oem-button",
                          "as-components-oem-background-color",
                          !(
                            (authconfig!.rememberpass ||
                              authconfig!.vcode_server_status
                                .send_vcode_by_sms ||
                              authconfig!.vcode_server_status
                                .send_vcode_by_email) &&
                            !authconfig!.enable_secret_mode
                          ) && "signin-button"
                        )}
                        type="primary"
                        onClick={() => {
                          if (
                            store.loginStatus === LoginStatus.READY &&
                            store.checkInput()
                          ) {
                            store.auth();
                          }
                        }}
                      >
                        {store.loginStatus === LoginStatus.READY
                          ? t("log-in")
                          : t("logging-in")}
                      </Button>
                    </Form.Item>
                  </Form>
                  {store.errorStatus ===
                  ErrorCode.Normal ? null : isElectronOpenExternal &&
                    store.errorStatus === 404 ? (
                    <div className="error-message-text">
                      {t(`signin-error-404-electron`)}
                      <Button
                        type="link"
                        onClick={() => {
                          if (isElectronOpenExternal) {
                            window.parent.postMessage(
                              {
                                msg: "OauthUIReload",
                              },
                              "*"
                            );
                          }
                        }}
                      >
                        {t("reload")}
                      </Button>
                    </div>
                  ) : (
                    <div className="error-message-text">{store.getError}</div>
                  )}

                  {store.errorStatus === ErrorCode.PasswordFailure ||
                  store.errorStatus === ErrorCode.PasswordInsecure ||
                  store.errorStatus === ErrorCode.PasswordIsInitial ? (
                    <div className="change-password-text">
                      <a
                        onClick={() => {
                          location.href = `${urlPrefix}/oauth2/change?login_challenge=${challenge}&${
                            oemconfig?.productId
                              ? `product=${oemconfig?.productId}&`
                              : ""
                          }redirect=${location.protocol}//${
                            location.hostname
                          }:${
                            location.port
                          }${urlPrefix}/oauth2/signin?login_challenge=${challenge}&account=${encodeURIComponent(
                            store.account
                          )}`;
                        }}
                      >
                        {t("change")}
                      </a>
                    </div>
                  ) : null}
                  {authconfig!.thirdPartyExists ? (
                    <div
                      className={
                        isWebMobile ? "third-signin-mobile" : "third-signin"
                      }
                    >
                      {!authconfig!.thirdAuthFirst ? (
                        device?.client_type === "web" ? (
                          <a
                            onClick={() => {
                              openThirdAuth(authconfig);
                            }}
                          >
                            {(authconfig!.thirdauth &&
                              authconfig!.thirdauth.config &&
                              authconfig!.thirdauth.config.loginButtonText) ||
                            authconfig!.third_auth_only
                              ? authconfig!.thirdauth.config.loginButtonText
                              : t("third")}
                          </a>
                        ) : device?.client_type === "console_web" ? (
                          <a
                            onClick={() => {
                              openThirdAuth(authconfig);
                            }}
                          >
                            {authconfig!.thirdauth &&
                            authconfig!.thirdauth.config &&
                            authconfig!.thirdauth.config.loginButtonText
                              ? authconfig!.thirdauth.config.loginButtonText
                              : t("third")}
                          </a>
                        ) : (
                          <a
                            onClick={() => {
                              openThirdAuth(authconfig);
                            }}
                          >
                            {authconfig!.thirdauth &&
                            authconfig!.thirdauth.config &&
                            authconfig!.thirdauth.config.loginButtonText
                              ? authconfig!.thirdauth.config.loginButtonText
                              : t("third")}
                          </a>
                        )
                      ) : store.displayAccountLoginUnderTheThirdAuthFirst ? (
                        <a onClick={switchLoginMode.bind(null, false)}>
                          {t("back")}
                        </a>
                      ) : null}
                    </div>
                  ) : null}
                </div>
              )}
              {device?.client_type === "ios" ||
              device?.client_type === "android" ||
              isWebMobile ? (
                oemconfig?.showUserAgreement || oemconfig?.showPrivacyPolicy ? (
                  <div
                    className="login-view-all"
                    style={{
                      top: oemconfig?.showPortalBanner
                        ? store.languageUser === "EN"
                          ? "414px"
                          : "392px"
                        : "404px",
                    }}
                  >
                    {t("login-agree")}
                    {oemconfig?.showUserAgreement ? (
                      <a
                        onClick={() => {
                          window.open(
                            `${location.protocol}//${location.hostname}:${location.port}${urlPrefix}/Agreement/UserAgreement/ServiceAgreement-${store.languageUser}.html`
                          );
                        }}
                        style={{
                          marginLeft: store.languageUser === "EN" ? "5px" : 0,
                        }}
                      >
                        {t("user-agreement")}
                      </a>
                    ) : null}

                    {oemconfig?.showUserAgreement &&
                    oemconfig?.showPrivacyPolicy ? (
                      <span>{t("login-symbol")}</span>
                    ) : null}

                    {oemconfig?.showPrivacyPolicy ? (
                      <span>
                        <a
                          onClick={() => {
                            window.open(
                              `${location.protocol}//${location.hostname}:${location.port}${urlPrefix}/Agreement/Privacy/Privacy-${store.languageUser}.html`
                            );
                          }}
                          style={{
                            marginLeft: store.languageUser === "EN" ? "5px" : 0,
                          }}
                        >
                          {t("privacy-policy")}
                        </a>
                      </span>
                    ) : null}
                  </div>
                ) : null
              ) : null}
            </div>
          ) : null}
          {store.needAuthBySMS ? (
            <AuthBySMS
              account={store.account}
              password={store.password}
              cellphone={store.cellphone}
              timeInterval={store.timeInterval}
              isDuplicateSend={store.isDuplicateSend}
              device={device!}
              challenge={challenge!}
              rememberLogin={store.rememberLogin}
              t={t}
              csrftoken={csrftoken}
              refreshAuthMethod={store.refreshAuthMethod}
              oemConfig={oemconfig}
            />
          ) : null}
        </div>
      </div>
    );
  });
};

export default withI18n<ISigninProps>(({ error, urlPrefix, ...otherProps }) => {
  const { lang } = useI18n();
  if (error) {
    return <ErrorPage {...error} lang={lang} urlPrefix={urlPrefix} />;
  }

  return <SignInForm {...otherProps} />;
});

export const getServerSideProps: GetI18nServerSideProps<ISigninProps> = async ({
  req,
  res,
  query,
}) => {
  const userAgent = req ? req.headers["user-agent"] : undefined;
  const SignoutLogin3rdPartyStatusValue = (req as Request).cookies[
    SignoutLogin3rdPartyStatusKey
  ];
  const csrftoken = (req as Request).csrfToken();
  const { login_challenge: queryChallenge = "", ...params } = query;
  const {
    login_challenge: cookieChallenge = "",
    ["client.hide_third_party"]: client_hide_third_party,
  } = (req as Request).cookies;
  const challenge = (queryChallenge || cookieChallenge || "") as string;
  const requestLanguage = getRequestLanguage(req);
  const isIE =
    (req.headers["user-agent"] && /rv:11.0/.test(req.headers["user-agent"])) ||
    false;
  delete params["lang"];
  const urlPrefix = getServerPrefix(req);

  if (challenge) {
    try {
      const { skip, subject, request_url, client, session_id } =
        await getLoginRequest(challenge);
      const loginRequstQuery = querystring.parse(request_url);
      const {
        udids,
        sso,
        oauth2ui_after_authorization_call,
        product: productKey = "anyshare",
      } = loginRequstQuery;

      let { lang } = loginRequstQuery;

      if (!lang || !supportLanguages.includes(lang as string)) {
        lang = requestLanguage;
      }

      const metadata = (client.metadata && client.metadata.device) || {
        name: "",
        description: "",
        client_type: "unknown",
      };
      const { name = "", description = "", client_type = "" } = metadata;

      // client数据是客户端注册时，自行配置的数据，由部署统一注册，配置文件在AnyShareConfig仓库
      const {
        third_party_login_visible = true,
        remember_password_visible = true,
        reset_password_visible = true,
        sms_login_visible = true,
      } = (client.metadata && client.metadata.login_form) || {};

      (res as Response).cookie("lang", lang, {
        httpOnly: false,
        secure: true,
      });

      let device = {
        name: name,
        description: description,
        client_type: client_type,
        udids: typeof udids === "string" ? [udids] : [...(udids || [])],
      } as Auth1SendauthvcodeReqDeviceinfo;

      const isRealSkip =
        skip &&
        SignoutLogin3rdPartyStatusValue !==
          SignoutLogin3rdPartyStatus.firstSignout;

      async function beforeRendererSignin() {
        let lastTimestamp = Date.now();
        const productId = (productKey || "") as string;
        const isWebMobile = getIsMobile(device?.client_type, userAgent);

        const configSection = isWebMobile ? "mobile" : "anyshare";

        if (!oemConfig?.[productId]) {
          oemConfig[productId] = {};
        }

        if (!oemConfig?.[productId]?.[configSection]) {
          oemConfig[productId][configSection] = {};
        }

        if (!oemConfig?.[productId]?.[configSection]?.productOemConfig) {
          // 获取通用配置
          console.log(
            `[${Date()}] [INFO]  {/deploy-web-service/v1/oemconfig} get}  START`
          );

          const {
            data: {
              hideLogo,
              loginBoxStyle,
              theme,
              showPortalBanner: showBanner,
              showUserAgreement: showAgreement,
              showPrivacyPolicy: showPolicy,
              webTemplate,
              desktopTemplate,
              [`favicon.ico`]: favicon,
            },
          } = await deployPublicApi.get(
            `/api/deploy-web-service/v1/oemconfig?section=${
              isWebMobile ? "mobile" : "anyshare"
            }${productId ? `&product=${productId}` : ""}`
          );
          oemConfig[productId][configSection].productOemConfig = {
            isTransparentBoxStyle:
              webTemplate === "regular" && loginBoxStyle === "transparent",
            desktopTemplate,
            loginBoxStyle,
            theme,
            showBanner,
            showAgreement,
            showPolicy,
            favicon,
            webTemplate,
            hideLogo,
          };
        }
        let section:
          | "shareweb_zh-cn"
          | "shareweb_zh-tw"
          | "shareweb_en-us"
          | "mobile_zh-cn"
          | "mobile_zh-tw"
          | "mobile_en-us";
        switch (lang) {
          case "zh":
          case "zh-cn":
            section = isWebMobile ? "mobile_zh-cn" : "shareweb_zh-cn";
            break;
          case "zh-tw":
          case "zh-hk":
            section = isWebMobile ? "mobile_zh-tw" : "shareweb_zh-tw";
            break;
          default:
            section = isWebMobile ? "mobile_en-us" : "shareweb_en-us";
            break;
        }

        if (!oemConfig?.[productId]?.[section]) {
          oemConfig[productId][section] = {};
          // 获取oem配置
          const {
            data: {
              [`logo.png`]: logo,
              [`darklogo.png`]: darklogo,
              product,
              portalBanner,
            },
          } = await deployPublicApi.get(
            `/api/deploy-web-service/v1/oemconfig?section=${section}${
              productId ? `&product=${productId}` : ""
            }`
          );

          oemConfig[productId][section].config = {
            logo,
            darklogo,
            product,
            portalBanner,
          };

          console.log(
            `[${Date()}] [INFO]  {/deploy-web-service/v1/oemconfig} get}  SUCCESS +${
              Date.now() - lastTimestamp
            }ms`
          );
        }
        const product = oemConfig?.[productId]?.[section]?.config?.product;
        const portalBanner =
          oemConfig?.[productId]?.[section]?.config?.portalBanner;

        lastTimestamp = Date.now();
        console.log(
          `[${Date()}] [INFO]  {/api/eacp/v1/auth1/login-configs} GET}  START`
        );
        // 获取认证配置
        const {
          data: {
            enable_secret_mode,
            vcode_server_status,
            oemconfig: { rememberpass },
            vcode_login_config,
            dualfactor_auth_server_status,
            thirdauth,
          },
        } = await eacpPublicApi.get("/api/eacp/v1/auth1/login-configs", {
          headers: {
            "X-Forwarded-For": (req as Request).ip as string,
          },
        });
        console.log(
          `[${Date()}] [INFO]  {/api/eacp/v1/auth1/login-configs} GET}  SUCCESS +${
            Date.now() - lastTimestamp
          }ms`
        );
        const hiddenPage = Boolean(
          device?.client_type !== "console_web" &&
            device?.client_type !== "deploy_web" &&
            thirdauth &&
            thirdauth.config &&
            (sso === "true" || thirdauth.config.autoCasRedirect)
        );
        let remember_visible_value: boolean;
        try {
          if (!hiddenPage) {
            console.log(
              `[${Date()}] [INFO]  {/api/authentication/v1/config/remember_visible} GET}  START`
            );
            //获取remember_visible【记住登录状态】返回值
            const { remember_visible } = await getApplicationbyslug(
              "remember_visible"
            );
            console.log(
              `[${Date()}] [INFO]  {/api/authentication/v1/config/remember_visible get} SUCCESS  +${
                Date.now() - lastTimestamp
              }ms`
            );
            remember_visible_value = remember_visible;
          } else {
            remember_visible_value = true;
          }
        } catch (error) {
          console.log(
            `[${Date()}] [INFO]  {/api/authentication/v1/config/remember_visible get} FAILD  +${
              Date.now() - lastTimestamp
            }ms`,
            error
          );
          remember_visible_value = true;
        }
        const isTransparentBoxStyle =
          !isWebMobile &&
          (getIsElectronOpenExternal(device)
            ? oemConfig?.[productId]?.[configSection]?.productOemConfig
                ?.desktopTemplate === "regular" &&
              oemConfig?.[productId]?.[configSection]?.productOemConfig
                ?.loginBoxStyle === "transparent"
            : oemConfig?.[productId]?.[configSection]?.productOemConfig
                ?.isTransparentBoxStyle);

        const logo = isTransparentBoxStyle
          ? oemConfig?.[productId]?.[section]?.config?.darklogo
          : oemConfig?.[productId]?.[section]?.config?.logo;
        return {
          props: {
            lang: lang as string,
            challenge: challenge,
            csrftoken,
            device,
            ip: (req as Request).ip as string,
            authconfig: {
              thirdAuthFirst: Boolean(thirdauth?.config?.thirdAuthFirst),
              thirdPartyExists: Boolean(
                client_hide_third_party !== "true" &&
                  third_party_login_visible &&
                  thirdauth &&
                  thirdauth?.config &&
                  !thirdauth?.config?.hideThirdLogin
              ),
              enable_secret_mode,
              rememberpass,
              vcode_server_status,
              vcode_login_config,
              dualfactor_auth_server_status,
              thirdauth: thirdauth ? thirdauth : null,
              third_auth_only: Boolean(
                client_hide_third_party !== "true" &&
                  AccessThirdAuthOnlys.includes(device?.client_type) &&
                  third_party_login_visible &&
                  thirdauth &&
                  thirdauth.config &&
                  thirdauth.config.hideLogin &&
                  !thirdauth.config.hideThirdLogin
              ),
              third_party_login_visible,
              remember_password_visible,
              reset_password_visible,
              sms_login_visible,
              sso: sso || "",
            },
            oemconfig: {
              isTransparentBoxStyle,
              theme:
                oemConfig?.[productId]?.[configSection]?.productOemConfig
                  ?.theme,
              logo,
              product,
              productId,
              favicon:
                oemConfig?.[productId]?.[configSection]?.productOemConfig
                  ?.favicon,
              showPortalBanner:
                oemConfig?.[productId]?.[configSection]?.productOemConfig
                  ?.showBanner,
              portalBanner,
              showUserAgreement:
                oemConfig?.[productId]?.[configSection]?.productOemConfig
                  ?.showAgreement,
              showPrivacyPolicy:
                oemConfig?.[productId]?.[configSection]?.productOemConfig
                  ?.showPolicy,
              webTemplate:
                oemConfig?.[productId]?.[configSection]?.productOemConfig
                  ?.webTemplate,
              hideLogo:
                oemConfig?.[productId]?.[configSection]?.productOemConfig
                  ?.hideLogo,
            },
            isIE,
            userAgent,
            hiddenPage,
            remember_visible: remember_visible_value,
            isThirdRememberLogin,
          },
        };
      }

      if (isRealSkip) {
        const lastTimestamp = Date.now();

        console.log(
          `[${Date()}] [INFO]  {/api/authentication/v1/session/${session_id}} GET}  START`
        );

        let context;
        try {
          const { data } = await authenticationPrivateApi.get(
            `/api/authentication/v1/session/${session_id}`
          );
          context = data.context;
        } catch (e: any) {
          // session_id无效，相关上下文信息已被删除
          if (e?.response?.data?.code === 404000000) {
            console.log(
              `[${Date()}] [ERROR]  {/api/authentication/v1/session/${session_id}} GET}  ERROR ${JSON.stringify(
                e?.response?.data
              )}`
            );

            return await beforeRendererSignin();
          }
          throw e;
        }

        console.log(
          `[${Date()}] [INFO]  {/api/authentication/v1/session/${session_id}} GET}  SUCCESS +${
            Date.now() - lastTimestamp
          }ms`
        );
        const { redirect_to } = await acceptLoginRequest(challenge, {
          subject,
          context,
        });
        (res as Response).redirect(redirect_to);
        return {
          props: {
            lang: lang as string,
            csrftoken,
          },
        };
      } else if (queryChallenge) {
        // 增加oauth2ui_after_authorization_call参数，在此处执行完授权直接重定向到自定义的位置
        if (Boolean(oauth2ui_after_authorization_call)) {
          console.log(
            `[${Date()}] [INFO] redirect to oauth2ui_after_authorization_call: ${oauth2ui_after_authorization_call}`
          );

          (res as Response).cookie("login_challenge", challenge);
          (res as Response).redirect(
            oauth2ui_after_authorization_call as string
          );
          return {
            props: {
              lang: lang as string,
              csrftoken,
            },
          };
        }

        return await beforeRendererSignin();
      } else if (Object.keys(params).filter((key) => params[key]).length) {
        let lastTimestamp = Date.now();
        console.log(
          `[${Date()}] [INFO] thirdparty START params: ${JSON.stringify(
            params
          )}`
        );

        console.log(
          `[${Date()}] [INFO]  {/api/eacp/v1/auth1/login-configs} GET}  START`
        );

        // 获取第三方认证id
        const {
          data: { thirdauth },
        } = await eacpPublicApi.get("/api/eacp/v1/auth1/login-configs", void 0);
        console.log(
          `[${Date()}] [INFO]  {/api/eacp/v1/auth1/login-configs} GET}  SUCCESS +${
            Date.now() - lastTimestamp
          }ms`
        );

        let user_id: string, context: any;

        // 第三方页面跳回oauth2-ui时，支持携带udids
        const { udids } = params;
        if (Boolean(udids)) {
          if (typeof udids === "string") {
            device = Object.assign(device, { udids: [udids] });
          } else if (Array.isArray(udids)) {
            device = Object.assign(device, { udids });
          }
        }

        console.log(
          `================device: ${JSON.stringify(device)}================`
        );

        if (
          device!.client_type === "console_web" ||
          device!.client_type === "deploy_web"
        ) {
          lastTimestamp = Date.now();
          console.log(
            `[${Date()}] [INFO]  {/api/eacp/v1/auth1/consolelogin} POST}  START`
          );
          // 管理控制台第三方登录
          ({
            data: { user_id, context },
          } = await eacpPrivateApi.post("/api/eacp/v1/auth1/consolelogin", {
            credential: {
              type: "third_party",
              params,
            },
            device,
            ip: (req as Request).ip || "",
          }));
          console.log(
            `[${Date()}] [INFO]  {/api/eacp/v1/auth1/consolelogin} POST}  SUCCESS +${
              Date.now() - lastTimestamp
            }ms`
          );
        } else {
          lastTimestamp = Date.now();
          console.log(
            `[${Date()}] [INFO]  {/api/eacp/v1/auth1/getbythirdparty} POST}  START`
          );
          // 客户端第三方认证
          ({
            data: { user_id, context },
          } = await eacpPrivateApi.post("/api/eacp/v1/auth1/getbythirdparty", {
            thirdpartyid: thirdauth.id,
            params,
            device,
            ip: (req as Request).ip || "",
          }));
          console.log(
            `[${Date()}] [INFO]  {/api/eacp/v1/auth1/getbythirdparty} POST}  SUCCESS +${
              Date.now() - lastTimestamp
            }ms`
          );
        }

        (res as Response).clearCookie("login_challenge");

        const { redirect_to } = await acceptLoginRequest(challenge, {
          subject: user_id,
          context,
          remember: true,
        });

        console.log(
          `[${Date()}] [INFO]  {/api/authentication/v1/config/remember_for get} START  +${
            Date.now() - lastTimestamp
          }ms`
        );
        const { remember_for } = await getApplicationbyslug("remember_for");

        console.log(
          `[${Date()}] [INFO]  {/api/authentication/v1/config/remember_for get} SUCCESS  +${
            Date.now() - lastTimestamp
          }ms`
        );

        console.log(
          `[${Date()}] [INFO]  {/api/authentication/v1/session/${session_id} PUT} START`
        );
        await authenticationPrivateApi.put(
          `/api/authentication/v1/session/${session_id}`,
          {
            subject: user_id,
            client_id: client.client_id,
            remember_for: remember_for ? remember_for : 30 * 24 * 60 * 60,
            context,
          }
        );
        console.log(
          `[${Date()}] [INFO]  {/api/authentication/v1/session/${session_id} PUT} SUCCESS  +${
            Date.now() - lastTimestamp
          }ms`
        );
        (res as Response).cookie(
          SignoutLogin3rdPartyStatusKey,
          SignoutLogin3rdPartyStatus.null
        );
        (res as Response).redirect(redirect_to);

        return {
          props: {
            lang: lang as string,
            csrftoken,
          },
        };
      } else {
        console.error(`参数不合法，可能第三方登录时缺少票据`);
        res.statusCode = 400;
        return {
          props: {
            lang: lang as string,
            csrftoken,
            error: {
              code: 400000000,
              cause: "参数不合法",
              message: "第三方登录时缺少票据",
            },
            urlPrefix,
          },
        };
      }
    } catch (e: any) {
      const path =
        e &&
        e.request &&
        (e.request.path || (e.request._options && e.request._options.path));
      const errorLog = { data: "", e: "" };
      try {
        errorLog.data = JSON.stringify(e && e.response && e.response.data);
        errorLog.e = JSON.stringify(e);
      } catch (error) {}
      console.error(`[${Date()}] [ERROR]  ${path}  ERROR ${errorLog.data}`);
      console.error(`[${Date()}] [ERROR]  ${path}  ERROR ${errorLog.e}`);

      if (e && e.response && e.response.status !== 503) {
        const { status, data } = e.response;
        res.statusCode = status;
        return {
          props: {
            lang: requestLanguage,
            csrftoken,
            error: data,
            urlPrefix,
          },
        };
      } else {
        const service = getServiceNameFromApi(path);
        console.error(`内部错误，连接${service}服务失败`);
        res.statusCode = 500;
        return {
          props: {
            lang: requestLanguage,
            csrftoken,
            error: {
              code: getErrorCodeFromService(service),
              cause: "内部错误",
              message: `连接${service}服务失败`,
            },
            urlPrefix,
          },
        };
      }
    }
  } else {
    console.error(`参数不合法，缺少login_challenge参数`);
    res.statusCode = 400;
    return {
      props: {
        lang: requestLanguage,
        csrftoken,
        error: {
          code: 400000000,
          cause: "参数不合法",
          message: "缺少login_challenge参数",
        },
        urlPrefix,
      },
    };
  }
};
