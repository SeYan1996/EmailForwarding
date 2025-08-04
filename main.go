package main

import (
	"email-forwarding/config"
	"email-forwarding/database"
	"email-forwarding/handlers"
	"email-forwarding/services"
	"email-forwarding/utils"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	if err := godotenv.Load(); err != nil {
		log.Println("没有找到.env文件，使用默认配置")
	}

	// 初始化日志
	utils.InitLogger()
	logger := utils.GetLogger()

	// 加载配置
	cfg := config.LoadConfig()

	// 初始化数据库
	if err := database.InitDatabase(cfg); err != nil {
		logger.Fatalf("数据库初始化失败: %v", err)
	}

	// 创建默认转发目标
	if err := database.CreateDefaultForwardTargets(); err != nil {
		logger.Errorf("创建默认转发目标失败: %v", err)
	}

	// 初始化Gmail服务
	gmailService, err := services.NewGmailService(
		cfg.Gmail.CredentialsFile,
		cfg.Gmail.TokenFile,
		cfg.Gmail.UserEmail,
	)
	if err != nil {
		logger.Fatalf("Gmail服务初始化失败: %v", err)
	}

	// 初始化邮件服务
	emailService := services.NewEmailService(gmailService)

	// 启动定时任务
	go startScheduler(emailService, cfg.App.CheckInterval)

	// 设置Gin模式
	gin.SetMode(cfg.Server.Mode)

	// 创建路由
	router := setupRoutes(emailService)

	// 启动服务器
	logger.Infof("服务器启动在端口 %s", cfg.Server.Port)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		logger.Fatalf("服务器启动失败: %v", err)
	}
}

// setupRoutes 设置路由
func setupRoutes(emailService *services.EmailService) *gin.Engine {
	router := gin.Default()

	// 创建处理器
	emailHandler := handlers.NewEmailHandler(emailService)

	// 添加CORS中间件
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// API路由组
	api := router.Group("/api/v1")
	{
		// 邮件处理相关
		api.POST("/emails/process", emailHandler.ProcessEmails)
		api.GET("/emails/logs", emailHandler.GetEmailLogs)
		api.GET("/stats", emailHandler.GetStats)

		// 转发目标管理
		targets := api.Group("/targets")
		{
			targets.GET("", emailHandler.GetForwardTargets)
			targets.POST("", emailHandler.CreateForwardTarget)
			targets.PUT("/:id", emailHandler.UpdateForwardTarget)
			targets.DELETE("/:id", emailHandler.DeleteForwardTarget)
		}
	}

	// 健康检查
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"message": "邮件转发系统运行正常",
			"timestamp": time.Now().Unix(),
		})
	})

	// 首页
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "欢迎使用邮件转发系统",
			"version": "1.0.0",
			"endpoints": gin.H{
				"health": "/health",
				"api": "/api/v1",
				"process_emails": "/api/v1/emails/process",
				"email_logs": "/api/v1/emails/logs",
				"targets": "/api/v1/targets",
			},
		})
	})

	return router
}

// startScheduler 启动定时任务
func startScheduler(emailService *services.EmailService, interval time.Duration) {
	logger := utils.GetLogger()
	logger.Infof("定时任务已启动，检查间隔: %v", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			logger.Info("开始定时检查邮件...")
			
			if err := emailService.ProcessEmails(); err != nil {
				logger.Errorf("定时处理邮件失败: %v", err)
			} else {
				logger.Info("定时邮件检查完成")
			}
		}
	}
}