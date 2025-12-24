package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// 定义配置结构体，方便其他包调用
type Config struct {
	ServerPort string
	DBDsn      string
	JwtSecret  []byte // 转为 []byte 方便 JWT 库直接使用
	PinataJWT  string
}

// 全局配置变量
var AppConfig Config

// LoadConfig 初始化配置
// 在 main 函数最开始调用
func LoadConfig() {
	// 1. 加载 .env 文件
	// 如果文件不存在（比如在生产环境容器里），不会报错，会继续从系统环境变量读
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️  未找到 .env 文件，将尝试从系统环境变量读取")
	}

	// 2. 读取并赋值
	AppConfig = Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		DBDsn:      getEnv("DB_DSN", ""),
		JwtSecret:  []byte(getEnv("JWT_SECRET", "unsafe_default_secret")), // 给个默认值防止 panic
		PinataJWT:  getEnv("PINATA_JWT", ""),
	}

	// 3. 简单的必填项检查
	if AppConfig.DBDsn == "" {
		log.Fatal("❌ 致命错误: 未配置 DB_DSN")
	}
	if len(AppConfig.PinataJWT) == 0 {
		log.Println("⚠️  警告: 未配置 PINATA_JWT，上传功能将不可用")
	}
}

// 辅助函数: 获取环境变量，带默认值
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
