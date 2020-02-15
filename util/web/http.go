package web

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/labstack/gommon/log"
	"io"
	"net/http"
	"net/url"
	"os"
)

var client = &http.Client{}

func init() {
	proxy := os.Getenv("PROXY")
	if proxy != "" {
		if proxyUrl, err := url.Parse(proxy); err == nil {
			log.Info("设置代理服务器: ", proxy)
			client = &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
		}
	}
}

func GetContent(url string) (*goquery.Document, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Arch Linux kernel 4.6.5) AppleWebKit/537.36 (KHTML, like Gecko) Maxthon/4.0 Chrome/39.0.2146.0 Safari/537.36")
	req.Header.Set("Set-Cookie", "r18=ok")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("DNT", "1")
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("目标网站无法连接: %w", err)
	}
	return goquery.NewDocumentFromReader(res.Body)
}

func Download(url string, dir string) error {
	res, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("目标网站无法连接: %w", err)
	}
	f, err := os.Create(dir)
	if err != nil {
		return fmt.Errorf("服务器错误: 无法创建文件")
	}
	_, _ = io.Copy(f, res.Body)
	return nil
}
