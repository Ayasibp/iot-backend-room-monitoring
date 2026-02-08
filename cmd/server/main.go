package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"iot-backend-room-monitoring/internal/config"
	"iot-backend-room-monitoring/internal/database"
	"iot-backend-room-monitoring/internal/handler"
	"iot-backend-room-monitoring/internal/middleware"
	"iot-backend-room-monitoring/internal/repository"
	"iot-backend-room-monitoring/internal/service"
	"iot-backend-room-monitoring/pkg/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load configuration
	cfg := config.LoadConfig()
	log.Println("Configuration loaded successfully")

	// 2. Initialize JWT utilities with config
	utils.InitJWT(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
	)

	// 3. Initialize database connection
	db := database.Connect(cfg)

	// 4. Initialize repositories
	userRepo := repository.NewUserRepo(db)
	theaterRepo := repository.NewTheaterRepo(db)
	auditRepo := repository.NewAuditRepo(db)

	// Ensure live state exists for OT-01
	if err := theaterRepo.CreateLiveStateIfNotExists("OT-01"); err != nil {
		log.Printf("Warning: Failed to ensure live state exists: %v", err)
	}

	// 5. Initialize services
	authService := service.NewAuthService(userRepo, auditRepo)
	theaterService := service.NewTheaterService(theaterRepo, auditRepo)
	workerService := service.NewWorkerService(theaterRepo)

	// 6. Start background worker in goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go workerService.Start(ctx)

	// 7. Setup Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// 8. Setup Gin router
	r := gin.Default()

	// Apply CORS middleware
	r.Use(middleware.CORS(cfg))

	// 9. Register handlers
	authHandler := handler.NewAuthHandler(authService)
	theaterHandler := handler.NewTheaterHandler(theaterService)

	// 10. Define routes
	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		utils.SuccessResponse(c, gin.H{
			"status":  "healthy",
			"service": "iot-backend-room-monitoring",
		})
	})

	// Auth routes (public)
	auth := r.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
	}

	// Theater routes (authenticated)
	theater := r.Group("/theater")
	theater.Use(middleware.AuthMiddleware())
	{
		theater.GET("/state", theaterHandler.GetState)       // Get single room state
		theater.GET("/states", theaterHandler.GetAllStates)  // Get all room states
		theater.GET("/rooms", theaterHandler.GetRooms)       // Get list of room names

		// Admin-only routes
		theater.POST("/timer/op", middleware.RequireAdmin(), theaterHandler.UpdateTimer)
		theater.POST("/timer/cd", middleware.RequireAdmin(), theaterHandler.UpdateCountdownTimer)
		theater.PATCH("/timer/cd/adjust", middleware.RequireAdmin(), theaterHandler.AdjustCountdownTimer)
	}

	// 11. Setup graceful shutdown
	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := r.Run(":" + cfg.Server.Port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Cancel background worker context
	cancel()
	log.Println("Server exited")
}
