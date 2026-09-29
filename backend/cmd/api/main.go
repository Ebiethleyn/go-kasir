package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-kasir/backend/internal/config"
	"github.com/go-kasir/backend/internal/handler"
	"github.com/go-kasir/backend/internal/middleware"
	"github.com/go-kasir/backend/internal/model"
	jwtutil "github.com/go-kasir/backend/internal/pkg/jwt"
	"github.com/go-kasir/backend/internal/repository"
	"github.com/go-kasir/backend/internal/service"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	if err := ensureDatabase(cfg); err != nil {
		log.Fatalf("gagal membuat database: %v", err)
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		log.Fatalf("koneksi DB gagal: %v", err)
	}

	if err := autoMigrate(db); err != nil {
		log.Fatalf("migrasi gagal: %v", err)
	}
	if err := seed(db); err != nil {
		log.Fatalf("seed gagal: %v", err)
	}

	_ = os.MkdirAll(cfg.UploadDir, 0755)

	// Repositories
	ownerRepo := repository.NewOwnerRepo(db)
	catRepo := repository.NewCategoryRepo(db)
	prodRepo := repository.NewProductRepo(db)
	stockRepo := repository.NewStockRepo(db)
	custRepo := repository.NewCustomerRepo(db)
	trxRepo := repository.NewTransactionRepo(db)
	setRepo := repository.NewSettingRepo(db)
	logRepo := repository.NewActivityRepo(db)
	reportRepo := repository.NewReportRepo(db)

	jwtMgr := jwtutil.NewManager(cfg.JWTSecret, cfg.JWTExpireHours, cfg.JWTRefreshExpireHrs)

	// Services
	authSvc := service.NewAuthService(ownerRepo, jwtMgr, logRepo)
	prodSvc := service.NewProductService(prodRepo, catRepo, stockRepo, setRepo, db)
	trxSvc := service.NewTransactionService(trxRepo, prodRepo, setRepo, logRepo, db)
	custSvc := service.NewCustomerService(custRepo)
	setSvc := service.NewSettingService(setRepo)
	dashSvc := service.NewDashboardService(trxRepo, reportRepo, prodRepo)
	reportSvc := service.NewReportService(reportRepo, trxRepo, prodRepo, stockRepo, custRepo)

	h := &handler.Handlers{
		Auth: authSvc, Product: prodSvc, Trx: trxSvc, Customer: custSvc,
		Setting: setSvc, Dash: dashSvc, Report: reportSvc, Cfg: cfg,
		Products: prodRepo, Cats: catRepo, Custs: custRepo, Trxs: trxRepo, Logs: logRepo,
	}

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Content-Disposition"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Static uploads
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.UploadDir))))

	r.Route("/api", func(api chi.Router) {
		api.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"success":true,"message":"ok"}`))
		})

		api.Post("/auth/login", h.Login)
		api.Post("/auth/refresh", h.Refresh)

		api.Group(func(priv chi.Router) {
			priv.Use(middleware.Auth(jwtMgr))

			priv.Get("/auth/profile", h.Profile)
			priv.Put("/auth/profile", h.UpdateProfile)
			priv.Post("/auth/change-password", h.ChangePassword)

			priv.Get("/dashboard", h.Dashboard)

			priv.Get("/categories", h.ListCategories)
			priv.Post("/categories", h.CreateCategory)
			priv.Put("/categories/{id}", h.UpdateCategory)
			priv.Delete("/categories/{id}", h.DeleteCategory)

			priv.Get("/products", h.ListProducts)
			priv.Get("/products/search", h.SearchProducts)
			priv.Get("/products/lookup", h.LookupProduct)
			priv.Get("/products/low-stock", h.LowStock)
			priv.Post("/products", h.CreateProduct)
			priv.Post("/products/import", h.ImportProducts)
			priv.Get("/products/{id}", h.GetProduct)
			priv.Put("/products/{id}", h.UpdateProduct)
			priv.Delete("/products/{id}", h.DeleteProduct)
			priv.Post("/products/{id}/stock", h.AdjustStock)
			priv.Post("/products/{id}/image", h.UploadProductImage)
			priv.Get("/stock-movements", h.ListStockMovements)

			priv.Get("/customers", h.ListCustomers)
			priv.Post("/customers", h.CreateCustomer)
			priv.Get("/customers/{id}", h.GetCustomer)
			priv.Put("/customers/{id}", h.UpdateCustomer)
			priv.Delete("/customers/{id}", h.DeleteCustomer)

			priv.Get("/transactions", h.ListTransactions)
			priv.Post("/transactions", h.CreateTransaction)
			priv.Get("/transactions/{id}", h.GetTransaction)
			priv.Post("/transactions/{id}/void", h.VoidTransaction)
			priv.Get("/transactions/{id}/receipt", h.ReceiptPDF)

			priv.Get("/reports/sales", h.ReportSales)
			priv.Get("/reports/sales/pdf", h.ExportSalesPDF)
			priv.Get("/reports/sales/excel", h.ExportSalesExcel)
			priv.Get("/reports/products", h.ReportProducts)
			priv.Get("/reports/products/pdf", h.ExportProductsPDF)
			priv.Get("/reports/products/excel", h.ExportProductsExcel)
			priv.Get("/reports/profit-loss", h.ReportProfitLoss)
			priv.Get("/reports/profit-loss/pdf", h.ExportProfitPDF)
			priv.Get("/reports/profit-loss/excel", h.ExportProfitExcel)
			priv.Get("/reports/transactions", h.ReportTransactions)
			priv.Get("/reports/transactions/pdf", h.ExportTrxPDF)
			priv.Get("/reports/transactions/excel", h.ExportTrxExcel)
			priv.Get("/reports/customers/{id}", h.ReportCustomer)
			priv.Get("/reports/customers/{id}/pdf", h.ExportCustomerPDF)
			priv.Get("/reports/customers/{id}/excel", h.ExportCustomerExcel)

			priv.Get("/settings", h.GetSettings)
			priv.Put("/settings", h.UpdateSettings)

			priv.Get("/backup", h.BackupNow)
			priv.Post("/backup/restore", h.RestoreBackup)
			priv.Get("/export/master", h.ExportMasterExcel)
			priv.Get("/activity-logs", h.ActivityLogs)
		})
	})

	// Serve frontend AFTER /api so API routes take precedence
	frontendDir := resolveFrontendDir()
	if frontendDir != "" {
		fileServer(r, "/", http.Dir(frontendDir))
		log.Printf("frontend served from %s", frontendDir)
	}

	addr := ":" + cfg.AppPort
	log.Printf("Kasir API berjalan di http://localhost%s", addr)
	log.Printf("Login default: admin / admin123")
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}

func ensureDatabase(cfg *config.Config) error {
	db, err := sql.Open("mysql", cfg.DSNWithoutDB())
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		cfg.DBName,
	))
	return err
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Owner{},
		&model.Category{},
		&model.Product{},
		&model.StockMovement{},
		&model.Customer{},
		&model.Transaction{},
		&model.TransactionItem{},
		&model.Setting{},
		&model.ActivityLog{},
	)
}

func seed(db *gorm.DB) error {
	var n int64
	db.Model(&model.Owner{}).Count(&n)
	if n == 0 {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		owner := model.Owner{
			Username: "admin", PasswordHash: string(hash),
			FullName: "Pemilik Toko", Email: "admin@toko.local", Phone: "081234567890",
		}
		if err := db.Create(&owner).Error; err != nil {
			return err
		}
	}

	db.Model(&model.Setting{}).Count(&n)
	if n == 0 {
		s := model.Setting{
			StoreName: "Toko Maju Jaya", StoreAddress: "Jl. Merdeka No. 10, Jakarta",
			StorePhone: "021-1234567", InvoicePrefix: "INV", TaxPercent: 11,
			DefaultMinStock: 5, ReceiptWidthMM: 80, CurrencySymbol: "Rp",
		}
		if err := db.Create(&s).Error; err != nil {
			return err
		}
	}

	db.Model(&model.Category{}).Count(&n)
	if n == 0 {
		cats := []model.Category{
			{Name: "Makanan", Description: "Produk makanan"},
			{Name: "Minuman", Description: "Produk minuman"},
			{Name: "Sembako", Description: "Kebutuhan pokok"},
			{Name: "ATK", Description: "Alat tulis kantor"},
		}
		if err := db.Create(&cats).Error; err != nil {
			return err
		}

		var makanan, minuman, sembako, atk model.Category
		db.Where("name = ?", "Makanan").First(&makanan)
		db.Where("name = ?", "Minuman").First(&minuman)
		db.Where("name = ?", "Sembako").First(&sembako)
		db.Where("name = ?", "ATK").First(&atk)

		products := []model.Product{
			{CategoryID: &makanan.ID, SKU: "MKN-001", Barcode: "8991001001", Name: "Indomie Goreng", BuyPrice: 2500, SellPrice: 3500, Stock: 100, MinStock: 20, Unit: "pcs", IsActive: true},
			{CategoryID: &makanan.ID, SKU: "MKN-002", Barcode: "8991001002", Name: "Chitato 68g", BuyPrice: 8000, SellPrice: 11000, Stock: 40, MinStock: 10, Unit: "pcs", IsActive: true},
			{CategoryID: &minuman.ID, SKU: "MNM-001", Barcode: "8992002001", Name: "Aqua 600ml", BuyPrice: 2000, SellPrice: 3000, Stock: 150, MinStock: 30, Unit: "botol", IsActive: true},
			{CategoryID: &minuman.ID, SKU: "MNM-002", Barcode: "8992002002", Name: "Teh Botol Sosro", BuyPrice: 3500, SellPrice: 5000, Stock: 80, MinStock: 15, Unit: "botol", IsActive: true},
			{CategoryID: &sembako.ID, SKU: "SMB-001", Barcode: "8993003001", Name: "Beras 5kg", BuyPrice: 55000, SellPrice: 65000, Stock: 25, MinStock: 5, Unit: "karung", IsActive: true},
			{CategoryID: &sembako.ID, SKU: "SMB-002", Barcode: "8993003002", Name: "Minyak Goreng 1L", BuyPrice: 14000, SellPrice: 17000, Stock: 8, MinStock: 10, Unit: "botol", IsActive: true},
			{CategoryID: &atk.ID, SKU: "ATK-001", Barcode: "8994004001", Name: "Pulpen Standard", BuyPrice: 1500, SellPrice: 2500, Stock: 200, MinStock: 50, Unit: "pcs", IsActive: true},
			{CategoryID: &atk.ID, SKU: "ATK-002", Barcode: "8994004002", Name: "Buku Tulis 38 Lbr", BuyPrice: 3000, SellPrice: 4500, Stock: 60, MinStock: 20, Unit: "pcs", IsActive: true},
		}
		if err := db.Create(&products).Error; err != nil {
			return err
		}
	}

	db.Model(&model.Customer{}).Count(&n)
	if n == 0 {
		customers := []model.Customer{
			{Name: "Budi Santoso", Phone: "081111111111", Address: "Jakarta"},
			{Name: "Siti Aminah", Phone: "082222222222", Address: "Depok"},
		}
		_ = db.Create(&customers).Error
	}
	return nil
}

func resolveFrontendDir() string {
	candidates := []string{
		filepath.Join("..", "frontend"),
		filepath.Join("..", "..", "frontend"),
		"frontend",
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append([]string{filepath.Join(filepath.Dir(exe), "..", "frontend")}, candidates...)
	}
	wd, _ := os.Getwd()
	candidates = append(candidates, filepath.Join(wd, "..", "frontend"), filepath.Join(wd, "frontend"))
	for _, c := range candidates {
		if abs, err := filepath.Abs(c); err == nil {
			if st, err := os.Stat(filepath.Join(abs, "index.html")); err == nil && !st.IsDir() {
				return abs
			}
		}
	}
	return ""
}

func fileServer(r chi.Router, path string, root http.FileSystem) {
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", 301).ServeHTTP)
		path += "/"
	}
	r.Get(path+"*", func(w http.ResponseWriter, req *http.Request) {
		fs := http.StripPrefix(path, http.FileServer(root))
		// Don't hijack /api
		if len(req.URL.Path) >= 4 && req.URL.Path[:4] == "/api" {
			http.NotFound(w, req)
			return
		}
		fs.ServeHTTP(w, req)
	})
}
