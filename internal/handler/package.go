package handler

import (
	"net/http"
	"strconv"
	"strings"

	"webservice/internal/logger"
	"webservice/internal/middleware"
	"webservice/internal/models"
	"webservice/internal/service"

	"github.com/gin-gonic/gin"
)

// PackageHandler 包管理处理器
type PackageHandler struct {
	packageService *service.PackageService
}

// NewPackageHandler 创建包管理处理器
func NewPackageHandler(packageService *service.PackageService) *PackageHandler {
	return &PackageHandler{
		packageService: packageService,
	}
}

// CreatePackage 创建包
func (h *PackageHandler) CreatePackage(c *gin.Context) {
	var req models.CreatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		middleware.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	pkg, err := h.packageService.CreatePackage(c.Request.Context(), &req, userID.(uint))
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			middleware.ErrorResponse(c, http.StatusConflict, "Package already exists")
			return
		}
		middleware.ErrorResponse(c, http.StatusInternalServerError, "Failed to create package")
		return
	}

	middleware.SuccessResponse(c, pkg)
}

// GetPackage 获取包信息
func (h *PackageHandler) GetPackage(c *gin.Context) {
	packageName := c.Param("package")
	if packageName == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Package name is required")
		return
	}

	pkg, err := h.packageService.GetPackage(c.Request.Context(), packageName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.ErrorResponse(c, http.StatusNotFound, "Package not found")
			return
		}
		middleware.ErrorResponse(c, http.StatusInternalServerError, "Failed to get package")
		return
	}

	middleware.SuccessResponse(c, pkg)
}

// UpdatePackage 更新包信息
func (h *PackageHandler) UpdatePackage(c *gin.Context) {
	packageName := c.Param("package")
	if packageName == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Package name is required")
		return
	}

	var req models.UpdatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		middleware.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	pkg, err := h.packageService.UpdatePackage(c.Request.Context(), packageName, &req, userID.(uint))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.ErrorResponse(c, http.StatusNotFound, "Package not found")
			return
		}
		if strings.Contains(err.Error(), "permission denied") {
			middleware.ErrorResponse(c, http.StatusForbidden, "Permission denied")
			return
		}
		middleware.ErrorResponse(c, http.StatusInternalServerError, "Failed to update package")
		return
	}

	middleware.SuccessResponse(c, pkg)
}

// DeletePackage 删除包
func (h *PackageHandler) DeletePackage(c *gin.Context) {
	packageName := c.Param("package")
	if packageName == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Package name is required")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		middleware.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	err := h.packageService.DeletePackage(c.Request.Context(), packageName, userID.(uint))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.ErrorResponse(c, http.StatusNotFound, "Package not found")
			return
		}
		if strings.Contains(err.Error(), "permission denied") {
			middleware.ErrorResponse(c, http.StatusForbidden, "Permission denied")
			return
		}
		middleware.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete package")
		return
	}

	middleware.SuccessResponse(c, gin.H{"message": "Package deleted successfully"})
}

// UploadPackageVersion 上传包版本
func (h *PackageHandler) UploadPackageVersion(c *gin.Context) {
	packageName := c.Param("package")
	if packageName == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Package name is required")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		middleware.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Failed to parse form data")
		return
	}

	version := c.PostForm("version")
	description := c.PostForm("description")
	changelog := c.PostForm("changelog")
	isPrerelease := c.PostForm("is_prerelease") == "true"

	if version == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Version is required")
		return
	}

	file, header, err := c.Request.FormFile("package_file")
	if err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Package file is required")
		return
	}
	defer file.Close()

	req := &models.CreatePackageVersionRequest{
		Version:      version,
		Description:  description,
		Changelog:    changelog,
		IsPrerelease: isPrerelease,
		Dependencies: make(map[string]string),
	}

	pkgVersion, err := h.packageService.UploadPackageVersion(
		c.Request.Context(),
		packageName,
		req,
		file,
		header.Size,
		userID.(uint),
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.ErrorResponse(c, http.StatusNotFound, "Package not found")
			return
		}
		if strings.Contains(err.Error(), "permission denied") {
			middleware.ErrorResponse(c, http.StatusForbidden, "Permission denied")
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			middleware.ErrorResponse(c, http.StatusConflict, "Version already exists")
			return
		}
		middleware.ErrorResponse(c, http.StatusInternalServerError, "Failed to upload package version")
		return
	}

	middleware.SuccessResponse(c, pkgVersion)
}

// DownloadPackageVersion 下载包版本
func (h *PackageHandler) DownloadPackageVersion(c *gin.Context) {
	packageName := c.Param("package")
	version := c.Param("version")

	if packageName == "" || version == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Package name and version are required")
		return
	}

	var userID *uint
	if id, exists := c.Get("user_id"); exists {
		uid := id.(uint)
		userID = &uid
	}

	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	reader, pkgVersion, err := h.packageService.DownloadPackageVersion(
		c.Request.Context(),
		packageName,
		version,
		userID,
		ipAddress,
		userAgent,
	)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.ErrorResponse(c, http.StatusNotFound, "Package version not found")
			return
		}
		if strings.Contains(err.Error(), "access denied") {
			middleware.ErrorResponse(c, http.StatusForbidden, "Access denied")
			return
		}
		middleware.ErrorResponse(c, http.StatusInternalServerError, "Failed to download package")
		return
	}
	defer reader.Close()

	filename := packageName + "-" + version + ".pkg"
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", strconv.FormatInt(pkgVersion.FileSize, 10))
	c.Header("X-Package-Name", packageName)
	c.Header("X-Package-Version", version)
	c.Header("X-Package-Hash", pkgVersion.FileHash)

	c.DataFromReader(http.StatusOK, pkgVersion.FileSize, "application/octet-stream", reader, map[string]string{})
}

// GetPackageVersions 获取包的所有版本
func (h *PackageHandler) GetPackageVersions(c *gin.Context) {
	packageName := c.Param("package")
	if packageName == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Package name is required")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	response, err := h.packageService.GetPackageVersions(c.Request.Context(), packageName, page, pageSize)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.ErrorResponse(c, http.StatusNotFound, "Package not found")
			return
		}
		middleware.ErrorResponse(c, http.StatusInternalServerError, "Failed to get package versions")
		return
	}

	middleware.SuccessResponse(c, response)
}

// DeletePackageVersion 删除包版本
func (h *PackageHandler) DeletePackageVersion(c *gin.Context) {
	packageName := c.Param("package")
	version := c.Param("version")

	if packageName == "" || version == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Package name and version are required")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		middleware.ErrorResponse(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	err := h.packageService.DeletePackageVersion(c.Request.Context(), packageName, version, userID.(uint))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.ErrorResponse(c, http.StatusNotFound, "Package version not found")
			return
		}
		if strings.Contains(err.Error(), "permission denied") {
			middleware.ErrorResponse(c, http.StatusForbidden, "Permission denied")
			return
		}
		middleware.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete package version")
		return
	}

	middleware.SuccessResponse(c, gin.H{"message": "Package version deleted successfully"})
}

// SearchPackages 搜索包
func (h *PackageHandler) SearchPackages(c *gin.Context) {
	var req models.SearchPackagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Invalid query parameters"+err.Error())
		return
	}
	logger.Info("SearchPackages request", "query", req.Query, "page", req.Page, "page_size", req.PageSize)
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	response, err := h.packageService.SearchPackages(c.Request.Context(), &req)
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, "Failed to search packages")
		return
	}

	middleware.SuccessResponse(c, response)
}

// GetPackageStats 获取包统计信息
func (h *PackageHandler) GetPackageStats(c *gin.Context) {
	stats, err := h.packageService.GetPackageStats(c.Request.Context())
	if err != nil {
		middleware.ErrorResponse(c, http.StatusInternalServerError, "Failed to get package stats "+err.Error())
		return
	}

	middleware.SuccessResponse(c, stats)
}

// GetDownloadURL 获取下载URL
func (h *PackageHandler) GetDownloadURL(c *gin.Context) {
	packageName := c.Param("package")
	version := c.Param("version")

	if packageName == "" || version == "" {
		middleware.ErrorResponse(c, http.StatusBadRequest, "Package name and version are required")
		return
	}

	var userID *uint
	if id, exists := c.Get("user_id"); exists {
		uid := id.(uint)
		userID = &uid
	}

	url, err := h.packageService.GetDownloadURL(c.Request.Context(), packageName, version, userID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			middleware.ErrorResponse(c, http.StatusNotFound, "Package version not found")
			return
		}
		if strings.Contains(err.Error(), "access denied") {
			middleware.ErrorResponse(c, http.StatusForbidden, "Access denied")
			return
		}
		middleware.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate download URL")
		return
	}

	middleware.SuccessResponse(c, gin.H{
		"download_url": url,
		"expires_in":   3600,
	})
}
