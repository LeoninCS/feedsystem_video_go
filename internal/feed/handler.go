package feed

import (
	"time"

	"github.com/gin-gonic/gin"
)

type ListLatestRequest struct {
	Limit      int   `json:"limit"`
	LatestTime int64 `json:"latest_time"`
}

type FeedHandler struct {
	service *FeedService
}

func NewFeedHandler(service *FeedService) *FeedHandler {
	return &FeedHandler{service: service}
}

func (f *FeedHandler) ListLatest(c *gin.Context) {
	var req ListLatestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 10
	}
	ctx := c.Request.Context()
	var latestTime time.Time
	if req.LatestTime > 0 {
		latestTime = time.Unix(req.LatestTime, 0)
	}
	feeds, err := f.service.ListLatest(ctx, req.Limit, latestTime)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, feeds)
}
