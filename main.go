package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"main.go/initializers"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type FileData struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	SecureURL string `json:"secure_url,omitempty"`
	Bytes     int    `json:"bytes"`
	Format    string `json:"format"`
}

type UrlData struct {
	URL       string `json:"url"`
	SecureURL string `json:"secure_url"`
}

var (
	secureURLs          = make(map[string]string)
	secureURLExtensions = make(map[string]string)
	mu                  sync.Mutex
)

func generateSecureURL(fileURL, originalExtension string) string {
	id := uuid.New().String()
	mu.Lock()
	secureURLs[id] = fileURL
	secureURLExtensions[id] = originalExtension
	mu.Unlock()
	return id
}

func isURLFile(url string) (bool, string, int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return false, "", 0, err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	contentLength := resp.ContentLength

	isFile := strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "application/") || strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/")

	return isFile, contentType, contentLength, nil
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

func getActualURL(secureID string) (string, bool) {
	mu.Lock()
	defer mu.Unlock()
	fileURL, exists := secureURLs[secureID]
	return fileURL, exists
}

func init() {
	initializers.LoadEnvironment()
}

func main() {
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	r.Use(cors.New(config))

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Hello World!",
		})
	})

	r.POST("/send", func(c *gin.Context) {
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

			fileID, err = sendDocument(botToken, chatID, file, fileHeader.Filename)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to send document: %v", err)})
				return
			}

			fileURL, fileSize, err = getFileInfo(botToken, fileID)
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
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to check URL: %v", err)})
				return
			}

			if !isFile {
				c.JSON(http.StatusBadRequest, gin.H{"error": "URL does not point to a file"})
				return
			}

			fmt.Printf("Content Type: %s, Content Length: %d\n", contentType, contentLength)

			fileExt = filepath.Ext(fileURL)
			if fileExt != "" {
				fileExt = fileExt[1:] // Remove the leading dot
			}
		}

		scheme := c.Request.Header.Get("X-Forwarded-Proto")
		if scheme == "" {
			scheme = "http"
		}

		secureID := generateSecureURL(fileURL, fileExt)
		secureURL := fmt.Sprintf("%s://%s/drive/%s", scheme, c.Request.Host, secureID)

		fileData := FileData{
			ID:     fileID,
			URL:    fileURL,
			SecureURL: secureURL,
			Bytes:  fileSize,
			Format: fileExt,
		}

		response := Response{
			Success: true,
			Message: "Upload file successfully!",
			Data:    fileData,
		}

		c.JSON(http.StatusOK, response)
	})

	r.GET("/url", func(c *gin.Context) {
		botToken := c.Query("bot_token")
		fileID := c.Query("file_id")

		fileURL, _, err := getFileInfo(botToken, fileID)
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

		urlData := UrlData{
			URL:       fileURL,
			SecureURL: secureURL,
		}

		response := Response{
			Success: true,
			Message: "File URL retrieved successfully!",
			Data:    urlData,
		}

		c.JSON(http.StatusOK, response)
	})

	r.GET("/drive/:id", func(c *gin.Context) {
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
	})

	r.GET("/info", func(c *gin.Context) {
		botToken := c.Query("bot_token")
		fileID := c.Query("file_id")

		fileURL, fileSize, err := getFileInfo(botToken, fileID)
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

		fileData := FileData{
			ID:        fileID,
			URL:       fileURL,
			SecureURL: secureURL,
			Bytes:     fileSize,
			Format:    fileExt,
		}

		response := Response{
			Success: true,
			Message: "Get file information successfully!",
			Data:    fileData,
		}

		c.JSON(http.StatusOK, response)
	})

	r.Run(":" + os.Getenv("PORT"))
}

func sendDocument(botToken, chatID string, file io.Reader, fileName string) (string, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", botToken)

	secureID := uuid.New().String()
	fileExt := filepath.Ext(fileName)
	newFileName := secureID + fileExt

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	_ = writer.WriteField("chat_id", chatID)

	part, err := writer.CreateFormFile("document", newFileName)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %v", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file contents: %v", err)
	}

	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %v", err)
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	var sendDocResp struct {
		Ok     bool `json:"ok"`
		Result struct {
			Document struct {
				FileID string `json:"file_id"`
			} `json:"document"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&sendDocResp); err != nil {
		return "", fmt.Errorf("failed to decode JSON response: %v", err)
	}

	if !sendDocResp.Ok {
		return "", fmt.Errorf("telegram API returned not ok status")
	}

	fileID := sendDocResp.Result.Document.FileID

	return fileID, nil
}

func getFileInfo(botToken, fileID string) (string, int, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", botToken, fileID)

	resp, err := http.Get(url)
	if err != nil {
		return "", 0, fmt.Errorf("failed to send GET request: %v", err)
	}
	defer resp.Body.Close()

	var getFileResp struct {
		Ok     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
			FileSize int    `json:"file_size"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&getFileResp); err != nil {
		return "", 0, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	if !getFileResp.Ok {
		return "", 0, fmt.Errorf("telegram API returned not ok status: %s", resp.Status)
	}

	finalURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botToken, getFileResp.Result.FilePath)

	return finalURL, getFileResp.Result.FileSize, nil
}