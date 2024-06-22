package services

import (
	"io"

	"main.go/repositories"
)

type FileService interface {
	SendFile(botToken, chatID string, file io.Reader, fileName string) (string, error)
	GetFileInfo(botToken, fileID string) (string, int, error)
	CheckBotAndChat(botToken, chatID string) (botInfo, chatInfo interface{}, botInChat, botIsAdmin bool, err error)
}

type fileService struct {
	repo repositories.FileRepository
}

func NewFileService(repo repositories.FileRepository) FileService {
	return &fileService{repo: repo}
}

func (s *fileService) SendFile(botToken, chatID string, file io.Reader, fileName string) (string, error) {
	return s.repo.SendDocument(botToken, chatID, file, fileName)
}

func (s *fileService) GetFileInfo(botToken, fileID string) (string, int, error) {
	fileURL, fileSize, err := s.repo.GetFileInfo(botToken, fileID)
	if err != nil {
		return "", 0, err
	}

	return fileURL, fileSize, nil
}

func (s *fileService) CheckBotAndChat(botToken, chatID string) (botInfo, chatInfo interface{}, botInChat, botIsAdmin bool, err error) {
	return s.repo.CheckBotAndChat(botToken, chatID)
}