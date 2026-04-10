import { withI18n, useI18n, getRequestLanguage } from "../i18n";
import { GetServerSideProps } from "next";
import React, { useEffect, FunctionComponent } from "react";
import { useLocalStore, useObserver } from "mobx-react-lite";
import "mobx-react-lite/batchingForReactDom";
import { pki } from "node-forge";
import classNames from "classnames";
import md5 from "md5";
import openapi, { Auth1ModifypasswordReq } from "../http/index";
import {
  eacpPublicApi,
  deployPublicApi,
  getServiceNameFromApi,
  getErrorCodeFromService,
} from "../core/config";
import { ErrorCode, getErrorMessage } from "../core/errorcode";
import { AppHead } from "../components";
import Button from "antd/lib/button";
import Input from "antd/lib/input";
import { Password } from "../controls";
import ErrorPage from "./_error";
import AccountIcon from "@icons/account.svg";
import NewPasswordIcon from "@icons/new-password.svg";
import PasswordIcon from "@icons/password.svg";
import ConfirmPasswordIcon from "@icons/confirm-password.svg";
import { getLoginRequest } from "../services/hydra";
import { getUrlPrefix, getServerPrefix } from "../common/getUrlPrefix";
import {
  getIsElectronOpenExternal,
  getIsMobile,
} from "../common/getClientType";
import { defaultOem, dipOem } from "../common";
/**
 * 验证码信息
 */
interface CaptchaInfo {
  uuid: string; // 验证码标识
  vcode: string; // 验证码图片
}

interface IChangePasswordProps {
  redirect: string; // 修改成功或取消修改的重定向地址
  account?: string; // 账号
  oemconfig?: { [key: string]: any }; // oem配置信息
  authconfig?: { [key: string]: any }; // 认证配置信息
  error?: { [key: string]: any }; // 错误信息
  client_type?: string; // 客户端类型
  isIE?: boolean; //IE浏览器
  urlPrefix?: string; //前缀
}

interface IChangePasswordState {
  password: string; // 旧密码
  newPassword: string; // 新密码
  confirmPassword: string; // 确认密码
  errorStatus: number; // 错误状态码
  errorInfo: any; // 错误信息
  captchaStatus: boolean; // 是否需要验证码
  captchaInfo: CaptchaInfo; // 验证码信息
  captcha: string; // 验证码值
  strongPasswordLength: number; // 强密码最小长度
  strongPasswordStatus: boolean; // 是否开启强密码模式
  checkInput: () => boolean; // 校验输入
  change: () => void; // 修改密码
  getCaptchaInfo: () => void; // 获取图形验证码信息
  updateStrongPasswordConfig: () => void; // 更新强密码配置信息
  [key: string]: any;
}

const PublicKey: any = pki.publicKeyFromPem(`-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQDB2fhLla9rMx+6LWTXajnK11Kd
p520s1Q+TfPfIXI/7G9+L2YC4RA3M5rgRi32s5+UFQ/CVqUFqMqVuzaZ4lw/uEdk
1qHcP0g6LB3E9wkl2FclFR0M+/HrWmxPoON+0y/tFQxxfNgsUodFzbdh0XY1rIVU
IbPLvufUBbLKXHDPpwIDAQAB
-----END PUBLIC KEY-----`);

export const ChangePasswordForm: FunctionComponent<
  Omit<IChangePasswordProps, "error">
> = ({ account, redirect, oemconfig, authconfig, isIE }) => {
  const { t } = useI18n();
  const urlPrefix = getUrlPrefix();
  const store = useLocalStore<IChangePasswordState>(() => {
    const { vcode_login_config: vcodeLoginConfig } = authconfig!;
    let captchaStatus;
    if (!vcodeLoginConfig.isenable) {
      // 图形验证码不可用
      process.browser && sessionStorage.removeItem("captchaStatus");
      captchaStatus = false;
    } else {
      if (
        vcodeLoginConfig.passwderrcnt === 0 ||
        (process.browser && sessionStorage.getItem("captchaStatus") === "true")
      ) {
        process.browser && sessionStorage.setItem("captchaStatus", "true");
        captchaStatus = true;
      } else {
        process.browser && sessionStorage.removeItem("captchaStatus");
        captchaStatus = false;
      }
    }
    return {
      password: "",
      newPassword: "",
      confirmPassword: "",
      errorStatus: ErrorCode.Normal,
      errorInfo: null,
      captchaStatus: captchaStatus,
      captchaInfo: {
        uuid: "",
        vcode: "",
      },
      captcha: "",
      strongPasswordLength: authconfig!.strong_pwd_length,
      strongPasswordStatus: authconfig!.enable_strong_pwd,
      get getError(): string {
        switch (store.errorStatus) {
          case ErrorCode.PasswordInvalidLocked:
          case ErrorCode.OldPasswordInvalidLocked:
            return getErrorMessage(store.errorStatus, t, {
              time: store.errorInfo.detail.remainlockTime,
            });
          case ErrorCode.PasswordWeak:
            return getErrorMessage(store.errorStatus, t, {
              length: store.strongPasswordLength.toString(),
            });
          default:
            return getErrorMessage(store.errorStatus, t);
        }
      },
      checkInput(): boolean {
        switch (true) {
          case !store.password:
            store.errorStatus = ErrorCode.NoOldPassword;
            return false;
          case !store.newPassword:
            store.errorStatus = ErrorCode.NoNewPassword;
            return false;
          case !store.confirmPassword:
            store.errorStatus = ErrorCode.NoConfirmPassword;
            return false;
          case !store.captcha && store.captchaStatus:
            store.errorStatus = ErrorCode.NoCaptcha;
            return false;
          case store.newPassword !== store.confirmPassword:
            store.errorStatus = ErrorCode.NewConfirmInconsitent;
            return false;
          case store.newPassword === store.password:
            store.errorStatus = ErrorCode.NewIsOld;
            return false;
          default:
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
      async updateStrongPasswordConfig() {
        const {
          data: { enable_strong_pwd, strong_pwd_length },
        } = await openapi.get("/eacp/v1/auth1/login-configs");
        store.strongPasswordStatus = enable_strong_pwd;
        store.strongPasswordLength = strong_pwd_length;
      },
      async change() {
        if (store.checkInput()) {
          try {
            const body = {
              account,
              oldpwd: btoa(
                PublicKey.encrypt(store.password, "RSAES-PKCS1-V1_5")
              ),
              newpwd: btoa(
                PublicKey.encrypt(store.newPassword, "RSAES-PKCS1-V1_5")
              ),
              vcodeinfo: {
                uuid: store.captchaInfo.uuid,
                vcode: store.captcha,
              },
            } as Auth1ModifypasswordReq;
            const sign = md5(JSON.stringify(body) + "eisoo.com");
            await openapi.post(
              `/eacp/v1/auth1/modifypassword?sign=${sign}` as any,
              body
            );
            sessionStorage.removeItem("captchaStatus");
            window.location.href =
              redirect ||
              `${location.protocol}//${location.hostname}:${location.port}${urlPrefix}`;
          } catch (e: any) {
            if (e.response) {
              const {
                response: { data: err },
              } = e;
              store.errorInfo = err;
              switch (err.code) {
                case ErrorCode.AuthFailed:
                  store.errorStatus = ErrorCode.OldPasswordInvalid;
                  break;
                case ErrorCode.PasswordInvalidLocked:
                  store.errorStatus = ErrorCode.OldPasswordInvalidLocked;
                  break;
                case ErrorCode.NewPasswordIsInitial:
                  store.errorStatus = ErrorCode.NewIsInitial;
                  break;
                default:
                  store.errorStatus = err.code;
              }

              if (store.captchaStatus) {
                if (err.detail && !err.detail.isShowStatus) {
                  // 控制台关闭图形验证码
                  sessionStorage.removeItem("captchaStatus");
                  store.captchaStatus = false;
                  store.captchaInfo = {
                    uuid: "",
                    vcode: "",
                  };
                  store.captcha = "";
                } else {
                  // 刷新图形验证码
                  await store.getCaptchaInfo();
                }
              }
              if (
                !store.captchaStatus &&
                err.detail &&
                err.detail.isShowStatus
              ) {
                // 密码错误达指定次数 或 控制台开启图形验证码
                sessionStorage.setItem("captchaStatus", "true");
                store.captchaStatus = true;
                await store.getCaptchaInfo();
              }

              if (store.strongPasswordStatus) {
                switch (err.code) {
                  case ErrorCode.PasswordInvalid: // 控制台开启弱密码
                    store.strongPasswordStatus = false;
                    break;
                  case ErrorCode.PasswordWeak: // 控制台更新强密码配置 或 密码格式错误
                    await store.updateStrongPasswordConfig();
                    break;
                  default:
                    break;
                }
              } else {
                if (err.code === ErrorCode.PasswordWeak) {
                  // 控制台开启强密码
                  await store.updateStrongPasswordConfig();
                }
              }
            } else {
              store.errorInfo = null;
              store.errorStatus = ErrorCode.NoNetwork;
            }
          }
        }
      },
    };
  });

  useEffect(() => {
    if (store.captchaStatus) {
      store.getCaptchaInfo();
    }
  }, []);

  return useObserver(() => {
    return (
      <div className="top-change-password">
        <div
          className={classNames(
            "oauth2-ui-wrapper",
            isIE ? "oauth2-ui-wrapper-ie" : null
          )}
        >
          <AppHead oemconfig={oemconfig!} />
          <div className="change-password-wrapper">
            <div className="title title-change-password">
              {t("change-password-title")}
            </div>
            <div className="content">
              <div className="tip">
                {store.strongPasswordStatus
                  ? t("strong-password-tip", {
                      length: store.strongPasswordLength,
                    })
                  : t("weak-password-tip")}
              </div>
              <div className="account">
                <span className="icon">
                  <AccountIcon />
                </span>
                {account}
              </div>
              <Password
                className="input-item change-password-item"
                type="password"
                prefix={
                  <span className="icon">
                    <PasswordIcon />
                  </span>
                }
                placeholder={t("old-password")}
                value={store.password}
                onChange={(e) => {
                  store.password = e.target.value;
                  store.errorInfo = null;
                  store.errorStatus = ErrorCode.Normal;
                }}
                onDrop={(e) => {
                  e.preventDefault();
                }}
              />
              <Password
                className="input-item change-password-item"
                type="password"
                prefix={
                  <span className="icon">
                    <NewPasswordIcon />
                  </span>
                }
                placeholder={t("new-password")}
                value={store.newPassword}
                onChange={(e) => {
                  store.newPassword = e.target.value;
                  store.errorInfo = null;
                  store.errorStatus = ErrorCode.Normal;
                }}
                onDrop={(e) => {
                  e.preventDefault();
                }}
              />
              <Password
                className="input-item change-password-item"
                type="password"
                prefix={
                  <span className="icon">
                    <ConfirmPasswordIcon />
                  </span>
                }
                placeholder={t("confirm-password")}
                value={store.confirmPassword}
                onChange={(e) => {
                  store.confirmPassword = e.target.value;
                  store.errorInfo = null;
                  store.errorStatus = ErrorCode.Normal;
                }}
                onDrop={(e) => {
                  e.preventDefault();
                }}
              />
              {store.captchaStatus ? (
                <>
                  <Input
                    className="input-item captcha-item change-password-item"
                    type="text"
                    placeholder={t("captcha")}
                    value={store.captcha}
                    onChange={(e) => {
                      store.captcha = e.target.value;
                      store.errorInfo = null;
                      store.errorStatus = ErrorCode.Normal;
                    }}
                    onDrop={(e) => {
                      e.preventDefault();
                    }}
                  />
                  <img
                    className="captcha-item-image"
                    src={`data:image/jpeg;base64,${store.captchaInfo.vcode}`}
                    onClick={async () => {
                      await store.getCaptchaInfo();
                    }}
                  />
                </>
              ) : null}
              {store.errorStatus !== ErrorCode.Normal ? (
                <div className="error-message-text">{store.getError}</div>
              ) : null}
              <div
                className={classNames(
                  "change-button",
                  store.errorStatus && "change-button-space"
                )}
              >
                <Button
                  className="confirm as-components-oem-background-color"
                  type="primary"
                  onClick={store.change}
                >
                  {t("confirm")}
                </Button>
                <Button
                  className="cancel"
                  onClick={() => {
                    sessionStorage.removeItem("captchaStatus");
                    window.location.href =
                      redirect ||
                      `${location.protocol}//${location.hostname}:${location.port}${urlPrefix}`;
                  }}
                >
                  {t("cancel")}
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  });
};

export default withI18n<IChangePasswordProps>(
  ({ error, urlPrefix, ...otherProps }) => {
    const { lang } = useI18n();

    if (error) {
      return <ErrorPage {...error} lang={lang} urlPrefix={urlPrefix} />;
    }
    return <ChangePasswordForm {...otherProps} />;
  }
);

export const getServerSideProps: GetServerSideProps<
  IChangePasswordProps
> = async ({ req, res, query }) => {
  const {
    account,
    redirect = "",
    login_challenge: challenge,
    product: productId,
  } = query;
  const requestLanguage = getRequestLanguage(req);
  const isIE =
    (req.headers["user-agent"] && /rv:11.0/.test(req.headers["user-agent"])) ||
    false;
  const urlPrefix = getServerPrefix(req);
  const userAgent = req.headers["user-agent"];

  if (account) {
    try {
      // 获取客户端类型
      let client_type = "unknown";
      if (challenge) {
        const { client } = await getLoginRequest(challenge as string);
        const metadata = (client.metadata && client.metadata.device) || {
          name: "",
          description: "",
          client_type: "unknown",
        };

        ({ client_type = "unknown" } = metadata);
      }
      const isWebMobile = getIsMobile(client_type, userAgent);

      let lastTimestamp = Date.now();
      let theme: string = defaultOem.config.theme;
      let product: string = defaultOem.sectionConfig.product;
      let favicon: string = defaultOem.config.favicon;
      let loginBoxStyle: string = defaultOem.config.loginBoxStyle;
      let webTemplate: string = defaultOem.config.webTemplate;
      let desktopTemplate: string = defaultOem.config.desktopTemplate;
      console.log(
        `[${Date()}] [INFO]  {/api/deploy-web-service/v1/oemconfig} GET}  START`
      );
      // 获取通用配置
      try {
        const {
          data: {
            loginBoxStyle: currentLoginBoxStyle,
            webTemplate: currentWebTemplate,
            desktopTemplate: currentDesktopTemplate,
            theme: currentTheme,
            [`favicon.ico`]: currentFIcon,
          },
        } = await deployPublicApi.get(
          `/api/deploy-web-service/v1/oemconfig?section=${
            isWebMobile ? "mobile" : "anyshare"
          }${productId ? `&product=${productId}` : ""}`,
        );
        loginBoxStyle = currentLoginBoxStyle;
        webTemplate = currentWebTemplate;
        desktopTemplate = currentDesktopTemplate;
        theme = currentTheme;
        favicon = currentFIcon;
      } catch (error: any) {
        console.error(
          `[${Date()}] [ERROR]  {/api/deploy-web-service/v1/oemconfig} GET}  ERROR`,
          error
        );
        // 当状态码为404且product为dip时，使用默认值作为降级策略
        if (error?.response?.status === 404 && productId === "dip") {
          console.log(
            `[${Date()}] [INFO]  product is dip and deploy-web-service not found`
          );
          theme = dipOem.config.theme;
          favicon = dipOem.config.favicon;
          loginBoxStyle = dipOem.config.loginBoxStyle;
          webTemplate = dipOem.config.webTemplate;
          desktopTemplate = dipOem.config.desktopTemplate;
        } else {
          theme = defaultOem.config.theme;
          favicon = defaultOem.config.favicon;
          loginBoxStyle = defaultOem.config.loginBoxStyle;
          webTemplate = defaultOem.config.webTemplate;
          desktopTemplate = defaultOem.config.desktopTemplate;
        }
      }

      let section: string;

      switch (requestLanguage) {
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

      // 获取oem配置
      try {
        const {
          data: { product: currentProduct },
        } = await deployPublicApi.get(
          `/api/deploy-web-service/v1/oemconfig?section=${section}${
            productId ? `&product=${productId}` : ""
          }`,
        );
        product = currentProduct;
      } catch (error: any) {
        console.error(
          `[${Date()}] [ERROR]  {/api/deploy-web-service/v1/oemconfig} GET}  ERROR`,
          error
        );
        // 当状态码为404且product为dip时，使用默认值作为降级策略
        if (error?.response?.status === 404 && productId === "dip") {
          console.log(
            `[${Date()}] [INFO]  product is dip and deploy-web-service not found, use default oem config`
          );
          product = dipOem.sectionConfig.product;
        } else {
          product = defaultOem.sectionConfig.product;
        }
      }
      console.log(
        `[${Date()}] [INFO]  {/api/deploy-web-service/v1/oemconfig} GET}  SUCCESS +${
          Date.now() - lastTimestamp
        }ms`
      );
      lastTimestamp = Date.now();
      console.log(
        `[${Date()}] [INFO]  {/api/eacp/v1/auth1/login-configs} GET}  START`
      );
      // 获取认证配置
      const {
        data: { enable_strong_pwd, strong_pwd_length, vcode_login_config },
      } = await eacpPublicApi.get("/api/eacp/v1/auth1/login-configs");
      console.log(
        `[${Date()}] [INFO]  {/api/eacp/v1/auth1/login-configs} GET}  SUCCESS +${
          Date.now() - lastTimestamp
        }ms`
      );

      return {
        props: {
          lang: requestLanguage,
          redirect: redirect as string,
          account: account as string,
          client_type: client_type,
          oemconfig: {
            theme,
            product,
            productId,
            favicon,
            isTransparentBoxStyle: getIsElectronOpenExternal({ client_type })
              ? desktopTemplate === "transparent" &&
                loginBoxStyle === "transparent"
              : webTemplate === "regular" && loginBoxStyle === "transparent",
          },
          authconfig: {
            enable_strong_pwd,
            strong_pwd_length,
            vcode_login_config,
          },
          isIE,
        },
      };
    } catch (e: any) {
      const path =
        e &&
        e.request &&
        (e.request.path || (e.request._options && e.request._options.path));
      console.error(
        `[${Date()}] [ERROR]  ${path}  ERROR ${JSON.stringify(
          e && e.response && e.response.data
        )}`
      );
      console.error(`[${Date()}] [ERROR]  ${path}  ERROR ${JSON.stringify(e)}`);
      if (e && e.response && e.response.status !== 503) {
        const { status, data } = e.response;
        res.statusCode = status;
        return {
          props: {
            lang: requestLanguage,
            redirect: redirect as string,
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
            redirect: redirect as string,
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
    console.error(`参数不合法，缺少account参数`);
    res.statusCode = 400;
    return {
      props: {
        lang: requestLanguage,
        redirect: redirect as string,
        error: {
          code: 400000000,
          cause: "参数不合法",
          message: "缺少account参数",
        },
        urlPrefix,
      },
    };
  }
};
