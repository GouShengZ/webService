package migration

import (
	"webservice/internal/logger"
	"webservice/internal/models"

	"gorm.io/gorm"
)

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(db *gorm.DB) error {
	logger.Info("Starting database migration...")

	// 一次性迁移所有模型，这样更高效
	if err := db.AutoMigrate(
		&models.User{},
		&models.Package{},
		&models.PackageVersion{},
		&models.PackageDownload{},
	); err != nil {
		logger.Errorf("Failed to migrate database: %v", err)
		return err
	}

	logger.Info("Database migration completed successfully")
	return nil
}

// CreateIndexes 创建数据库索引
func CreateIndexes(db *gorm.DB) error {
	logger.Info("Skipping database indexes creation for faster startup...")
	// 暂时跳过索引创建以加快启动速度
	return nil
}

// SeedData 初始化种子数据
func SeedData(db *gorm.DB) error {
	logger.Info("Seeding initial data...")

	// 检查是否已存在管理员用户
	var adminCount int64
	if err := db.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&adminCount).Error; err != nil {
		logger.Errorf("Failed to count admin users: %v", err)
		return err
	}

	if adminCount == 0 {
		// 创建默认管理员用户
		adminUser := &models.User{
			Username: "admin",
			Email:    "admin@example.com",
			Password: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password
			Nickname: "Administrator",
			Role:     models.RoleAdmin,
			Status:   models.UserStatusActive,
		}

		if err := db.Create(adminUser).Error; err != nil {
			logger.Errorf("Failed to create admin user: %v", err)
			return err
		}
		logger.Info("Default admin user created successfully (username: admin, password: password)")
	} else {
		logger.Info("Admin user already exists, skipping creation")
	}

	// 检查是否已存在测试用户
	var userCount int64
	if err := db.Model(&models.User{}).Where("role = ? AND username = ?", models.RoleUser, "testuser").Count(&userCount).Error; err != nil {
		logger.Errorf("Failed to count test users: %v", err)
		return err
	}

	if userCount == 0 {
		// 创建测试用户
		testUser := &models.User{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password
			Nickname: "Test User",
			Role:     models.RoleUser,
			Status:   models.UserStatusActive,
		}

		if err := db.Create(testUser).Error; err != nil {
			logger.Errorf("Failed to create test user: %v", err)
			return err
		}
		logger.Info("Test user created successfully (username: testuser, password: password)")
	} else {
		logger.Info("Test user already exists, skipping creation")
	}

	logger.Info("Data seeding completed successfully")
	return nil
}

// RunMigrations 运行所有迁移
func RunMigrations(db *gorm.DB) error {
	logger.Info("Starting migrations...")

	// 自动迁移表结构
	logger.Info("Running AutoMigrate...")
	if err := AutoMigrate(db); err != nil {
		logger.Errorf("AutoMigrate failed: %v", err)
		return err
	}
	logger.Info("AutoMigrate completed successfully")

	// 创建索引
	logger.Info("Running CreateIndexes...")
	if err := CreateIndexes(db); err != nil {
		logger.Errorf("CreateIndexes failed: %v", err)
		return err
	}
	logger.Info("CreateIndexes completed successfully")

	// 初始化种子数据
	logger.Info("Running SeedData...")
	if err := SeedData(db); err != nil {
		logger.Errorf("SeedData failed: %v", err)
		return err
	}
	logger.Info("SeedData completed successfully")

	logger.Info("All migrations completed successfully")
	return nil
}
