package models

import (
	"time"

	"gorm.io/gorm"
)

// Package 包模型
type Package struct {
	ID          uint             `json:"id" gorm:"primarykey"`
	Name        string           `json:"name" gorm:"uniqueIndex:idx_package_name;not null;size:100" binding:"required,min=1,max=100"`
	Description string           `json:"description" gorm:"size:500"`
	Author      string           `json:"author" gorm:"size:100"`
	Homepage    string           `json:"homepage" gorm:"size:255"`
	Repository  string           `json:"repository" gorm:"size:255"`
	License     string           `json:"license" gorm:"size:50"`
	Keywords    string           `json:"keywords" gorm:"size:500"` // JSON数组存储为字符串
	IsPrivate   bool             `json:"is_private" gorm:"default:false"`
	OwnerID     uint             `json:"owner_id" gorm:"not null"`
	Owner       User             `json:"owner" gorm:"foreignKey:OwnerID"`
	Versions    []PackageVersion `json:"versions,omitempty" gorm:"foreignKey:PackageID"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	DeletedAt   gorm.DeletedAt   `json:"-" gorm:"index"`
}

// PackageVersion 包版本模型
type PackageVersion struct {
	ID            uint           `json:"id" gorm:"primarykey"`
	PackageID     uint           `json:"package_id" gorm:"not null"`
	Package       Package        `json:"package,omitempty" gorm:"foreignKey:PackageID"`
	Version       string         `json:"version" gorm:"uniqueIndex:idx_package_version;not null;size:50" binding:"required"`
	Description   string         `json:"description" gorm:"size:500"`
	Changelog     string         `json:"changelog" gorm:"type:text"`
	Dependencies  string         `json:"dependencies" gorm:"type:text"` // JSON存储依赖关系
	FileSize      int64          `json:"file_size" gorm:"not null"`
	FileHash      string         `json:"file_hash" gorm:"size:64"`   // SHA256哈希
	MinIOPath     string         `json:"minio_path" gorm:"size:255"` // MinIO中的存储路径
	DownloadCount int64          `json:"download_count" gorm:"default:0"`
	IsPrerelease  bool           `json:"is_prerelease" gorm:"default:false"`
	UploaderID    uint           `json:"uploader_id" gorm:"not null"`
	Uploader      User           `json:"uploader" gorm:"foreignKey:UploaderID"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// PackageDownload 包下载记录模型
type PackageDownload struct {
	ID               uint           `json:"id" gorm:"primarykey"`
	PackageVersionID uint           `json:"package_version_id" gorm:"not null"`
	PackageVersion   PackageVersion `json:"package_version,omitempty" gorm:"foreignKey:PackageVersionID"`
	UserID           *uint          `json:"user_id"` // 可选，匿名下载时为空
	User             *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
	IPAddress        string         `json:"ip_address" gorm:"size:45"` // 支持IPv6
	UserAgent        string         `json:"user_agent" gorm:"size:500"`
	DownloadTime     time.Time      `json:"download_time" gorm:"autoCreateTime"`
}

// CreatePackageRequest 创建包请求
type CreatePackageRequest struct {
	Name        string   `json:"name" binding:"required,min=1,max=100"`
	Description string   `json:"description" binding:"max=500"`
	Author      string   `json:"author" binding:"max=100"`
	Homepage    string   `json:"homepage" binding:"max=255,url"`
	Repository  string   `json:"repository" binding:"max=255,url"`
	License     string   `json:"license" binding:"max=50"`
	Keywords    []string `json:"keywords"`
	IsPrivate   bool     `json:"is_private"`
}

// UpdatePackageRequest 更新包请求
type UpdatePackageRequest struct {
	Description string   `json:"description" binding:"max=500"`
	Author      string   `json:"author" binding:"max=100"`
	Homepage    string   `json:"homepage" binding:"max=255,url"`
	Repository  string   `json:"repository" binding:"max=255,url"`
	License     string   `json:"license" binding:"max=50"`
	Keywords    []string `json:"keywords"`
	IsPrivate   *bool    `json:"is_private"` // 使用指针以区分false和未设置
}

// CreatePackageVersionRequest 创建包版本请求
type CreatePackageVersionRequest struct {
	Version      string            `json:"version" binding:"required,max=50"`
	Description  string            `json:"description" binding:"max=500"`
	Changelog    string            `json:"changelog"`
	Dependencies map[string]string `json:"dependencies"` // package_name: version
	IsPrerelease bool              `json:"is_prerelease"`
}

// PackageListResponse 包列表响应
type PackageListResponse struct {
	Packages   []Package `json:"packages"`
	Total      int64     `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalPages int       `json:"total_pages"`
}

// PackageVersionListResponse 包版本列表响应
type PackageVersionListResponse struct {
	Versions   []PackageVersion `json:"versions"`
	Total      int64            `json:"total"`
	Page       int              `json:"page"`
	PageSize   int              `json:"page_size"`
	TotalPages int              `json:"total_pages"`
}

// SearchPackagesRequest 包搜索请求
type SearchPackagesRequest struct {
	Query     string `json:"query" form:"query"`
	Author    string `json:"author" form:"author"`
	Keywords  string `json:"keywords" form:"keywords"`
	License   string `json:"license" form:"license"`
	IsPrivate *bool  `json:"is_private" form:"is_private"`
	Page      int    `json:"page" form:"page"`
	PageSize  int    `json:"page_size" form:"page_size"`
}

// PackageStatsResponse 包统计响应
type PackageStatsResponse struct {
	TotalPackages   int64            `json:"total_packages"`
	TotalVersions   int64            `json:"total_versions"`
	TotalDownloads  int64            `json:"total_downloads"`
	RecentDownloads int64            `json:"recent_downloads"` // 最近30天下载量
	PopularPackages []Package        `json:"popular_packages"` // 热门包
	RecentPackages  []Package        `json:"recent_packages"`  // 最新包
	RecentVersions  []PackageVersion `json:"recent_versions"`  // 最新版本
}

// TableName 指定Package表名
func (Package) TableName() string {
	return "packages"
}

// TableName 指定PackageVersion表名
func (PackageVersion) TableName() string {
	return "package_versions"
}

// TableName 指定PackageDownload表名
func (PackageDownload) TableName() string {
	return "package_downloads"
}
