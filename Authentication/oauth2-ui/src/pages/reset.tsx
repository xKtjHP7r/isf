import { withI18n, useI18n, getRequestLanguage } from "../i18n";
import { GetServerSideProps } from "next";
import React, { FunctionComponent } from "react";
import { useLocalStore, useObserver } from "mobx-react-lite";
import "mobx-react-lite/batchingForReactDom";
import openapi from "../http/index";
import {
  eacpPublicApi,
  deployPublicApi,
  getServiceNameFromApi,
  getErrorCodeFromService,
} from "../core/config";
import {
  AppHead,
  IForgetPasswordState,
  VerificationType,
  VerificationContext,
  IForgetPasswordProps,
  SendVerification,
  ResetPassword,
  UserVerification,
  verificationValueType,
} from "../components";
import ErrorPage from "./_error";
import { getLoginRequest } from "../services/hydra";
import classNames from "classnames";
import { getServerPrefix, getUrlPrefix } from "../common/getUrlPrefix";
import {
  getIsElectronOpenExternal,
  getIsMobile,
} from "../common/getClientType";
import { defaultOem, dipOem } from "../common";

export const ForgetPasswordForm: FunctionComponent<
  Omit<IForgetPasswordProps, "error">
> = ({ redirect, oemconfig, authconfig, isIE }) => {
  const { t } = useI18n();
  const urlPrefix = getUrlPrefix();
  const redirectUrl = new URL(redirect);
  const prefixRedirect =
    urlPrefix && !redirectUrl.pathname.startsWith(urlPrefix)
      ? redirectUrl.origin +
        urlPrefix +
        redirectUrl.pathname +
        redirectUrl.search
      : redirect;
  const store = useLocalStore<IForgetPasswordState>(() => {
    const { vcode_server_status, strong_pwd_length, enable_strong_pwd } =
      authconfig!;
    const { send_vcode_by_sms, send_vcode_by_email } = vcode_server_status;
    return {
      isVerifying: true,
      verificationId: "",
      verificationValue: undefined,
      verificationType: VerificationType.NONE,
      strongPasswordStatus: enable_strong_pwd,
      strongPasswordLength: strong_pwd_length,
      sendVcodeType: {
        sendVcodeBySMS: send_vcode_by_sms,
        sendVcodeByEmail: send_vcode_by_email,
      },
      isUserVerification: true,
      account: "",
      updateVerificationValue(value: verificationValueType) {
        store.verificationValue = value;
        store.verificationType = value.email
          ? VerificationType.EMAIL
          : VerificationType.PHONE;
      },
      updateVerificationId(id: string) {
        store.verificationId = id;
      },
      sendVcodeSuccess(uuid: string) {
        store.isVerifying = false;
        store.isUserVerification = false;
        store.verificationId = uuid;
      },
      returnSendVcode() {
        store.isVerifying = true;
        store.isUserVerification = false;
      },
      returnUserVerification() {
        store.isUserVerification = true;
        store.isVerifying = false;
        store.account = "";
      },
      checkAccountSuccess() {
        store.isUserVerification = false;
        store.isVerifying = true;
      },
      updateAccount(account: string) {
        store.account = account;
      },
      updateVerificationType(type: VerificationType) {
        store.verificationType = type;
      },
      async updatePasswordConfig() {
        const {
          data: { vcode_server_status, strong_pwd_length, enable_strong_pwd },
        } = await openapi.get("/eacp/v1/auth1/login-configs");
        store.sendVcodeType = {
          sendVcodeBySMS: (vcode_server_status as any).send_vcode_by_sms,
          sendVcodeByEmail: (vcode_server_status as any).send_vcode_by_email,
        };
        store.strongPasswordLength = strong_pwd_length;
        store.strongPasswordStatus = enable_strong_pwd;
      },
    };
  });

  return useObserver(() => {
    return (
      <div className="top">
        <div
          className={classNames(
            "oauth2-ui-wrapper",
            isIE ? "oauth2-ui-wrapper-ie" : null
          )}
        >
          <AppHead oemconfig={oemconfig!} />
          <div className="forget-password-wrapper">
            <div className="title">{t("reset-password-title")}</div>
            <VerificationContext.Provider value={store}>
              {store.isUserVerification ? (
                <UserVerification
                  t={t}
                  redirect={
                    prefixRedirect ||
                    `${location.protocol}//${location.hostname}:${location.port}${urlPrefix}`
                  }
                />
              ) : store.isVerifying ? (
                <SendVerification t={t} />
              ) : (
                <ResetPassword
                  redirect={
                    prefixRedirect ||
                    `${location.protocol}//${location.hostname}:${location.port}${urlPrefix}`
                  }
                  t={t}
                />
              )}
            </VerificationContext.Provider>
          </div>
        </div>
      </div>
    );
  });
};

export default withI18n<IForgetPasswordProps>(
  ({ error, urlPrefix, ...otherProps }) => {
    const { lang } = useI18n();

    if (error) {
      return <ErrorPage {...error} lang={lang} urlPrefix={urlPrefix} />;
    }
    return <ForgetPasswordForm {...otherProps} />;
  }
);

export const getServerSideProps: GetServerSideProps<
  IForgetPasswordProps
> = async ({ req, res, query }) => {
  const {
    redirect = "",
    login_challenge: challenge,
    product: productId,
  } = query;
  const requestLanguage = getRequestLanguage(req);
  const isIE =
    (req.headers["user-agent"] && /rv:11.0/.test(req.headers["user-agent"])) ||
    false;
  const userAgent = req.headers["user-agent"];
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
        }${productId ? `&product=${productId}` : ""}`
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
      // 当状态码为404且product为dip时，使用dip默认值
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
        `api/deploy-web-service/v1/oemconfig?section=${section}${
          productId ? `&product=${productId}` : ""
        }`
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
      data: { strong_pwd_length, vcode_server_status, enable_strong_pwd },
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
          strong_pwd_length,
          vcode_server_status,
          enable_strong_pwd,
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
    const urlPrefix = getServerPrefix(req);
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
};
