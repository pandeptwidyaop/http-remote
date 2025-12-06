package handlers

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"image/png"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pandeptwidyaop/http-remote/internal/models"
	"github.com/pandeptwidyaop/http-remote/internal/services"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TwoFAHandler handles two-factor authentication operations
type TwoFAHandler struct {
	authService  *services.AuthService
	auditService *services.AuditService
}

// NewTwoFAHandler creates a new TwoFAHandler instance
func NewTwoFAHandler(authService *services.AuthService, auditService *services.AuditService) *TwoFAHandler {
	return &TwoFAHandler{
		authService:  authService,
		auditService: auditService,
	}
}

// SetupTOTPRequest represents the request to setup 2FA.
type SetupTOTPRequest struct {
	Code string `json:"code" binding:"required"`
}

// SetupTOTPResponse represents the response for 2FA setup
type SetupTOTPResponse struct {
	Secret    string `json:"secret"`
	QRCodeURL string `json:"qr_code_url"`
}

// VerifyTOTPRequest represents the request to verify TOTP code
type VerifyTOTPRequest struct {
	Code string `json:"code" binding:"required"`
}

// GenerateSecret generates a new TOTP secret for the user
func (h *TwoFAHandler) GenerateSecret(c *gin.Context) {
	userObj, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}

	// Generate TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "HTTP Remote",
		AccountName: user.Username,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate secret"})
		return
	}

	// Store secret (but don't enable yet)
	if err := h.authService.SetTOTPSecret(user.ID, key.Secret()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store secret"})
		return
	}

	c.JSON(http.StatusOK, SetupTOTPResponse{
		Secret:    key.Secret(),
		QRCodeURL: key.URL(),
	})
}

// GetQRCode generates and returns QR code image
func (h *TwoFAHandler) GetQRCode(c *gin.Context) {
	userObj, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}

	if user.TOTPSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA not setup. Call /generate-secret first"})
		return
	}

	// Create OTP key from existing secret
	// Use otp.NewKeyFromURL to reconstruct the key properly
	url := fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		"HTTP Remote",
		user.Username,
		user.TOTPSecret,
		"HTTP Remote",
	)

	key, err := otp.NewKeyFromURL(url)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create key from secret"})
		return
	}

	// Generate QR code image
	img, err := key.Image(200, 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate QR code"})
		return
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode QR code"})
		return
	}

	// Return as base64 data URL
	base64Img := base64.StdEncoding.EncodeToString(buf.Bytes())
	c.JSON(http.StatusOK, gin.H{
		"qr_code": "data:image/png;base64," + base64Img,
	})
}

// EnableTOTP enables 2FA after verifying the TOTP code
func (h *TwoFAHandler) EnableTOTP(c *gin.Context) {
	userObj, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}

	var req VerifyTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if user.TOTPSecret == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA not setup. Call /generate-secret first"})
		return
	}

	// Verify the code
	valid := totp.Validate(req.Code, user.TOTPSecret)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code"})
		return
	}

	// Enable 2FA
	if err := h.authService.EnableTOTP(user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable 2FA"})
		return
	}

	// Generate and store backup codes
	backupCodes := generateBackupCodes()
	if err := h.authService.SetBackupCodes(user.ID, backupCodes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate backup codes"})
		return
	}

	// Audit log
	if h.auditService != nil {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &user.ID,
			Username:     user.Username,
			Action:       "enable_2fa",
			ResourceType: "auth",
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "2FA enabled successfully",
		"enabled":      true,
		"backup_codes": backupCodes,
	})
}

// DisableTOTP disables 2FA after verifying the TOTP code
func (h *TwoFAHandler) DisableTOTP(c *gin.Context) {
	userObj, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}

	var req VerifyTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !user.TOTPEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA is not enabled"})
		return
	}

	// Verify the code
	valid := totp.Validate(req.Code, user.TOTPSecret)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code"})
		return
	}

	// Disable 2FA
	if err := h.authService.DisableTOTP(user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable 2FA"})
		return
	}

	// Audit log
	if h.auditService != nil {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &user.ID,
			Username:     user.Username,
			Action:       "disable_2fa",
			ResourceType: "auth",
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "2FA disabled successfully",
		"enabled": false,
	})
}

// GetStatus returns the current 2FA status
func (h *TwoFAHandler) GetStatus(c *gin.Context) {
	userObj, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"enabled": user.TOTPEnabled,
		"setup":   user.TOTPSecret != "",
	})
}

// generateBackupCodes generates 10 random backup codes
func generateBackupCodes() []string {
	codes := make([]string, 10)
	for i := 0; i < 10; i++ {
		code := make([]byte, 4)
		_, _ = rand.Read(code)
		codes[i] = fmt.Sprintf("%08x", code)
	}
	return codes
}

// GetBackupCodes returns the count of remaining backup codes (not the codes themselves for security)
func (h *TwoFAHandler) GetBackupCodes(c *gin.Context) {
	userObj, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}

	if !user.TOTPEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA not enabled"})
		return
	}

	count, err := h.authService.GetBackupCodesCount(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get backup codes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"remaining_codes": count,
		"message":         "For security, backup codes are only shown when generated. If you need new codes, regenerate them.",
	})
}

// RegenerateBackupCodes regenerates backup codes for the user
func (h *TwoFAHandler) RegenerateBackupCodes(c *gin.Context) {
	userObj, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, ok := userObj.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user context"})
		return
	}

	if !user.TOTPEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA not enabled"})
		return
	}

	codes := generateBackupCodes()

	// Store hashed backup codes
	if err := h.authService.SetBackupCodes(user.ID, codes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store backup codes"})
		return
	}

	// Audit log
	if h.auditService != nil {
		_ = h.auditService.Log(services.AuditLog{
			UserID:       &user.ID,
			Username:     user.Username,
			Action:       "regenerate_backup_codes",
			ResourceType: "auth",
			IPAddress:    c.ClientIP(),
			UserAgent:    c.GetHeader("User-Agent"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"codes":   codes,
		"message": "Backup codes regenerated. Save these codes in a secure location. Each code can only be used once.",
	})
}
