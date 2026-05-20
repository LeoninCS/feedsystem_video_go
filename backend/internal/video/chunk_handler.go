package video

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"feedsystem_video_go/internal/middleware/jwt"
	rediscache "feedsystem_video_go/internal/middleware/redis"

	"github.com/gin-gonic/gin"
)

const sessionTTL = 24 * time.Hour

var errChunkCacheUnavailable = errors.New("chunk upload requires redis")

type ChunkUploadHandler struct {
	cache *rediscache.Client
}

func NewChunkUploadHandler(cache *rediscache.Client) *ChunkUploadHandler {
	return &ChunkUploadHandler{cache: cache}
}

func (h *ChunkUploadHandler) sessionKey(uploadID string) string {
	return h.cache.Key("chunk_upload:%s", uploadID)
}

func (h *ChunkUploadHandler) hashKey(accountID uint, fileHash string) string {
	return h.cache.Key("chunk_upload_hash:%d:%s", accountID, fileHash)
}

func (h *ChunkUploadHandler) getSession(ctx *gin.Context, uploadID string) (*ChunkUploadSession, error) {
	if h.cache == nil {
		return nil, errChunkCacheUnavailable
	}
	b, err := h.cache.GetBytes(ctx.Request.Context(), h.sessionKey(uploadID))
	if err != nil {
		return nil, fmt.Errorf("upload session not found")
	}
	var s ChunkUploadSession
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("invalid session data")
	}
	return &s, nil
}

func (h *ChunkUploadHandler) saveSession(ctx *gin.Context, s *ChunkUploadSession) error {
	if h.cache == nil {
		return errChunkCacheUnavailable
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return h.cache.SetBytes(ctx.Request.Context(), h.sessionKey(s.UploadID), b, sessionTTL)
}

func (h *ChunkUploadHandler) InitChunkUpload(c *gin.Context) {
	if h.cache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": errChunkCacheUnavailable.Error()})
		return
	}

	var req InitChunkUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	const maxSize = 200 << 20
	if req.FileSize > maxSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size exceeds 200MB limit"})
		return
	}

	// Check for existing session (resume)
	hashKey := h.hashKey(accountID, req.FileHash)
	existingID, err := h.cache.GetBytes(c.Request.Context(), hashKey)
	if err == nil && len(existingID) > 0 {
		session, sessErr := h.getSession(c, string(existingID))
		if sessErr == nil {
			// Refresh TTL on resume
			_ = h.cache.SetBytes(c.Request.Context(), hashKey, existingID, sessionTTL)
			_ = h.saveSession(c, session)
			c.JSON(http.StatusOK, gin.H{
				"upload_id":       session.UploadID,
				"uploaded_chunks": session.UploadedChunks(),
			})
			return
		}
	}

	id, _ := randHex(16)
	uploadID := id + fmt.Sprintf("%d", time.Now().UnixNano())
	session := &ChunkUploadSession{
		UploadID:     uploadID,
		AccountID:    accountID,
		Filename:     req.Filename,
		FileSize:     req.FileSize,
		ChunkSize:    req.ChunkSize,
		TotalChunks:  req.TotalChunks,
		FileHash:     req.FileHash,
		UploadedBits: make([]bool, req.TotalChunks),
	}

	if err := h.saveSession(c, session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	if err := h.cache.SetBytes(c.Request.Context(), hashKey, []byte(uploadID), sessionTTL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_id":       uploadID,
		"uploaded_chunks": []int{},
	})
}

func (h *ChunkUploadHandler) UploadChunk(c *gin.Context) {
	if h.cache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": errChunkCacheUnavailable.Error()})
		return
	}

	var req UploadChunkRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.getSession(c, req.UploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if session.AccountID != accountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if req.ChunkIndex < 0 || req.ChunkIndex >= session.TotalChunks {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid chunk_index"})
		return
	}

	if session.UploadedBits[req.ChunkIndex] {
		c.JSON(http.StatusOK, gin.H{"chunk_index": req.ChunkIndex})
		return
	}

	f, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}

	chunkFile, err := f.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read chunk"})
		return
	}
	defer chunkFile.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, chunkFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash chunk"})
		return
	}
	actualHash := fmt.Sprintf("%x", hash.Sum(nil))

	if actualHash != req.ChunkHash {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chunk hash mismatch", "expected": req.ChunkHash, "actual": actualHash})
		return
	}

	tmpDir := filepath.Join(".run", "uploads", "tmp", req.UploadID)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp dir"})
		return
	}

	chunkPath := filepath.Join(tmpDir, fmt.Sprintf("%d", req.ChunkIndex))
	if _, seekErr := chunkFile.Seek(0, io.SeekStart); seekErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read chunk"})
		return
	}

	dst, err := os.Create(chunkPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save chunk"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, chunkFile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save chunk"})
		return
	}

	session.UploadedBits[req.ChunkIndex] = true
	if err := h.saveSession(c, session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"chunk_index": req.ChunkIndex})
}

func (h *ChunkUploadHandler) ChunkStatus(c *gin.Context) {
	if h.cache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": errChunkCacheUnavailable.Error()})
		return
	}

	var req ChunkStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.getSession(c, req.UploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if session.AccountID != accountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_id":       session.UploadID,
		"uploaded_chunks": session.UploadedChunks(),
		"total_chunks":    session.TotalChunks,
	})
}

func (h *ChunkUploadHandler) CompleteChunkUpload(c *gin.Context) {
	if h.cache == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": errChunkCacheUnavailable.Error()})
		return
	}

	var req CompleteChunkUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	session, err := h.getSession(c, req.UploadID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	accountID, err := jwt.GetAccountID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if session.AccountID != accountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if !session.IsComplete() {
		missing := 0
		for _, uploaded := range session.UploadedBits {
			if !uploaded {
				missing++
				if missing > 5 {
					missing = 5
					break
				}
			}
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     "not all chunks uploaded",
			"missing":   missing,
			"completed": len(session.UploadedChunks()),
			"total":     session.TotalChunks,
		})
		return
	}

	date := time.Now().Format("20060102")
	relDir := filepath.Join("videos", fmt.Sprintf("%d", accountID), date)
	root := filepath.Join(".run", "uploads")
	absDir := filepath.Join(root, relDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create output dir"})
		return
	}

	filename, err := randHex(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate filename"})
		return
	}
	finalPath := filepath.Join(absDir, filename+".mp4")

	finalFile, err := os.Create(finalPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create final file"})
		return
	}
	defer finalFile.Close()

	tmpDir := filepath.Join(".run", "uploads", "tmp", req.UploadID)
	for i := 0; i < session.TotalChunks; i++ {
		chunkPath := filepath.Join(tmpDir, fmt.Sprintf("%d", i))
		cf, err := os.Open(chunkPath)
		if err != nil {
			finalFile.Close()
			os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("chunk %d missing", i)})
			return
		}
		_, err = io.Copy(finalFile, cf)
		cf.Close()
		if err != nil {
			finalFile.Close()
			os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to merge chunks"})
			return
		}
	}
	finalFile.Close()

	// Clean up temp chunks
	_ = os.RemoveAll(tmpDir)

	// Clean up Redis session
	_ = h.cache.Del(c.Request.Context(), h.sessionKey(req.UploadID))
	_ = h.cache.Del(c.Request.Context(), h.hashKey(accountID, session.FileHash))

	urlPath := fmt.Sprintf("/static/videos/%d/%s/%s.mp4", accountID, date, filename)
	playURL := buildAbsoluteURL(c, urlPath)

	c.JSON(http.StatusOK, gin.H{
		"url":      playURL,
		"play_url": playURL,
	})
}
