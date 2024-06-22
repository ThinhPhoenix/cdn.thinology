package repositories

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
)

type FileRepository interface {
	SendDocument(botToken, chatID string, file io.Reader, fileName string) (string, error)
	GetFileInfo(botToken, fileID string) (string, int, error)
	CheckBotAndChat(botToken, chatID string) (botInfo, chatInfo interface{}, botInChat, botIsAdmin bool, err error)
}

type fileRepository struct{}

func NewFileRepository() FileRepository {
	return &fileRepository{}
}

func (r *fileRepository) SendDocument(botToken, chatID string, file io.Reader, fileName string) (string, error) {
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

func (r *fileRepository) GetFileInfo(botToken, fileID string) (string, int, error) {
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

func (r *fileRepository) CheckBotAndChat(botToken, chatID string) (botInfo, chatInfo interface{}, botInChat, botIsAdmin bool, err error) {
	// Get bot info
	botInfo, err = r.getBotInfo(botToken)
	if err != nil {
		return nil, nil, false, false, fmt.Errorf("failed to get bot info: %v", err)
	}

	// Get chat info
	chatInfo, err = r.getChatInfo(botToken, chatID)
	if err != nil {
		return nil, nil, false, false, fmt.Errorf("failed to get chat info: %v", err)
	}

	// Check if bot is in chat and if it's an admin
	botInChat, botIsAdmin, err = r.checkBotStatus(botToken, chatID, botInfo.(map[string]interface{})["id"].(float64))
	if err != nil {
		return nil, nil, false, false, fmt.Errorf("failed to check bot status: %v", err)
	}

	return botInfo, chatInfo, botInChat, botIsAdmin, nil
}

func (r *fileRepository) getBotInfo(botToken string) (interface{}, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", botToken)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %v", err)
	}
	defer resp.Body.Close()

	var getMeResp struct {
		Ok     bool                   `json:"ok"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&getMeResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	if !getMeResp.Ok {
		return nil, fmt.Errorf("telegram API returned not ok status: %s", resp.Status)
	}

	return getMeResp.Result, nil
}

func (r *fileRepository) getChatInfo(botToken, chatID string) (interface{}, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getChat", botToken)

	data := map[string]string{
		"chat_id": chatID,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	var getChatResp struct {
		Ok     bool                   `json:"ok"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&getChatResp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	if !getChatResp.Ok {
		return nil, fmt.Errorf("telegram API returned not ok status: %s", resp.Status)
	}

	return getChatResp.Result, nil
}

func (r *fileRepository) checkBotStatus(botToken, chatID string, botID float64) (bool, bool, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getChatMember", botToken)

	data := map[string]interface{}{
		"chat_id": chatID,
		"user_id": botID,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return false, false, fmt.Errorf("failed to marshal JSON: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return false, false, fmt.Errorf("failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	var getChatMemberResp struct {
		Ok     bool `json:"ok"`
		Result struct {
			Status string `json:"status"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&getChatMemberResp); err != nil {
		return false, false, fmt.Errorf("failed to decode JSON response: %v", err)
	}

	if !getChatMemberResp.Ok {
		return false, false, fmt.Errorf("telegram API returned not ok status: %s", resp.Status)
	}

	botInChat := getChatMemberResp.Result.Status != "left" && getChatMemberResp.Result.Status != "kicked"
	botIsAdmin := getChatMemberResp.Result.Status == "administrator" || getChatMemberResp.Result.Status == "creator"

	return botInChat, botIsAdmin, nil
}