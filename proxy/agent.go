package proxy

import (
	"math/rand"
	"strconv"
	"strings"
	"time"
)

const (
	DEVICE_PC = "pc"
	DEVICE_APP= "mobile"
	DEVICE_ALL= "all"
)

var windowSys = []string{"Win32; x86", "Win64; x64", "WOW64"}

type AgentSegSt []string

//生产随机数处理逻辑
func (s AgentSegSt) RandStr(start, end int) string {
	n := rand.Intn(end)
	if start > 0 && start < end {
		n = start + n % (end - start)
	}
	return strconv.FormatInt(int64(n), 10)
}

//获取系统的处理逻辑
func (s AgentSegSt) getWindowSys(seed int64) string {
	if seed < 1 { //如果小于0的情况
		seed = time.Now().UnixNano()
	}
	idx := seed%int64(len(windowSys))
	return "Windows NT "+s.RandStr(6, 11)+"."+s.RandStr(1, 8)+"; " + windowSys[idx]
}

//获取mac系统的数据资料信息
func (s AgentSegSt) getMacOsX() string {
	return "(Macintosh; U;  Intel Mac OS X "+s.RandStr(9, 10)+"_"+s.RandStr(9, 16)+"_"+s.RandStr(4, 8)+")"
}

//获取浏览器版本Chrome
func (s AgentSegSt) getChromeWebkit() string {
	vMin := s.RandStr(537, 539)+"."+s.RandStr(35, 38)
	vMain:= s.RandStr(97, 101)+".0."+s.RandStr(3951, 4968)+"."+s.RandStr(26, 58)
	return "AppleWebKit/"+vMin+" (KHTML, like Gecko) Chrome/"+vMain+" Safari/"+vMin
}

//获取浏览器版本firefox
func (s AgentSegSt) getFileFoxWebkit() string {
	vMin := s.RandStr(2016, 2021)+"0"+s.RandStr(1, 8)+"0"+s.RandStr(1, 8)
	vMain:= s.RandStr(101, 108)+".0"
	return "Gecko/"+vMin+" Firefox/"+vMain
}

//获取浏览器版本apple
func (s AgentSegSt) getAppleWebkit() string {
	vMin := s.RandStr(532, 538)+"."+s.RandStr(20, 26)
	return "AppleWebKit/"+vMin+".25 (KHTML, like Gecko) Version/"+s.RandStr(3,8)+".0."+s.RandStr(3,6)+" Safari/"+vMin+".27"
}

//获取随机的浏览器地址
func UserAgent(device string) string {
	if device == DEVICE_APP {
		return UserAgentMobile()
	}
	return UserAgentPc()
}

//获取PC端的浏览器头
func UserAgentPc() string {
	rand.Seed(time.Now().UnixNano())
	seed  := rand.Int63()
	agent := AgentSegSt{"Mozilla/5.0"}
	rate  := seed % 10
	if rate < 6 {//window 60%
		agent = append(agent, agent.getWindowSys(seed))
	} else {
		agent = append(agent, agent.getMacOsX())
	}
	if rate <= 4 {
		agent = append(agent, agent.getChromeWebkit())
	} else if rate <= 7 {
		agent = append(agent, agent.getAppleWebkit())
	} else {
		agent = append(agent, agent.getFileFoxWebkit())
	}
	return strings.Join(agent, " ")
}

//获取随机的浏览器地址
func UserAgentMobile() string {
	rand.Seed(time.Now().UnixNano())
	agent := AgentSegSt{"Mozilla/5.0"}
	agent  = append(agent, agent.linuxSystem())
	agent  = append(agent, agent.phoneSubffix())
	return strings.Join(agent, " ")
}

func (s AgentSegSt) phoneDevice() string {
	ratio := rand.Int() % 100
	if ratio < 30 {
		return "MI" + s.RandStr(6, 13)
	} else if ratio < 40 {
		return "Redmi Note" + s.RandStr(6, 13)
	} else if ratio < 70 {
		v := s.RandStr(10, 99)
		return "EVR-AL"+v+" Build/HUAWEIVCE-AL" + v
	} else if ratio < 80 {
		return "baiduboxapp/"+s.RandStr(10, 99)+".0."+s.RandStr(3, 9)+"."+s.RandStr(10, 99)
	} else {
		return "Nexus One Build/FRF"+s.RandStr(10, 99)
	}
}

func (s AgentSegSt) linuxSystem() string {
	v := s.RandStr(5, 10)+"."+s.RandStr(1, 10)+s.RandStr(1, 99)
	return "(Linux; U; Android "+v+"; en-us; "+s.phoneDevice()+")"
}

//后缀处理逻辑
func (s AgentSegSt) phoneSubffix() string {
	v  := s.RandStr(4, 8)
	wv := s.RandStr(533, 539)+"."+s.RandStr(1, 8)
	return "AppleWebKit/"+wv+" (KHTML, like Gecko) Version/"+v+".0 Mobile Safari/"+wv
}