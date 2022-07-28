package core

import (
	"github.com/leicc520/go-orm/log"
)

//启动执行APP业务处理逻辑
func (app *Application) Start() {
	if len(app.handler) > 0 {
		for _, handle := range app.handler {
			handle(app.app)
		}
	}
	httpStr, wsStr, isSsl := app.httpProto()
	log.Write(-1, "=======================start app window=====================")
	log.Write(-1, "===http server{", httpStr, "} jwt={", string(gJwtSecret), "}")
	if len(wsStr) > 1 {
		log.Write(-1, "===websocket server{", wsStr, "}")
	}
	log.Write(-1, "============================================================")
	if isSsl {//针对https 热更新的处理逻辑
		if err := app.app.RunTLS(app.config.Host, app.config.CertFile, app.config.KeyFile); err != nil {
			log.Write(log.FATAL, "start app failed:"+err.Error())
		}
	} else {//针对http 热更新的处理逻辑
		if err := app.app.Run(app.config.Host); err != nil {
			log.Write(log.FATAL, "start app failed:"+err.Error())
		}
	}
}

