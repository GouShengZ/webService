package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"webservice/internal/minio"
	"webservice/internal/models"

	"gorm.io/gorm"
)

// PackageService 包管理服务
type PackageService struct {
	db          *gorm.DB
	minioClient *minio.Client
}

// NewPackageService 创建包管理服务实例
func NewPackageService(db *gorm.DB, minioClient *minio.Client) *PackageService {
	return &PackageService{
		db:          db,
		minioClient: minioClient,
	}
}

// CreatePackage 创建包
func (s *PackageService) CreatePackage(ctx context.Context, req *models.CreatePackageRequest, ownerID uint) (*models.Package, error) {
	// 检查包名是否已存在
	var existingPackage models.Package
	if err := s.db.Where("name = ?", req.Name).First(&existingPackage).Error; err == nil {
		return nil, errors.New("package name already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check package existence: %w", err)
	}

	// 处理关键词
	keywordsJSON := ""
	if len(req.Keywords) > 0 {
		keywordsBytes, _ := json.Marshal(req.Keywords)
		keywordsJSON = string(keywordsBytes)
	}

	// 创建包
	pkg := &models.Package{
		Name:        req.Name,
		Description: req.Description,
		Author:      req.Author,
		Homepage:    req.Homepage,
		Repository:  req.Repository,
		License:     req.License,
		Keywords:    keywordsJSON,
		IsPrivate:   req.IsPrivate,
		OwnerID:     ownerID,
	}

	if err := s.db.Create(pkg).Error; err != nil {
		return nil, fmt.Errorf("failed to create package: %w", err)
	}

	// 预加载关联数据
	if err := s.db.Preload("Owner").First(pkg, pkg.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to load package with associations: %w", err)
	}

	return pkg, nil
}

// GetPackage 获取包信息
func (s *PackageService) GetPackage(ctx context.Context, packageName string) (*models.Package, error) {
	var pkg models.Package
	err := s.db.Preload("Owner").Preload("Versions").Where("name = ?", packageName).First(&pkg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("package not found")
		}
		return nil, fmt.Errorf("failed to get package: %w", err)
	}

	return &pkg, nil
}

// UpdatePackage 更新包信息
func (s *PackageService) UpdatePackage(ctx context.Context, packageName string, req *models.UpdatePackageRequest, userID uint) (*models.Package, error) {
	var pkg models.Package
	if err := s.db.Where("name = ?", packageName).First(&pkg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("package not found")
		}
		return nil, fmt.Errorf("failed to find package: %w", err)
	}

	// 检查权限
	if pkg.OwnerID != userID {
		return nil, errors.New("permission denied")
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Author != "" {
		updates["author"] = req.Author
	}
	if req.Homepage != "" {
		updates["homepage"] = req.Homepage
	}
	if req.Repository != "" {
		updates["repository"] = req.Repository
	}
	if req.License != "" {
		updates["license"] = req.License
	}
	if req.IsPrivate != nil {
		updates["is_private"] = *req.IsPrivate
	}
	if len(req.Keywords) > 0 {
		keywordsBytes, _ := json.Marshal(req.Keywords)
		updates["keywords"] = string(keywordsBytes)
	}

	if len(updates) > 0 {
		if err := s.db.Model(&pkg).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("failed to update package: %w", err)
		}
	}

	// 重新加载数据
	if err := s.db.Preload("Owner").Preload("Versions").First(&pkg, pkg.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload package: %w", err)
	}

	return &pkg, nil
}

// DeletePackage 删除包
func (s *PackageService) DeletePackage(ctx context.Context, packageName string, userID uint) error {
	var pkg models.Package
	if err := s.db.Where("name = ?", packageName).First(&pkg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("package not found")
		}
		return fmt.Errorf("failed to find package: %w", err)
	}

	// 检查权限
	if pkg.OwnerID != userID {
		return errors.New("permission denied")
	}

	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 获取所有版本
	var versions []models.PackageVersion
	if err := tx.Where("package_id = ?", pkg.ID).Find(&versions).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to get package versions: %w", err)
	}

	// 删除MinIO中的文件
	for _, version := range versions {
		if err := s.minioClient.DeletePackage(ctx, packageName, version.Version); err != nil {
			// 记录错误但不中断删除流程
			fmt.Printf("Warning: failed to delete package file from MinIO: %v\n", err)
		}
	}

	// 删除下载记录
	if err := tx.Where("package_version_id IN (SELECT id FROM package_versions WHERE package_id = ?)", pkg.ID).Delete(&models.PackageDownload{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete download records: %w", err)
	}

	// 删除版本
	if err := tx.Where("package_id = ?", pkg.ID).Delete(&models.PackageVersion{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete package versions: %w", err)
	}

	// 删除包
	if err := tx.Delete(&pkg).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete package: %w", err)
	}

	return tx.Commit().Error
}

// UploadPackageVersion 上传包版本
func (s *PackageService) UploadPackageVersion(ctx context.Context, packageName string, req *models.CreatePackageVersionRequest, fileReader io.Reader, fileSize int64, uploaderID uint) (*models.PackageVersion, error) {
	// 查找包
	var pkg models.Package
	if err := s.db.Where("name = ?", packageName).First(&pkg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("package not found")
		}
		return nil, fmt.Errorf("failed to find package: %w", err)
	}

	// 检查权限
	if pkg.OwnerID != uploaderID {
		return nil, errors.New("permission denied")
	}

	// 检查版本是否已存在
	var existingVersion models.PackageVersion
	if err := s.db.Where("package_id = ? AND version = ?", pkg.ID, req.Version).First(&existingVersion).Error; err == nil {
		return nil, errors.New("version already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check version existence: %w", err)
	}

	// 计算文件哈希
	hasher := sha256.New()
	fileReader = io.TeeReader(fileReader, hasher)

	// 上传到MinIO
	packageInfo, err := s.minioClient.UploadPackage(ctx, packageName, req.Version, fileReader, fileSize, &minio.UploadOptions{
		ContentType: "application/octet-stream",
		Metadata: map[string]string{
			"uploader-id": fmt.Sprintf("%d", uploaderID),
			"description": req.Description,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload package to storage: %w", err)
	}

	// 处理依赖关系
	dependenciesJSON := ""
	if len(req.Dependencies) > 0 {
		dependenciesBytes, _ := json.Marshal(req.Dependencies)
		dependenciesJSON = string(dependenciesBytes)
	}

	// 创建版本记录
	version := &models.PackageVersion{
		PackageID:    pkg.ID,
		Version:      req.Version,
		Description:  req.Description,
		Changelog:    req.Changelog,
		Dependencies: dependenciesJSON,
		FileSize:     packageInfo.Size,
		FileHash:     fmt.Sprintf("%x", hasher.Sum(nil)),
		MinIOPath:    fmt.Sprintf("packages/%s/%s", packageName, req.Version),
		IsPrerelease: req.IsPrerelease,
		UploaderID:   uploaderID,
	}

	if err := s.db.Create(version).Error; err != nil {
		// 如果数据库操作失败，尝试删除已上传的文件
		s.minioClient.DeletePackage(ctx, packageName, req.Version)
		return nil, fmt.Errorf("failed to create version record: %w", err)
	}

	// 预加载关联数据
	if err := s.db.Preload("Package").Preload("Uploader").First(version, version.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to load version with associations: %w", err)
	}

	return version, nil
}

// DownloadPackageVersion 下载包版本
func (s *PackageService) DownloadPackageVersion(ctx context.Context, packageName, version string, userID *uint, ipAddress, userAgent string) (io.ReadCloser, *models.PackageVersion, error) {
	// 查找包版本
	var pkgVersion models.PackageVersion
	err := s.db.Preload("Package").Where("package_id = (SELECT id FROM packages WHERE name = ?) AND version = ?", packageName, version).First(&pkgVersion).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("package version not found")
		}
		return nil, nil, fmt.Errorf("failed to find package version: %w", err)
	}

	// 检查私有包权限
	if pkgVersion.Package.IsPrivate && (userID == nil || pkgVersion.Package.OwnerID != *userID) {
		return nil, nil, errors.New("access denied to private package")
	}

	// 从MinIO下载文件
	reader, _, err := s.minioClient.DownloadPackage(ctx, packageName, version)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download package from storage: %w", err)
	}

	// 记录下载
	go func() {
		downloadRecord := &models.PackageDownload{
			PackageVersionID: pkgVersion.ID,
			UserID:           userID,
			IPAddress:        ipAddress,
			UserAgent:        userAgent,
		}
		if err := s.db.Create(downloadRecord).Error; err != nil {
			fmt.Printf("Warning: failed to record download: %v\n", err)
		}

		// 更新下载计数
		if err := s.db.Model(&pkgVersion).UpdateColumn("download_count", gorm.Expr("download_count + ?", 1)).Error; err != nil {
			fmt.Printf("Warning: failed to update download count: %v\n", err)
		}
	}()

	return reader, &pkgVersion, nil
}

// GetPackageVersions 获取包的所有版本
func (s *PackageService) GetPackageVersions(ctx context.Context, packageName string, page, pageSize int) (*models.PackageVersionListResponse, error) {
	var pkg models.Package
	if err := s.db.Where("name = ?", packageName).First(&pkg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("package not found")
		}
		return nil, fmt.Errorf("failed to find package: %w", err)
	}

	var total int64
	if err := s.db.Model(&models.PackageVersion{}).Where("package_id = ?", pkg.ID).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count versions: %w", err)
	}

	offset := (page - 1) * pageSize
	var versions []models.PackageVersion
	err := s.db.Preload("Uploader").Where("package_id = ?", pkg.ID).
		Order("created_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&versions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get versions: %w", err)
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	return &models.PackageVersionListResponse{
		Versions:   versions,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// DeletePackageVersion 删除包版本
func (s *PackageService) DeletePackageVersion(ctx context.Context, packageName, version string, userID uint) error {
	// 查找包版本
	var pkgVersion models.PackageVersion
	err := s.db.Preload("Package").Where("package_id = (SELECT id FROM packages WHERE name = ?) AND version = ?", packageName, version).First(&pkgVersion).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("package version not found")
		}
		return fmt.Errorf("failed to find package version: %w", err)
	}

	// 检查权限
	if pkgVersion.Package.OwnerID != userID {
		return errors.New("permission denied")
	}

	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除下载记录
	if err := tx.Where("package_version_id = ?", pkgVersion.ID).Delete(&models.PackageDownload{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete download records: %w", err)
	}

	// 删除版本记录
	if err := tx.Delete(&pkgVersion).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete version: %w", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 删除MinIO中的文件
	if err := s.minioClient.DeletePackage(ctx, packageName, version); err != nil {
		// 记录错误但不返回失败
		fmt.Printf("Warning: failed to delete package file from MinIO: %v\n", err)
	}

	return nil
}

// SearchPackages 搜索包
func (s *PackageService) SearchPackages(ctx context.Context, req *models.SearchPackagesRequest) (*models.PackageListResponse, error) {
	query := s.db.Model(&models.Package{}).Preload("Owner")

	// 构建搜索条件
	if req.Query != "" {
		searchTerm := "%" + strings.ToLower(req.Query) + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ?", searchTerm, searchTerm)
	}

	if req.Author != "" {
		query = query.Where("LOWER(author) LIKE ?", "%"+strings.ToLower(req.Author)+"%")
	}

	if req.Keywords != "" {
		query = query.Where("LOWER(keywords) LIKE ?", "%"+strings.ToLower(req.Keywords)+"%")
	}

	if req.License != "" {
		query = query.Where("LOWER(license) = ?", strings.ToLower(req.License))
	}

	if req.IsPrivate != nil {
		query = query.Where("is_private = ?", *req.IsPrivate)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count packages: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	var packages []models.Package
	err := query.Order("created_at DESC").
		Limit(req.PageSize).Offset(offset).
		Find(&packages).Error
	if err != nil {
		return nil, fmt.Errorf("failed to search packages: %w", err)
	}

	totalPages := int((total + int64(req.PageSize) - 1) / int64(req.PageSize))

	return &models.PackageListResponse{
		Packages:   packages,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetPackageStats 获取包统计信息
func (s *PackageService) GetPackageStats(ctx context.Context) (*models.PackageStatsResponse, error) {
	stats := &models.PackageStatsResponse{}

	// 总包数
	if err := s.db.Model(&models.Package{}).Count(&stats.TotalPackages).Error; err != nil {
		return nil, fmt.Errorf("failed to count packages: %w", err)
	}

	// 总版本数
	if err := s.db.Model(&models.PackageVersion{}).Count(&stats.TotalVersions).Error; err != nil {
		return nil, fmt.Errorf("failed to count versions: %w", err)
	}

	// 总下载数
	if err := s.db.Model(&models.PackageVersion{}).Select("SUM(download_count)").Scan(&stats.TotalDownloads).Error; err != nil {
		return nil, fmt.Errorf("failed to count downloads: %w", err)
	}

	// 最近30天下载数
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	if err := s.db.Model(&models.PackageDownload{}).Where("download_time >= ?", thirtyDaysAgo).Count(&stats.RecentDownloads).Error; err != nil {
		return nil, fmt.Errorf("failed to count recent downloads: %w", err)
	}

	// 热门包（按下载量排序）
	err := s.db.Preload("Owner").
		Joins("JOIN (SELECT package_id, SUM(download_count) as total_downloads FROM package_versions GROUP BY package_id ORDER BY total_downloads DESC LIMIT 10) pv ON packages.id = pv.package_id").
		Order("pv.total_downloads DESC").
		Find(&stats.PopularPackages).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get popular packages: %w", err)
	}

	// 最新包
	if err := s.db.Preload("Owner").Order("created_at DESC").Limit(10).Find(&stats.RecentPackages).Error; err != nil {
		return nil, fmt.Errorf("failed to get recent packages: %w", err)
	}

	// 最新版本
	if err := s.db.Preload("Package").Preload("Uploader").Order("created_at DESC").Limit(10).Find(&stats.RecentVersions).Error; err != nil {
		return nil, fmt.Errorf("failed to get recent versions: %w", err)
	}

	return stats, nil
}

// GetDownloadURL 获取下载URL
func (s *PackageService) GetDownloadURL(ctx context.Context, packageName, version string, userID *uint) (string, error) {
	// 查找包版本
	var pkgVersion models.PackageVersion
	err := s.db.Preload("Package").Where("package_id = (SELECT id FROM packages WHERE name = ?) AND version = ?", packageName, version).First(&pkgVersion).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("package version not found")
		}
		return "", fmt.Errorf("failed to find package version: %w", err)
	}

	// 检查私有包权限
	if pkgVersion.Package.IsPrivate && (userID == nil || pkgVersion.Package.OwnerID != *userID) {
		return "", errors.New("access denied to private package")
	}

	// 生成下载URL（1小时有效期）
	url, err := s.minioClient.GetDownloadURL(ctx, packageName, version, time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return url, nil
}
