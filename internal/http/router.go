package http

import (
	"feedsystem_video_go/internal/account"
	"feedsystem_video_go/internal/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()

	userRepository := account.NewUserRepository(db)
	userService := account.NewUserService(userRepository)
	userHandler := NewUserHandler(userService)
	userGroup := r.Group("/user")
	{
		userGroup.POST("/register", userHandler.CreateUser)
		userGroup.POST("/rename", userHandler.RenameByID)
		userGroup.POST("/changePassword", userHandler.ChangePassword)
		userGroup.POST("/findByID", userHandler.FindByID)
		userGroup.POST("/findByUsername", userHandler.FindByUsername)
	}
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/login", userHandler.Login)
	}

	protectedAuthGroup := authGroup.Group("")
	protectedAuthGroup.Use(middleware.JWTAuth())
	{
		protectedAuthGroup.POST("/logout", userHandler.Logout)
	}

	return r
}
