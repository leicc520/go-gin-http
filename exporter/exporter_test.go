package exporter

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func DemoType(ss IFExporter, filename string) {
	header := []string{"ASIN", "站点", "重量", "体积", "状态", "操作时间", "备注"}
	dir, _ := os.Getwd()
	err := ss.SetHeader(header).Open(filepath.Join(dir, filename))
	fmt.Println(err, "==========", dir)
	rows := []string{"B0B2K75GVB", "GB", "0.699 kilograms", "6.096 X 36.5 X 38.278 centimeters", "成功", "2022-09-07 03:20", "该ASIN在此站点无搜索结果"}
	err = ss.WriteRow(rows)
	fmt.Println(err, ss, "==========")
	err = ss.Close()
	fmt.Println(err, ss, "==========")
}

func TestType(t *testing.T) {

	ss := NewWebXlsExporter(nil)
	DemoType(ss, "webxls")

	aa := NewExcelExporter(nil)
	DemoType(aa, "demo")

	cc := NewCsvExporter(nil)
	DemoType(cc, "demo")
}
