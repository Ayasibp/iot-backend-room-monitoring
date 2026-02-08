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
	hospitalRepo := repository.NewHospitalRepo(db)
	roomRepo := repository.NewRoomRepo(db)
	userHospitalRepo := repository.NewUserHospitalRepo(db)

	// Ensure live state exists for OT-01
	if err := theaterRepo.CreateLiveStateIfNotExists("OT-01"); err != nil {
		log.Printf("Warning: Failed to ensure live state exists: %v", err)
	}

	// 5. Initialize services
	authService := service.NewAuthService(userRepo, auditRepo)
	theaterService := service.NewTheaterService(theaterRepo, auditRepo)
	workerService := service.NewWorkerService(theaterRepo)
	hospitalService := service.NewHospitalService(hospitalRepo, userHospitalRepo, auditRepo)
	roomService := service.NewRoomService(roomRepo, hospitalRepo, userHospitalRepo, auditRepo)

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
	hospitalHandler := handler.NewHospitalHandler(hospitalService)
	roomHandler := handler.NewRoomHandler(roomService)

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

	// API v1 routes
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware())
	{
		// Hospital Management
		hospitals := api.Group("/hospitals")
		{
			hospitals.GET("", hospitalHandler.GetAllHospitals)     // List hospitals (filtered by user access)
			hospitals.GET("/:id", hospitalHandler.GetHospital)     // Get hospital details
			hospitals.GET("/:id/rooms", roomHandler.GetRoomsByHospital) // Get rooms in hospital

			// Admin-only operations
			hospitals.POST("", middleware.RequireAdmin(), hospitalHandler.CreateHospital)
			hospitals.PUT("/:id", middleware.RequireAdmin(), hospitalHandler.UpdateHospital)
			hospitals.DELETE("/:id", middleware.RequireAdmin(), hospitalHandler.DeleteHospital)
		}

		// Room Management
		rooms := api.Group("/rooms")
		{
			rooms.GET("", roomHandler.GetAllRooms)             // List all rooms (filtered by user access)
			rooms.GET("/:id", roomHandler.GetRoom)             // Get room details

			// Admin-only operations
			rooms.POST("", middleware.RequireAdmin(), roomHandler.CreateRoom)
			rooms.PUT("/:id", middleware.RequireAdmin(), roomHandler.UpdateRoom)
			rooms.DELETE("/:id", middleware.RequireAdmin(), roomHandler.DeleteRoom)
		}

		// Dashboard endpoints
		dashboard := api.Group("/dashboard")
		{
			dashboard.GET("/rooms/:room_id", theaterHandler.GetRoomDashboard)
			
			// Admin-only timer operations by room_id
			dashboard.POST("/rooms/:room_id/timer/op", middleware.RequireAdmin(), theaterHandler.UpdateTimerByRoomID)
			dashboard.POST("/rooms/:room_id/timer/cd", middleware.RequireAdmin(), theaterHandler.UpdateCountdownTimerByRoomID)
			dashboard.PATCH("/rooms/:room_id/timer/cd/adjust", middleware.RequireAdmin(), theaterHandler.AdjustCountdownTimerByRoomID)
		}
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
