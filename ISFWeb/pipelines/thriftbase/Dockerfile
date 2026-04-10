FROM acr.aishu.cn/public/ubuntu:22.04.20260128

ENV DEBIAN_FRONTEND noninteractive

ADD ./Python-3.8.17.tgz /usr/local/

COPY thrift-0.19.0.tar.gz /home

ENV LD_LIBRARY_PATH=/lib:/usr/lib:/usr/local/lib

RUN set -xue && \
    apt-get update && apt-get install --no-install-recommends -y \
    cmake \
    gettext \
    gcc \
    build-essential \
    python3-httplib2 \
    zlib1g-dev \
    libncurses5-dev \
    libgdbm-dev \
    libgdbm-compat-dev \
    libnss3-dev \
    libssl-dev \
    libreadline-dev \
    libffi-dev \
    libsqlite3-dev \
    libbz2-dev \
    liblzma-dev \
    uuid-dev \
    tk-dev \
    locales \
    wget \
    g++ \
    tzdata curl gnupg2 lsb-release ubuntu-keyring && \
    curl https://mirrors.ustc.edu.cn/nginx/keys/nginx_signing.key | gpg --dearmor | tee /usr/share/keyrings/nginx-archive-keyring.gpg >/dev/null && \
    echo "deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] https://mirrors.ustc.edu.cn/nginx/ubuntu/ `lsb_release -cs` nginx" | tee /etc/apt/sources.list.d/nginx.list && \
    apt-get update && \
    apt-get install -y nginx=1.26.1-2~jammy && \
    rm -rf /var/lib/apt/lists/* && \
    localedef -c -f UTF-8 -i en_US en_US.utf8 && \
    localedef -c -f UTF-8 -i zh_CN zh_CN.utf8 && \
    localedef -c -f UTF-8 -i zh_TW zh_TW.utf8 && \
    cd /usr/local/Python-3.8.17 && \
    ./configure --enable-shared && \
    make -j $(nproc) && make install && \
    update-alternatives --install /usr/bin/python python /usr/local/Python-3.8.17/python 1 && \
    cd /home && \
    tar -zxvf thrift-0.19.0.tar.gz && cd thrift-0.19.0 && ./configure && make -j $(nproc) && make install && \
    cd ../ && thrift --version && rm -rf thrift-0.19.0.tar.gz && cp -r /home/thrift-0.19.0/lib/py/src /usr/lib/python3/dist-packages/thrift && \
    pip3 config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple && \
    pip3 --default-timeout=1000 install -U pip setuptools==70.0.0 && \
    pip3 --default-timeout=1000 install --use-pep517 uwsgi==2.0.26 && \
    pip3 --default-timeout=1000 install six==1.16.0 django==4.1.13 requests==2.32.3 pyyaml==6.0.1 && \
    cd / && python3 -m pip uninstall pip -y &&\
    apt-get purge -y curl \
    gnupg2 \    
    gcc \
    wget \
    g++ \
    build-essential \
    zlib1g-dev \
    libncurses5-dev \
    libgdbm-dev \
    libgdbm-compat-dev \
    libnss3-dev \
    libssl-dev \
    libreadline-dev \
    libffi-dev \
    libsqlite3-dev \
    libbz2-dev \
    liblzma-dev \
    uuid-dev \
    tk-dev && \
    apt-get autoremove -y && \
    apt-get clean && \
    rm -rf /usr/local/Python-3.8.17 && \
    rm -rf /usr/local/lib/python3.8/distutils/command/wininst-*.exe && \
    rm -rf /usr/local/lib/python3.8/test/*.pem && \
    rm -rf /usr/local/lib/python3.8/distutils/tests/xxmodule.c && \
    rm -rf /usr/local/lib/python3.8/config-3.8-x86_64-linux-gnu/config.c && \
    rm -rf /usr/local/lib/python3.8/config-3.8-x86_64-linux-gnu/python.o && \
    rm -rf /usr/share/cmake-3.22/Templates/Windows/Windows_TemporaryKey.pfx && \
    rm -rf /usr/share/cmake-3.22/Modules/ && \
    rm -rf /home

### 配置pythonpath环境变量
ENV PYTHONPATH=/usr/lib/python3/dist-packages