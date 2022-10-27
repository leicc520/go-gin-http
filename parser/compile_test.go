package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"git.ziniao.com/webscraper/go-gin-http/proxy"
)

func init() {
	os.Chdir("../../")
}

func TestDemo(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()

	dir, _ := os.Getwd()
	tt := TemplateSt{Request: &BaseRequest{}}
	tt.LoadFile("./config/template/amazon-product-101.yml")
	link := "https://www.amazon.com/dp/B0BBSLF2GT?th=1"
	file := "/cachedir/once/20221011/aaa.html"
	fmt.Println(filepath.Join(dir, file))
	result, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		fmt.Println(err)
		return
	}
	item, err := NewCompiler(string(result), proxy.DEVICE_PC).Parse(link, tt.DataFields)
	fmt.Println(item, err)
	fmt.Println(tt.IsAllCollected(item))
}
