package exporter

import (
	"errors"
	"os"
	"strings"
)

type WebXlsExporterSt struct {
	file   string //保存的文件
	fp     *os.File
	rowIdx int
	header []string
}

// 创建对象
func NewWebXlsExporter(header []string) *WebXlsExporterSt {
	return &WebXlsExporterSt{header: header}
}

// 返回文件名称
func (s *WebXlsExporterSt) File() string {
	return s.file
}

// 写一行写入句柄当中
func (s *WebXlsExporterSt) WriteRow(row []string) error {
	s.rowIdx++
	rStr := "<tr>"
	if s.rowIdx != 1 {
		rStr = `<tr height="` + imgHeight + `">`
	}
	for _, str := range row {
		if s.rowIdx == 1 { //首航标题栏
			rStr += "<td align=\"left\">" + str + "</td>"
		} else {
			if s.isImgSrc(str) {
				str = "<img src=\"" + str + "\" width=\"" + imgWidth + "\" height=\"" + imgHeight + "\" />"
			}
			rStr += "<td>" + str + "</td>"
		}
	}
	rStr += "</tr>\r\n"
	_, err := s.fp.WriteString(rStr)
	return err
}

// 设置请求头信息
func (s *WebXlsExporterSt) SetHeader(header []string) IFExporter {
	s.header = header
	return s
}

// 配置图片地址的处理逻辑
func (s *WebXlsExporterSt) isImgSrc(str string) bool {
	if !strings.HasPrefix(str, "http") {
		return false
	}
	str = strings.ToLower(str)
	for _, prefix := range imgPrefix {
		if strings.Contains(str, prefix) {
			return true
		}
	}
	return false
}

// 关闭数据处理逻辑
func (s *WebXlsExporterSt) Close() error {
	if s.fp == nil {
		return errors.New("Save WebXls File Error")
	}
	s.fp.WriteString(`</table></body></html>`)
	s.fp.Close()
	s.fp = nil
	return nil
}

// 打开文件记录逻辑，记得使用完close关闭
func (s *WebXlsExporterSt) Open(file string) (err error) {
	if s.fp != nil { //关闭旧文件
		s.fp.Close()
		s.fp = nil
	}
	s.file = file + ".xlsx"
	s.fp, err = os.OpenFile(s.file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil { //失败的情况处理逻辑
		return err
	}
	s.fp.WriteString(`<html xmlns:o="urn:schemas-microsoft-com:office:office"
xmlns:x="urn:schemas-microsoft-com:office:excel"
xmlns="[url=http://www.w3.org/TR/REC-html40]http://www.w3.org/TR/REC-html40[/url]">
<head>
<meta http-equiv="expires" content="Mon, 06 Jan 1999 00:00:01 GMT">
<meta http-equiv=Content-Type content="text/html; charset="utf-8">
<!--[if gte mso 9]><xml>
<x:ExcelWorkbook>
<x:ExcelWorksheets>
<x:ExcelWorksheet>
<x:Name></x:Name>
<x:WorksheetOptions>
<x:DisplayGridlines/>
</x:WorksheetOptions>
</x:ExcelWorksheet>
</x:ExcelWorksheets>
</x:ExcelWorkbook>
</xml><![endif]-->
</head>
<body link=blue vlink=purple leftmargin=0 topmargin=0>
<table width="100%" border="0" cellspacing="0" cellpadding="0">`)
	return s.WriteRow(s.header) //写入头部文件
}
