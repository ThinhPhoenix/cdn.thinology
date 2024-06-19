package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"main.go/initializers"
)

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type FileData struct {
	FileID   string `json:"file_id"`
	FileURL  string `json:"file_url"`
	FileSize int    `json:"file_size"`
}

type UrlData struct {
	FileURL  string `json:"file_url"`
}

func init() {
	initializers.LoadEnvironment()
}

func main() {
	r := gin.Default()

	// CORS middleware
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	r.Use(cors.New(config))

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Hello World!",
		})
	})

	r.POST("/upload", func(c *gin.Context) {
		botToken := c.PostForm("bot_token")
		chatID := c.PostForm("chat_id")

		fileHeader, err := c.FormFile("document")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File upload failed"})
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
			return
		}
		defer file.Close()

		fileID, err := sendDocument(botToken, chatID, file, fileHeader.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to send document: %v", err)})
			return
		}

		fileURL, fileSize, err := getFileInfo(botToken, fileID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get file info: %v", err)})
			return
		}

		fileData := FileData{
			FileID:   fileID,
			FileURL:  fileURL,
			FileSize: fileSize,
		}

		response := Response{
			Success: true,
			Message: "File uploaded successfully",
			Data:    fileData,
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal JSON response"})
			return
		}

		c.Header("Content-Type", "application/json")
		c.Status(http.StatusOK)
		c.Writer.Write(jsonResponse)
	})

	r.GET("/url", func(c *gin.Context) {
		botToken := c.Query("bot_token")
		fileID := c.Query("file_id")

		fileURL, fileSize, err := getFileInfo(botToken, fileID)
		_ = fileSize
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to get file info: %v", err)})
			return
		}

		urlData := UrlData {
			FileURL: fileURL,
		}

		response := Response{
			Success: true,
			Message: "File URL retrieved successfully",
			Data:    urlData,
		}

		jsonResponse, err := json.Marshal(response)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal JSON response"})
			return
		}

		c.Header("Content-Type", "application/json")
		c.Status(http.StatusOK)
		c.Writer.Write(jsonResponse)
	})

	r.Run(":" + os.Getenv("PORT"))
}

// Function to send a document using Telegram Bot API sendDocument method
func sendDocument(botToken, chatID string, file io.Reader, fileName string) (string, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", botToken)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add chat_id field
	_ = writer.WriteField("chat_id", chatID)

	// Add document file
	part, err := writer.CreateFormFile("document", fileName)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %v", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file contents: %v", err)
	}

	// Close multipart writer
	err = writer.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close multipart writer: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
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
		return "", fmt.Errorf("!Telegram API returned not ok status")
	}

	fileID := sendDocResp.Result.Document.FileID

	return fileID, nil
}

// Function to get file information using Telegram Bot API getFile method
func getFileInfo(botToken, fileID string) (string, int, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", botToken, fileID)

	resp, err := http.Get(url)
	if err != nil {
		return "", 0, fmt.Errorf("failed to send GET request: %v", err)
	}
	defer resp.Body.Close()

	// Read response body
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
		return "", 0, fmt.Errorf("!Telegram API returned not ok status")
	}

	finalURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botToken, getFileResp.Result.FilePath)

	return finalURL, getFileResp.Result.FileSize, nil
}
