# GitHub Container Registry (GHCR) 使用指南

## 🎯 优势

使用 GitHub Container Registry (ghcr.io) 相比 Docker Hub 有以下优势：

- ✅ **无需额外配置** - 使用 GitHub 内置的 `GITHUB_TOKEN`
- ✅ **与仓库集成** - 镜像与代码仓库直接关联
- ✅ **权限继承** - 自动继承 GitHub 仓库的访问权限
- ✅ **无费用限制** - 公共仓库免费，私有仓库有慷慨的免费额度
- ✅ **更好的集成** - 在 GitHub 界面直接查看镜像

## 📦 镜像命名规则

使用 GitHub Container Registry 时，镜像会按以下格式命名：

```
ghcr.io/{github用户名或组织名}/{镜像名}:{标签}
```

### 示例

如果您的 GitHub 用户名是 `zhangyuchen`，镜像将会是：

```bash
# 主分支最新版本
ghcr.io/zhangyuchen/webservice:latest

# 分支版本
ghcr.io/zhangyuchen/webservice:main
ghcr.io/zhangyuchen/webservice:develop

# 带提交哈希的版本
ghcr.io/zhangyuchen/webservice:main-abc1234
```

## 🚀 使用方法

### 1. 推送代码触发构建

只需推送代码到任何分支：

```bash
git add .
git commit -m "your commit message"
git push origin your-branch-name
```

### 2. 查看构建状态

1. 在 GitHub 仓库页面点击 `Actions` 标签
2. 查看最新的工作流运行状态
3. 点击具体运行查看详细日志

### 3. 查看镜像

构建完成后，可以在以下位置查看镜像：

1. **GitHub 仓库页面**：右侧边栏的 "Packages" 部分
2. **直接链接**：`https://github.com/users/{用户名}/packages/container/webservice`
3. **组织仓库**：`https://github.com/orgs/{组织名}/packages/container/webservice`

## 🔧 拉取和使用镜像

### 拉取镜像

```bash
# 拉取最新版本
docker pull ghcr.io/zhangyuchen/webservice:latest

# 拉取特定分支版本
docker pull ghcr.io/zhangyuchen/webservice:develop

# 拉取特定提交版本
docker pull ghcr.io/zhangyuchen/webservice:main-abc1234
```

### 运行容器

```bash
# 基本运行
docker run -p 8080:8080 ghcr.io/zhangyuchen/webservice:latest

# 带环境变量运行
docker run -p 8080:8080 \
  -e SERVER_PORT=8080 \
  -e LOG_LEVEL=info \
  ghcr.io/zhangyuchen/webservice:latest

# 挂载配置文件运行
docker run -p 8080:8080 \
  -v $(pwd)/config.yaml:/app/config.yaml \
  ghcr.io/zhangyuchen/webservice:latest
```

## 🔒 私有仓库权限配置

如果您的 GitHub 仓库是私有的，需要配置拉取权限：

### 1. 创建个人访问令牌 (PAT)

1. 前往 GitHub Settings → Developer settings → Personal access tokens
2. 点击 "Generate new token"
3. 选择权限：`read:packages`
4. 复制生成的令牌

### 2. 登录到 GitHub Container Registry

```bash
echo "YOUR_PAT_TOKEN" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
```

### 3. 在 Kubernetes 中使用

创建 Secret：

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ghcr-secret
type: kubernetes.io/dockerconfigjson
data:
  .dockerconfigjson: BASE64_ENCODED_DOCKER_CONFIG
```

在 Pod 中使用：

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: webservice
spec:
  containers:
  - name: webservice
    image: ghcr.io/zhangyuchen/webservice:latest
  imagePullSecrets:
  - name: ghcr-secret
```

## 📊 镜像标签策略

工作流会自动创建以下标签：

| 触发条件 | 标签示例 | 说明 |
|---------|---------|------|
| 推送到 main | `latest`, `main`, `main-abc1234` | 主分支标签 |
| 推送到 develop | `develop`, `develop-abc1234` | 开发分支标签 |
| 推送到 feature/xxx | `feature-xxx`, `feature-xxx-abc1234` | 功能分支标签 |
| Pull Request | `pr-123` | PR 标签 |

## 🛡️ 安全最佳实践

### 1. 镜像扫描

工作流已集成 Trivy 安全扫描，会自动：
- 扫描镜像漏洞
- 将结果上传到 GitHub Security 标签
- 在严重漏洞时发出警告

### 2. 最小权限原则

工作流使用的权限：
- `contents: read` - 读取仓库内容
- `packages: write` - 写入容器镜像
- `security-events: write` - 上传安全扫描结果

### 3. 签名验证 (可选)

可以使用 cosign 对镜像进行签名：

```bash
# 安装 cosign
curl -O -L "https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64"
sudo mv cosign-linux-amd64 /usr/local/bin/cosign
sudo chmod +x /usr/local/bin/cosign

# 签名镜像
cosign sign ghcr.io/zhangyuchen/webservice:latest
```

## 🔍 故障排除

### 常见问题

1. **权限错误**
   ```
   Error: denied: permission_denied
   ```
   - 检查仓库的 Actions 权限设置
   - 确认 `GITHUB_TOKEN` 有 `packages: write` 权限

2. **镜像推送失败**
   ```
   Error: failed to push to registry
   ```
   - 检查网络连接
   - 查看 GitHub Actions 日志获取详细错误信息

3. **镜像拉取失败**
   ```
   Error: pull access denied
   ```
   - 确认镜像名称和标签正确
   - 私有仓库需要登录认证

### 调试命令

```bash
# 检查本地 Docker 配置
docker system info

# 测试登录
echo $GITHUB_TOKEN | docker login ghcr.io -u $GITHUB_USERNAME --password-stdin

# 查看镜像信息
docker inspect ghcr.io/zhangyuchen/webservice:latest
```

## 📈 监控和维护

### 1. 定期清理旧镜像

可以设置策略自动删除旧的镜像版本：

1. 前往 GitHub Package 设置
2. 配置保留策略
3. 设置最大保留版本数

### 2. 监控镜像大小

```bash
# 查看镜像大小
docker images ghcr.io/zhangyuchen/webservice

# 优化建议
# - 使用多阶段构建
# - 清理不必要的文件
# - 使用 .dockerignore
```

### 3. 更新依赖

定期更新：
- GitHub Actions 版本
- Docker 基础镜像
- Go 版本和依赖包

## 🎉 总结

使用 GitHub Container Registry 的好处：

1. **开箱即用** - 无需额外配置账户和令牌
2. **安全可靠** - 与 GitHub 安全模型集成
3. **费用友好** - 大多数用例下免费
4. **易于管理** - 在 GitHub 界面统一管理代码和镜像

现在您可以直接推送代码，GitHub Actions 会自动构建并推送镜像到 GitHub Container Registry！
