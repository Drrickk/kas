package services

import (
	"fmt"
	"github.com/bmaupin/go-epub"
	"github.com/ystyle/kas/core"
	"github.com/ystyle/kas/model"
	"github.com/ystyle/kas/util/file"
	"github.com/ystyle/kas/util/kindlegen"
	"github.com/ystyle/kas/util/site"
	"github.com/ystyle/kas/util/web"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
)

func ArticleSubmit(client *core.WsClient, message core.Message) {
	var book model.ArticleInfo
	err := message.JsonParse(&book)
	if err != nil {
		client.WsSend <- core.NewMessage("Error", "参数解析失败")
		return
	}
	model.Statistics(message.DriveID)
	book.SetDefault()
	client.WsSend <- core.NewMessage("article:bookid", book.ID)

	client.WsSend <- core.NewMessage("info", "开始解析文章内容")
	var wg sync.WaitGroup
	wg.Add(len(book.UrlList))
	for _, item := range book.UrlList {
		client.WsSend <- core.NewMessage("info", fmt.Sprintf("解析文章: %s", item.Title))
		getPage(&wg, item)
	}
	wg.Wait()
	client.WsSend <- core.NewMessage("info", "全部文章解析完成, 正在生成epub")
	e := epub.NewEpub(book.Title)
	e.SetAuthor("KAF")
	for _, item := range book.UrlList {
		for key, img := range item.Images {
			placeholder := fmt.Sprintf("{{ %s }}", key)
			imageSource, err := e.AddImage(img, path.Base(img))
			if err != nil {
				continue
			}

			item.Content = strings.ReplaceAll(item.Content, placeholder, fmt.Sprintf("<img src='%s' />", imageSource))
		}
		e.AddSection(item.Content, item.Title, "", "")
	}
	file.CheckDir(path.Dir(book.EpubFile))
	err = e.Write(book.EpubFile)
	if err != nil {
		client.WsSend <- core.NewMessage("Error", "生成epub失败")
		return
	}
	client.WsSend <- core.NewMessage("info", "epub制作完成，正在转为mobi")
	err = kindlegen.Conver(book.EpubFile, path.Base(book.MobiFile), false)
	if err != nil {
		fmt.Println(err)
		client.WsSend <- core.NewMessage("Error", "生成mobi失败")
		return
	}
	file.CheckDir(path.Dir(book.ZipFile))
	client.WsSend <- core.NewMessage("info", "mobi转换完成，正在压缩文件")
	err = file.CompressZipToFile(book.MobiFile, book.ZipFile)
	if err != nil {
		client.WsSend <- core.NewMessage("Error", "压缩mobi失败")
		return
	}
	client.WsSend <- core.NewMessage("info", "压缩完成，准备下载")
	articleDownload(client, book)
	os.Remove(book.EpubFile)
	os.Remove(book.MobiFile)
}

func getPage(wg *sync.WaitGroup, item *model.ArticleItem) {
	defer wg.Done()
	node, err := web.GetHtmlNode(item.Url)
	if err != nil {
		return
	}
	site.ParseContent(node.Find("body"), item)
}

func articleDownload(client *core.WsClient, book model.ArticleInfo) {
	buff, err := ioutil.ReadFile(book.ZipFile)
	filename := path.Base(book.ZipFile)
	if err != nil {
		readErr := fmt.Sprintf("读取文件失败: %s", filename)
		client.WsSend <- core.NewMessage("Error", readErr)
		return
	}
	client.WsSend <- core.NewMessage("info", fmt.Sprintf("正在下载: %s， 文件大小: %s", filename, file.FormatBytesLength(len(buff))))
	client.WsSend <- core.NewMessage("article:download", buff)
}
