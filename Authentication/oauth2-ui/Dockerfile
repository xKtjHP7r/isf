FROM acr.aishu.cn/public/node:24.13.0-ubuntu22.04.20260128-1

WORKDIR /oauth2

# 设置环境变量
ENV NODE_ENV=production
ENV PORT=4015

COPY .next .next
COPY dist dist

COPY package.prod.json package.json
COPY yarn.lock yarn.lock

COPY next.config.prod.js next.config.js

# 创建并设置 yarn 缓存目录权限
RUN mkdir -p /.yarn /.cache/yarn && \
    chmod -R 777 /.yarn /.cache/yarn && \
    npm install -g yarn --force && \
    yarn install --force --ignore-engines && \
    rm -rf node_modules/public-encrypt/test && \
    chown -R 101:101 /.yarn /.cache/yarn

EXPOSE 4015
CMD ["yarn", "start"]
