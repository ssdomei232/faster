package main

import (
	"github.com/gin-gonic/gin"
	"github.com/ssdomei232/faster/internal/booster"
)

func main() {
	r := gin.Default()

	r.GET("/b/*url", booster.HandleHttpRequest)

	r.Run(":8089")
}
