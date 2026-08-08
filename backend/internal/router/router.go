package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlog "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/ispcms/backend/internal/config"
	"github.com/ispcms/backend/internal/handlers"
	"github.com/ispcms/backend/internal/middleware"
	"github.com/ispcms/backend/internal/repositories"
	"github.com/ispcms/backend/internal/services"
	"gorm.io/gorm"
)

func Setup(db *gorm.DB, cfg *config.Config) (*fiber.App, *services.OLTScheduler) {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"error":   err.Error(),
			})
		},
	})

	// Middleware
	app.Use(recover.New())
	app.Use(fiberlog.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PUT, PATCH, DELETE, OPTIONS",
		AllowCredentials: true,
	}))

	// ── Repositories ─────────────────────────────────────────────────────────────
	userRepo := repositories.NewUserRepository(db)
	roleRepo := repositories.NewRoleRepository(db)
	routerRepo := repositories.NewRouterRepository(db)
	pppoeRepo := repositories.NewPPPoERepository(db)
	iaRepo := repositories.NewInternetAccountRepository(db)
	packageRepo := repositories.NewPackageRepository(db)
	profileMappingRepo := repositories.NewProfileMappingRepository(db)
	subRepo := repositories.NewSubscriptionRepository(db)
	billRepo := repositories.NewBillRepository(db)
	notifRepo := repositories.NewNotificationRepository(db)
	paymentRepo := repositories.NewPaymentRepository(db)
	expenseCatRepo := repositories.NewExpenseCategoryRepository(db)
	expenseRepo := repositories.NewExpenseRepository(db)
	activityRepo := repositories.NewActivityLogRepository(db)
	snmpProfileRepo := repositories.NewSNMPProfileRepository(db)
	oltRepo := repositories.NewOLTRepository(db)
	ponPortRepo := repositories.NewPONPortRepository(db)
	onuRepo := repositories.NewONURepository(db)
	oltSyncLogRepo := repositories.NewOLTSyncLogRepository(db)

	// ── Services ──────────────────────────────────────────────────────────────────
	authSvc := services.NewAuthService(userRepo, cfg)
	userSvc := services.NewUserService(userRepo, cfg)
	roleSvc := services.NewRoleService(roleRepo)
	routerSvc := services.NewRouterService(routerRepo, cfg)
	notifSvc := services.NewNotificationService(notifRepo)
	packageSvc := services.NewPackageService(packageRepo)
	profileMappingSvc := services.NewProfileMappingService(profileMappingRepo)
	activitySvc := services.NewActivityService(activityRepo)
	billingSvc := services.NewBillingService(packageRepo, subRepo, billRepo, notifRepo, paymentRepo, activitySvc, db)
	syncSvc := services.NewSyncService(routerRepo, pppoeRepo, iaRepo, profileMappingRepo, billingSvc, notifSvc, activitySvc, db, cfg)
	dashboardSvc := services.NewDashboardService(routerRepo, pppoeRepo, iaRepo, packageRepo, profileMappingRepo, subRepo, billRepo, expenseRepo, activityRepo, db)
	expenseCatSvc := services.NewExpenseCategoryService(expenseCatRepo)
	expenseSvc := services.NewExpenseService(expenseRepo, expenseCatRepo)
	snmpProfileSvc := services.NewSNMPProfileService(snmpProfileRepo)
	oltSvc := services.NewOLTService(oltRepo, ponPortRepo, oltSyncLogRepo)
	oltSyncSvc := services.NewOLTSyncService(oltRepo, ponPortRepo, onuRepo, oltSyncLogRepo, activitySvc, cfg.JWTSecret)
	onuSvc := services.NewONUService(onuRepo)

	// ── Handlers ──────────────────────────────────────────────────────────────────
	authH := handlers.NewAuthHandler(authSvc, userSvc)
	userH := handlers.NewUserHandler(userSvc)
	roleH := handlers.NewRoleHandler(roleSvc)
	routerH := handlers.NewRouterHandler(routerSvc, syncSvc)
	pppoeH := handlers.NewPPPoEHandler(pppoeRepo)
	dashH := handlers.NewDashboardHandler(dashboardSvc, activitySvc)
	iaH := handlers.NewInternetAccountHandler(iaRepo, roleRepo, syncSvc)
	packageH := handlers.NewPackageHandler(packageSvc)
	profileMappingH := handlers.NewProfileMappingHandler(profileMappingSvc)
	billH := handlers.NewBillHandler(billingSvc)
	notifH := handlers.NewNotificationHandler(notifSvc)
	expenseH := handlers.NewExpenseHandler(expenseSvc, expenseCatSvc, activitySvc)
	snmpProfileH := handlers.NewSNMPProfileHandler(snmpProfileSvc)
	oltH := handlers.NewOLTHandler(oltSvc, oltSyncSvc, cfg.JWTSecret)
	onuH := handlers.NewONUHandler(onuSvc)

	// Auth middleware factory
	jwtAuth := middleware.JWTAuth(cfg.JWTSecret)
	perm := func(module, action string) fiber.Handler {
		return middleware.RequirePermission(roleRepo, module, action)
	}

	api := app.Group("/api/v1")

	// ── Public routes ─────────────────────────────────────────────────────────────
	auth := api.Group("/auth")
	auth.Post("/login", authH.Login)
	auth.Post("/refresh", authH.RefreshToken)

	// ── Protected routes ──────────────────────────────────────────────────────────
	auth.Post("/logout", jwtAuth, authH.Logout)
	auth.Get("/profile", jwtAuth, authH.Profile)
	auth.Put("/change-password", jwtAuth, authH.ChangePassword)

	// Users
	users := api.Group("/users", jwtAuth)
	users.Get("/", perm("users", "view"), userH.List)
	users.Post("/", perm("users", "create"), userH.Create)
	users.Get("/:id", perm("users", "view"), userH.Get)
	users.Put("/:id", perm("users", "update"), userH.Update)
	users.Delete("/:id", perm("users", "delete"), userH.Delete)
	users.Post("/:id/roles", perm("users", "update"), userH.AssignRole)
	users.Delete("/:id/roles/:roleId", perm("users", "update"), userH.RemoveRole)
	users.Patch("/:id/status", perm("users", "update"), userH.SetStatus)
	users.Post("/:id/reset-password",
		middleware.RequireRole("super_admin", "admin"),
		authH.ResetPassword,
	)

	// Roles
	roles := api.Group("/roles", jwtAuth)
	roles.Get("/", perm("roles", "view"), roleH.List)
	roles.Post("/", perm("roles", "create"), roleH.Create)
	roles.Get("/permissions", perm("roles", "view"), roleH.ListPermissions)
	roles.Get("/:id", perm("roles", "view"), roleH.Get)
	roles.Put("/:id", perm("roles", "update"), roleH.Update)
	roles.Delete("/:id", perm("roles", "delete"), roleH.Delete)
	roles.Post("/:id/permissions", middleware.RequireRole("super_admin"), roleH.AssignPermission)
	roles.Delete("/:id/permissions/:permId", middleware.RequireRole("super_admin"), roleH.RemovePermission)
	roles.Put("/:id/permissions", middleware.RequireRole("super_admin"), roleH.SetPermissions)
	roles.Put("/:id/account-prefixes", perm("roles", "update"), roleH.SetAccountPrefixes)

	// Routers
	rts := api.Group("/routers", jwtAuth)
	rts.Get("/", perm("routers", "view"), routerH.List)
	rts.Post("/", perm("routers", "create"), routerH.Create)
	rts.Post("/test-connection", perm("routers", "view"), routerH.TestConnectionRaw)
	rts.Get("/:id", perm("routers", "view"), routerH.Get)
	rts.Put("/:id", perm("routers", "update"), routerH.Update)
	rts.Delete("/:id", perm("routers", "delete"), routerH.Delete)
	rts.Post("/:id/test", perm("routers", "view"), routerH.TestConnection)
	rts.Post("/:id/sync", perm("routers", "update"), routerH.Sync)
	rts.Get("/:id/sync-logs", perm("routers", "view"), routerH.GetSyncLogs)

	// PPPoE (legacy)
	pppoe := api.Group("/pppoe", jwtAuth)
	pppoe.Get("/secrets", perm("pppoe", "view"), pppoeH.ListSecrets)
	pppoe.Get("/secrets/:id", perm("pppoe", "view"), pppoeH.GetSecret)
	pppoe.Get("/sessions", perm("pppoe", "view"), pppoeH.ListSessions)

	// Internet Accounts
	ia := api.Group("/internet-accounts", jwtAuth)
	ia.Get("/", perm("accounts", "view"), iaH.List)
	ia.Get("/stats", perm("accounts", "view"), iaH.Stats)
	ia.Get("/profiles", perm("accounts", "view"), iaH.Profiles)
	ia.Post("/sync-all", perm("routers", "update"), iaH.SyncAll)
	ia.Get("/:id/payment-history", perm("billing", "view"), billH.PaymentHistory)
	ia.Get("/:id/billing-history", perm("billing", "view"), billH.BillingHistory)
	ia.Get("/:id", perm("accounts", "view"), iaH.Get)

	// Packages
	pkgs := api.Group("/packages", jwtAuth)
	pkgs.Get("/", perm("packages", "view"), packageH.List)
	pkgs.Get("/active", perm("packages", "view"), packageH.ListActive)
	pkgs.Post("/", perm("packages", "create"), packageH.Create)
	pkgs.Get("/:id", perm("packages", "view"), packageH.Get)
	pkgs.Put("/:id", perm("packages", "update"), packageH.Update)
	pkgs.Delete("/:id", perm("packages", "delete"), packageH.Delete)

	// Profile Mappings
	pm := api.Group("/profile-mappings", jwtAuth)
	pm.Get("/", perm("billing", "view"), profileMappingH.List)
	pm.Get("/unmapped", perm("billing", "view"), profileMappingH.UnmappedProfiles)
	pm.Post("/", perm("billing", "create"), profileMappingH.Create)
	pm.Get("/:id", perm("billing", "view"), profileMappingH.Get)
	pm.Put("/:id", perm("billing", "update"), profileMappingH.Update)
	pm.Delete("/:id", perm("billing", "delete"), profileMappingH.Delete)

	// Subscriptions
	subs := api.Group("/subscriptions", jwtAuth)
	subs.Post("/auto-assign", perm("subscriptions", "create"), billH.AutoAssignSubscriptions)
	subs.Get("/", perm("subscriptions", "view"), billH.ListSubscriptions)
	subs.Post("/", perm("subscriptions", "create"), billH.AssignSubscription)
	subs.Get("/account/:accountId", perm("subscriptions", "view"), billH.GetActiveSubscription)

	// Bills
	bills := api.Group("/bills", jwtAuth)
	bills.Get("/", perm("billing", "view"), billH.List)
	bills.Get("/status", perm("billing", "view"), billH.BillingStatus)
	bills.Get("/generation-logs", perm("billing", "view"), billH.ListGenerationLogs)
	bills.Post("/generate", perm("billing", "create"), billH.GenerateBills)
	bills.Get("/:id", perm("billing", "view"), billH.Get)
	bills.Patch("/:id/status", perm("billing", "update"), billH.UpdateStatus)

	// Notifications
	notifs := api.Group("/notifications", jwtAuth)
	notifs.Get("/", notifH.List)
	notifs.Get("/unread-count", notifH.CountUnread)
	notifs.Patch("/:id/read", notifH.MarkRead)
	notifs.Post("/mark-all-read", notifH.MarkAllRead)

	// Expenses
	expCats := api.Group("/expense-categories", jwtAuth)
	expCats.Get("/", perm("expenses", "view"), expenseH.ListCategories)
	expCats.Post("/", perm("expenses", "create"), expenseH.CreateCategory)
	expCats.Put("/:id", perm("expenses", "update"), expenseH.UpdateCategory)
	expCats.Delete("/:id", perm("expenses", "delete"), expenseH.DeleteCategory)

	exps := api.Group("/expenses", jwtAuth)
	exps.Get("/summary", perm("expenses", "view"), expenseH.Summary)
	exps.Get("/", perm("expenses", "view"), expenseH.List)
	exps.Post("/", perm("expenses", "create"), expenseH.Create)
	exps.Get("/:id", perm("expenses", "view"), expenseH.Get)
	exps.Put("/:id", perm("expenses", "update"), expenseH.Update)
	exps.Delete("/:id", perm("expenses", "delete"), expenseH.Delete)

	// Dashboard
	api.Get("/dashboard/stats", jwtAuth, perm("dashboard", "view"), dashH.Stats)
	api.Get("/dashboard/activities", jwtAuth, perm("dashboard", "view"), dashH.Activities)

	// SNMP Profiles
	snmpProfiles := api.Group("/snmp-profiles", jwtAuth)
	snmpProfiles.Get("/", perm("network", "view"), snmpProfileH.List)
	snmpProfiles.Post("/", perm("network", "create"), snmpProfileH.Create)
	snmpProfiles.Get("/:id", perm("network", "view"), snmpProfileH.Get)
	snmpProfiles.Put("/:id", perm("network", "update"), snmpProfileH.Update)
	snmpProfiles.Delete("/:id", perm("network", "delete"), snmpProfileH.Delete)

	// OLTs
	olts := api.Group("/olts", jwtAuth)
	olts.Get("/", perm("network", "view"), oltH.List)
	olts.Get("/stats", perm("network", "view"), oltH.Stats)
	olts.Get("/sync-logs", perm("network", "view"), oltH.RecentSyncLogs)
	olts.Post("/test-connection", perm("network", "view"), oltH.TestConnectionRaw)
	olts.Post("/", perm("network", "create"), oltH.Create)
	olts.Get("/:id", perm("network", "view"), oltH.Get)
	olts.Put("/:id", perm("network", "update"), oltH.Update)
	olts.Delete("/:id", perm("network", "delete"), oltH.Delete)
	olts.Post("/:id/sync", perm("network", "update"), oltH.Sync)
	olts.Post("/:id/test", perm("network", "view"), oltH.TestConnection)
	olts.Get("/:id/sync-logs", perm("network", "view"), oltH.SyncLogs)
	olts.Get("/:id/pon-ports", perm("network", "view"), oltH.PONPorts)

	// ONUs
	onus := api.Group("/onus", jwtAuth)
	onus.Get("/", perm("network", "view"), onuH.List)
	onus.Get("/:id", perm("network", "view"), onuH.Get)
	onus.Patch("/:id/link", perm("network", "update"), onuH.Link)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	scheduler := services.NewOLTScheduler(oltRepo, oltSyncSvc)
	return app, scheduler
}
