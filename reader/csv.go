package reader

import (
	"bufio"
	"os"
	"strings"

	"github.com/leicc520/go-orm/log"
)

type CsvReaderSt struct {
	file 	string
	fp   	*os.File
	reader 	*bufio.Reader
}

//创建读取csv格式数据
func NewCsvReader(file string) *CsvReaderSt {
	var reader *bufio.Reader = nil
	fp, err := os.Open(file)
	if err != nil {
		log.Write(-1, "打开文件失败", file)
	} else {
		reader = bufio.NewReader(fp)
	}
	return &CsvReaderSt{file:file, fp: fp, reader: reader}
}

//关闭链接句柄的处理逻辑
func (s *CsvReaderSt) Close() {
	if s.fp != nil {
		s.fp.Close()
		s.fp = nil
	}
}

//读取一行数据信息
func (s *CsvReaderSt) ReadRow() []string {
	if s.fp == nil || s.reader == nil {
		return nil
	}
	line, _, err := s.reader.ReadLine()
	if err != nil {//结束
		return nil
	}
	return strings.Split(string(line),",")
}
