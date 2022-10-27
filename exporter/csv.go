package exporter

import (
	"errors"
	"os"
	"strings"
)

type CsvExporterSt struct {
	file   string //保存的文件
	fp     *os.File
	header []string
}

// 创建对象
func NewCsvExporter(header []string) *CsvExporterSt {
	return &CsvExporterSt{header: header}
}

// 返回文件名称
func (s *CsvExporterSt) File() string {
	return s.file
}

// 写一行写入句柄当中
func (s *CsvExporterSt) WriteRow(row []string) error {
	//替换一下逗号，否则格式出来可能会有问题
	for idx, str := range row {
		row[idx] = strings.Replace(str, ",", "，", -1)
	}
	str := strings.Join(row, ",")
	_, err := s.fp.WriteString(str + "\r\n")
	return err
}

// 关闭数据处理逻辑
func (s *CsvExporterSt) Close() error {
	if s.fp == nil {
		return errors.New("Save CSV File Error")
	}
	s.fp.Close()
	s.fp = nil
	return nil
}

// 设置请求头信息
func (s *CsvExporterSt) SetHeader(header []string) IFExporter {
	s.header = header
	return s
}

// 打开文件记录逻辑，记得使用完close关闭
func (s *CsvExporterSt) Open(file string) (err error) {
	if s.fp != nil { //关闭旧文件
		s.fp.Close()
		s.fp = nil
	}
	s.file = file + ".csv"
	s.fp, err = os.OpenFile(s.file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil { //失败的情况处理逻辑
		return err
	}
	s.WriteRow(s.header) //写入头部文件
	return err
}
