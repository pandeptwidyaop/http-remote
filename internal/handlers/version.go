package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pandeptwidyaop/http-remote/internal/upgrade"
	"github.com/pandeptwidyaop/http-remote/internal/version"
)

// VersionHandler handles version information requests.
type VersionHandler struct{}

// NewVersionHandler creates a new VersionHandler instance.
func NewVersionHandler() *VersionHandler {
	return &VersionHandler{}
}

// Get returns version information.
func (h *VersionHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, version.Info())
}

// CheckUpdate checks if a new version is available
// GET /api/version/check
func (h *VersionHandler) CheckUpdate(c *gin.Context) {
	release, err := upgrade.CheckLatestVersion()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"current":          version.Version,
			"latest":           "",
			"update_available": false,
			"error":            err.Error(),
		})
		return
	}

	needsUpgrade := upgrade.NeedsUpgrade(release.TagName)

	c.JSON(http.StatusOK, gin.H{
		"current":          version.Version,
		"latest":           release.TagName,
		"update_available": needsUpgrade,
		"release_name":     release.Name,
	})
}
