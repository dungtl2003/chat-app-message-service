package model

import "dungtl2003/chat-app-message-service/internal/types"

type UserGender string
type UserRole string

const (
	UserGenderMale   = "MALE"
	UserGenderFemale = "FEMALE"

	UserRoleAdmin = "ADMIN"
	UserRoleUser  = "USER"
)

var (
	AllowedUserGenders = map[UserGender]bool{
		UserGenderMale:   true,
		UserGenderFemale: true,
	}
	AllowedUserRoles = map[UserRole]bool{
		UserRoleAdmin: true,
		UserRoleUser:  true,
	}
)

type ChatUser struct {
	Id             types.JsonInt64      `json:"id"`
	Email          string               `json:"email"`
	Username       string               `json:"username"`
	Password       string               `json:"password"`
	Role           UserRole             `json:"role"`
	FirstName      types.JsonNullString `json:"first_name"`
	LastName       types.JsonNullString `json:"last_name"`
	Birthday       types.JsonNullTime   `json:"birthday"`
	Gender         types.JsonNullString `json:"gender"`
	PhoneNumber    types.JsonNullString `json:"phone_number"`
	Privacy        types.JsonNullString `json:"privacy"`
	AvatarId       types.JsonNullInt64  `json:"avatar_id"`
	CreatedAt      types.JsonTime       `json:"created_at"`
	UpdatedAt      types.JsonTime       `json:"updated_at"`
	DeletedAt      types.JsonNullTime   `json:"deleted_at"`
	SessionVersion types.JsonInt64      `json:"session_version"`
	Version        types.JsonInt64      `json:"version"`

	// generated field
	FullName string `json:"full_name"`

	Sessions  []Session            `json:"sessions"`
	AvatarURL types.JsonNullString `json:"avatar_url"`
}
