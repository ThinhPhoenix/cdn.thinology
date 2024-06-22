package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"main.go/models"
	"main.go/services"
)

type Handlers struct {
	service services.FileService
}

var (
	secureURLs          = make(map[string]string)
	secureURLExtensions = make(map[string]string)
	mu                  sync.Mutex
)

func NewHandlers(service services.FileService) *Handlers {
	return &Handlers{service: service}
}

func (h *Handlers) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func (h *Handlers) SendFile(c *gin.Context) {
	botToken := c.PostForm("bot_token")
	chatID := c.PostForm("chat_id")

	var fileID string
	var fileURL string
	var fileSize int
	var err error
	var fileExt string

	fileHeader, err := c.FormFile("document")
	if err == nil {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
			return
		}
		defer file.Close()

		fileID, err = h.service.SendFile(botToken, chatID, file, fileHeader.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to send document: %v", err)})
			return
		}

		fileURL, fileSize, err = h.service.GetFileInfo(botToken, fileID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get file info: %v", err)})
			return
		}

		fileExt = filepath.Ext(fileHeader.Filename)
		if fileExt != "" {
			fileExt = fileExt[1:] // Remove the leading dot
		}
	} else {
		fileURL = c.PostForm("document")
		fileSize = 0

		isFile, contentType, contentLength, err := isURLFile(fileURL)
		_ = contentLength

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to check URL: %v", err)})
			return
		}

		if !isFile {
			c.JSON(http.StatusBadRequest, gin.H{"error": "URL does not point to a file"})
			return
		}

		fileExt = getExtensionFromContentType(contentType)
	}

	scheme := c.Request.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
	}

	secureID := generateSecureURL(fileURL, fileExt)
	secureURL := fmt.Sprintf("%s://%s/drive/%s", scheme, c.Request.Host, secureID)

	response := models.Response{
		Success: true,
		Message: "Upload file successfully!",
		Data: models.FileData{
			ID:        fileID,
			URL:       fileURL,
			SecureURL: secureURL,
			Bytes:     fileSize,
			Format:    fileExt,
		},
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handlers) GetFileURL(c *gin.Context) {
	botToken := c.Query("bot_token")
	fileID := c.Query("file_id")

	fileURL, _, err := h.service.GetFileInfo(botToken, fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get file info: %v", err)})
		return
	}

	scheme := c.Request.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
	}

	fileExt := filepath.Ext(fileURL)
	if fileExt != "" {
		fileExt = fileExt[1:] // Remove the leading dot
	}

	secureID := generateSecureURL(fileURL, fileExt)
	secureURL := fmt.Sprintf("%s://%s/drive/%s", scheme, c.Request.Host, secureID)

	response := models.Response{
		Success: true,
		Message: "File URL retrieved successfully!",
		Data: models.FileData{
			URL:       fileURL,
			SecureURL: secureURL,
		},
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handlers) DownloadFile(c *gin.Context) {
	secureID := c.Param("id")

	fileURL, exists := getActualURL(secureID)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	resp, err := http.Get(fileURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch file"})
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	mu.Lock()
	originalExtension := secureURLExtensions[secureID]
	mu.Unlock()

	if originalExtension == "" {
		originalExtension = getExtensionFromContentType(contentType)
	}

	filename := secureID
	if originalExtension != "" {
		filename += "." + originalExtension
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

func (h *Handlers) GetFileInfo(c *gin.Context) {
	botToken := c.Query("bot_token")
	fileID := c.Query("file_id")

	fileURL, fileSize, err := h.service.GetFileInfo(botToken, fileID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("Failed to get file info: %v", err),
		})
		return
	}

	scheme := c.Request.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
	}

	fileExt := filepath.Ext(fileURL)
	if fileExt != "" {
		fileExt = fileExt[1:] // Remove the leading dot
	}

	secureID := generateSecureURL(fileURL, fileExt)
	secureURL := fmt.Sprintf("%s://%s/drive/%s", scheme, c.Request.Host, secureID)

	response := models.Response{
		Success: true,
		Message: "Get file information successfully!",
		Data: models.FileData{
			ID:        fileID,
			URL:       fileURL,
			SecureURL: secureURL,
			Bytes:     fileSize,
			Format:    fileExt,
		},
	}

	c.JSON(http.StatusOK, response)
}

func generateSecureURL(fileURL, originalExtension string) string {
	id := uuid.New().String()
	mu.Lock()
	defer mu.Unlock()
	secureURLs[id] = fileURL
	secureURLExtensions[id] = originalExtension
	return id
}

func getActualURL(secureID string) (string, bool) {
	mu.Lock()
	defer mu.Unlock()
	fileURL, exists := secureURLs[secureID]
	return fileURL, exists
}

func getExtensionFromContentType(contentType string) string {
	switch contentType {
	case "application/zip":
		return "zip"
	case "application/x-7z-compressed":
		return "7z"
	case "application/pdf":
		return "pdf"
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	// Add more mappings as needed
	default:
		return ""
	}
}

func isURLFile(url string) (bool, string, int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return false, "", 0, err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	contentLength := resp.ContentLength

	isFile := strings.HasPrefix(contentType, "image/") ||
		strings.HasPrefix(contentType, "application/") ||
		strings.HasPrefix(contentType, "video/") ||
		strings.HasPrefix(contentType, "audio/")

	return isFile, contentType, contentLength, nil
}

func (h *Handlers) CheckBotAndChat(c *gin.Context) {
	botToken := c.Query("bot_token")
	chatID := c.Query("chat_id")

	botInfo, chatInfo, botInChat, botIsAdmin, err := h.service.CheckBotAndChat(botToken, chatID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("Failed to check bot and chat info: %v", err),
		})
		return
	}

	response := models.Response{
		Success: true,
		Message: "Bot and chat information retrieved successfully!",
		Data: gin.H{
			"bot_info":     botInfo,
			"chat_info":    chatInfo,
			"bot_in_chat":  botInChat,
			"bot_is_admin": botIsAdmin,
		},
	}

	c.JSON(http.StatusOK, response)
}
