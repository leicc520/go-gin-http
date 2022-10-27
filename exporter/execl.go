package exporter

import (
	"errors"
	"fmt"
	"github.com/xuri/excelize/v2"
)

type ExcelExporterSt struct {
	file      string //保存的文件
	fp        *excelize.File
	activeIdx int
	rowIdx    int
	sheet     string
	header    []string
}

// 创建对象
func NewExcelExporter(header []string) *ExcelExporterSt {
	return &ExcelExporterSt{header: header}
}

// 返回文件名称
func (s *ExcelExporterSt) File() string {
	return s.file
}

// 写一行写入句柄当中
func (s *ExcelExporterSt) WriteRow(row []string) error {
	s.rowIdx++
	axis := fmt.Sprintf("A%d", s.rowIdx)
	return s.fp.SetSheetRow(s.sheet, axis, &row)
}

// 关闭数据处理逻辑
func (s *ExcelExporterSt) Close() error {
	if s.fp == nil {
		return errors.New("Save Excel File Error")
	}
	s.fp.SaveAs(s.file)
	s.fp.Close()
	s.fp = nil
	return nil
}

// 设置当前活动分页
func (s *ExcelExporterSt) SetActiveSheet(idx int) *ExcelExporterSt {
	s.activeIdx = idx
	s.fp.SetActiveSheet(s.activeIdx)
	return s
}

// 添加一个分页处理逻辑
func (s *ExcelExporterSt) AddSheet(name string) *ExcelExporterSt {
	s.rowIdx = 0
	s.sheet = name
	s.activeIdx = s.fp.NewSheet(name)
	s.fp.SetActiveSheet(s.activeIdx)
	return s
}

// 设置请求头信息
func (s *ExcelExporterSt) SetHeader(header []string) IFExporter {
	s.header = header
	return s
}

// 打开文件记录逻辑，记得使用完close关闭
func (s *ExcelExporterSt) Open(file string) error {
	if s.fp != nil { //关闭旧文件
		s.fp.Close()
		s.fp = nil
	}
	s.file = file + ".xlsx"
	s.fp = excelize.NewFile()
	//写入头部文件
	return s.AddSheet("Sheet1").WriteRow(s.header)
}
