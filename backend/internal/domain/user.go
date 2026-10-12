package domain

import (
	"time"
)

type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
<<<<<<< HEAD
=======
	NationalID   string
	PhoneNumber  string
>>>>>>> origin/main
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	ExpiresAt        time.Time
	RevokedAt        time.Time
	UserAgent        string
	IPAddress        string
	CreatedAt        time.Time
}

type PasswordResetToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    time.Time
	CreatedAt time.Time
}

<<<<<<< HEAD
=======
type RegisterRequest struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	NationalID  string `json:"national_id,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
}

>>>>>>> origin/main
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserDTO struct {
<<<<<<< HEAD
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role,omitempty"`
=======
	ID          string `json:"id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Role        string `json:"role,omitempty"`
	NationalID  string `json:"national_id,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
>>>>>>> origin/main
}

type SessionData struct {
	AccessToken string  `json:"access_token"`
	TokenType   string  `json:"token_type"`
	ExpiresIn   int64   `json:"expires_in"`
	User        UserDTO `json:"user"`
}

type AuthResponse struct {
	Data SessionData `json:"data"`
}

type PasswordResetRequestDTO struct {
	Email string `json:"email"`
}

type PasswordResetConfirmDTO struct {
	Password string `json:"password"`
}

type MessageResponse struct {
	Message string `json:"message"`
}
