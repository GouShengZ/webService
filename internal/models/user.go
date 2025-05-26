package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	Username  string         `json:"username" gorm:"uniqueIndex;not null;size:50" binding:"required,min=3,max=50"`
	Email     string         `json:"email" gorm:"uniqueIndex;not null;size:100" binding:"required,email"`
	Password  string         `json:"-" gorm:"not null;size:255" binding:"required,min=6"`
	Nickname  string         `json:"nickname" gorm:"size:50"`
	Avatar    string         `json:"avatar" gorm:"size:255"`
	Role      string         `json:"role" gorm:"not null;default:user;size:20"`
	Status    UserStatus     `json:"status" gorm:"not null;default:1"`
	LastLogin *time.Time     `json:"last_login"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// UserStatus 用户状态枚举
type UserStatus int

const (
	// UserStatusInactive 未激活
	UserStatusInactive UserStatus = 0
	// UserStatusActive 正常
	UserStatusActive UserStatus = 1
	// UserStatusSuspended 暂停
	UserStatusSuspended UserStatus = 2
	// UserStatusBanned 禁用
	UserStatusBanned UserStatus = 3
)

// String 返回用户状态的字符串表示
func (s UserStatus) String() string {
	switch s {
	case UserStatusInactive:
		return "inactive"
	case UserStatusActive:
		return "active"
	case UserStatusSuspended:
		return "suspended"
	case UserStatusBanned:
		return "banned"
	default:
		return "unknown"
	}
}

// UserRole 用户角色常量
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
	RoleSuper = "super"
)

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// BeforeCreate GORM钩子：创建前
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// 设置默认角色
	if u.Role == "" {
		u.Role = RoleUser
	}
	// 设置默认状态
	if u.Status == 0 {
		u.Status = UserStatusActive
	}
	// 设置默认昵称
	if u.Nickname == "" {
		u.Nickname = u.Username
	}
	return nil
}

// IsActive 检查用户是否为活跃状态
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsAdmin 检查用户是否为管理员
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin || u.Role == RoleSuper
}

// IsSuper 检查用户是否为超级管理员
func (u *User) IsSuper() bool {
	return u.Role == RoleSuper
}

// ToPublicUser 转换为公开用户信息（隐藏敏感信息）
func (u *User) ToPublicUser() *PublicUser {
	return &PublicUser{
		ID:        u.ID,
		Username:  u.Username,
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
		Status:    u.Status.String(),
		CreatedAt: u.CreatedAt,
	}
}

// PublicUser 公开用户信息结构体
type PublicUser struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginRequest 登录请求结构体
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest 注册请求结构体
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname" binding:"max=50"`
}

// UpdateProfileRequest 更新个人资料请求结构体
type UpdateProfileRequest struct {
	Nickname string `json:"nickname" binding:"max=50"`
	Avatar   string `json:"avatar" binding:"max=255"`
	Email    string `json:"email" binding:"email"`
}

// UpdateUserRequest 更新用户请求结构体（管理员使用）
type UpdateUserRequest struct {
	Nickname string     `json:"nickname" binding:"max=50"`
	Avatar   string     `json:"avatar" binding:"max=255"`
	Email    string     `json:"email" binding:"email"`
	Role     string     `json:"role" binding:"oneof=user admin super"`
	Status   UserStatus `json:"status" binding:"oneof=0 1 2 3"`
}

// LoginResponse 登录响应结构体
type LoginResponse struct {
	User  *PublicUser `json:"user"`
	Token string      `json:"token"`
}
