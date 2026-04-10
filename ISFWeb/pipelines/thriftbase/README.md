# ThriftBase 基础镜像

## 镜像构建

镜像使用流水线构建，镜像名称维护在 azure-pipelines.yml ImageName 变量。

## 镜像tag说明

1. **ThriftBase 版本**
   - ThriftBase的版本

2. **基础镜像版本**
   - 由 proton 提供的 Ubuntu 基础镜像的版本

3. **镜像小版本号**
   - 初始值为 `1`
   - 当存在非第一部分或第二部分的实质性改动时，数字 +1（例如：1, 2, 3 依次递增）
   - 当一部分或第二部分改动时，镜像小版本号重置为 `1`
