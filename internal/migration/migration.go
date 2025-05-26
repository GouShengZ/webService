package migration

import (
	"strings"
	"webservice/internal/logger"
	"webservice/internal/models"

	"gorm.io/gorm"
)

// AutoMigrate 自动迁移数据库表结构
func AutoMigrate(db *gorm.DB) error {
	logger.Info("Starting database migration...")

	// 定义需要迁移的模型
	modelsToMigrate := []interface{}{
		&models.User{},
		// 在这里添加其他模型
	}

	// 执行自动迁移
	for _, model := range modelsToMigrate {
		if err := db.AutoMigrate(model); err != nil {
			logger.Errorf("Failed to migrate model %T: %v", model, err)
			return err
		}
		logger.Infof("Successfully migrated model: %T", model)
	}

	logger.Info("Database migration completed successfully")
	return nil
}

// CreateIndexes 创建数据库索引
func CreateIndexes(db *gorm.DB) error {
	logger.Info("Creating database indexes...")

	// 用户表索引
	indexes := []struct {
		table string
		name  string
		sql   string
	}{
		{
			table: "users",
			name:  "idx_users_username",
			sql:   "CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)",
		},
		{
			table: "users",
			name:  "idx_users_email",
			sql:   "CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)",
		},
		{
			table: "users",
			name:  "idx_users_role",
			sql:   "CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)",
		},
		{
			table: "users",
			name:  "idx_users_status",
			sql:   "CREATE INDEX IF NOT EXISTS idx_users_status ON users(status)",
		},
		{
			table: "users",
			name:  "idx_users_created_at",
			sql:   "CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at)",
		},
	}

	for _, index := range indexes {
		// 检查索引是否存在
		var count int64
		db.Raw("SELECT COUNT(1) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = ? AND index_name = ?", index.table, index.name).Scan(&count)

		if count == 0 {
			// 索引不存在，创建索引
			// 注意：这里的 CREATE INDEX 语句可能需要根据实际的列名进行调整
			// 例如，对于 idx_users_username，SQL 应该是 CREATE INDEX idx_users_username ON users(username)
			// 这里我们假设 index.sql 已经是正确的 CREATE INDEX 语句，只是去掉了 IF NOT EXISTS
			createSql := strings.Replace(index.sql, "IF NOT EXISTS ", "", 1)
			if err := db.Exec(createSql).Error; err != nil {
				logger.Errorf("Failed to create index %s on table %s: %v", index.name, index.table, err)
				return err
			}
			logger.Infof("Successfully created index: %s on table %s", index.name, index.table)
		} else {
			logger.Infof("Index %s on table %s already exists", index.name, index.table)
		}
	}

	logger.Info("Database indexes created successfully")
	return nil
}

// SeedData 初始化种子数据
func SeedData(db *gorm.DB) error {
	logger.Info("Seeding initial data...")

	// 检查是否已存在管理员用户
	var adminCount int64
	db.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&adminCount)

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
	db.Model(&models.User{}).Where("role = ? AND username = ?", models.RoleUser, "testuser").Count(&userCount)

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
	// 自动迁移表结构
	if err := AutoMigrate(db); err != nil {
		return err
	}

	// 创建索引
	if err := CreateIndexes(db); err != nil {
		return err
	}

	// 初始化种子数据
	if err := SeedData(db); err != nil {
		return err
	}

	return nil
}
