package management

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed assets/requests.html
var requestsPageAsset embed.FS

// requestsPagePath is the embed.FS path to the static request-logs page.
const requestsPagePath = "assets/requests.html"

// ServeRequestsPage returns the embedded /requests.html static page.
// The page is served unauthenticated (like /management.html); the page
// itself prompts the user for the management password and stores it in
// browser localStorage to use as a Bearer token for subsequent
// /v0/management/* calls.
func (h *Handler) ServeRequestsPage(c *gin.Context) {
	data, err := requestsPageAsset.ReadFile(requestsPagePath)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}
