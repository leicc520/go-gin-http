package reader

import (
	"git.ziniao.com/webscraper/go-orm/log"
	"github.com/xuri/excelize/v2"
)

type ExcelReaderSt struct {
	file  string
	sheet string
	fp    *excelize.File
	rows  *excelize.Rows
}

//创建读取csv格式数据
func NewExcelReader(file string) *ExcelReaderSt {
	fp, err := excelize.OpenFile(file)
	if err != nil {
		log.Write(-1, "打开文件失败", file)
	}
	sheets := fp.GetSheetList()
	if len(sheets) < 1 { //页面为空的情况
		fp.Close()
		fp = nil
	}
	//默认取第一个页面的数据
	return &ExcelReaderSt{file: file, fp: fp, sheet: sheets[0]}
}

//关闭链接句柄的处理逻辑
func (s *ExcelReaderSt) Close() {
	if s.fp != nil {
		s.fp.Close()
		s.fp = nil
	}
	if s.rows != nil {
		s.rows.Close()
		s.rows = nil
	}
}

//切换页面处理逻辑
func (s *ExcelReaderSt) SwitchSheet(sheet string) {
	s.sheet = sheet
	if s.rows != nil {
		s.rows.Close()
	}
	s.rows = nil
}

//读取一行数据信息
func (s *ExcelReaderSt) ReadRow() []string {
	if s.fp == nil {
		return nil
	}
	var err error = nil
	if s.rows == nil {
		s.rows, err = s.fp.Rows(s.sheet)
		if err != nil || s.rows == nil {
			log.Write(-1, "读取页面"+s.sheet+"出错", err)
			return nil
		}
	}
	var row []string = nil
	if s.rows.Next() {
		row, err = s.rows.Columns()
		if err != nil {
			return row
		}
	}
	return row
}