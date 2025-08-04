package database

import (
	"email-forwarding/config"
	"email-forwarding/models"
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDatabase 初始化数据库连接
func InitDatabase(cfg *config.Config) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}

	// 自动迁移数据表
	if err := autoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate database: %v", err)
	}

	log.Println("数据库连接成功")
	return nil
}

// autoMigrate 自动迁移数据表
func autoMigrate() error {
	return DB.AutoMigrate(
		&models.ForwardTarget{},
		&models.EmailLog{},
	)
}

// GetDB 获取数据库连接
func GetDB() *gorm.DB {
	return DB
}

// CreateDefaultForwardTargets 创建默认的转发目标
func CreateDefaultForwardTargets() error {
	var count int64
	if err := DB.Model(&models.ForwardTarget{}).Count(&count).Error; err != nil {
		return err
	}

	// 如果没有数据，创建一些示例数据
	if count == 0 {
		defaultTargets := []models.ForwardTarget{
			{
				Name:     "客服部门",
				Email:    "customer-service@company.com",
				Keywords: "客户,投诉,咨询",
				IsActive: true,
			},
			{
				Name:     "技术支持",
				Email:    "tech-support@company.com",
				Keywords: "技术,故障,bug",
				IsActive: true,
			},
			{
				Name:     "销售部门",
				Email:    "sales@company.com",
				Keywords: "销售,合作,商务",
				IsActive: true,
			},
		}

		for _, target := range defaultTargets {
			if err := DB.Create(&target).Error; err != nil {
				return err
			}
		}

		log.Println("创建默认转发目标成功")
	}

	return nil
}