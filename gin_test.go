package core

import (
	"github.com/gin-gonic/gin"
	"github.com/leicc520/go-orm"
	"testing"
)

func TestAPP(t *testing.T) {
	config := AppConfigSt{Host: "127.0.0.1:8081", Name: "go.test.srv", Domain: "127.0.0.1:8081"}
	NewApp(&config).RegHandler(func(c *gin.Engine) {
		c.GET("/demo", func(context *gin.Context) {
			context.JSON(200, orm.SqlMap{"demo":"test"})
		})
	}).Start()
}
