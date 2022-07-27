package core

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io"
	"strconv"

	"github.com/leicc520/go-orm/log"
)

type Crypt struct {
	JKey []byte
}

var DefaultCrypt Crypt

func init()  {
	DefaultCrypt.SetJKey([]byte{44,56,77,90,12,123,65,77,58,96,87,80,25,66,99})
}

//设置密钥数据信息
func (c *Crypt) SetJKey(jkey []byte)  {
	c.JKey = jkey
}

//抑或加密处理逻辑
func (c Crypt) jhash(bstr []byte)  {
	nsize := len(bstr)
	msize := len(c.JKey)
	for idx, j := 0, 0; idx < nsize; idx++ {
		if idx % 2 == 0 {//偶数单位抑或加密
			bstr[idx] ^= c.JKey[j%msize]
			j++
		}
	}
}

//数据的加密处理逻辑 涉及压缩+抑或加密
func (c Crypt) Encrypt(str []byte) (string, error)  {
	var buf bytes.Buffer
	zwer  := zlib.NewWriter(&buf)
	if nsize, err := zwer.Write(str); err != nil {
		log.Write(log.ERROR, "Encrypt data error", err)
		return "", err
	} else {
		zwer.Close()
		bstr:= buf.Bytes()
		c.jhash(bstr)
		ostr := strconv.FormatInt(int64(nsize), 10)
		ostr += "$"+base64.StdEncoding.EncodeToString(bstr)
		return ostr, nil
	}
}

//解密算法处理逻辑
func (c Crypt) Decrypt(str []byte) []byte {
	nidx := bytes.IndexByte(str, '$')
	if nidx < 1 {
		return nil
	}
	nsize, err := strconv.ParseInt(string(str[0:nidx]), 10, 64)
	if err != nil || nsize < 1 {//参数错误的情况
		log.Write(log.ERROR, "Decrypt data nsize", err)
		return nil
	}
	bstr  := make([]byte, len(str))
	nlen, err := base64.StdEncoding.Decode(bstr, str[nidx+1:])
	if err != nil {
		log.Write(log.ERROR, "Decrypt data error", err)
		return nil
	}
	bstr  = bstr[0:nlen]
	c.jhash(bstr)
	var buf bytes.Buffer
	rd   := bytes.NewReader(bstr)
	zrer, err := zlib.NewReader(rd)
	if err != nil {
		log.Write(log.ERROR, "zlib.NewReader data error", err)
		return nil
	}
	defer zrer.Close()
	io.Copy(&buf, zrer)
	return buf.Bytes()
}