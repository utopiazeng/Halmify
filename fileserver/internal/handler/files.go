package handler

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"syscall"

	"fileserver/internal/service"
	"fileserver/internal/storage"
	"fileserver/pkg/resp"

	"github.com/gin-gonic/gin"
)

type FileHandler struct {
	svc         *service.FileService
	storageRoot string
}

func NewFileHandler(svc *service.FileService, storageRoot string) *FileHandler {
	return &FileHandler{svc: svc, storageRoot: storageRoot}
}

type checkReq struct {
	MD5      string `json:"md5"`
	FileSize uint64 `json:"file_size"`
}

func tenantID(c *gin.Context) uint64 {
	v, _ := c.Get("tenant_id")
	n, _ := v.(uint64)
	return n
}

func (h *FileHandler) Check(c *gin.Context) {
	var req checkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Err(c, http.StatusBadRequest, "invalid request")
		return
	}
	req.MD5 = strings.ToLower(strings.TrimSpace(req.MD5))
	if len(req.MD5) != 32 {
		resp.Err(c, http.StatusBadRequest, "invalid md5")
		return
	}
	exist, fileID, err := h.svc.Check(tenantID(c), req.MD5, req.FileSize)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	if exist {
		resp.OK(c, gin.H{"exist": true, "file_id": fileID})
		return
	}
	resp.OK(c, gin.H{"exist": false})
}

func (h *FileHandler) Upload(c *gin.Context) {
	f, header, err := c.Request.FormFile("file")
	if err != nil {
		resp.Err(c, http.StatusBadRequest, "missing file")
		return
	}
	defer f.Close()
	uploader := c.GetHeader("X-Uploader")
	if uploader == "" {
		uploader = "system"
	}
	result, err := h.svc.Upload(tenantID(c), uploader, header.Filename, f)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.OK(c, result)
}

func (h *FileHandler) Info(c *gin.Context) {
	fileID := strings.TrimSpace(c.Param("file_id"))
	if fileID == "" {
		resp.Err(c, http.StatusBadRequest, "missing file_id")
		return
	}
	result, err := h.svc.Info(tenantID(c), fileID)
	if err != nil {
		resp.Err(c, http.StatusNotFound, "file not found")
		return
	}
	resp.OK(c, result)
}

func (h *FileHandler) Delete(c *gin.Context) {
	fileID := strings.TrimSpace(c.Param("file_id"))
	if fileID == "" {
		resp.Err(c, http.StatusBadRequest, "missing file_id")
		return
	}
	if err := h.svc.Delete(tenantID(c), fileID); err != nil {
		resp.Err(c, http.StatusNotFound, "file not found")
		return
	}
	resp.OK(c, gin.H{"message": "deleted"})
}

func (h *FileHandler) TenantUsage(c *gin.Context) {
	result, err := h.svc.TenantUsage(tenantID(c))
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.OK(c, result)
}

func (h *FileHandler) DeleteTenantFiles(c *gin.Context) {
	result, err := h.svc.DeleteTenantFiles(tenantID(c))
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.OK(c, result)
}

func (h *FileHandler) Download(c *gin.Context) {
	fileID := strings.TrimSpace(c.Param("file_id"))
	if fileID == "" {
		resp.Err(c, http.StatusBadRequest, "missing file_id")
		return
	}
	relPath, downloadName, err := h.svc.FilePathForDownload(tenantID(c), fileID)
	if err != nil {
		resp.Err(c, http.StatusNotFound, "file not found")
		return
	}
	store := storage.New(h.storageRoot)
	abs, err := store.AbsPath(relPath)
	if err != nil {
		resp.Err(c, http.StatusNotFound, "file not found")
		return
	}
	file, err := store.OpenForRead(relPath)
	if err != nil {
		resp.Err(c, http.StatusNotFound, "file not found")
		return
	}
	defer file.Close()
	fi, err := file.Stat()
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, "stat file failed")
		return
	}
	if queryName := strings.TrimSpace(c.Query("download_name")); queryName != "" {
		downloadName = queryName
	}
	if downloadName == "" {
		downloadName = filepath.Base(abs)
	}
	ctype := mime.TypeByExtension(filepath.Ext(downloadName))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	c.Header("Content-Type", ctype)
	c.Header("Accept-Ranges", "bytes")
	c.Header("Content-Disposition", contentDisposition(downloadName))
	http.ServeContent(c.Writer, c.Request, downloadName, fi.ModTime(), file)
}

func (h *FileHandler) Health(c *gin.Context) {
	info, err := h.svc.NodeInfo()
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.OK(c, gin.H{"status": "ok", "node_id": info["node_id"]})
}

func (h *FileHandler) Space(c *gin.Context) {
	total, free, err := getDiskSpace(h.storageRoot)
	if err != nil {
		resp.Err(c, http.StatusInternalServerError, err.Error())
		return
	}
	used := total - free
	resp.OK(c, gin.H{
		"total_bytes": total,
		"free_bytes":  free,
		"used_bytes":  used,
		"total_gb":    fmt.Sprintf("%.2f", float64(total)/float64(1024*1024*1024)),
		"free_gb":     fmt.Sprintf("%.2f", float64(free)/float64(1024*1024*1024)),
		"used_gb":     fmt.Sprintf("%.2f", float64(used)/float64(1024*1024*1024)),
	})
}

func getDiskSpace(path string) (uint64, uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	return total, free, nil
}

func contentDisposition(filename string) string {
	filename = strings.ReplaceAll(filename, "\r", "")
	filename = strings.ReplaceAll(filename, "\n", "")
	ascii := make([]rune, 0, len(filename))
	for _, r := range filename {
		if r >= 32 && r <= 126 && r != '"' && r != '\\' {
			ascii = append(ascii, r)
		}
	}
	fallback := string(ascii)
	if fallback == "" {
		fallback = "download"
	}
	return fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", fallback, url.PathEscape(filename))
}
