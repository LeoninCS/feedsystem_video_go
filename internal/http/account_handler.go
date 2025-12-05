package http

import (
	"feedsystem_video_go/internal/account"

	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	accountService *account.AccountService
}

type CreateAccountRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateAccountResponse struct {
}

type RenameByIDRequest struct {
	ID          uint   `json:"id"`
	NewUsername string `json:"new_username"`
}

type RenameByIDResponse struct {
}

type FindByIDRequest struct {
	ID uint `json:"id"`
}

type FindByIDResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type FindByUsernameRequest struct {
	Username string `json:"username"`
}

type FindByUsernameResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type ChangePasswordRequest struct {
	Username    string `json:"username"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type ChangePasswordResponse struct {
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type LoginResponse struct {
	Token string `json:"token"`
}
type LogoutRequest struct {
	ID uint `json:"id"`
}
type LogoutResponse struct {
}

func NewAccountHandler(accountService *account.AccountService) *AccountHandler {
	return &AccountHandler{accountService: accountService}
}
func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := h.accountService.CreateAccount(&account.Account{
		Username: req.Username,
		Password: req.Password,
	}); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "account created"})
}

func (h *AccountHandler) RenameByID(c *gin.Context) {
	var req RenameByIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := h.accountService.RenameByID(req.ID, req.NewUsername); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "account renamed"})
}

func (h *AccountHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := h.accountService.ChangePassword(req.Username, req.OldPassword, req.NewPassword); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "password changed"})
}

func (h *AccountHandler) FindByID(c *gin.Context) {
	var req FindByIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if account, err := h.accountService.FindByID(req.ID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	} else {
		c.JSON(200, account)
	}
}

func (h *AccountHandler) FindByUsername(c *gin.Context) {
	var req FindByUsernameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if account, err := h.accountService.FindByUsername(req.Username); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	} else {
		c.JSON(200, account)
	}
}

func (h *AccountHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if token, err := h.accountService.Login(req.Username, req.Password); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	} else {
		c.JSON(200, LoginResponse{Token: token})
	}
}

func (h *AccountHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := h.accountService.Logout(req.ID); err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, LogoutResponse{})
}
