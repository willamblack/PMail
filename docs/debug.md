测试imap协议返回

`openssl s_client  -crlf  -connect imap.xxxx.com:993`
`openssl s_client  -crlf  -connect 127.0.0.1:993`


删除邮件
```bash
A1 LOGIN testCase testCase
A3 SELECT "Deleted Messages" 
A4 UID STORE 1:* +FLAGS.SILENT (\Deleted) 
A5 EXPUNGE
```

搜索邮件
```bash
A1 LOGIN testCase testCase
114 SELECT "INBOX"
115 UID SEARCH 1:5 NOT DELETED
```


普通回归测试使用临时 SQLite 和本机随机端口，不读取/删除运行配置，也不需要 sudo：

```bash
make build_fe
make test
cd server && go vet ./... && go test -race -p 1 ./...
```

旧全服务器端到端测试改为显式 `make test_integration`，只在隔离开发环境执行，
它仍会绑定 25/465/587/110/995/993/17888，不能在生产邮件服务器上运行。
测试配置与数据库由独立临时目录隔离；失败/超时会返回非零退出码。
当前旧 `test.domain` 夹具没有真实 SPF/DKIM，移除原有校验绕过后，
它仍在收件箱断言处失败；不能把默认跳过此测试称为全服务器验收通过。
旧 MySQL/PostgreSQL 测试使用固定数据库的方式已禁用，不能用于生产库。

执行单个测试用例
`go test -v -run ^Test ./services/del_email`
