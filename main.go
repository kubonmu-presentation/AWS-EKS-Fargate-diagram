package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Item struct {
	ID    string `json:"id" dynamodbav:"id"`
	Name  string `json:"name" dynamodbav:"name"`
	Price int    `json:"price" dynamodbav:"price"`
}

type CreateItemRequest struct {
	Name  string `json:"name" binding:"required"`
	Price int    `json:"price" binding:"required,gt=0"`
}

type App struct {
	dynamoClient *dynamodb.Client
	secretClient *secretsmanager.Client
	tableName    string
	secretName   string
}

func loadAWSConfig(ctx context.Context) (aws.Config, error) {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "ap-northeast-2"
	}

	// IRSA 환경에서는 AWS SDK가 Web Identity 자격증명을 자동으로 사용한다.
	return awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(region),
	)
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		log.Printf(
			"method=%s path=%s status=%d latency=%s",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(start),
		)
	}
}

func health(c *gin.Context) {
	// 외부 AWS 서비스 상태와 무관하게 항상 200 반환
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

func (a *App) getItems(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := a.dynamoClient.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(a.tableName),
	})
	if err != nil {
		log.Printf("method=GET path=/items status=500 error=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to retrieve items",
		})
		return
	}

	items := make([]Item, 0)

	if err := attributevalue.UnmarshalListOfMaps(result.Items, &items); err != nil {
		log.Printf("method=GET path=/items status=500 error=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to decode items",
		})
		return
	}

	c.JSON(http.StatusOK, items)
}

func (a *App) createItem(c *gin.Context) {
	var req CreateItemRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request",
		})
		return
	}

	item := Item{
		ID:    uuid.NewString(),
		Name:  req.Name,
		Price: req.Price,
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		log.Printf("method=POST path=/items status=500 error=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to encode item",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	_, err = a.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(a.tableName),
		Item:      av,
	})
	if err != nil {
		log.Printf("method=POST path=/items status=500 error=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to store item",
		})
		return
	}

	c.JSON(http.StatusCreated, item)
}

func (a *App) getConfig(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	result, err := a.secretClient.GetSecretValue(
		ctx,
		&secretsmanager.GetSecretValueInput{
			SecretId: aws.String(a.secretName),
		},
	)

	if err != nil {
		log.Printf("method=GET path=/config status=500 error=secret_load_failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"secret_loaded": false,
		})
		return
	}

	if result.SecretString == nil && len(result.SecretBinary) == 0 {
		log.Printf("method=GET path=/config status=500 error=secret_empty")
		c.JSON(http.StatusInternalServerError, gin.H{
			"secret_loaded": false,
		})
		return
	}

	// 실제 Secret 값은 응답이나 로그에 절대 출력하지 않는다.
	c.JSON(http.StatusOK, gin.H{
		"secret_loaded": true,
	})
}

func main() {
	gin.SetMode(gin.ReleaseMode)

	tableName := os.Getenv("DDB_TABLE")
	secretName := os.Getenv("SECRET_NAME")

	if tableName == "" {
		log.Fatal("DDB_TABLE environment variable is required")
	}

	if secretName == "" {
		log.Fatal("SECRET_NAME environment variable is required")
	}

	awsCfg, err := loadAWSConfig(context.Background())
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	app := &App{
		dynamoClient: dynamodb.NewFromConfig(awsCfg),
		secretClient: secretsmanager.NewFromConfig(awsCfg),
		tableName:    tableName,
		secretName:   secretName,
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger())

	router.GET("/health", health)
	router.GET("/items", app.getItems)
	router.POST("/items", app.createItem)
	router.GET("/config", app.getConfig)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Println("Gin REST API listening on :8080")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit

	log.Println("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		return
	}

	log.Println("server stopped")
}
