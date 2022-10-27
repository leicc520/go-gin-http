package exporter

var (
	imgWidth  = "120"
	imgHeight = "80"
	imgPrefix = []string{".jpg", ".jpeg", ".png", ".gif", ".bmp"}
)

type IFExporter interface {
	Open(file string) (err error)
	Close() error
	File() string
	WriteRow(row []string) error
	SetHeader(header []string) IFExporter
}
