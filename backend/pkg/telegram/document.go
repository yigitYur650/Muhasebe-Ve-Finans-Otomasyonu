package telegram

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// SendDocument sends a file (e.g. backup .sql.gz) directly to the configured Telegram chat.
func (c *Client) SendDocument(filePath, caption string) error {
	if !c.IsEnabled() {
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open backup file for telegram: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add chat_id field
	if err := writer.WriteField("chat_id", c.chatID); err != nil {
		return fmt.Errorf("failed to write chat_id field: %w", err)
	}

	// Add caption field
	if caption != "" {
		if err := writer.WriteField("caption", caption); err != nil {
			return fmt.Errorf("failed to write caption field: %w", err)
		}
		if err := writer.WriteField("parse_mode", "HTML"); err != nil {
			return fmt.Errorf("failed to write parse_mode field: %w", err)
		}
	}

	// Add document file part
	part, err := writer.CreateFormFile("document", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file to multipart: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", c.botToken)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("failed to create sendDocument request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	uploadClient := &http.Client{
		Timeout: 60 * time.Second,
	}
	resp, err := uploadClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute sendDocument request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram sendDocument returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
