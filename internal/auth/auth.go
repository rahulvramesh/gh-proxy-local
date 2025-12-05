// Package auth provides authentication and credentials management for GitHub Copilot.
package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/rahulvramesh/gh-proxy-local/internal/config"
	"github.com/rahulvramesh/gh-proxy-local/internal/models"
)

// Manager handles authentication and token management.
type Manager struct {
	credentialsFile string
	credentials     *models.Credentials
	mu              sync.RWMutex
	httpClient      *http.Client
	debug           bool
}

// NewManager creates a new authentication manager.
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		credentialsFile: cfg.CredentialsFile,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		debug: cfg.Debug,
	}
}

// LoadCredentials loads credentials from the credentials file.
func (m *Manager) LoadCredentials() (*models.Credentials, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.credentialsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	var creds models.Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	m.credentials = &creds
	return &creds, nil
}

// SaveCredentials saves credentials to the credentials file.
func (m *Manager) SaveCredentials(creds *models.Credentials) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	if err := os.WriteFile(m.credentialsFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	m.credentials = creds
	return nil
}

// GetCredentials returns the current credentials, performing auth if needed.
func (m *Manager) GetCredentials() (*models.Credentials, error) {
	m.mu.RLock()
	if m.credentials != nil && m.credentials.GitHubToken != "" {
		creds := m.credentials
		m.mu.RUnlock()
		return creds, nil
	}
	m.mu.RUnlock()

	creds, err := m.LoadCredentials()
	if err != nil {
		return nil, err
	}

	if creds == nil || creds.GitHubToken == "" {
		// Need to perform device flow authentication
		token, err := m.DeviceFlowAuth()
		if err != nil {
			return nil, err
		}
		creds = &models.Credentials{GitHubToken: token}
		if err := m.SaveCredentials(creds); err != nil {
			return nil, err
		}
	}

	return creds, nil
}

// DeviceFlowAuth performs GitHub OAuth device flow authentication.
func (m *Manager) DeviceFlowAuth() (string, error) {
	fmt.Println("\n=== GitHub Copilot Authentication ===")
	fmt.Println()

	// Request device code
	reqBody := map[string]string{
		"client_id": config.ClientID,
		"scope":     "read:user",
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", config.DeviceCodeURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GitHubCopilotChat/0.35.0")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("device code request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to initiate device authorization: %s", string(respBody))
	}

	var deviceResp models.DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&deviceResp); err != nil {
		return "", fmt.Errorf("failed to decode device response: %w", err)
	}

	fmt.Printf("Please visit: %s\n", deviceResp.VerificationURI)
	fmt.Printf("Enter code: %s\n", deviceResp.UserCode)
	fmt.Println("\nWaiting for authorization...")

	interval := deviceResp.Interval
	if interval == 0 {
		interval = 5
	}

	// Poll for access token
	for {
		time.Sleep(time.Duration(interval) * time.Second)

		tokenReqBody := map[string]string{
			"client_id":   config.ClientID,
			"device_code": deviceResp.DeviceCode,
			"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		}
		tokenBody, _ := json.Marshal(tokenReqBody)

		tokenReq, err := http.NewRequest("POST", config.AccessTokenURL, bytes.NewReader(tokenBody))
		if err != nil {
			return "", fmt.Errorf("failed to create token request: %w", err)
		}
		tokenReq.Header.Set("Accept", "application/json")
		tokenReq.Header.Set("Content-Type", "application/json")
		tokenReq.Header.Set("User-Agent", "GitHubCopilotChat/0.35.0")

		tokenResp, err := m.httpClient.Do(tokenReq)
		if err != nil {
			return "", fmt.Errorf("token request failed: %w", err)
		}

		var tokenData models.AccessTokenResponse
		if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
			tokenResp.Body.Close()
			return "", fmt.Errorf("failed to decode token response: %w", err)
		}
		tokenResp.Body.Close()

		if tokenData.AccessToken != "" {
			fmt.Println("\nAuthorization successful!")
			return tokenData.AccessToken, nil
		}

		switch tokenData.Error {
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5
			continue
		case "":
			continue
		default:
			errMsg := tokenData.ErrorDescription
			if errMsg == "" {
				errMsg = tokenData.Error
			}
			return "", fmt.Errorf("authorization error: %s", errMsg)
		}
	}
}

// GetCopilotToken gets a short-lived Copilot API token.
func (m *Manager) GetCopilotToken(creds *models.Credentials) (string, error) {
	currentTimeMS := time.Now().UnixMilli()

	// Check if current token is still valid
	if creds.CopilotToken != "" && creds.CopilotExpires > (currentTimeMS+config.RefreshBufferMS) {
		return creds.CopilotToken, nil
	}

	if m.debug {
		fmt.Println("[DEBUG] Refreshing Copilot API token...")
	}

	req, err := http.NewRequest("GET", config.CopilotTokenURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+creds.GitHubToken)
	for k, v := range config.CopilotHeaders {
		req.Header.Set(k, v)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("GitHub token expired. Re-authenticate by deleting %s", m.credentialsFile)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get Copilot token: %s", string(body))
	}

	var tokenResp models.CopilotTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	creds.CopilotToken = tokenResp.Token
	creds.CopilotExpires = tokenResp.ExpiresAt * 1000

	if err := m.SaveCredentials(creds); err != nil {
		return "", err
	}

	return tokenResp.Token, nil
}

// GetCopilotAccountInfo fetches account information from Copilot API.
func (m *Manager) GetCopilotAccountInfo(creds *models.Credentials) (*models.CopilotTokenResponse, error) {
	req, err := http.NewRequest("GET", config.CopilotTokenURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+creds.GitHubToken)
	for k, v := range config.CopilotHeaders {
		req.Header.Set(k, v)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch account info: %s", string(body))
	}

	var info models.CopilotTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Remove token from response for security
	info.Token = ""
	return &info, nil
}

// GetGitHubUser fetches GitHub user information.
func (m *Manager) GetGitHubUser(creds *models.Credentials) (*models.GitHubUser, error) {
	req, err := http.NewRequest("GET", config.GitHubUserURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+creds.GitHubToken)
	req.Header.Set("User-Agent", "GitHubCopilotChat/0.32.4")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user info: %s", string(body))
	}

	var user models.GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &user, nil
}
