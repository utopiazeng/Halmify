package controllers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/beego/beego/v2/server/web"
)

// OfficialWebsiteController serves the public marketing website from the
// repository-level officialwebsite directory.
type OfficialWebsiteController struct {
	web.Controller
}

func (c *OfficialWebsiteController) Home() {
	c.serveFile("index.html")
}

func (c *OfficialWebsiteController) Asset() {
	name := strings.TrimSpace(c.Ctx.Input.Param(":name"))
	if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		c.Ctx.Output.SetStatus(http.StatusNotFound)
		return
	}
	c.serveFile(name)
}

func (c *OfficialWebsiteController) serveFile(name string) {
	root := filepath.Clean(filepath.Join("..", "officialwebsite"))
	path := filepath.Clean(filepath.Join(root, name))
	if !strings.HasPrefix(path, root+string(filepath.Separator)) {
		c.Ctx.Output.SetStatus(http.StatusNotFound)
		return
	}
	content, err := os.ReadFile(path)
	if err != nil {
		c.Ctx.Output.SetStatus(http.StatusNotFound)
		return
	}
	c.Ctx.Output.ContentType(filepath.Ext(path))
	_ = c.Ctx.Output.Body(content)
}
