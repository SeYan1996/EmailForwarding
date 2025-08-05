package services

import (
	"email-forwarding/database"
	"email-forwarding/models"
	"email-forwarding/utils"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type EmailService struct {
	gmailService *GmailService
}

// NewEmailService 创建邮件服务实例
func NewEmailService(gmailService *GmailService) *EmailService {
	return &EmailService{
		gmailService: gmailService,
	}
}

// ProcessEmails 处理邮件
func (es *EmailService) ProcessEmails() error {
	logger := utils.GetLogger()
	
	// 获取未读邮件（使用配置的数量限制）
	emails, err := es.gmailService.GetUnreadEmails()
	if err != nil {
		return fmt.Errorf("获取未读邮件失败: %v", err)
	}

	logger.Infof("获取到 %d 封未读邮件", len(emails))

	for _, email := range emails {
		if err := es.processEmail(email); err != nil {
			logger.Errorf("处理邮件失败 [%s]: %v", email.ID, err)
		}
	}

	return nil
}

// processEmail 处理单封邮件
func (es *EmailService) processEmail(email *EmailMessage) error {
	logger := utils.GetLogger()
	db := database.GetDB()

	// 检查邮件是否已处理
	var existingLog models.EmailLog
	if err := db.Where("gmail_message_id = ?", email.ID).First(&existingLog).Error; err == nil {
		logger.Infof("邮件 [%s] 已处理，跳过", email.ID)
		return nil
	}

	// 创建邮件日志记录
	emailLog := models.EmailLog{
		GmailMessageID: email.ID,
		Subject:        email.Subject,
		FromEmail:      email.From,
		ToEmail:        email.To,
		Content:        email.Body,
		ForwardStatus:  models.StatusPending,
	}

	// 解析邮件标题，提取关键字和转发目标
	keyword, targetName := es.parseEmailSubject(email.Subject)
	
	if keyword == "" || targetName == "" {
		// 不符合转发规则，标记邮件为已读但不转发
		emailLog.ForwardStatus = models.StatusFailed
		emailLog.ErrorMessage = "邮件标题不符合转发规则"
		
		if err := db.Create(&emailLog).Error; err != nil {
			return fmt.Errorf("保存邮件记录失败: %v", err)
		}

		// 标记为已读
		if err := es.gmailService.MarkAsRead(email.ID); err != nil {
			logger.Errorf("标记邮件为已读失败: %v", err)
		}
		
		logger.Infof("邮件 [%s] 不符合转发规则，已跳过", email.ID)
		return nil
	}

	emailLog.Keyword = keyword
	emailLog.ForwardTarget = targetName

	// 查找转发目标
	target, err := es.findForwardTarget(keyword, targetName)
	if err != nil {
		emailLog.ForwardStatus = models.StatusFailed
		emailLog.ErrorMessage = fmt.Sprintf("查找转发目标失败: %v", err)
		
		if err := db.Create(&emailLog).Error; err != nil {
			return fmt.Errorf("保存邮件记录失败: %v", err)
		}
		
		return fmt.Errorf("查找转发目标失败: %v", err)
	}

	emailLog.ForwardEmail = target.Email

	// 转发邮件
	if err := es.forwardEmail(email, target); err != nil {
		emailLog.ForwardStatus = models.StatusFailed
		emailLog.ErrorMessage = fmt.Sprintf("转发邮件失败: %v", err)
		
		logger.Errorf("转发邮件失败 [%s]: %v", email.ID, err)
	} else {
		emailLog.ForwardStatus = models.StatusSuccess
		now := time.Now()
		emailLog.ProcessedAt = &now
		
		logger.Infof("邮件 [%s] 转发成功到 %s", email.ID, target.Email)
	}

	// 保存处理记录
	if err := db.Create(&emailLog).Error; err != nil {
		return fmt.Errorf("保存邮件记录失败: %v", err)
	}

	// 标记邮件为已读
	if err := es.gmailService.MarkAsRead(email.ID); err != nil {
		logger.Errorf("标记邮件为已读失败: %v", err)
	}

	return nil
}

// parseEmailSubject 解析邮件标题
func (es *EmailService) parseEmailSubject(subject string) (keyword, targetName string) {
	// 标题格式：指定关键字 - 转发对象名字
	// 使用正则表达式匹配
	re := regexp.MustCompile(`^(.+?)\s*-\s*(.+?)$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(subject))
	
	if len(matches) == 3 {
		keyword = strings.TrimSpace(matches[1])
		targetName = strings.TrimSpace(matches[2])
	}
	
	return keyword, targetName
}

// findForwardTarget 查找转发目标
func (es *EmailService) findForwardTarget(keyword, targetName string) (*models.ForwardTarget, error) {
	db := database.GetDB()
	
	var target models.ForwardTarget
	
	// 首先根据名字精确匹配
	if err := db.Where("name = ? AND is_active = ?", targetName, true).First(&target).Error; err == nil {
		// 验证关键字是否匹配
		if es.matchKeyword(keyword, target.Keywords) {
			return &target, nil
		}
	}

	// 如果名字匹配失败，尝试根据关键字模糊匹配
	var targets []models.ForwardTarget
	if err := db.Where("is_active = ?", true).Find(&targets).Error; err != nil {
		return nil, fmt.Errorf("查询转发目标失败: %v", err)
	}

	for _, t := range targets {
		if es.matchKeyword(keyword, t.Keywords) && strings.Contains(strings.ToLower(t.Name), strings.ToLower(targetName)) {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("未找到匹配的转发目标，关键字: %s, 目标名字: %s", keyword, targetName)
}

// matchKeyword 匹配关键字
func (es *EmailService) matchKeyword(keyword, targetKeywords string) bool {
	if targetKeywords == "" {
		return false
	}

	keywords := strings.Split(targetKeywords, ",")
	for _, k := range keywords {
		k = strings.TrimSpace(k)
		if strings.Contains(strings.ToLower(keyword), strings.ToLower(k)) {
			return true
		}
	}
	
	return false
}

// forwardEmail 转发邮件
func (es *EmailService) forwardEmail(email *EmailMessage, target *models.ForwardTarget) error {
	// 构建转发邮件的主题和内容
	forwardSubject := fmt.Sprintf("[转发] %s", email.Subject)
	
	forwardBody := fmt.Sprintf(`
		<div style="border-left: 4px solid #ccc; padding-left: 10px; margin: 10px 0;">
			<h3>原邮件信息</h3>
			<p><strong>发件人:</strong> %s</p>
			<p><strong>收件人:</strong> %s</p>
			<p><strong>主题:</strong> %s</p>
			<p><strong>时间:</strong> %s</p>
			<hr style="margin: 10px 0;">
			<div>
				%s
			</div>
		</div>
		<br>
		<p style="font-size: 12px; color: #666;">此邮件由邮件转发系统自动转发</p>
	`,
		email.From,
		email.To,
		email.Subject,
		email.ReceivedAt.Format("2006-01-02 15:04:05"),
		email.Body,
	)

	return es.gmailService.SendEmail(target.Email, forwardSubject, forwardBody)
}

// GetEmailLogs 获取邮件处理日志
func (es *EmailService) GetEmailLogs(page, pageSize int, status string) ([]models.EmailLog, int64, error) {
	db := database.GetDB()
	
	var logs []models.EmailLog
	var total int64
	
	query := db.Model(&models.EmailLog{})
	
	if status != "" {
		query = query.Where("forward_status = ?", status)
	}
	
	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	
	return logs, total, nil
}

// GetForwardTargets 获取转发目标列表
func (es *EmailService) GetForwardTargets() ([]models.ForwardTarget, error) {
	db := database.GetDB()
	
	var targets []models.ForwardTarget
	if err := db.Where("is_active = ?", true).Find(&targets).Error; err != nil {
		return nil, err
	}
	
	return targets, nil
}

// CreateForwardTarget 创建转发目标
func (es *EmailService) CreateForwardTarget(target *models.ForwardTarget) error {
	db := database.GetDB()
	
	// 检查邮箱是否已存在
	var existingTarget models.ForwardTarget
	if err := db.Where("email = ?", target.Email).First(&existingTarget).Error; err == nil {
		return fmt.Errorf("邮箱 %s 已存在", target.Email)
	}
	
	return db.Create(target).Error
}

// UpdateForwardTarget 更新转发目标
func (es *EmailService) UpdateForwardTarget(id uint, target *models.ForwardTarget) error {
	db := database.GetDB()
	
	return db.Model(&models.ForwardTarget{}).Where("id = ?", id).Updates(target).Error
}

// DeleteForwardTarget 删除转发目标
func (es *EmailService) DeleteForwardTarget(id uint) error {
	db := database.GetDB()
	
	return db.Delete(&models.ForwardTarget{}, id).Error
}