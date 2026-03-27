import React, { FunctionComponent } from "react";
import { AppProps } from "next/app";
import openapi from "../http/index";
import "../style/index.less";
import AntdConfigProvider from "antd/lib/config-provider";
import { getUrlPrefix } from "../common";

if (process.browser) {
  openapi.defaults.https = location.protocol === "https:";
  openapi.defaults.hostname = location.hostname;
  openapi.defaults.port = parseInt(location.port, 10) || undefined;
  openapi.defaults.urlPrefix = getUrlPrefix();
}

const App: FunctionComponent<AppProps> = ({ Component, pageProps }) => {
  return (
    <AntdConfigProvider autoInsertSpaceInButton={false}>
      <Component {...pageProps} />
    </AntdConfigProvider>
  );
};

export default App;
