package http

import (
	"feedsystem_video_go/internal/account"
	"feedsystem_video_go/internal/feed"
	"feedsystem_video_go/internal/middleware"
	"feedsystem_video_go/internal/video"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetRouter(db *gorm.DB) *gin.Engine {
	r := gin.Default()
	// account
	accountRepository := account.NewAccountRepository(db)
	accountService := account.NewAccountService(accountRepository)
	accountHandler := account.NewAccountHandler(accountService)
	accountGroup := r.Group("/account")
	{
		accountGroup.POST("/register", accountHandler.CreateAccount)
		accountGroup.POST("/login", accountHandler.Login)
		accountGroup.POST("/changePassword", accountHandler.ChangePassword)
		accountGroup.POST("/findByID", accountHandler.FindByID)
		accountGroup.POST("/findByUsername", accountHandler.FindByUsername)
	}
	protectedAccountGroup := accountGroup.Group("")
	protectedAccountGroup.Use(middleware.JWTAuth(accountRepository))
	{
		protectedAccountGroup.POST("/logout", accountHandler.Logout)
		protectedAccountGroup.POST("/rename", accountHandler.RenameByID)
	}
	// video
	videoRepository := video.NewVideoRepository(db)
	videoService := video.NewVideoService(videoRepository)
	videoHandler := video.NewVideoHandler(videoService)
	videoGroup := r.Group("/video")
	{
		videoGroup.POST("/listByAuthorID", videoHandler.ListByAuthorID)
		videoGroup.POST("/getDetail", videoHandler.GetDetail)
	}
	protectedVideoGroup := videoGroup.Group("")
	protectedVideoGroup.Use(middleware.JWTAuth(accountRepository))
	{
		protectedVideoGroup.POST("/publish", videoHandler.PublishVideo)
	}

	// feed
	feedRepository := feed.NewFeedRepository(db)
	feedService := feed.NewFeedService(feedRepository)
	feedHandler := feed.NewFeedHandler(feedService)
	feedGroup := r.Group("/feed")
	{
		feedGroup.POST("/listLatest", feedHandler.ListLatest)
	}
	return r
}
