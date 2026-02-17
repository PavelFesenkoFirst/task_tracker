package main

import (
	"log"
	"os"

	"github.com/PavelFesenkoFirst/task_tracker/internal/user/model"
	"github.com/PavelFesenkoFirst/task_tracker/pkg/config/mysql"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	db := mysql.NewConnection()
	err := db.AutoMigrate(&model.User{})
	if err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	port := os.Getenv("APP_PORT")
	if port == "" {
		log.Println("Error loading .env file, proceeding with system environment variables")
		port = "8080"
	}

	r.Run(":" + port)
}
