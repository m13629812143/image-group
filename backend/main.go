package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// getEnv 获取环境变量，不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// FileRecord 文件上传记录
type FileRecord struct {
	ID         bson.ObjectID `json:"id" bson:"_id,omitempty"`
	Filename   string        `json:"filename" bson:"filename"`
	Size       int64         `json:"size" bson:"size"`
	UploadTime time.Time     `json:"upload_time" bson:"upload_time"`
}

var (
	mongoClient    *mongo.Client
	redisClient    *redis.Client
	fileCollection *mongo.Collection
)

func main() {
	// 连接 MongoDB
	initMongoDB()
	// 连接 Redis
	initRedis()

	// 创建 Gin 路由
	r := gin.Default()

	// 配置跨域（允许前端 Vue 访问）
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "DELETE"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	// API 路由
	api := r.Group("/api")
	{
		api.POST("/upload", uploadFile)
		api.GET("/files", getFiles)
		api.DELETE("/files/:filename", deleteFile)
		api.GET("/health", healthCheck)
	}

	// 提供上传文件的静态访问
	r.Static("/uploads", "./uploads")

	fmt.Println("========================================")
	fmt.Println("  后端服务启动成功!")
	fmt.Println("  地址: http://localhost:8080")
	fmt.Println("========================================")

	r.Run(":8080")
}

// 初始化 MongoDB 连接
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
	fileCollection = client.Database("imagegroup").Collection("files")
	fmt.Println("[OK] MongoDB 连接成功")
}

// 初始化 Redis 连接
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

// 上传文件
func uploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "请选择要上传的文件",
		})
		return
	}

	// 生成唯一文件名（防止重名覆盖）
	ext := filepath.Ext(file.Filename)
	newFilename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)

	// 保存文件到 uploads 目录
	savePath := filepath.Join("./uploads", newFilename)
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "文件保存失败",
		})
		return
	}

	// 文件信息存入 MongoDB
	record := FileRecord{
		Filename:   file.Filename,
		Size:       file.Size,
		UploadTime: time.Now(),
	}

	ctx := context.Background()
	_, err = fileCollection.InsertOne(ctx, record)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "数据库写入失败",
		})
		return
	}

	// 清除 Redis 中的文件列表缓存（因为有新文件了）
	redisClient.Del(ctx, "file:list")

	// 记录上传次数到 Redis
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

// 获取文件列表
func getFiles(c *gin.Context) {
	ctx := context.Background()

	// 先查 Redis 缓存
	cached, err := redisClient.Get(ctx, "file:list").Result()
	if err == nil && cached != "" {
		// 缓存命中，直接返回
		c.Header("X-Cache", "HIT")
		c.Data(http.StatusOK, "application/json", []byte(cached))
		return
	}

	// 缓存没有，查 MongoDB
	cursor, err := fileCollection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "查询失败",
		})
		return
	}

	var files []FileRecord
	if err := cursor.All(ctx, &files); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "数据解析失败",
		})
		return
	}

	if files == nil {
		files = []FileRecord{}
	}

	// 构建响应
	response := gin.H{
		"code":    0,
		"message": "查询成功",
		"data":    files,
	}

	// 存入 Redis 缓存，60 秒过期
	jsonBytes, _ := json.Marshal(response)
	redisClient.Set(ctx, "file:list", string(jsonBytes), 60*time.Second)

	c.Header("X-Cache", "MISS")
	c.JSON(http.StatusOK, response)
}

// 删除文件记录
func deleteFile(c *gin.Context) {
	filename := c.Param("filename")
	ctx := context.Background()

	_, err := fileCollection.DeleteOne(ctx, bson.M{"filename": filename})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "删除失败",
		})
		return
	}

	// 清除缓存
	redisClient.Del(ctx, "file:list")

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// 健康检查
func healthCheck(c *gin.Context) {
	ctx := context.Background()

	// 检查 MongoDB
	mongoOK := "正常"
	if err := mongoClient.Ping(ctx, nil); err != nil {
		mongoOK = "异常: " + err.Error()
	}

	// 检查 Redis
	redisOK := "正常"
	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		redisOK = "异常: " + err.Error()
	}

	// 获取上传次数
	count, _ := redisClient.Get(ctx, "upload:count").Result()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "服务正常",
		"data": gin.H{
			"mongodb":      mongoOK,
			"redis":        redisOK,
			"upload_count": count,
		},
	})
}
