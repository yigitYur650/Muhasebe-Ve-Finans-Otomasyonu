package gdrive

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ServiceAccountKey represents Google Cloud Service Account credentials JSON.
type ServiceAccountKey struct {
	Type        string `json:"type"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
	TokenURI    string `json:"token_uri"`
}

// Config holds Google Drive configuration.
type Config struct {
	CredentialsFile string
	FolderID        string
	Enabled         bool
	RetentionDays   int
}

// Client manages uploading backups and applying retention policies to Google Drive via native REST API.
type Client struct {
	creds         *ServiceAccountKey
	folderID      string
	enabled       bool
	retentionDays int
	httpClient    *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// DriveFile represents a Google Drive file resource.
type DriveFile struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CreatedTime string `json:"createdTime"`
	Size        string `json:"size"`
}

type fileListResponse struct {
	Files []DriveFile `json:"files"`
}

type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// NewClient initializes a native Google Drive API v3 client using Service Account credentials.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if !cfg.Enabled || cfg.CredentialsFile == "" || cfg.FolderID == "" {
		return &Client{enabled: false}, nil
	}

	data, err := os.ReadFile(cfg.CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("google drive credentials file not found at '%s': %w", cfg.CredentialsFile, err)
	}

	var key ServiceAccountKey
	if err := json.Unmarshal(data, &key); err != nil {
		return nil, fmt.Errorf("invalid service account JSON credentials: %w", err)
	}

	if key.ClientEmail == "" || key.PrivateKey == "" {
		return nil, fmt.Errorf("service account credentials missing client_email or private_key")
	}

	if key.TokenURI == "" {
		key.TokenURI = "https://oauth2.googleapis.com/token"
	}

	retention := cfg.RetentionDays
	if retention <= 0 {
		retention = 30
	}

	return &Client{
		creds:         &key,
		folderID:      cfg.FolderID,
		enabled:       true,
		retentionDays: retention,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// IsEnabled returns true if Google Drive backup is configured and active.
func (c *Client) IsEnabled() bool {
	return c != nil && c.enabled && c.creds != nil
}

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiry.Add(-2*time.Minute)) {
		return c.accessToken, nil
	}

	rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(c.creds.PrivateKey))
	if err != nil {
		return "", fmt.Errorf("failed to parse service account private key: %w", err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   c.creds.ClientEmail,
		"scope": "https://www.googleapis.com/auth/drive.file https://www.googleapis.com/auth/drive",
		"aud":   c.creds.TokenURI,
		"exp":   now.Add(1 * time.Hour).Unix(),
		"iat":   now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedJWT, err := token.SignedString(rsaKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign OAuth2 JWT assertion: %w", err)
	}

	formData := url.Values{}
	formData.Set("grant_type", "urn:ietf:params:oauth:grant-type:jwt-bearer")
	formData.Set("assertion", signedJWT)

	req, err := http.NewRequestWithContext(ctx, "POST", c.creds.TokenURI, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to exchange OAuth2 token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("oauth token endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp oauthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return c.accessToken, nil
}

// UploadBackup uploads a local file (.sql.gz) into the configured Google Drive folder.
func (c *Client) UploadBackup(ctx context.Context, localFilePath, remoteFilename string) (*DriveFile, error) {
	if !c.IsEnabled() {
		return nil, fmt.Errorf("google drive backup is disabled or not configured")
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to obtain Google Drive access token: %w", err)
	}

	file, err := os.Open(localFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open local backup file: %w", err)
	}
	defer file.Close()

	if remoteFilename == "" {
		remoteFilename = filepath.Base(localFilePath)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Part 1: Metadata
	metaHeader := fmt.Sprintf("form-data; name=\"metadata\"")
	metaPart, err := writer.CreatePart(map[string][]string{
		"Content-Type":        {"application/json; charset=UTF-8"},
		"Content-Disposition": {metaHeader},
	})
	if err != nil {
		return nil, err
	}

	metadata := map[string]interface{}{
		"name":    remoteFilename,
		"parents": []string{c.folderID},
	}
	if err := json.NewEncoder(metaPart).Encode(metadata); err != nil {
		return nil, err
	}

	// Part 2: Media
	mediaPart, err := writer.CreatePart(map[string][]string{
		"Content-Type":        {"application/gzip"},
		"Content-Disposition": {fmt.Sprintf("form-data; name=\"file\"; filename=\"%s\"", remoteFilename)},
	})
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(mediaPart, file); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	uploadURL := "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart&fields=id,name,size,createdTime"
	req, err := http.NewRequestWithContext(ctx, "POST", uploadURL, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to upload backup to Google Drive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google drive upload returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var driveFile DriveFile
	if err := json.NewDecoder(resp.Body).Decode(&driveFile); err != nil {
		return nil, err
	}

	return &driveFile, nil
}

// CleanOldBackups deletes backups older than retentionDays from the configured folder.
func (c *Client) CleanOldBackups(ctx context.Context) (int, error) {
	if !c.IsEnabled() {
		return 0, nil
	}

	token, err := c.getAccessToken(ctx)
	if err != nil {
		return 0, err
	}

	cutoffTime := time.Now().AddDate(0, 0, -c.retentionDays)
	q := fmt.Sprintf("'%s' in parents and trashed = false and name contains 'backup_oncu_otogaz_'", c.folderID)
	listURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files?q=%s&fields=files(id,name,createdTime)", url.QueryEscape(q))

	req, err := http.NewRequestWithContext(ctx, "GET", listURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("drive list returned status %d", resp.StatusCode)
	}

	var fileList fileListResponse
	if err := json.NewDecoder(resp.Body).Decode(&fileList); err != nil {
		return 0, err
	}

	deletedCount := 0
	for _, f := range fileList.Files {
		created, err := time.Parse(time.RFC3339, f.CreatedTime)
		if err != nil {
			continue
		}

		if created.Before(cutoffTime) {
			delURL := fmt.Sprintf("https://www.googleapis.com/drive/v3/files/%s", f.ID)
			delReq, _ := http.NewRequestWithContext(ctx, "DELETE", delURL, nil)
			delReq.Header.Set("Authorization", "Bearer "+token)
			delResp, err := c.httpClient.Do(delReq)
			if err == nil {
				delResp.Body.Close()
				deletedCount++
			}
		}
	}

	return deletedCount, nil
}
