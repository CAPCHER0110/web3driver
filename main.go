package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"web3driver/config"
	"web3driver/middleware"
	"web3driver/models"
	"web3driver/utils"
)

// 全局数据库对象
var db *gorm.DB

func main() {
	// 0. 加载配置
	config.LoadConfig()

	// 1. 初始化数据库连接
	initDB()

	// 2. 初始化 Gin 引擎
	r := gin.Default()

	// 设置上传文件大小限制 (例如 50MB)
	r.MaxMultipartMemory = 50 << 20

	// 3. 公开接口 (无需鉴权)
	// 获取登录挑战码
	r.GET("/auth/nonce", getNonce)
	// 提交签名进行登录
	r.POST("/auth/login", login)

	// 4. 受保护接口 (需要 JWT 鉴权)
	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware()) // 挂载中间件
	{
		// 上传文件 (存到 IPFS)
		api.POST("/upload", uploadFile)
		// 获取我的文件列表
		api.GET("/files", listFiles)
	}

	port := ":" + config.AppConfig.ServerPort
	fmt.Printf("🚀 D-Drive Server running on port %s\n", config.AppConfig.ServerPort)
	// 启动服务
	if err := r.Run(port); err != nil {
		panic("无法启动服务器: " + err.Error())
	}
}

// ---------------------------------------------------------
// 数据库初始化逻辑
// ---------------------------------------------------------
func initDB() {
	// 使用配置文件里的 DSN
	dsn := config.AppConfig.DBDsn

	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("❌ 数据库连接失败: " + err.Error())
	}

	// 自动迁移表结构 (引用 models 包中的结构体)
	err = db.AutoMigrate(&models.User{}, &models.File{})
	if err != nil {
		panic("❌ 数据表迁移失败: " + err.Error())
	}

	fmt.Println("✅ 数据库连接 & 表结构迁移成功")
}

// ---------------------------------------------------------
// 业务处理逻辑 (Handlers)
// ---------------------------------------------------------

// 1. 获取 Nonce
// 前端传入 ?address=0x...
func getNonce(c *gin.Context) {
	address := c.Query("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address is required"})
		return
	}

	// 调用 utils 生成随机字符串
	// 格式示例: "Login to D-Drive: 8a7b9c..."
	nonceMsg := fmt.Sprintf("Login to D-Drive: %s", utils.GenerateNonce())

	// Upsert: 如果用户存在则更新 nonce，不存在则创建用户
	// 使用 models.User
	user := models.User{Address: address, Nonce: nonceMsg}
	if err := db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"nonce": nonceMsg})
}

// 2. 登录 (验签换 Token)
func login(c *gin.Context) {
	// 定义请求参数结构
	var req struct {
		Address   string `json:"address"`
		Signature string `json:"signature"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// A. 查库获取该用户当前的 Nonce
	var user models.User
	if err := db.First(&user, "address = ?", req.Address).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found or nonce expired. Please call /auth/nonce first."})
		return
	}

	// B. 验证签名 (调用 utils 包)
	// 核心逻辑: Verify(地址, 消息, 签名)
	if !utils.VerifySignature(req.Address, user.Nonce, req.Signature) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Signature verification failed"})
		return
	}

	// C. 验证通过，颁发 JWT (调用 middleware 包)
	token, err := middleware.GenerateJWT(req.Address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	// D. 销毁 Nonce (防重放攻击)
	// 将 Nonce 置空，下次登录必须重新请求 /auth/nonce
	db.Model(&user).Update("nonce", "")

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"msg":   "Login successful",
	})
}

// 3. 上传文件
func uploadFile(c *gin.Context) {
	// 从 JWT Context 中获取当前用户地址 (由 AuthMiddleware 注入)
	userAddress := c.GetString("user_address")
	if userAddress == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User context missing"})
		return
	}

	// 获取上传的文件
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	// A. 创建临时目录并保存文件
	tempDir := "./tmp"
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		os.Mkdir(tempDir, os.ModePerm)
	}

	tempPath := filepath.Join(tempDir, fileHeader.Filename)
	if err := c.SaveUploadedFile(fileHeader, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save temp file"})
		return
	}
	// 函数结束时删除临时文件
	defer os.Remove(tempPath)

	// B. 上传到 IPFS (调用 utils 包)
	cid, err := utils.UploadToIPFS(tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "IPFS upload failed: " + err.Error()})
		return
	}

	// C. 存入 MySQL (仅存储元数据)
	newFile := models.File{
		Cid:          cid,
		Filename:     fileHeader.Filename,
		Size:         fileHeader.Size,
		OwnerAddress: userAddress,
	}

	if err := db.Create(&newFile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"msg": "File uploaded successfully",
		"cid": cid,
		"url": "https://gateway.pinata.cloud/ipfs/" + cid,
	})
}

// 4. 获取文件列表
func listFiles(c *gin.Context) {
	userAddress := c.GetString("user_address")

	var files []models.File
	// 查询属于该地址的所有文件，按时间倒序
	if err := db.Where("owner_address = ?", userAddress).Order("created_at desc").Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database query error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}
