package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// ========== 配置 ==========

// JWT 密钥（生产环境应从环境变量读取）
var jwtSecret = []byte(getEnv("JWT_SECRET", "image-group-secret-key-2026"))

// getEnv 获取环境变量，不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ========== 数据模型 ==========

// User 用户模型
type User struct {
	ID        bson.ObjectID `json:"id" bson:"_id,omitempty"`
	Username  string        `json:"username" bson:"username"`
	Password  string        `json:"-" bson:"password"` // json:"-" 表示不返回给前端
	Role      string        `json:"role" bson:"role"`  // "user" 或 "admin"
	CreatedAt time.Time     `json:"created_at" bson:"created_at"`
}

// FileRecord 文件上传记录
type FileRecord struct {
	ID         bson.ObjectID `json:"id" bson:"_id,omitempty"`
	Filename   string        `json:"filename" bson:"filename"`
	StoreName  string        `json:"store_name" bson:"store_name"`
	Size       int64         `json:"size" bson:"size"`
	UploaderID bson.ObjectID `json:"uploader_id" bson:"uploader_id"`
	Uploader   string        `json:"uploader" bson:"uploader"` // 上传者用户名（方便显示）
	UploadTime time.Time     `json:"upload_time" bson:"upload_time"`
}

// ========== 请求结构 ==========

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// ========== 全局变量 ==========

var (
	mongoClient    *mongo.Client
	redisClient    *redis.Client
	userCollection *mongo.Collection
	fileCollection *mongo.Collection
)

// ========== 主函数 ==========

func main() {
	initMongoDB()
	initRedis()
	createDefaultAdmin() // 创建默认管理员账号

	r := gin.Default()

	// 配置跨域
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
	}))

	// ===== 公开接口（不需要登录） =====
	api := r.Group("/api")
	{
		api.POST("/register", register)
		api.POST("/login", login)
		api.GET("/health", healthCheck)
	}

	// ===== 用户接口（需要登录） =====
	auth := r.Group("/api")
	auth.Use(authMiddleware())
	{
		auth.GET("/profile", getProfile)
		auth.POST("/upload", uploadFile)
		auth.GET("/files", getFiles)
		auth.GET("/download/:id", downloadFile)
		auth.DELETE("/files/:id", deleteFile)
	}

	// ===== 管理员接口（需要管理员权限） =====
	admin := r.Group("/api/admin")
	admin.Use(authMiddleware(), adminMiddleware())
	{
		admin.GET("/users", adminGetUsers)
		admin.DELETE("/users/:id", adminDeleteUser)
		admin.GET("/files", adminGetFiles)
		admin.DELETE("/files/:id", adminDeleteFile)
		admin.GET("/stats", adminGetStats)
	}

	// 提供上传文件的静态访问
	r.Static("/uploads", "./uploads")

	fmt.Println("========================================")
	fmt.Println("  后端服务启动成功!")
	fmt.Println("  地址: http://localhost:8080")
	fmt.Println("  默认管理员: admin / admin123")
	fmt.Println("========================================")

	r.Run(":8080")
}

// ========== 初始化 ==========

func initMongoDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")
	client, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("MongoDB 连接失败:", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("MongoDB Ping 失败:", err)
	}

	mongoClient = client
	db := client.Database("imagegroup")
	userCollection = db.Collection("users")
	fileCollection = db.Collection("files")

	// 创建用户名唯一索引
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	userCollection.Indexes().CreateOne(ctx, indexModel)

	fmt.Println("[OK] MongoDB 连接成功")
}

func initRedis() {
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisClient = redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})

	ctx := context.Background()
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Redis 连接失败:", err)
	}

	fmt.Println("[OK] Redis 连接成功")
}

// 创建默认管理员账号（如果不存在）
func createDefaultAdmin() {
	ctx := context.Background()
	var existing User
	err := userCollection.FindOne(ctx, bson.M{"username": "admin"}).Decode(&existing)
	if err == nil {
		return // 已存在，不重复创建
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	admin := User{
		Username:  "admin",
		Password:  string(hashedPassword),
		Role:      "admin",
		CreatedAt: time.Now(),
	}
	userCollection.InsertOne(ctx, admin)
	fmt.Println("[OK] 默认管理员账号已创建: admin / admin123")
}

// ========== JWT 工具函数 ==========

// 生成 JWT Token
func generateToken(user User) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  user.ID.Hex(),
		"username": user.Username,
		"role":     user.Role,
		"exp":      time.Now().Add(24 * time.Hour).Unix(), // 24 小时过期
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// 解析 JWT Token
func parseToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("无效的 Token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("无效的 Token")
	}
	return claims, nil
}

// ========== 中间件 ==========

// 登录鉴权中间件
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "message": "请先登录"})
			c.Abort()
			return
		}

		// Token 格式: "Bearer xxxxx"
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := parseToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 1, "message": "登录已过期，请重新登录"})
			c.Abort()
			return
		}

		// 把用户信息存到请求上下文中，后续接口可以直接使用
		c.Set("user_id", claims["user_id"].(string))
		c.Set("username", claims["username"].(string))
		c.Set("role", claims["role"].(string))
		c.Next()
	}
}

// 管理员权限中间件
func adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"code": 1, "message": "无权限，仅管理员可操作"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// ========== 公开接口 ==========

// 注册
func register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "请输入用户名和密码"})
		return
	}

	// 验证用户名长度
	if len(req.Username) < 2 || len(req.Username) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "用户名长度需要 2-20 个字符"})
		return
	}

	// 验证密码长度
	if len(req.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "密码至少 6 个字符"})
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "系统错误"})
		return
	}

	// 创建用户
	user := User{
		Username:  req.Username,
		Password:  string(hashedPassword),
		Role:      "user", // 注册的都是普通用户
		CreatedAt: time.Now(),
	}

	ctx := context.Background()
	_, err = userCollection.InsertOne(ctx, user)
	if err != nil {
		// 用户名重复（唯一索引冲突）
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "用户名已存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "注册成功"})
}

// 登录
func login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "请输入用户名和密码"})
		return
	}

	ctx := context.Background()
	var user User
	err := userCollection.FindOne(ctx, bson.M{"username": req.Username}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "用户名或密码错误"})
		return
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "用户名或密码错误"})
		return
	}

	// 生成 Token
	token, err := generateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "系统错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "登录成功",
		"data": gin.H{
			"token":    token,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

// ========== 用户接口（需要登录） ==========

// 获取当前用户信息
func getProfile(c *gin.Context) {
	userID := c.GetString("user_id")
	ctx := context.Background()

	objectID, _ := bson.ObjectIDFromHex(userID)
	var user User
	err := userCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "用户不存在"})
		return
	}

	// 统计该用户的文件数量和总大小
	cursor, _ := fileCollection.Find(ctx, bson.M{"uploader_id": objectID})
	var files []FileRecord
	cursor.All(ctx, &files)

	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "查询成功",
		"data": gin.H{
			"id":         user.ID,
			"username":   user.Username,
			"role":       user.Role,
			"created_at": user.CreatedAt,
			"file_count": len(files),
			"total_size": totalSize,
		},
	})
}

// 上传文件
func uploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "请选择要上传的文件"})
		return
	}

	// 获取当前登录用户信息
	userID := c.GetString("user_id")
	username := c.GetString("username")
	uploaderID, _ := bson.ObjectIDFromHex(userID)

	// 生成唯一文件名
	ext := filepath.Ext(file.Filename)
	newFilename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)

	// 保存文件到 uploads 目录
	savePath := filepath.Join("./uploads", newFilename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "文件保存失败"})
		return
	}

	// 文件信息存入 MongoDB（包含上传者信息）
	record := FileRecord{
		Filename:   file.Filename,
		StoreName:  newFilename,
		Size:       file.Size,
		UploaderID: uploaderID,
		Uploader:   username,
		UploadTime: time.Now(),
	}

	ctx := context.Background()
	_, err = fileCollection.InsertOne(ctx, record)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "数据库写入失败"})
		return
	}

	// 清除缓存
	redisClient.Del(ctx, "file:list:"+userID)
	redisClient.Incr(ctx, "upload:count")

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "上传成功",
		"data": gin.H{
			"filename": file.Filename,
			"size":     file.Size,
		},
	})
}

// 获取文件列表（只返回当前用户的文件）
func getFiles(c *gin.Context) {
	userID := c.GetString("user_id")
	uploaderID, _ := bson.ObjectIDFromHex(userID)
	ctx := context.Background()

	// 查 Redis 缓存
	cacheKey := "file:list:" + userID
	cached, err := redisClient.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		c.Header("X-Cache", "HIT")
		c.Data(http.StatusOK, "application/json", []byte(cached))
		return
	}

	// 查 MongoDB（只查当前用户的文件）
	cursor, err := fileCollection.Find(ctx, bson.M{"uploader_id": uploaderID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败"})
		return
	}

	var files []FileRecord
	if err := cursor.All(ctx, &files); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "数据解析失败"})
		return
	}

	if files == nil {
		files = []FileRecord{}
	}

	response := gin.H{"code": 0, "message": "查询成功", "data": files}

	// 缓存 60 秒
	jsonBytes, _ := json.Marshal(response)
	redisClient.Set(ctx, cacheKey, string(jsonBytes), 60*time.Second)

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, response)
}

// 下载文件
func downloadFile(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	role := c.GetString("role")
	ctx := context.Background()

	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的文件ID"})
		return
	}

	var record FileRecord
	err = fileCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&record)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "文件不存在"})
		return
	}

	// 普通用户只能下载自己的文件，管理员可以下载所有文件
	if role != "admin" && record.UploaderID.Hex() != userID {
		c.JSON(http.StatusForbidden, gin.H{"code": 1, "message": "无权下载此文件"})
		return
	}

	filePath := filepath.Join("./uploads", record.StoreName)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "文件已丢失"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, record.Filename))
	c.File(filePath)
}

// 删除文件（只能删除自己的文件）
func deleteFile(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("user_id")
	ctx := context.Background()

	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的文件ID"})
		return
	}

	// 查找文件记录
	var record FileRecord
	err = fileCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&record)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "文件不存在"})
		return
	}

	// 只能删除自己的文件
	if record.UploaderID.Hex() != userID {
		c.JSON(http.StatusForbidden, gin.H{"code": 1, "message": "无权删除此文件"})
		return
	}

	// 删除磁盘文件
	if record.StoreName != "" {
		filePath := filepath.Join("./uploads", record.StoreName)
		os.Remove(filePath)
	}

	// 删除数据库记录
	fileCollection.DeleteOne(ctx, bson.M{"_id": objectID})

	// 清除缓存
	redisClient.Del(ctx, "file:list:"+userID)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// ========== 管理员接口 ==========

// 获取所有用户
func adminGetUsers(c *gin.Context) {
	ctx := context.Background()
	cursor, err := userCollection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败"})
		return
	}

	var users []User
	if err := cursor.All(ctx, &users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "数据解析失败"})
		return
	}

	if users == nil {
		users = []User{}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "查询成功", "data": users})
}

// 删除用户（及其所有文件）
func adminDeleteUser(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()

	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的用户ID"})
		return
	}

	// 不能删除自己
	currentUserID := c.GetString("user_id")
	if id == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "不能删除自己"})
		return
	}

	// 删除该用户的所有文件（先删磁盘文件）
	cursor, _ := fileCollection.Find(ctx, bson.M{"uploader_id": objectID})
	var files []FileRecord
	cursor.All(ctx, &files)
	for _, f := range files {
		if f.StoreName != "" {
			os.Remove(filepath.Join("./uploads", f.StoreName))
		}
	}

	// 删除该用户的所有文件记录
	fileCollection.DeleteMany(ctx, bson.M{"uploader_id": objectID})

	// 删除用户
	userCollection.DeleteOne(ctx, bson.M{"_id": objectID})

	// 清除该用户的缓存
	redisClient.Del(ctx, "file:list:"+id)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "用户及其文件已删除"})
}

// 获取所有文件（管理员）
func adminGetFiles(c *gin.Context) {
	ctx := context.Background()
	cursor, err := fileCollection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "查询失败"})
		return
	}

	var files []FileRecord
	if err := cursor.All(ctx, &files); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": "数据解析失败"})
		return
	}

	if files == nil {
		files = []FileRecord{}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "查询成功", "data": files})
}

// 管理员删除任意文件
func adminDeleteFile(c *gin.Context) {
	id := c.Param("id")
	ctx := context.Background()

	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1, "message": "无效的文件ID"})
		return
	}

	var record FileRecord
	err = fileCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&record)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "文件不存在"})
		return
	}

	// 删除磁盘文件
	if record.StoreName != "" {
		os.Remove(filepath.Join("./uploads", record.StoreName))
	}

	// 删除数据库记录
	fileCollection.DeleteOne(ctx, bson.M{"_id": objectID})

	// 清除上传者的缓存
	redisClient.Del(ctx, "file:list:"+record.UploaderID.Hex())

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "文件已删除"})
}

// 管理员统计数据
func adminGetStats(c *gin.Context) {
	ctx := context.Background()

	// 用户总数
	userCount, _ := userCollection.CountDocuments(ctx, bson.M{})

	// 文件总数
	fileCount, _ := fileCollection.CountDocuments(ctx, bson.M{})

	// 总存储大小
	cursor, _ := fileCollection.Find(ctx, bson.M{})
	var files []FileRecord
	cursor.All(ctx, &files)

	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}

	// 上传次数
	uploadCount, _ := redisClient.Get(ctx, "upload:count").Result()

	// MongoDB 状态
	mongoOK := "正常"
	if err := mongoClient.Ping(ctx, nil); err != nil {
		mongoOK = "异常: " + err.Error()
	}

	// Redis 状态
	redisOK := "正常"
	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		redisOK = "异常: " + err.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "查询成功",
		"data": gin.H{
			"user_count":   userCount,
			"file_count":   fileCount,
			"total_size":   totalSize,
			"upload_count": uploadCount,
			"mongodb":      mongoOK,
			"redis":        redisOK,
		},
	})
}

// 健康检查（公开接口）
func healthCheck(c *gin.Context) {
	ctx := context.Background()

	mongoOK := "正常"
	if err := mongoClient.Ping(ctx, nil); err != nil {
		mongoOK = "异常: " + err.Error()
	}

	redisOK := "正常"
	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		redisOK = "异常: " + err.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "服务正常",
		"data": gin.H{
			"mongodb": mongoOK,
			"redis":   redisOK,
		},
	})
}
