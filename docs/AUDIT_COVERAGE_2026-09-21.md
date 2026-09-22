# 全仓审查覆盖清单

固定快照：03f9eee40de5779edf0c7ad057a3cee6cf19a482；定制基线：0ff28fbd67daa380278ef0aba1a05f8d032c5c14。

初始 Git 跟踪文件 233 个：reviewed 217，skipped 16；全部有明确归属，没有静默遗漏。
reviewed 表示源码/配置审读；锁文件表示依赖解析与漏洞审查，不代表已审计每一个第三方依赖实现。也不代表每一行均有测试覆盖。
测试、新增代码和审计报告属于修复后的增量，单独列出，不用它们扩大初始覆盖率。

## 初始快照逐文件清单

| Path | 结果 | 说明 |
| --- | --- | --- |
| `.github/ISSUE_TEMPLATE/bug_report.yml` | reviewed | 源码或配置审读 |
| `.github/ISSUE_TEMPLATE/config.yml` | reviewed | 源码或配置审读 |
| `.github/ISSUE_TEMPLATE/feature_request.md` | reviewed | 源码或配置审读 |
| `.github/workflows/docker_build.yml` | reviewed | 源码或配置审读 |
| `.github/workflows/docker_build_admin_catchall.yml` | reviewed | 源码或配置审读 |
| `.github/workflows/docker_build_pre.yml` | reviewed | 源码或配置审读 |
| `.github/workflows/release.yml` | reviewed | 源码或配置审读 |
| `.github/workflows/unitTest.yml` | reviewed | 源码或配置审读 |
| `.gitignore` | reviewed | 源码或配置审读 |
| `Dockerfile` | reviewed | 源码或配置审读 |
| `DockerfileGithubAction` | reviewed | 源码或配置审读 |
| `LICENSE` | skipped | 许可证，不是代码 |
| `Makefile` | reviewed | 源码或配置审读 |
| `README.md` | skipped | 上游宣传/介绍文档，未逐行作为功能证据 |
| `README_ADMIN_CATCHALL.md` | reviewed | 源码或配置审读 |
| `README_CN.md` | skipped | 上游宣传/介绍文档，未逐行作为功能证据 |
| `config.admin-catchall.sqlite.example.json` | reviewed | 源码或配置审读 |
| `docker-compose.admin-catchall.yml` | reviewed | 源码或配置审读 |
| `docs/CONFIGURATION_CN.md` | reviewed | 源码或配置审读 |
| `docs/cn.gif` | skipped | 二进制演示图 |
| `docs/debug.md` | reviewed | 源码或配置审读 |
| `docs/en.gif` | skipped | 二进制演示图 |
| `docs/nuisance/demo.txt` | skipped | 样例数据 |
| `docs/settings.jpg` | skipped | 二进制演示图 |
| `fe/.eslintrc.cjs` | reviewed | 源码或配置审读 |
| `fe/.gitignore` | reviewed | 源码或配置审读 |
| `fe/.vscode/extensions.json` | reviewed | 源码或配置审读 |
| `fe/README.md` | reviewed | 源码或配置审读 |
| `fe/index.html` | reviewed | 源码或配置审读 |
| `fe/package.json` | reviewed | 源码或配置审读 |
| `fe/public/favicon.ico` | skipped | 二进制图标 |
| `fe/src/App.vue` | reviewed | 源码或配置审读 |
| `fe/src/assets/base.css` | reviewed | 源码或配置审读 |
| `fe/src/assets/logo.svg` | reviewed | 源码或配置审读 |
| `fe/src/assets/main.css` | reviewed | 源码或配置审读 |
| `fe/src/components/GroupSettings.vue` | reviewed | 源码或配置审读 |
| `fe/src/components/HomeAside.vue` | reviewed | 源码或配置审读 |
| `fe/src/components/HomeHeader.vue` | reviewed | 源码或配置审读 |
| `fe/src/components/LanguageSelector.vue` | reviewed | 源码或配置审读 |
| `fe/src/components/PluginSettings.vue` | reviewed | 源码或配置审读 |
| `fe/src/components/RuleSettings.vue` | reviewed | 源码或配置审读 |
| `fe/src/components/SecuritySettings.vue` | reviewed | 源码或配置审读 |
| `fe/src/components/UserManagement.vue` | reviewed | 源码或配置审读 |
| `fe/src/i18n/i18n.js` | reviewed | 源码或配置审读 |
| `fe/src/main.js` | reviewed | 源码或配置审读 |
| `fe/src/router/index.js` | reviewed | 源码或配置审读 |
| `fe/src/stores/group.js` | reviewed | 源码或配置审读 |
| `fe/src/stores/useGlobalStatusStore.js` | reviewed | 源码或配置审读 |
| `fe/src/utils/axios.js` | reviewed | 源码或配置审读 |
| `fe/src/utils/email.js` | reviewed | 源码或配置审读 |
| `fe/src/views/EditerView.vue` | reviewed | 源码或配置审读 |
| `fe/src/views/EmailDetailView.vue` | reviewed | 源码或配置审读 |
| `fe/src/views/ListView.vue` | reviewed | 源码或配置审读 |
| `fe/src/views/LoginView.vue` | reviewed | 源码或配置审读 |
| `fe/src/views/SetupView.vue` | reviewed | 源码或配置审读 |
| `fe/vite.config.js` | reviewed | 源码或配置审读 |
| `fe/yarn.lock` | reviewed | 机器生成依赖文件，解析/扫描审查 |
| `server/config/config.dev.json` | reviewed | 源码或配置审读 |
| `server/config/config.go` | reviewed | 源码或配置审读 |
| `server/config/config.json` | reviewed | 源码或配置审读 |
| `server/config/config_mysql.json` | reviewed | 源码或配置审读 |
| `server/config/dkim/README.md` | reviewed | 源码或配置审读 |
| `server/config/dkim/dkim.priv` | skipped | 开发密码学数据夹具，不打印/逐字审读私钥 |
| `server/config/dkim/dkim.public` | skipped | 开发密码学数据夹具，不打印/逐字审读私钥 |
| `server/config/ssl/README.md` | reviewed | 源码或配置审读 |
| `server/config/ssl/private.key` | skipped | 开发密码学数据夹具，不打印/逐字审读私钥 |
| `server/config/ssl/public.crt` | skipped | 开发密码学数据夹具，不打印/逐字审读私钥 |
| `server/config/ssl/server.csr` | skipped | 开发密码学数据夹具，不打印/逐字审读私钥 |
| `server/consts/consts.go` | reviewed | 源码或配置审读 |
| `server/controllers/attachments.go` | reviewed | 源码或配置审读 |
| `server/controllers/base.go` | reviewed | 源码或配置审读 |
| `server/controllers/email/delete.go` | reviewed | 源码或配置审读 |
| `server/controllers/email/detail.go` | reviewed | 源码或配置审读 |
| `server/controllers/email/list.go` | reviewed | 源码或配置审读 |
| `server/controllers/email/list_test.go` | reviewed | 源码或配置审读 |
| `server/controllers/email/move.go` | reviewed | 源码或配置审读 |
| `server/controllers/email/read.go` | reviewed | 源码或配置审读 |
| `server/controllers/email/send.go` | reviewed | 源码或配置审读 |
| `server/controllers/group.go` | reviewed | 源码或配置审读 |
| `server/controllers/interceptor.go` | reviewed | 源码或配置审读 |
| `server/controllers/login.go` | reviewed | 源码或配置审读 |
| `server/controllers/ping.go` | reviewed | 源码或配置审读 |
| `server/controllers/plugin.go` | reviewed | 源码或配置审读 |
| `server/controllers/rule.go` | reviewed | 源码或配置审读 |
| `server/controllers/settings.go` | reviewed | 源码或配置审读 |
| `server/controllers/setup.go` | reviewed | 源码或配置审读 |
| `server/controllers/user.go` | reviewed | 源码或配置审读 |
| `server/db/init.go` | reviewed | 源码或配置审读 |
| `server/db/retry.go` | reviewed | 源码或配置审读 |
| `server/db/retry_test.go` | reviewed | 源码或配置审读 |
| `server/dto/parsemail/dkim.go` | reviewed | 源码或配置审读 |
| `server/dto/parsemail/dkim_test.go` | reviewed | 源码或配置审读 |
| `server/dto/parsemail/email.go` | reviewed | 源码或配置审读 |
| `server/dto/parsemail/email_authentication_test.go` | reviewed | 源码或配置审读 |
| `server/dto/parsemail/email_forward_test.go` | reviewed | 源码或配置审读 |
| `server/dto/parsemail/email_test.go` | reviewed | 源码或配置审读 |
| `server/dto/parsemail/email_xss_test.go` | reviewed | 源码或配置审读 |
| `server/dto/parsemail/encodedword.go` | reviewed | 源码或配置审读 |
| `server/dto/response/email.go` | reviewed | 源码或配置审读 |
| `server/dto/response/response.go` | reviewed | 源码或配置审读 |
| `server/dto/rule.go` | reviewed | 源码或配置审读 |
| `server/dto/tag.go` | reviewed | 源码或配置审读 |
| `server/go.mod` | reviewed | 源码或配置审读 |
| `server/go.sum` | reviewed | 机器生成依赖文件，解析/扫描审查 |
| `server/hooks/base.go` | reviewed | 源码或配置审读 |
| `server/hooks/base_test.go` | reviewed | 源码或配置审读 |
| `server/hooks/debug/debug.go` | reviewed | 源码或配置审读 |
| `server/hooks/framework/framework.go` | reviewed | 源码或配置审读 |
| `server/hooks/framework/framework_test.go` | reviewed | 源码或配置审读 |
| `server/hooks/spam_block/README.md` | reviewed | 源码或配置审读 |
| `server/hooks/spam_block/export/Makefile` | reviewed | 源码或配置审读 |
| `server/hooks/spam_block/export/export.go` | reviewed | 源码或配置审读 |
| `server/hooks/spam_block/requirements.txt` | reviewed | 源码或配置审读 |
| `server/hooks/spam_block/spam_block.go` | reviewed | 源码或配置审读 |
| `server/hooks/spam_block/spam_block_test.go` | reviewed | 源码或配置审读 |
| `server/hooks/spam_block/static/index.html` | reviewed | 源码或配置审读 |
| `server/hooks/spam_block/static/jquery.js` | skipped | 第三方压缩脚本；未逐行/未纳入Yarn扫描 |
| `server/hooks/spam_block/test.py` | reviewed | 源码或配置审读 |
| `server/hooks/spam_block/testData/data.csv` | skipped | 训练数据 |
| `server/hooks/spam_block/tools/tools.go` | reviewed | 源码或配置审读 |
| `server/hooks/spam_block/train.py` | reviewed | 源码或配置审读 |
| `server/hooks/spam_block/trainData/data.csv` | skipped | 训练数据 |
| `server/hooks/spam_block/trec06c_format.py` | reviewed | 源码或配置审读 |
| `server/hooks/spam_block/trec07p_format.py` | reviewed | 源码或配置审读 |
| `server/hooks/wechat_push/README.md` | reviewed | 源码或配置审读 |
| `server/hooks/wechat_push/wechat_push.go` | reviewed | 源码或配置审读 |
| `server/hooks/wechat_push/wechat_push_test.go` | reviewed | 源码或配置审读 |
| `server/i18n/i18n.go` | reviewed | 源码或配置审读 |
| `server/listen/cron_server/ssl_update.go` | reviewed | 源码或配置审读 |
| `server/listen/http_server/http_server.go` | reviewed | 源码或配置审读 |
| `server/listen/http_server/http_server_test.go` | reviewed | 源码或配置审读 |
| `server/listen/http_server/https_server.go` | reviewed | 源码或配置审读 |
| `server/listen/http_server/setup_server.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/imap_server.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/imap_server_test.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/server.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_copy.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_create.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_delete.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_expunge.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_fetch.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_idle.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_list.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_login.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_move.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_namespace.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_poll.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_rename.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_search.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_select.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_status.go` | reviewed | 源码或配置审读 |
| `server/listen/imap_server/session_store.go` | reviewed | 源码或配置审读 |
| `server/listen/pop3_server/action.go` | reviewed | 源码或配置审读 |
| `server/listen/pop3_server/action_test.go` | reviewed | 源码或配置审读 |
| `server/listen/pop3_server/pop3server.go` | reviewed | 源码或配置审读 |
| `server/listen/smtp_server/action.go` | reviewed | 源码或配置审读 |
| `server/listen/smtp_server/action_recipient_test.go` | reviewed | 源码或配置审读 |
| `server/listen/smtp_server/audit_persistence.go` | reviewed | 源码或配置审读 |
| `server/listen/smtp_server/audit_persistence_test.go` | reviewed | 源码或配置审读 |
| `server/listen/smtp_server/delivery_error.go` | reviewed | 源码或配置审读 |
| `server/listen/smtp_server/delivery_error_test.go` | reviewed | 源码或配置审读 |
| `server/listen/smtp_server/email_authentication_test.go` | reviewed | 源码或配置审读 |
| `server/listen/smtp_server/login.go` | reviewed | 源码或配置审读 |
| `server/listen/smtp_server/read_content.go` | reviewed | 源码或配置审读 |
| `server/listen/smtp_server/read_content_test.go` | reviewed | 源码或配置审读 |
| `server/listen/smtp_server/smtp.go` | reviewed | 源码或配置审读 |
| `server/listen/smtp_server/smtp_test/sendEmailTest.py` | reviewed | 源码或配置审读 |
| `server/main.go` | reviewed | 源码或配置审读 |
| `server/main_test.go` | reviewed | 源码或配置审读 |
| `server/models/User.go` | reviewed | 源码或配置审读 |
| `server/models/email.go` | reviewed | 源码或配置审读 |
| `server/models/group.go` | reviewed | 源码或配置审读 |
| `server/models/rule.go` | reviewed | 源码或配置审读 |
| `server/models/session.go` | reviewed | 源码或配置审读 |
| `server/models/user_email.go` | reviewed | 源码或配置审读 |
| `server/models/version.go` | reviewed | 源码或配置审读 |
| `server/res_init/init.go` | reviewed | 源码或配置审读 |
| `server/services/attachments/attachments.go` | reviewed | 源码或配置审读 |
| `server/services/auth/auth.go` | reviewed | 源码或配置审读 |
| `server/services/del_email/del_email.go` | reviewed | 源码或配置审读 |
| `server/services/detail/detail.go` | reviewed | 源码或配置审读 |
| `server/services/group/group.go` | reviewed | 源码或配置审读 |
| `server/services/group/group_test.go` | reviewed | 源码或配置审读 |
| `server/services/list/imap_search.go` | reviewed | 源码或配置审读 |
| `server/services/list/list.go` | reviewed | 源码或配置审读 |
| `server/services/list/list_test.go` | reviewed | 源码或配置审读 |
| `server/services/rule/match/base.go` | reviewed | 源码或配置审读 |
| `server/services/rule/match/contains_match.go` | reviewed | 源码或配置审读 |
| `server/services/rule/match/equal_match.go` | reviewed | 源码或配置审读 |
| `server/services/rule/match/regex_match.go` | reviewed | 源码或配置审读 |
| `server/services/rule/match/regex_match_test.go` | reviewed | 源码或配置审读 |
| `server/services/rule/rule.go` | reviewed | 源码或配置审读 |
| `server/services/rule/rule_forward_test.go` | reviewed | 源码或配置审读 |
| `server/services/setup/db.go` | reviewed | 源码或配置审读 |
| `server/services/setup/db_test.go` | reviewed | 源码或配置审读 |
| `server/services/setup/dns.go` | reviewed | 源码或配置审读 |
| `server/services/setup/dns_test.go` | reviewed | 源码或配置审读 |
| `server/services/setup/domain.go` | reviewed | 源码或配置审读 |
| `server/services/setup/domain_test.go` | reviewed | 源码或配置审读 |
| `server/services/setup/finish.go` | reviewed | 源码或配置审读 |
| `server/services/setup/ssl/challenge.go` | reviewed | 源码或配置审读 |
| `server/services/setup/ssl/dnsProvide.go` | reviewed | 源码或配置审读 |
| `server/services/setup/ssl/ssl.go` | reviewed | 源码或配置审读 |
| `server/services/setup/ssl/ssl_test.go` | reviewed | 源码或配置审读 |
| `server/session/init.go` | reviewed | 源码或配置审读 |
| `server/signal/signal.go` | reviewed | 源码或配置审读 |
| `server/utils/address/address.go` | reviewed | 源码或配置审读 |
| `server/utils/address/address_test.go` | reviewed | 源码或配置审读 |
| `server/utils/array/array.go` | reviewed | 源码或配置审读 |
| `server/utils/async/async.go` | reviewed | 源码或配置审读 |
| `server/utils/consts/consts.go` | reviewed | 源码或配置审读 |
| `server/utils/context/context.go` | reviewed | 源码或配置审读 |
| `server/utils/errors/error.go` | reviewed | 源码或配置审读 |
| `server/utils/file/file.go` | reviewed | 源码或配置审读 |
| `server/utils/id/logid.go` | reviewed | 源码或配置审读 |
| `server/utils/ip/ip.go` | reviewed | 源码或配置审读 |
| `server/utils/maildomain/domain.go` | reviewed | 源码或配置审读 |
| `server/utils/maildomain/domain_test.go` | reviewed | 源码或配置审读 |
| `server/utils/password/encode.go` | reviewed | 源码或配置审读 |
| `server/utils/password/encode_test.go` | reviewed | 源码或配置审读 |
| `server/utils/send/send.go` | reviewed | 源码或配置审读 |
| `server/utils/send/send_test.go` | reviewed | 源码或配置审读 |
| `server/utils/smtp/smtp.go` | reviewed | 源码或配置审读 |
| `server/utils/smtp/smtp_test.go` | reviewed | 源码或配置审读 |
| `server/utils/utf7/LICENSE` | reviewed | 源码或配置审读 |
| `server/utils/utf7/README.md` | reviewed | 源码或配置审读 |
| `server/utils/utf7/decoder.go` | reviewed | 源码或配置审读 |
| `server/utils/utf7/decoder_test.go` | reviewed | 源码或配置审读 |
| `server/utils/utf7/encoder.go` | reviewed | 源码或配置审读 |
| `server/utils/utf7/encoder_test.go` | reviewed | 源码或配置审读 |
| `server/utils/utf7/utf7.go` | reviewed | 源码或配置审读 |
| `server/utils/version/version.go` | reviewed | 源码或配置审读 |
| `server/utils/version/version_test.go` | reviewed | 源码或配置审读 |

## OCR 原始差异覆盖

预览的 59 个 (path,status) 全部列出。42 个 reviewable 条目完成审查；17 个默认 excluded 条目另行人工审读/依赖扫描，不因默认排除而忽略用户要求。

| Path | Diff status | 预览 | 本次处理 |
| --- | --- | --- | --- |
| `.github/workflows/docker_build_admin_catchall.yml` | added | reviewable | reviewed |
| `Dockerfile` | modified | reviewable | reviewed |
| `DockerfileGithubAction` | modified | reviewable | reviewed |
| `config.admin-catchall.sqlite.example.json` | added | reviewable | reviewed |
| `docker-compose.admin-catchall.yml` | added | reviewable | reviewed |
| `fe/index.html` | modified | reviewable | reviewed |
| `fe/src/App.vue` | modified | reviewable | reviewed |
| `fe/src/assets/base.css` | modified | reviewable | reviewed |
| `fe/src/assets/main.css` | modified | reviewable | reviewed |
| `fe/src/components/HomeAside.vue` | modified | reviewable | reviewed |
| `fe/src/components/HomeHeader.vue` | modified | reviewable | reviewed |
| `fe/src/components/LanguageSelector.vue` | added | reviewable | reviewed |
| `fe/src/i18n/i18n.js` | modified | reviewable | reviewed |
| `fe/src/stores/group.js` | modified | reviewable | reviewed |
| `fe/src/utils/email.js` | added | reviewable | reviewed |
| `fe/src/views/EmailDetailView.vue` | modified | reviewable | reviewed |
| `fe/src/views/ListView.vue` | modified | reviewable | reviewed |
| `fe/src/views/LoginView.vue` | modified | reviewable | reviewed |
| `fe/src/views/SetupView.vue` | modified | reviewable | reviewed |
| `server/config/config.dev.json` | modified | reviewable | reviewed |
| `server/config/config.go` | modified | reviewable | reviewed |
| `server/config/config.json` | modified | reviewable | reviewed |
| `server/config/config_mysql.json` | modified | reviewable | reviewed |
| `server/controllers/email/list.go` | modified | reviewable | reviewed |
| `server/controllers/email/send.go` | modified | reviewable | reviewed |
| `server/dto/parsemail/dkim.go` | modified | reviewable | reviewed |
| `server/dto/parsemail/email.go` | modified | reviewable | reviewed |
| `server/i18n/i18n.go` | modified | reviewable | reviewed |
| `server/listen/http_server/http_server.go` | modified | reviewable | reviewed |
| `server/listen/http_server/https_server.go` | modified | reviewable | reviewed |
| `server/listen/http_server/setup_server.go` | modified | reviewable | reviewed |
| `server/listen/smtp_server/action.go` | modified | reviewable | reviewed |
| `server/listen/smtp_server/audit_persistence.go` | modified | reviewable | reviewed |
| `server/listen/smtp_server/read_content.go` | modified | reviewable | reviewed |
| `server/services/list/list.go` | modified | reviewable | reviewed |
| `server/services/rule/rule.go` | modified | reviewable | reviewed |
| `server/services/setup/db.go` | modified | reviewable | reviewed |
| `server/services/setup/dns.go` | modified | reviewable | reviewed |
| `server/services/setup/domain.go` | modified | reviewable | reviewed |
| `server/services/setup/ssl/ssl.go` | modified | reviewable | reviewed |
| `server/utils/maildomain/domain.go` | added | reviewable | reviewed |
| `server/utils/smtp/smtp.go` | modified | reviewable | reviewed |
| `README_ADMIN_CATCHALL.md` | added | excluded | reviewed |
| `README_CN.md` | modified | excluded | skipped: 上游宣传/介绍文档，未逐行作为功能证据 |
| `docs/CONFIGURATION_CN.md` | added | excluded | reviewed |
| `server/controllers/email/list_test.go` | added | excluded | reviewed |
| `server/dto/parsemail/dkim_test.go` | modified | excluded | reviewed |
| `server/dto/parsemail/email_test.go` | modified | excluded | reviewed |
| `server/go.mod` | modified | excluded | reviewed |
| `server/listen/http_server/http_server_test.go` | added | excluded | reviewed |
| `server/listen/smtp_server/action_recipient_test.go` | added | excluded | reviewed |
| `server/listen/smtp_server/audit_persistence_test.go` | modified | excluded | reviewed |
| `server/services/list/list_test.go` | modified | excluded | reviewed |
| `server/services/setup/db_test.go` | added | excluded | reviewed |
| `server/services/setup/dns_test.go` | added | excluded | reviewed |
| `server/services/setup/domain_test.go` | added | excluded | reviewed |
| `server/services/setup/ssl/ssl_test.go` | modified | excluded | reviewed |
| `server/utils/maildomain/domain_test.go` | added | excluded | reviewed |
| `server/utils/smtp/smtp_test.go` | modified | excluded | reviewed |

## 本轮新增源码/回归测试

- reviewed: `fe/src/utils/mailBody.js`
- reviewed: `fe/src/utils/requestScope.js`
- reviewed: `fe/src/utils/ruleForm.js`
- reviewed: `fe/src/utils/setup.js`
- reviewed: `fe/tests/browser-mail-security.mjs`
- reviewed: `fe/tests/browser-webmail-regressions.mjs`
- reviewed: `fe/tests/review-regressions.test.mjs`
- reviewed: `server/config/config_test.go`
- reviewed: `server/controllers/email/send_security_test.go`
- reviewed: `server/controllers/security_test.go`
- reviewed: `server/dto/parsemail/security_review_test.go`
- reviewed: `server/hooks/audit_http_test.go`
- reviewed: `server/integration_fixture_test.go`
- reviewed: `server/listen/http_server/auth_security_test.go`
- reviewed: `server/listen/imap_server/audit_safety_test.go`
- reviewed: `server/listen/imap_server/session_messages.go`
- reviewed: `server/listen/pop3_server/audit_safety_test.go`
- reviewed: `server/listen/smtp_server/security_review_test.go`
- reviewed: `server/services/auth/auth_test.go`
- reviewed: `server/services/del_email/audit_safety_test.go`
- reviewed: `server/services/detail/detail_security_test.go`
- reviewed: `server/services/group/audit_safety_test.go`
- reviewed: `server/services/list/audit_imap_test.go`
- reviewed: `server/services/rule/audit_move_test.go`
- reviewed: `server/services/setup/ssl/challenge_test.go`
- reviewed: `server/utils/async/async_test.go`
- reviewed: `server/utils/httputil/json.go`
- reviewed: `server/utils/httputil/json_test.go`
