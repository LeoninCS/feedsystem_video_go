package http

import (
	"feedsystem_video_go/internal/video"
	"time"

	"github.com/gin-gonic/gin"
)

type VideoHandler struct {
	service *video.VideoService
}

func NewVideoHandler(service *video.VideoService) *VideoHandler {
	return &VideoHandler{service: service}
}

type PublishVideoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PlayURL     string `json:"play_url"`
}

type ListByAuthorIDRequest struct {
	AuthorID uint `json:"author_id"`
}

type GetDetailRequest struct {
	ID uint `json:"id"`
}

func (vh *VideoHandler) PublishVideo(c *gin.Context) {
	var req PublishVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Title == "" {
		c.JSON(400, gin.H{"error": "title is required"})
		return
	}
	if req.PlayURL == "" {
		c.JSON(400, gin.H{"error": "play url is required"})
		return
	}
	uidValue, exists := c.Get("accountID")
	if !exists {
		c.JSON(400, gin.H{"error": "accountID not found"})
		return
	}
	authorID, ok := uidValue.(uint)
	if !ok {
		c.JSON(400, gin.H{"error": "accountID has invalid type"})
		return
	}
	video := &video.Video{
		AuthorID:    authorID,
		Title:       req.Title,
		Description: req.Description,
		PlayURL:     req.PlayURL,
		CreateTime:  time.Now(),
	}
	if err := vh.service.Publish(c, video); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, video)
}

func (vh *VideoHandler) ListByAuthorID(c *gin.Context) {
	var req ListByAuthorIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	videos, err := vh.service.ListByAuthorID(c, req.AuthorID)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, videos)
}

func (vh *VideoHandler) ListLatest(c *gin.Context) {
	videos, err := vh.service.ListLatest(c)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, videos)
}

func (vh *VideoHandler) GetDetail(c *gin.Context) {
	var req GetDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	video, err := vh.service.GetDetail(c, req.ID)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, video)
}
