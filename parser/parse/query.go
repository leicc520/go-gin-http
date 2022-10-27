package parse

import (
	"git.ziniao.com/webscraper/go-gin-http"
	"github.com/PuerkitoBio/goquery"
	"strings"
)

// Xpath解析器的使用情况逻辑
type QueryParseSt struct {
	node *goquery.Document
}

// 解析数据资料信息
func NewQueryParse(htmlStr string) (*QueryParseSt, error) {
	topNode, err := goquery.NewDocumentFromReader(strings.NewReader(htmlStr))
	if err != nil {
		return nil, err
	}
	return &QueryParseSt{node: topNode}, nil
}

// 通过文件获取解析器的逻辑
func NewQueryParseFromFile(file string) (*QueryParseSt, error) {
	htmlStr := core.ReadFile(file)
	return NewQueryParse(htmlStr)
}

// 获取节点的数据资料信息
func (s *QueryParseSt) TextHTML(expr string) (text string, err error) {
	sel := s.node.Find(expr)
	if sel.Length() < 1 {
		err = ErrNoExists
		return
	}
	text, err = sel.Html()
	return
}

// 获取节点的数据资料信息
func (s *QueryParseSt) InnerText(expr string) (text string, err error) {
	sel := s.node.Find(expr)
	if sel.Length() < 1 {
		err = ErrNoExists
		return
	}
	sel.Find("*").RemoveFiltered("style,noscript,script")
	text = strings.TrimSpace(sel.Text())
	return
}

// 获取节点的数据资料信息
func (s *QueryParseSt) InnerTexts(expr string) (texts []string, err error) {
	sel := s.node.Find(expr)
	if sel.Length() < 1 {
		err = ErrNoExists
		return
	}
	texts = make([]string, 0)
	sel.Find("*").RemoveFiltered("style,noscript,script")
	f := func(_ int, tmpSel *goquery.Selection) string {
		return strings.TrimSpace(tmpSel.Text())
	}
	texts = sel.Map(f)
	return
}
