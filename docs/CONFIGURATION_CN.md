# 单管理员通配版：Docker 与配置说明

本文适用于本仓库的 `codex/admin-wildcard-catchall` 版本，目标是：多个根域、任意层级子域、任意合法 local-part，统一投递给一个管理员账号。

## 1. 为什么容器配置目录是 `/work/config`

`server/config` 是源码树中的目录，其中既有 Go 源码，也有开发和数据库样例。镜像构建完成后，最终运行层不会保留这棵源码目录：

```text
源码构建阶段                         容器最终运行阶段
server/config/config.go   ──编译──>  /work/pmail
server/config/*.json      （样例）    /work/config/（运行数据）
```

两个 Dockerfile 的最终阶段都声明了 `WORKDIR /work`，并把二进制复制为 `/work/pmail`。PMail 根据可执行文件目录确定运行根目录，因此生产配置路径是 `/work/config/config.json`。

宿主机目录名可以任意选择。下面的挂载是正确的：

```bash
-v /www/wwwroot/mail.117799.xyz/config:/work/config
```

挂载后，宿主机 `/www/wwwroot/mail.117799.xyz/config/config.json` 对应容器 `/work/config/config.json`。

## 2. 需要持久化哪些文件

使用 SQLite 和手动 SSL 证书时，建议目录如下：

```text
/www/wwwroot/mail.117799.xyz/config/
├── config.json                 必需：唯一的生产运行配置
├── pmail.db                    必需：SQLite 数据库；首次安装时可由程序创建
├── dkim/
│   └── dkim.priv               必需：DKIM 私钥
└── ssl/
    ├── private.key             必需：SMTP/IMAP/POP3/可选 Web HTTPS 私钥
    └── public.crt              必需：对应证书，建议包含完整证书链
```

不需要复制：

- `config.go`：Go 源码，已经编译进 `/work/pmail`；复制到运行目录不会生效。
- `config.dev.json`：仅在可执行文件最后一个参数是 `dev` 时读取；默认 Docker `CMD /work/pmail` 不读取它。
- `config_mysql.json`：MySQL 样例，程序不会自动读取；SQLite 部署不需要。
- 整个源码 `server/config`：生产容器只需要上面的运行数据。

如果使用 PMail 自动 ACME，还可能生成 `ssl/issuerCert.crt`、`ssl/account_private.pem` 等文件，也应保留整个挂载目录。手动证书模式 `sslType: "1"` 不要求预先存在这些 ACME 文件。

## 3. 到底修改哪个文件

正常 Docker 启动只修改宿主机上的：

```text
/www/wwwroot/mail.117799.xyz/config/config.json
```

修改后执行：

```bash
docker restart pmail
```

JSON 不支持注释，不能把 `// ...` 或 `# ...` 写入 `config.json`。可以复制仓库根目录的 `config.admin-catchall.sqlite.example.json` 作为起点，说明以本文为准。

注意：网页或聊天中显示的 `&#x20;` 是 HTML 空格实体，实际 JSON 文件中必须是普通空格，不能原样写入 `&#x20;`。

## 4. 适合当前部署的完整 SQLite 配置

下面示例保留 `117799.xyz` 和 `nilinkeji.com`；其余根域继续加入 `domains` 数组。只填写根域，不要填写 `*.域名`，也不需要枚举子域名。

```json
{
  "logLevel": "info",
  "domain": "117799.xyz",
  "domains": [
    "117799.xyz",
    "nilinkeji.com"
  ],
  "acceptSubdomains": true,
  "catchAllAccount": "admin",
  "outboundHostname": "mail.117799.xyz",
  "tlsNames": [
    "mail.117799.xyz"
  ],
  "webDomain": "mail.117799.xyz",
  "dkimPrivateKeyPath": "/work/config/dkim/dkim.priv",
  "sslType": "1",
  "SSLPrivateKeyPath": "/work/config/ssl/private.key",
  "SSLPublicKeyPath": "/work/config/ssl/public.crt",
  "dbDSN": "/work/config/pmail.db",
  "dbType": "sqlite",
  "httpsEnabled": 2,
  "spamFilterLevel": 1,
  "httpPort": 80,
  "httpsPort": 443,
  "weChatPushAppId": "",
  "weChatPushSecret": "",
  "weChatPushTemplateId": "",
  "weChatPushUserId": "",
  "tgBotToken": "",
  "tgChatId": "",
  "webPushUrl": "",
  "webPushToken": "",
  "isInit": true
}
```

旧数据库可以继续使用，不需要迁移到 MySQL，也不需要新增 Alias、Domain 或 Catch-All 数据表。必须确认数据库内存在已启用且具有管理员权限的 `admin` 账号。

## 5. 所有配置字段

| 字段 | 推荐值 | 实际作用 |
| --- | --- | --- |
| `logLevel` | `"info"` | 日志级别：空值或 `info` 为普通日志；`debug` 还会输出 SQL，生产环境不建议长期启用；也支持 `warn`、`error`。 |
| `domain` | `"117799.xyz"` | 主根域。兼容旧逻辑，并用于 SMTP 服务标识、默认 Message-ID 等；应同时出现在 `domains`。 |
| `domains` | 50 个根域数组 | 所有允许收件/发件的根域。不要写 `*.`，不要逐个写子域；程序会安全匹配“完全相等”或 `.` 加根域的边界后缀。 |
| `acceptSubdomains` | `true` | 是否把配置根域的任意层级子域视为本地域。你的需求必须为 `true`。 |
| `catchAllAccount` | `"admin"` | 未创建的本地域 local-part 最终归属账号。该账号必须存在、启用且为管理员。 |
| `outboundHostname` | `"mail.117799.xyz"` | PMail 向外部 MX 发信时使用的 EHLO 主机名。应有 A/AAAA，并尽量与 VPS PTR 一致。它不是收件根域列表。 |
| `tlsNames` | `["mail.117799.xyz"]` | 证书应覆盖的服务主机名，也是内置 ACME 的申请名单。50 个收件根域和任意子域不需要都写入这里。 |
| `webDomain` | `"mail.117799.xyz"` | Web 管理端规范主机名，也用于 HTTP 到 HTTPS 的跳转目标。 |
| `dkimPrivateKeyPath` | `/work/config/dkim/dkim.priv` | DKIM 私钥路径。程序启动时必须能读取，否则会停止。当前实现让所有根域复用这把私钥，并按 From 根域动态设置 DKIM `d=`。 |
| `sslType` | `"1"` | `"0"`：内置 ACME HTTP-01；`"1"`：用户提供证书；`"2"`：内置 DNS-01。你已有 SSL 文件时使用 `"1"`。 |
| `SSLPrivateKeyPath` | `/work/config/ssl/private.key` | TLS 私钥。即使 Web HTTPS 关闭，SMTP 25 STARTTLS、465、587、IMAP 993、POP3 995 仍需加载。 |
| `SSLPublicKeyPath` | `/work/config/ssl/public.crt` | TLS 证书。邮件客户端连接 `mail.117799.xyz` 时，该名称必须在证书 SAN 中。 |
| `dbDSN` | `/work/config/pmail.db` | SQLite 文件路径。放在挂载目录中才能在换容器后保留邮件、账号和设置。 |
| `dbType` | `"sqlite"` | 数据库类型。你的单管理员部署使用 SQLite 即可。 |
| `httpsEnabled` | `2` | 只有值 `2` 表示关闭 PMail 内置 HTTPS，并让 HTTP 直接提供 Web 页面；其他值会启用内置 HTTPS，并把 HTTP 请求跳转到 `https://webDomain`。 |
| `spamFilterLevel` | `1` | 原有未知用户过滤级别：`0` 不按 SPF/DKIM 丢弃；`1` 为 SPF 与 DKIM 均失败；`2` 为 SPF 失败；`3` 为 DKIM 失败。显式 `catchAllAccount` 的本地域投递优先于这段旧未知用户丢弃逻辑。 |
| `httpPort` | `80` | 容器内部 HTTP 监听端口；`0` 也会回退到 80。Docker 使用 `16080:80` 时不要把这里改成 16080。 |
| `httpsPort` | `443` | 容器内部 HTTPS 监听端口；`0` 也会回退到 443。仅在 `httpsEnabled != 2` 时监听。 |
| `weChatPushAppId` 等 | 空字符串 | 原有微信模板消息参数；不用就留空。 |
| `tgBotToken`、`tgChatId` | 空字符串 | 原有 Telegram 推送参数；不用就留空。 |
| `webPushUrl`、`webPushToken` | 空字符串 | 原有 Web Push 参数；不用就留空。 |
| `isInit` | `true` | `true` 表示已经完成初始化并按正式服务启动；`false` 会进入初始化流程。沿用旧数据库时保持 `true`。 |

## 6. `config.json`、`config.dev.json`、`config_mysql.json` 的区别

| 文件 | 默认 Docker 是否读取 | 用途 |
| --- | --- | --- |
| `config.json` | 是 | 唯一的正式运行配置。 |
| `config.dev.json` | 否 | 本地源码开发配置；仅当命令最后一个参数为 `dev` 时读取。字段多少不决定优先级。 |
| `config_mysql.json` | 否 | MySQL 示例，需要人工复制/改名为 `config.json` 才可能使用。SQLite 部署忽略它。 |
| `config.go` | 否 | 定义配置结构与加载代码的 Go 源文件，只在构建镜像时编译。 |

## 7. 端口与 HTTPS 两种推荐方式

### 方式 A：宿主机 Nginx/宝塔终止 Web HTTPS（推荐）

保持 `httpsEnabled: 2`，Nginx 将 `https://mail.117799.xyz` 反向代理到 `http://127.0.0.1:16080`。容器 443 不会监听，所以无需映射 `16443:443`。

```bash
docker run -d \
  --name pmail \
  --restart unless-stopped \
  -p 25:25 \
  -p 465:465 \
  -p 587:587 \
  -p 993:993 \
  -p 995:995 \
  -p 127.0.0.1:16080:80 \
  -v /www/wwwroot/mail.117799.xyz/config:/work/config \
  ghcr.io/willamblack/pmail:0.02
```

不使用 POP3 时可省略 110/995；不建议把明文 POP3 110 暴露到公网。即使 Web TLS 由 Nginx 终止，PMail 自己仍需可读的 SSL 文件来服务 SMTP/IMAP TLS。

### 方式 B：PMail 自己提供 Web HTTPS

设置 `httpsEnabled: 1`，并使用标准映射 `80:80`、`443:443`。不建议用 `16443:443`：PMail 的 HTTP 重定向目标是 `https://mail.117799.xyz`，不会自动附加 `:16443`。

## 8. DKIM、证书和 50 个根域

- `tlsNames` 只保留统一服务主机 `mail.117799.xyz`。所有域的 MX 都可以指向它，邮件客户端也连接它；不需要为每个收件地址域或任意层级子域申请证书。
- 当前分支对发件地址按匹配到的根域设置 DKIM `d=`，但复用同一把 `dkim.priv`。因此每个允许发信的根域都应发布同一公钥：`default._domainkey.<根域>`。
- 从根域地址发信时，应在每个根域发布 SPF；SPF 不会从根域自动继承到子域。当前版本使用完整 From 地址作为 SMTP Envelope From，所以从 `abc@a.b.117799.xyz` 发信时，接收方会查询 `a.b.117799.xyz` 的 SPF。任意层级子域发信需要为实际 Envelope From 域提供可解析的 SPF（可按 DNS 提供商能力设计通配记录，但必须按 DNS closest-encloser 规则实测），不能只发布根域 SPF。
- DMARC 应按组织域部署。多层子域 From 由根域 DKIM `d=` 签名时，使用默认/宽松的 `adkim=r` 可以对齐；不要设置严格的 `adkim=s`，否则子域 From 与根域 `d=` 不严格相等。
- `mail.117799.xyz` 的 A/AAAA 必须指向 VPS；PTR 建议回指同一主机名，`outboundHostname` 也使用该名称。

## 9. 修改和启动前检查

先备份整个目录，尤其是 SQLite 和私钥：

```bash
cp -a /www/wwwroot/mail.117799.xyz/config /www/wwwroot/mail.117799.xyz/config.backup
```

检查 JSON 语法、挂载和日志：

```bash
jq empty /www/wwwroot/mail.117799.xyz/config/config.json
docker inspect pmail --format '{{json .Mounts}}'
docker exec pmail sh -c 'pwd && ls -la /work/config /work/config/dkim /work/config/ssl'
docker logs --tail 200 pmail
```

不要在日志或公开 Issue 中贴出 DKIM/SSL 私钥、Telegram token、微信 secret 或 Web Push token。
