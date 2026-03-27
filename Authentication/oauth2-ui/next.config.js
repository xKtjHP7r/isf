/* eslint-disable */
const withLess = require("@zeit/next-less");
const path = require("path");
const isDEV = process.env.NODE_ENV === "development";
const prdConfig = isDEV ? {} : {};

// SVG 加载器配置
const svgrLoaderConfig = {
  test: /\.svg$/,
  use: [
    {
      loader: "@svgr/webpack",
      options: {
        typescript: false,
        svgoConfig: {
          plugins: {
            removeViewBox: false,
          },
        },
      },
    },
  ],
};

// 配置别名
const aliasConfig = {
  "@icons": path.resolve(__dirname, "src/icons"),
};

module.exports = withLess({
  transpileModules: ["templite", "rosetta", "is-retry-allowed"],
  lessLoaderOptions: {
    lessOptions: {
      javascriptEnabled: true,
    },
  },
  typescript: {
    ignoreDevErrors: true,
  },
  basePath: "/oauth2-ui",
  experimental: {
    basePath: "/oauth2-ui",
  },
  assetPrefix: "./",
  ...prdConfig,
  webpack(config, options) {
    // IE11 兼容配置：修改现有 babel-loader 配置并设置 IE11 目标
    const babelLoaderRule = config.module.rules.find(
      (rule) =>
        rule.loader === "babel-loader" || rule.use?.loader === "babel-loader",
    );
    if (babelLoaderRule) {
      // 设置 test 匹配 js/jsx/ts/tsx 文件
      babelLoaderRule.test = /\.(js|jsx|ts|tsx)$/;

      // 扩展 include 到所有需要转译的文件
      babelLoaderRule.include = [
        /node_modules/,
        /src/,
        /pages/,
        /components/,
        /http/,
        /common/,
        /style/,
        /icons/,
        /is-retry-allowed/,
        /axios/,
        /axios-retry/,
      ];

      // 排除不需要转译的模块
      babelLoaderRule.exclude = [
        /node_modules\/core-js/,
        /node_modules\/next/,
        /node_modules\/webpack/,
        /node_modules\/react/,
        /node_modules\/react-dom/,
      ];

      // 修改 preset-env 配置为 IE11
      if (
        babelLoaderRule.use &&
        babelLoaderRule.use.options &&
        babelLoaderRule.use.options.presets
      ) {
        babelLoaderRule.use.options.presets =
          babelLoaderRule.use.options.presets.map((preset) => {
            if (Array.isArray(preset) && preset[0] === "@babel/preset-env") {
              return [
                "@babel/preset-env",
                {
                  useBuiltIns: "usage",
                  corejs: 3,
                  targets: {
                    ie: "11",
                  },
                },
              ];
            }
            if (Array.isArray(preset) && preset[0] === "next/babel") {
              const [name, options] = preset;
              return [
                name,
                {
                  ...options,
                  "preset-env": {
                    useBuiltIns: "usage",
                    corejs: 3,
                    targets: {
                      ie: "11",
                    },
                  },
                },
              ];
            }
            return preset;
          });
      }
    }

    // 确保 templite 和 rosetta 模块被转译
    config.module.rules[0].include.push(
      /templite[\/]dist/,
      /rosetta[\/]dist/,
      /is-retry-allowed/,
      /axios/,
      /axios-retry/,
    );
    const { exclude } = config.module.rules[0];
    config.module.rules[0].exclude = (excludePath) => {
      if (
        [
          /templite[\/]dist/,
          /rosetta[\/]dist/,
          /is-retry-allowed/,
          /axios/,
          /axios-retry/,
        ].some((reg) => reg.test(excludePath))
      ) {
        return false;
      }
      return exclude(excludePath);
    };

    // 添加 SVG 加载器配置
    config.module.rules.push(svgrLoaderConfig);

    // 添加处理图片文件的规则
    config.module.rules.push({
      test: /\.(png|jpe?g|gif)$/i,
      use: [
        {
          loader: "file-loader",
          options: {
            publicPath: "/oauth2/_next/static/images/",
            outputPath: `/static/images/`,
            name: "[name].[hash].[ext]",
          },
        },
      ],
    });

    config.resolve.alias = {
      ...config.resolve.alias,
      ...aliasConfig,
    };
    return config;
  },
});
