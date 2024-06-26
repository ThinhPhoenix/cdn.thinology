package main

import (
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"main.go/handlers"
	"main.go/initializers"
	"main.go/repositories"
	"main.go/services"
)

func main() {
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	r.Use(cors.New(config))

	h := initializeHandlers()

	r.GET("/ping", h.Ping)
	r.POST("/send", h.SendFile)
	r.GET("/url", h.GetFileURL)
	r.GET("/drive/:id", h.DownloadFile)
	r.GET("/info", h.GetFileInfo)
	r.GET("/verify", h.CheckBotAndChat)
	r.GET("/js", func(c *gin.Context) {
		c.File("./public.js")
	})

	r.Run(":" + os.Getenv("PORT"))
}

func initializeHandlers() *handlers.Handlers {
	initializers.LoadEnvironment()

	repo := initializeRepositories()
	service := services.NewFileService(repo)
	return handlers.NewHandlers(service)
}

func initializeRepositories() repositories.FileRepository {
	return repositories.NewFileRepository()
}
