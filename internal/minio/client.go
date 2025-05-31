package minio

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"webservice/internal/config"
	"webservice/internal/logger"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client MinIO客户端封装
type Client struct {
	client     *minio.Client
	bucketName string
	config     config.MinIOConfig
}

// PackageInfo 包信息
type PackageInfo struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Size        int64     `json:"size"`
	UploadTime  time.Time `json:"upload_time"`
	ContentType string    `json:"content_type"`
	ETag        string    `json:"etag"`
	DownloadURL string    `json:"download_url,omitempty"`
}

// UploadOptions 上传选项
type UploadOptions struct {
	ContentType string
	Metadata    map[string]string
}

// NewClient 创建MinIO客户端
func NewClient(cfg config.MinIOConfig) (*Client, error) {
	// 初始化MinIO客户端
	minioClient, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	client := &Client{
		client:     minioClient,
		bucketName: cfg.BucketName,
		config:     cfg,
	}

	// 确保bucket存在
	if err := client.ensureBucket(); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return client, nil
}

// ensureBucket 确保bucket存在
func (c *Client) ensureBucket() error {
	ctx := context.Background()

	// 检查bucket是否存在
	exists, err := c.client.BucketExists(ctx, c.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	// 如果bucket不存在，创建它
	if !exists {
		err = c.client.MakeBucket(ctx, c.bucketName, minio.MakeBucketOptions{
			Region: c.config.Region,
		})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		logger.Info(fmt.Sprintf("Created bucket: %s", c.bucketName))
	}

	return nil
}

// UploadPackage 上传包文件
func (c *Client) UploadPackage(ctx context.Context, packageName, version string, reader io.Reader, size int64, opts *UploadOptions) (*PackageInfo, error) {
	objectName := c.buildObjectName(packageName, version)

	// 设置默认选项
	if opts == nil {
		opts = &UploadOptions{
			ContentType: "application/octet-stream",
		}
	}

	// 准备上传选项
	uploadOpts := minio.PutObjectOptions{
		ContentType: opts.ContentType,
		UserMetadata: map[string]string{
			"package-name":    packageName,
			"package-version": version,
			"upload-time":     time.Now().Format(time.RFC3339),
		},
	}

	// 添加自定义元数据
	for k, v := range opts.Metadata {
		uploadOpts.UserMetadata[k] = v
	}

	// 上传文件
	info, err := c.client.PutObject(ctx, c.bucketName, objectName, reader, size, uploadOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to upload package: %w", err)
	}

	// 获取对象信息
	objInfo, err := c.client.StatObject(ctx, c.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	packageInfo := &PackageInfo{
		Name:        packageName,
		Version:     version,
		Size:        objInfo.Size,
		UploadTime:  objInfo.LastModified,
		ContentType: objInfo.ContentType,
		ETag:        objInfo.ETag,
	}

	logger.Info(fmt.Sprintf("Package uploaded successfully: %s@%s (size: %d bytes)", packageName, version, info.Size))
	return packageInfo, nil
}

// DownloadPackage 下载包文件
func (c *Client) DownloadPackage(ctx context.Context, packageName, version string) (io.ReadCloser, *PackageInfo, error) {
	objectName := c.buildObjectName(packageName, version)

	// 获取对象信息
	objInfo, err := c.client.StatObject(ctx, c.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("package not found: %w", err)
	}

	// 获取对象
	object, err := c.client.GetObject(ctx, c.bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download package: %w", err)
	}

	packageInfo := &PackageInfo{
		Name:        packageName,
		Version:     version,
		Size:        objInfo.Size,
		UploadTime:  objInfo.LastModified,
		ContentType: objInfo.ContentType,
		ETag:        objInfo.ETag,
	}

	return object, packageInfo, nil
}

// DeletePackage 删除包文件
func (c *Client) DeletePackage(ctx context.Context, packageName, version string) error {
	objectName := c.buildObjectName(packageName, version)

	err := c.client.RemoveObject(ctx, c.bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete package: %w", err)
	}

	logger.Info(fmt.Sprintf("Package deleted successfully: %s@%s", packageName, version))
	return nil
}

// ListPackageVersions 列出包的所有版本
func (c *Client) ListPackageVersions(ctx context.Context, packageName string) ([]*PackageInfo, error) {
	prefix := fmt.Sprintf("packages/%s/", packageName)

	objectCh := c.client.ListObjects(ctx, c.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var packages []*PackageInfo
	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}

		// 从对象名解析版本信息
		version := c.extractVersionFromObjectName(object.Key)
		if version == "" {
			continue
		}

		packageInfo := &PackageInfo{
			Name:        packageName,
			Version:     version,
			Size:        object.Size,
			UploadTime:  object.LastModified,
			ContentType: "application/octet-stream",
			ETag:        object.ETag,
		}

		packages = append(packages, packageInfo)
	}

	return packages, nil
}

// ListAllPackages 列出所有包
func (c *Client) ListAllPackages(ctx context.Context) (map[string][]*PackageInfo, error) {
	prefix := "packages/"

	objectCh := c.client.ListObjects(ctx, c.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	packages := make(map[string][]*PackageInfo)
	for object := range objectCh {
		if object.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", object.Err)
		}

		// 从对象名解析包名和版本
		packageName, version := c.extractPackageInfoFromObjectName(object.Key)
		if packageName == "" || version == "" {
			continue
		}

		packageInfo := &PackageInfo{
			Name:        packageName,
			Version:     version,
			Size:        object.Size,
			UploadTime:  object.LastModified,
			ContentType: "application/octet-stream",
			ETag:        object.ETag,
		}

		packages[packageName] = append(packages[packageName], packageInfo)
	}

	return packages, nil
}

// GetDownloadURL 获取包的下载URL
func (c *Client) GetDownloadURL(ctx context.Context, packageName, version string, expiry time.Duration) (string, error) {
	objectName := c.buildObjectName(packageName, version)

	// 生成预签名URL
	reqParams := make(url.Values)
	presignedURL, err := c.client.PresignedGetObject(ctx, c.bucketName, objectName, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate download URL: %w", err)
	}

	return presignedURL.String(), nil
}

// PackageExists 检查包是否存在
func (c *Client) PackageExists(ctx context.Context, packageName, version string) (bool, error) {
	objectName := c.buildObjectName(packageName, version)

	_, err := c.client.StatObject(ctx, c.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to check package existence: %w", err)
	}

	return true, nil
}

// buildObjectName 构建对象名称
func (c *Client) buildObjectName(packageName, version string) string {
	// 清理包名和版本中的特殊字符
	cleanPackageName := strings.ReplaceAll(packageName, "/", "_")
	cleanVersion := strings.ReplaceAll(version, "/", "_")

	return fmt.Sprintf("packages/%s/%s/%s-%s.pkg", cleanPackageName, cleanVersion, cleanPackageName, cleanVersion)
}

// extractVersionFromObjectName 从对象名中提取版本信息
func (c *Client) extractVersionFromObjectName(objectName string) string {
	// packages/package-name/version/package-name-version.pkg
	parts := strings.Split(objectName, "/")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

// extractPackageInfoFromObjectName 从对象名中提取包名和版本信息
func (c *Client) extractPackageInfoFromObjectName(objectName string) (string, string) {
	// packages/package-name/version/package-name-version.pkg
	parts := strings.Split(objectName, "/")
	if len(parts) >= 4 {
		packageName := parts[1]
		version := parts[2]
		return packageName, version
	}
	return "", ""
}

// GetPackageInfo 获取包信息
func (c *Client) GetPackageInfo(ctx context.Context, packageName, version string) (*PackageInfo, error) {
	objectName := c.buildObjectName(packageName, version)

	objInfo, err := c.client.StatObject(ctx, c.bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("package not found: %w", err)
	}

	return &PackageInfo{
		Name:        packageName,
		Version:     version,
		Size:        objInfo.Size,
		UploadTime:  objInfo.LastModified,
		ContentType: objInfo.ContentType,
		ETag:        objInfo.ETag,
	}, nil
}
