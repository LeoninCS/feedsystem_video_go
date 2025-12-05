package http

import (
	"feedsystem_video_go/internal/account"
	"feedsystem_video_go/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	accountRepository := account.NewAccountRepository(db)
	accountService := account.NewAccountService(accountRepository)
	accountHandler := NewAccountHandler(accountService)
	accountGroup := r.Group("/account")
	{
		accountGroup.POST("/register", accountHandler.CreateAccount)
		accountGroup.POST("/rename", accountHandler.RenameByID)
		accountGroup.POST("/changePassword", accountHandler.ChangePassword)
		accountGroup.POST("/findByID", accountHandler.FindByID)
		accountGroup.POST("/findByUsername", accountHandler.FindByUsername)
	}
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", accountHandler.Login)
	}

	protectedAuthGroup := authGroup.Group("")
	protectedAuthGroup.Use(middleware.JWTAuth())
	{
		protectedAuthGroup.POST("/logout", accountHandler.Logout)
	}

	return r
}
