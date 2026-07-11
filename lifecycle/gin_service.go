package lifecycle

import (
	"com.mutantcat.echoes/router"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// 初始化Gin服务
func InitGin() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	ginServer := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "OPTIONS", "PUT"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Health-Info"}
	ginServer.Use(cors.New(config))
	return ginServer
}

// 启动Gin服务
func StartGin(ginServer *gin.Engine, port string) error {
	err := ginServer.Run(":" + port)
	if err != nil {
		return err
	}
	return nil
}

// 注册路由
func RegisterRouter(ginServer *gin.Engine, router ...router.RouterTemplate) error {
	for _, r := range router {
		err := r.PrepareRouter()
		if err != nil {
			return err
		}
		err = r.InitRouter(ginServer)
		if err != nil {
			return err
		}
	}
	return nil
}
