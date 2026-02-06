# ShareServer (EACP)

## 依赖仓库的版本控制

### 一般仓库

构建过程中依赖的仓库及其版本定义在 `azurepipelines.yml` 中，一般为 `git clone -b xxx`。

如需引入依赖仓库的新变更，需在对应仓库创建 tag：

- **格式**：`v<年>.<月>.<日>.<序号>`
- **示例**：`v2026.01.01.1`
- **规则**：序号从 1 开始，同一天内发布多个版本时依次递增，日期变更后序号重置为 1

---

### Deps

Deps 仓库体积较大，不适合每次构建时克隆。当前方案是使用 `git archive` 打包后上传至 FTP，构建时直接下载。

**更新 Deps 包版本的步骤：**

1. 按上述规则创建 tag
2. 访问流水线，基于该 tag 运行：
   > https://devops.aishu.cn/AISHUDevOps/AnyShareFamily/_build?definitionId=6933
3. 根据流水线上传到 FTP 的路径，更新本仓库流水线中 Deps 包的下载链接
