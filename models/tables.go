package models

import (
	"gorm.io/gorm"
)

// User 用户表：存储钱包地址和登录用的随机数
type User struct {
	Address string `gorm:"primaryKey;type:char(42)" json:"address"` // 设置为主键，长度42
	Nonce   string `gorm:"type:varchar(255)" json:"nonce"`          // 随机挑战码
}

// File 文件表：存储 IPFS CID 和元数据
type File struct {
	gorm.Model          // 自动包含 ID, CreatedAt, UpdatedAt, DeletedAt
	Cid          string `gorm:"not null" json:"cid"`
	Filename     string `json:"filename"`
	Size         int64  `json:"size"`
	OwnerAddress string `gorm:"index;type:char(42)" json:"owner_address"` // 建立索引方便查询
}
