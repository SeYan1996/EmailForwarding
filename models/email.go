package models

import (
	"gorm.io/gorm"
	"time"
)

// EmailLog 邮件处理记录表
type EmailLog struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	GmailMessageID string         `gorm:"size:100;not null;uniqueIndex" json:"gmail_message_id"` // Gmail消息ID
	Subject        string         `gorm:"size:500;not null" json:"subject"`                      // 邮件主题
	FromEmail      string         `gorm:"size:255;not null" json:"from_email"`                   // 发件人
	ToEmail        string         `gorm:"size:255;not null" json:"to_email"`                     // 收件人
	Content        string         `gorm:"type:longtext" json:"content"`                          // 邮件内容
	Keyword        string         `gorm:"size:100" json:"keyword"`                               // 匹配的关键字
	ForwardTarget  string         `gorm:"size:100" json:"forward_target"`                        // 转发目标名字
	ForwardEmail   string         `gorm:"size:255" json:"forward_email"`                         // 转发目标邮箱
	ForwardStatus  string         `gorm:"size:50;default:'pending'" json:"forward_status"`       // 转发状态：pending/success/failed
	ErrorMessage   string         `gorm:"type:text" json:"error_message"`                        // 错误信息
	ProcessedAt    *time.Time     `json:"processed_at"`                                          // 处理时间
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (EmailLog) TableName() string {
	return "email_logs"
}

// ForwardStatus 转发状态常量
const (
	StatusPending = "pending"
	StatusSuccess = "success"
	StatusFailed  = "failed"
)