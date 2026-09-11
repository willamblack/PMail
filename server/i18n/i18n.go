package i18n

var (
	cn = map[string]string{
		"all_email":             "全部邮件数据",
		"inbox":                 "收件箱",
		"outbox":                "发件箱",
		"sketch":                "草稿箱",
		"aperror":               "账号或密码错误",
		"unknowError":           "未知错误",
		"succ":                  "成功",
		"send_fail":             "发送失败",
		"att_err":               "附件解码错误",
		"login_exp":             "登录已失效",
		"ip_taps":               "这是你服务器IP，确保这个IP正确",
		"service_ip_tips":       "统一邮件服务主机的IP；只需为这个实际连接主机配置A记录和TLS证书",
		"root_mx_tips":          "根域收件记录；MX目标必须是可解析A/AAAA的主机名，不能是CNAME",
		"wildcard_mx_tips":      "子域通配收件；显式存在的DNS节点会截断通配匹配，需在该分支补精确MX和分支通配MX",
		"root_spf_tips":         "根域发件SPF；上线验证稳定后可把~all收紧为-all",
		"wildcard_spf_tips":     "子域Envelope-From的通配SPF；受DNS显式节点和closest-encloser规则限制",
		"dkim_tips":             "本版本按匹配到的根域签名；每个允许发件的根域都要发布同一公钥",
		"dmarc_tips":            "监控模式DMARC；确认对齐和报告正常后再逐步改为quarantine或reject",
		"invalid_email_address": "无效的邮箱地址！",
		"deleted":               "已删除",
		"junk":                  "广告箱",
	}
	en = map[string]string{
		"all_email":             "All Email",
		"inbox":                 "Inbox",
		"outbox":                "Outbox",
		"sketch":                "Sketch",
		"aperror":               "Incorrect account number or password",
		"unknowError":           "Unknow Error",
		"succ":                  "Success",
		"send_fail":             "Send Failure",
		"att_err":               "Attachment decoding error",
		"login_exp":             "Login has expired.",
		"ip_taps":               "This is your server's IP, make sure it is correct.",
		"service_ip_tips":       "IP of the canonical mail service host; only this connection hostname needs the A record and TLS certificate.",
		"root_mx_tips":          "Inbound MX for the root domain. The target must resolve to A/AAAA and must not be a CNAME.",
		"wildcard_mx_tips":      "Wildcard inbound MX for subdomains. Explicit DNS nodes block wildcard synthesis; add exact and branch wildcard MX records there.",
		"root_spf_tips":         "SPF for root-domain senders. Tighten ~all to -all after successful rollout validation.",
		"wildcard_spf_tips":     "Wildcard SPF for subdomain envelope senders; DNS explicit-node and closest-encloser rules still apply.",
		"dkim_tips":             "This build signs with the matched root domain; publish this same public key for every sending root.",
		"dmarc_tips":            "Monitoring-mode DMARC. Move gradually to quarantine or reject after alignment and reports are verified.",
		"invalid_email_address": "Invalid e-mail address!",
		"deleted":               "Deleted",
		"junk":                  "Junk",
	}
)

func GetText(lang, key string) string {
	if lang == "zhCn" {
		text, exist := cn[key]
		if !exist {
			return ""
		}
		return text
	}
	text, exist := en[key]
	if !exist {
		return ""
	}
	return text
}
