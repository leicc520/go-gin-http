package reader

import (
	"errors"
	"strings"
)

type IFReader interface {
	ReadRow() []string
	Close()
}

func Factory(file string) (IFReader, error) {
	subffix := ""
	npos := strings.LastIndex(file, ".")
	if npos > 0 {//获取后缀处理逻辑
		subffix = strings.ToLower(file[npos:])
	}
	switch subffix {
	case ".csv":
		s := NewCsvReader(file)
		return s, nil
	case ".xlsx":
	case ".xls":
		s := NewExcelReader(file)
		return s, nil
	}
	return nil, errors.New("文件打开不支持"+file)
}


