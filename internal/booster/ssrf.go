package booster

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/imroc/req/v3"
)

// TODO: 拉黑域名
func downloadFile(url string) error {
	urlHash := getUrlHash(url)
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
		// 判断 IP
		ips, err := net.LookupIP(req.URL.Host)
		if err != nil {
			return err
		}
		for _, ip := range ips {
			if isLocalIP(ip) {
				return fmt.Errorf("ssrf detected")
			}
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

	// 1. 分离主机和端口（处理带端口的IP如192.168.1.1:8080）
	host, _, _ := net.SplitHostPort(parsedURL.Host)
	if host == "" {
		host = parsedURL.Host // 无端口时直接使用Host
	}

	// 2. 检查是否为纯IP（域名返回false）
	ip := net.ParseIP(host)
	if ip == nil {
		return false, nil // 主机是域名，无需检查
	}

	// 3. 调用您的isLocalIP函数检查
	return isLocalIP(ip), nil
}
