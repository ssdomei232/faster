// https://segmentfault.com/a/1190000039009572
package booster

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"

	"github.com/imroc/req/v3"
)

// TODO: 拉黑域名
func downloadFile(url string) error {
	urlHash := getUrlHash(url)
	const dnsCheckAttempts = 3
	// 在开始下载前，对原始 URL 做多次 DNS 检查，若发现内网 IP 则拒绝
	if ok, err := checkURLForLocalIPMultiple(url, dnsCheckAttempts); err != nil {
		return err
	} else if ok {
		return fmt.Errorf("ssrf detected: host resolves to local IP")
	}
	// anti-ssrf
	redirectPolicy := func(req *http.Request, via []*http.Request) error {
		// 跳转超过10次，拒绝继续跳转
		if len(via) >= 10 {
			return fmt.Errorf("redirect too much")
		}
		statusCode := req.Response.StatusCode
		if statusCode == 307 || statusCode == 308 {
			// 拒绝跳转访问
			return fmt.Errorf("unsupport redirect method")
		}
		// 多次 DNS 检查主机是否解析到内网 IP
		host := req.URL.Hostname()
		if host == "" {
			// 兜底：若 Hostname 为空，尝试使用原始 Host（可能含端口）
			host = req.URL.Host
		}
		ok, err := resolveHostHasLocalIP(host, dnsCheckAttempts)
		if err != nil {
			return err
		}
		if ok {
			return fmt.Errorf("ssrf detected")
		}
		return nil
	}

	// 创建 req 客户端并设置重定向策略
	client := req.C()
	client.SetRedirectPolicy(redirectPolicy)
	client.R().SetOutputFile("data/" + urlHash + getFileExtensionFromURL(url)).Get(url)
	return nil
}

// IsLocalIP 判断是否是内网ip
func isLocalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	// 判断是否是回环地址, ipv4时是127.0.0.1；ipv6时是::1
	if ip.IsLoopback() {
		return true
	}
	// 判断ipv4是否是内网
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 || // 10.0.0.0/8
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) || // 172.16.0.0/12
			(ip4[0] == 192 && ip4[1] == 168) // 192.168.0.0/16
	}
	// 判断ipv6是否是内网
	if ip16 := ip.To16(); ip16 != nil {
		// 参考 https://tools.ietf.org/html/rfc4193#section-3
		// 参考 https://en.wikipedia.org/wiki/Private_network#Private_IPv6_addresses
		// 判断ipv6唯一本地地址
		return 0xfd == ip16[0]
	}
	// 不是ip直接返回false
	return false
}

// CheckURLForLocalIP 检查URL中是否包含内网IP（仅当主机是纯IP时检查）
func checkURLForLocalIP(rawURL string) (bool, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false, err
	}
	host, _, _ := net.SplitHostPort(parsedURL.Host)
	if host == "" {
		host = parsedURL.Host // 无端口时直接使用Host
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false, nil // 主机是域名，无需检查
	}
	return isLocalIP(ip), nil
}

// resolveHostHasLocalIP 对给定主机名进行多次 DNS 查询（attempts 次）。
// 如果任意一次解析结果包含内网 IP，则返回 true。
// 若全部尝试都失败，会返回最后一次的错误。
func resolveHostHasLocalIP(host string, attempts int) (bool, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		ips, err := net.LookupIP(host)
		if err != nil {
			lastErr = err
			continue
		}
		if len(ips) == 0 {
			// 没有解析到 IP，记录错误并重试
			lastErr = fmt.Errorf("no IPs found for host %s", host)
			continue
		}
		if slices.ContainsFunc(ips, isLocalIP) {
			return true, nil
		}
		// 本次解析未发现内网 IP，继续下一次尝试
	}
	if lastErr != nil {
		return false, lastErr
	}
	return false, nil
}

// checkURLForLocalIPMultiple 检查 URL 的主机是否为内网 IP 或在多次 DNS 解析中解析到内网 IP。
func checkURLForLocalIPMultiple(rawURL string, attempts int) (bool, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return false, err
	}
	// 先尝试使用 Hostname 以剥离端口和方括号
	host := parsedURL.Hostname()
	if host == "" {
		// 兜底：尝试 SplitHostPort
		h, _, _ := net.SplitHostPort(parsedURL.Host)
		if h != "" {
			host = h
		} else {
			host = parsedURL.Host
		}
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return isLocalIP(ip), nil
	}
	return resolveHostHasLocalIP(host, attempts)
}
