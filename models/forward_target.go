package models

import (
	"gorm.io/gorm"
	"time"
)

// ForwardTarget 转发目标表
type ForwardTarget struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Name      string         `gorm:"size:100;not null;index" json:"name"`           // 转发对象名字
	Email     string         `gorm:"size:255;not null;index" json:"email"`          // 转发目标邮箱
	Keywords  string         `gorm:"type:text" json:"keywords"`                     // 关联的关键字，用逗号分隔
	IsActive  bool           `gorm:"default:true" json:"is_active"`                 // 是否启用
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ForwardTarget) TableName() string {
	return "forward_targets"
}