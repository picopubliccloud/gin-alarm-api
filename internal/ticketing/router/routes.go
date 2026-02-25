package router

import (
	"database/sql"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/auth"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/config"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/handlers"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/middleware"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/repo"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/service"
	"github.com/picopubliccloud/alarm-api/internal/ticketing/storage"
)

func RegisterRoutes(r *gin.RouterGroup, db *sql.DB) {
	// repos
	userRepo := repo.NewUserRepo(db)
	rbacRepo := repo.NewRBACRepo(db)
	updateRepo := repo.NewUpdateRepo(db)
	ticketRepo := repo.NewTicketRepo(db, updateRepo)
	assignRepo := repo.NewAssignmentRepo(db)
	lockRepo := repo.NewLockRepo(db)
	closureRepo := repo.NewClosureRepo(db)
	attachRepo := repo.NewAttachmentRepo(db)
	outboxRepo := repo.NewOutboxRepo(db)

	// meta repo + handler (NEW)
	metaRepo := repo.NewMetaRepo(db)
	metaHandler := handlers.NewMetaHandler(metaRepo)

	// --- Object storage (S3 / Ceph / Swift gateway) ---
	store, err := storage.NewS3Store(storage.Config{
		Endpoint:  config.MustEnv("S3_ENDPOINT"),
		Bucket:    config.MustEnv("S3_BUCKET"),
		Region:    config.GetEnvDefault("S3_REGION", "us-east-1"),
		AccessKey: config.MustEnv("S3_ACCESS_KEY"),
		SecretKey: config.MustEnv("S3_SECRET_KEY"),
		KeyPrefix: config.GetEnvDefault("S3_KEY_PREFIX", "tickets/"),
	})
	if err != nil {
		log.Fatalf("ticketing: failed to init S3 store: %v", err)
	}

	// services (UNCHANGED - do NOT pass metaRepo here)
	ticketSvc := service.NewTicketService(
		db,
		ticketRepo, updateRepo, assignRepo, lockRepo, closureRepo, attachRepo, outboxRepo,
		rbacRepo, metaRepo,
	)
	authSvc := service.NewAuthService(db, userRepo, rbacRepo)

	// handlers
	h := handlers.NewTicketsHandler(ticketSvc)
	attachmentsHandler := handlers.NewAttachmentsHandler(store, attachRepo)

	// --- Create Keycloak verifier ONCE (JWKS + issuer) ---
	verifier, err := auth.NewVerifier(
		"https://192.168.30.120:8443/realms/ppc/protocol/openid-connect/certs",
		"https://192.168.30.120:8443/realms/ppc",
	)
	if err != nil {
		log.Fatalf("ticketing: failed to init keycloak verifier: %v", err)
	}

	// --- Public endpoints (no auth) ---
	public := r.Group("")
	{
		public.GET("/health", h.Health)
	}

	// --- Protected endpoints (JWT verified + user upsert + rbac) ---
	api := r.Group("")
	api.Use(
		middleware.AuthContext(verifier), // verify JWT + extract sub/email/name/roles into CtxAuth
		middleware.UserUpsert(authSvc),   // upsert ops.users using CtxAuth
		middleware.RBAC(authSvc),         // resolve RBAC using subject from context
	)

	{
		api.GET("/me", h.Me)

		// Listing + pagination
		api.GET("", h.ListTickets)
		api.GET("/pool", h.ListPoolTickets)
		api.GET("/summary", h.SummaryTickets)

		// Meta (use /meta to avoid conflict with /:ticket_id)
		meta := api.Group("/meta")
		{
			meta.GET("/customers", metaHandler.ListCustomers)
			meta.GET("/services", metaHandler.ListServices)
			meta.GET("/users", metaHandler.ListUsers)
		}

		// Read single
		api.GET("/:ticket_id", h.GetTicket)

		// Create
		api.POST("", h.CreateTicket)

		// Updates
		api.POST("/:ticket_id/updates", h.AddUpdate)

		// Assignments
		api.POST("/:ticket_id/assign", h.AssignOwner)
		api.POST("/:ticket_id/unassign", h.UnassignOwner)

		// Locks
		api.POST("/:ticket_id/lock", h.LockTicket)
		api.POST("/:ticket_id/unlock", h.UnlockTicket)

		// Closure
		api.POST("/:ticket_id/close", h.CloseTicket)

		// attachments
		api.GET("/:ticket_id/attachments", attachmentsHandler.ListAttachments)
		api.POST("/:ticket_id/attachments/upload-temp-url", attachmentsHandler.CreateUploadTempURL)
		api.POST("/:ticket_id/attachments", attachmentsHandler.RegisterAttachment)
		api.GET("/:ticket_id/attachments/:attachment_id/temp-url", attachmentsHandler.GetDownloadTempURL)
	}
}
