package video

import (
	"feedsystem_video_go/internal/middleware"

	"github.com/gin-gonic/gin"
)

type LikeRequest struct {
	VideoID uint `json:"video_id"`
}

type LikeHandler struct {
	service *LikeService
}

func NewLikeHandler(service *LikeService) *LikeHandler {
	return &LikeHandler{service: service}
}

func (lh *LikeHandler) Like(c *gin.Context) {
	var req LikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.VideoID <= 0 {
		c.JSON(400, gin.H{"error": "video_id is required"})
		return
	}

	accountID, err := middleware.GetAccountID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	like := &Like{
		VideoID:   req.VideoID,
		AccountID: accountID,
	}
	if err := lh.service.Like(c.Request.Context(), like); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "like success"})
}

func (lh *LikeHandler) Unlike(c *gin.Context) {
	var req LikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.VideoID <= 0 {
		c.JSON(400, gin.H{"error": "video_id is required"})
		return
	}

	accountID, err := middleware.GetAccountID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	like := &Like{
		VideoID:   req.VideoID,
		AccountID: accountID,
	}
	if err := lh.service.Unlike(c.Request.Context(), like); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "unlike success"})
}

func (lh *LikeHandler) IsLiked(c *gin.Context) {
	var req LikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.VideoID <= 0 {
		c.JSON(400, gin.H{"error": "video_id is required"})
		return
	}

	accountID, err := middleware.GetAccountID(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	isLiked, err := lh.service.IsLiked(c.Request.Context(), req.VideoID, accountID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"is_liked": isLiked})
}

func (lh *LikeHandler) GetLikesCount(c *gin.Context) {
	var req LikeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.VideoID <= 0 {
		c.JSON(400, gin.H{"error": "video_id is required"})
		return
	}
	likesCount, err := lh.service.GetLikesCount(c.Request.Context(), req.VideoID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"likes_count": likesCount})
}
