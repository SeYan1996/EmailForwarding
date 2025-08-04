package handlers

import (
	"email-forwarding/models"
	"email-forwarding/services"
	"email-forwarding/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type EmailHandler struct {
	emailService *services.EmailService
}

// NewEmailHandler 创建邮件处理器
func NewEmailHandler(emailService *services.EmailService) *EmailHandler {
	return &EmailHandler{
		emailService: emailService,
	}
}

// ProcessEmails 手动处理邮件
func (h *EmailHandler) ProcessEmails(c *gin.Context) {
	logger := utils.GetLogger()
	
	if err := h.emailService.ProcessEmails(); err != nil {
		logger.Errorf("处理邮件失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "处理邮件失败",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "邮件处理完成",
	})
}

// GetEmailLogs 获取邮件日志
func (h *EmailHandler) GetEmailLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	logs, total, err := h.emailService.GetEmailLogs(page, pageSize, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取邮件日志失败",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": logs,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetForwardTargets 获取转发目标列表
func (h *EmailHandler) GetForwardTargets(c *gin.Context) {
	targets, err := h.emailService.GetForwardTargets()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取转发目标失败",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": targets,
	})
}

// CreateForwardTarget 创建转发目标
func (h *EmailHandler) CreateForwardTarget(c *gin.Context) {
	var target models.ForwardTarget
	if err := c.ShouldBindJSON(&target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "参数错误",
			"message": err.Error(),
		})
		return
	}

	// 验证必填字段
	if target.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "名称不能为空",
		})
		return
	}

	if target.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "邮箱不能为空",
		})
		return
	}

	// 设置默认值
	target.IsActive = true

	if err := h.emailService.CreateForwardTarget(&target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建转发目标失败",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "创建成功",
		"data": target,
	})
}

// UpdateForwardTarget 更新转发目标
func (h *EmailHandler) UpdateForwardTarget(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的ID",
		})
		return
	}

	var target models.ForwardTarget
	if err := c.ShouldBindJSON(&target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "参数错误",
			"message": err.Error(),
		})
		return
	}

	if err := h.emailService.UpdateForwardTarget(uint(id), &target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "更新转发目标失败",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "更新成功",
	})
}

// DeleteForwardTarget 删除转发目标
func (h *EmailHandler) DeleteForwardTarget(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的ID",
		})
		return
	}

	if err := h.emailService.DeleteForwardTarget(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除转发目标失败",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "删除成功",
	})
}

// GetStats 获取统计信息
func (h *EmailHandler) GetStats(c *gin.Context) {
	// 这里可以添加统计信息的逻辑
	// 比如：总处理邮件数、成功转发数、失败数等
	
	c.JSON(http.StatusOK, gin.H{
		"message": "统计功能待实现",
	})
}