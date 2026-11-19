package server

import (
	"net/http"
	"time"

	"app/docs"
	"app/internal/config"
	"app/internal/features/ai"
	"app/internal/features/auth"
	"app/internal/features/post"
	"app/internal/middleware"
	"app/internal/response"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

func NewRouter(db *gorm.DB, cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.RateLimiter(300, time.Minute))

	// OpenAPI / Swagger JSON dynamic endpoint
	r.GET("/api.json", func(c *gin.Context) {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", []byte(docs.GetSwaggerJSON(c.Request.Host, scheme)))
	})

	// Swagger UI interactive documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/api.json")))

	// Feature repositories, services, and handlers
	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, cfg.AppKey)
	authHandler := auth.NewHandler(authSvc)

	postRepo := post.NewRepository(db)
	postSvc := post.NewService(postRepo)
	postHandler := post.NewHandler(postSvc)

	searchClient := ai.NewSearchClient(cfg.SearchEngineEndpoint, cfg.OpenAIKey)
	aiClient := ai.NewAIClient(cfg.OpenAIEndpoint, cfg.OpenAIKey, cfg.OpenAIModel)
	aiSvc := ai.NewService(searchClient, aiClient)
	aiHandler := ai.NewHandler(aiSvc)

	api := r.Group("/api")
	{
		// Public routes
		pub := api.Group("/public")
		{
			pub.GET("/categories", postHandler.GetPublicCategories)
			pub.GET("/posts", postHandler.ListPublicPosts)
			pub.GET("/posts/:slug", postHandler.GetPublicPostBySlug)
		}

		// Authentication routes
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/logout", authHandler.Logout)
			authGroup.GET("/profile", middleware.Auth(cfg.AppKey), authHandler.GetProfile)
			authGroup.PUT("/profile", middleware.Auth(cfg.AppKey), authHandler.UpdateProfile)
		}

		// Author / Member Posts routes
		postsGroup := api.Group("/posts", middleware.Auth(cfg.AppKey))
		{
			postsGroup.GET("", postHandler.ListAuthorPosts)
			postsGroup.POST("", postHandler.CreatePost)
			postsGroup.GET("/stats", postHandler.GetAuthorStats)
			postsGroup.POST("/ai-generate", aiHandler.GenerateBlog)
			postsGroup.GET("/:id", postHandler.GetAuthorPostByID)
			postsGroup.PUT("/:id", postHandler.UpdatePost)
			postsGroup.PATCH("/:id/toggle-publish", postHandler.TogglePublish)
			postsGroup.DELETE("/:id", postHandler.DeletePost)
		}

		// Admin Only routes
		adminGroup := api.Group("/admin", middleware.Auth(cfg.AppKey), middleware.RequireRole("admin"))
		{
			adminGroup.GET("/users", postHandler.AdminListUsers)
			adminGroup.GET("/posts", postHandler.AdminListPosts)
			adminGroup.GET("/users/:id/posts", postHandler.AdminListUserPosts)
		}
	}

	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, "Endpoint not found")
	})

	return r
}
