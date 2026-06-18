package models

import "time"

type User struct {
	ID                  string    `json:"id"`
	Email               string    `json:"email"`
	PublicKey           string    `json:"publicKey"`
	EncryptedPrivateKey string    `json:"encryptedPrivateKey"`
	KDFSalt             string    `json:"kdfSalt"`
	CreatedAt           time.Time `json:"createdAt"`
}

type Project struct {
	ID                   string    `json:"id"`
	UserID               string    `json:"userId"`
	EncryptedName        string    `json:"encryptedName"`
	EncryptedDescription string    `json:"encryptedDescription,omitempty"`
	CreatedAt            time.Time `json:"createdAt"`
}

type Pipeline struct {
	ID            string    `json:"id"`
	UserID        string    `json:"userId"`
	ProjectID     string    `json:"projectId"`
	EncryptedName string    `json:"encryptedName"`
	CreatedAt     time.Time `json:"createdAt"`
}

type NotificationToken struct {
	ID            string     `json:"id"`
	UserID        string     `json:"userId"`
	ProjectID     string     `json:"projectId"`
	PipelineID    string     `json:"pipelineId,omitempty"`
	EncryptedName string     `json:"encryptedName"`
	TokenHash     string     `json:"-"`
	Active        bool       `json:"active"`
	CreatedAt     time.Time  `json:"createdAt"`
	LastUsedAt    *time.Time `json:"lastUsedAt,omitempty"`
}

type Run struct {
	ID               string    `json:"id"`
	UserID           string    `json:"userId"`
	ProjectID        string    `json:"projectId"`
	PipelineID       string    `json:"pipelineId"`
	TokenID          string    `json:"tokenId"`
	Status           string    `json:"status"`
	EncryptedPayload string    `json:"encryptedPayload"`
	ReceivedAt       time.Time `json:"receivedAt"`
}

type RunPayload struct {
	Status   string `json:"status"`
	Pipeline string `json:"pipeline,omitempty"`
	RunID    string `json:"runId,omitempty"`
	Commit   string `json:"commit,omitempty"`
	Branch   string `json:"branch,omitempty"`
	Duration string `json:"duration,omitempty"`
	Message  string `json:"message,omitempty"`
}

type VAPIDSubscription struct {
	ID         string    `json:"id"`
	UserID     string    `json:"userId"`
	Endpoint   string    `json:"endpoint"`
	P256DHKey  string    `json:"p256dhKey"`
	AuthKey    string    `json:"authKey"`
	DeviceName string    `json:"deviceName,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

// API request/response types

type RegisterRequest struct {
	Email               string `json:"email"`
	Password            string `json:"password"`
	PublicKey           string `json:"publicKey"`
	EncryptedPrivateKey string `json:"encryptedPrivateKey"`
	KDFSalt             string `json:"kdfSalt"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	JWT                 string `json:"jwt"`
	PublicKey           string `json:"publicKey"`
	EncryptedPrivateKey string `json:"encryptedPrivateKey"`
	KDFSalt             string `json:"kdfSalt"`
}

type WebhookRequest struct {
	Token    string `json:"token"`
	Status   string `json:"status"`
	Pipeline string `json:"pipeline,omitempty"`
	RunID    string `json:"runId,omitempty"`
	Commit   string `json:"commit,omitempty"`
	Branch   string `json:"branch,omitempty"`
	Duration string `json:"duration,omitempty"`
	Message  string `json:"message,omitempty"`
}

type PushSubscribeRequest struct {
	Endpoint   string `json:"endpoint"`
	P256DHKey  string `json:"p256dhKey"`
	AuthKey    string `json:"authKey"`
	DeviceName string `json:"deviceName,omitempty"`
}

type CreateProjectRequest struct {
	EncryptedName        string `json:"encryptedName"`
	EncryptedDescription string `json:"encryptedDescription,omitempty"`
}

type CreatePipelineRequest struct {
	EncryptedName string `json:"encryptedName"`
}

type CreateTokenRequest struct {
	EncryptedName string `json:"encryptedName"`
	ProjectID     string `json:"projectId"`
	PipelineID    string `json:"pipelineId,omitempty"`
}

type CreateTokenResponse struct {
	Token          NotificationToken `json:"token"`
	PlaintextToken string            `json:"plaintextToken"` // returned once, never stored
}

type SSEEvent struct {
	Type             string `json:"type"` // "run_update"
	RunID            string `json:"runId"`
	EncryptedPayload string `json:"encryptedPayload"`
	ReceivedAt       string `json:"receivedAt"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
