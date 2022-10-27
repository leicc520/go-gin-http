package proxy

//附加的请求头也允许定制化配置
type HeaderItemSt struct {
	Key string    `json:"key" yaml:"key"`
	Value string  `json:"value" yaml:"value"`
}

type HeaderSt []HeaderItemSt

//转换成map数据信息存在起来
func (s HeaderSt) ASMap() map[string]string {
	data := map[string]string{}
	for _, item := range s {
		data[item.Key] = item.Value
	}
	return data
}
