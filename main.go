package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 日志数据结构
type LogData struct {
	ApplicationID string `json:"application_id" binding:"required"`
	LogLevel      string `json:"log_level" binding:"required"`
	Timestamp     string `json:"timestamp" binding:"required"`
	LogMessage    string `json:"log_message" binding:"required"`
}

// 日志上传接口
func logUploadHandler(c *gin.Context) {
	var logData LogData

	// 解析请求体中的日志数据
	if err := c.ShouldBindJSON(&logData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format or missing required fields"})
		return
	}

	// 检查并创建对应的文件夹
	appFolder := filepath.Join("logs", logData.ApplicationID)
	err := os.MkdirAll(appFolder, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create application folder"})
		return
	}

	// 根据日期创建日志文件，文件名可以按日期生成
	logFileName := time.Now().Format("2006-01-02") + ".log"
	logFilePath := filepath.Join(appFolder, logFileName)

	// 将日志写入文件
	logEntry := fmt.Sprintf("[%s] [%s]: %s\n", logData.Timestamp, logData.LogLevel, logData.LogMessage)
	err = appendToFile(logFilePath, logEntry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to write log to file"})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{"message": "Log uploaded successfully"})
}

// 查询日志接口
func logQueryHandler(c *gin.Context) {
	// 从查询参数中获取 application_id、log_level 和 limit
	applicationID := c.Query("application_id")
	logLevel := c.Query("log_level")
	limitParam := c.DefaultQuery("limit", "100") // 默认返回100条

	// 解析 limit 参数
	limit, err := strconv.Atoi(limitParam)
	if err != nil || limit <= 0 {
		limit = 100 // 如果 limit 非法，设置默认值
	}

	// 检查参数是否存在
	if applicationID == "" || logLevel == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "application_id and log_level are required"})
		return
	}

	// 获取应用程序日志文件夹
	appFolder := filepath.Join("logs", applicationID)
	files, err := ioutil.ReadDir(appFolder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to read application logs"})
		return
	}

	var logs []LogData

	// 遍历日志文件，读取每个文件的内容
	for _, file := range files {
		if !file.IsDir() {
			logFilePath := filepath.Join(appFolder, file.Name())
			fileLogs, err := readLogsFromFile(logFilePath, logLevel)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Unable to read log file: %s", logFilePath)})
				return
			}

			// 将解析后的日志加入到列表中
			for _, logLine := range fileLogs {
				parsedLog, err := parseLogLine(logLine)
				if err == nil {
					logs = append(logs, parsedLog)
				}
			}
		}
	}

	// 限制返回的日志条目数量
	if len(logs) > limit {
		logs = logs[:limit]
	}

	// 返回结构化的日志结果
	c.JSON(http.StatusOK, gin.H{
		"application_id": applicationID,
		"log_level":      logLevel,
		"logs":           logs, // 返回的是结构化的日志对象数组
	})
}

// 从文件中读取指定级别的日志
func readLogsFromFile(filePath, logLevel string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var logs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, fmt.Sprintf("%s", logLevel)) {
			logs = append(logs, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

// 辅助函数：追加日志到文件
func appendToFile(filePath, logEntry string) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(logEntry)
	if err != nil {
		return err
	}

	return nil
}

// 解析日志行，将其转换为 LogData 结构体
func parseLogLine(logLine string) (LogData, error) {
	var log LogData
	parts := strings.SplitN(logLine, ": ", 2)
	if len(parts) != 2 {
		return log, fmt.Errorf("invalid log format")
	}

	metaParts := strings.SplitN(parts[0], "] [", 2)
	if len(metaParts) != 2 {
		return log, fmt.Errorf("invalid log format")
	}

	log.Timestamp = strings.Trim(metaParts[0], "[]")
	log.LogLevel = strings.Trim(metaParts[1], "[]")
	log.LogMessage = parts[1]

	return log, nil
}

func main() {
	// 初始化Gin路由
	router := gin.Default()

	// 定义日志上传和查询的路由
	router.POST("/upload", logUploadHandler)
	router.GET("/query", logQueryHandler)

	// 启动服务器
	port := ":8080"
	fmt.Printf("Server is running on port %s\n", port)
	log.Fatal(router.Run(port))
}
