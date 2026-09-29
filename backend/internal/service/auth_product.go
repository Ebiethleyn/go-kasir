package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-kasir/backend/internal/model"
	jwtutil "github.com/go-kasir/backend/internal/pkg/jwt"
	"github.com/go-kasir/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	owners *repository.OwnerRepo
	jwt    *jwtutil.Manager
	logs   *repository.ActivityRepo
}

func NewAuthService(owners *repository.OwnerRepo, jwt *jwtutil.Manager, logs *repository.ActivityRepo) *AuthService {
	return &AuthService{owners: owners, jwt: jwt, logs: logs}
}

type LoginResult struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresAt    time.Time   `json:"expires_at"`
	Owner        model.Owner `json:"owner"`
}

func (s *AuthService) Login(username, password, ip string) (*LoginResult, error) {
	owner, err := s.owners.FindByUsername(username)
	if err != nil {
		return nil, errors.New("username atau password salah")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(owner.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("username atau password salah")
	}
	access, exp, err := s.jwt.GenerateAccess(owner.ID, owner.Username)
	if err != nil {
		return nil, err
	}
	refresh, _, err := s.jwt.GenerateRefresh(owner.ID, owner.Username)
	if err != nil {
		return nil, err
	}
	_ = s.logs.Create(&model.ActivityLog{
		OwnerID: owner.ID, Action: "login", EntityType: "owner",
		EntityID: &owner.ID, Details: "Login berhasil", IPAddress: ip,
	})
	return &LoginResult{AccessToken: access, RefreshToken: refresh, ExpiresAt: exp, Owner: *owner}, nil
}

func (s *AuthService) Refresh(refreshToken string) (*LoginResult, error) {
	claims, err := s.jwt.Parse(refreshToken)
	if err != nil || claims.TokenTyp != "refresh" {
		return nil, errors.New("refresh token tidak valid")
	}
	owner, err := s.owners.FindByID(claims.OwnerID)
	if err != nil {
		return nil, errors.New("owner tidak ditemukan")
	}
	access, exp, err := s.jwt.GenerateAccess(owner.ID, owner.Username)
	if err != nil {
		return nil, err
	}
	refresh, _, err := s.jwt.GenerateRefresh(owner.ID, owner.Username)
	if err != nil {
		return nil, err
	}
	return &LoginResult{AccessToken: access, RefreshToken: refresh, ExpiresAt: exp, Owner: *owner}, nil
}

func (s *AuthService) Profile(ownerID uint) (*model.Owner, error) {
	return s.owners.FindByID(ownerID)
}

func (s *AuthService) UpdateProfile(ownerID uint, fullName, email, phone string) (*model.Owner, error) {
	o, err := s.owners.FindByID(ownerID)
	if err != nil {
		return nil, err
	}
	o.FullName = fullName
	o.Email = email
	o.Phone = phone
	if err := s.owners.Update(o); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *AuthService) ChangePassword(ownerID uint, oldPass, newPass string) error {
	o, err := s.owners.FindByID(ownerID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(o.PasswordHash), []byte(oldPass)); err != nil {
		return errors.New("password lama salah")
	}
	if len(newPass) < 6 {
		return errors.New("password baru minimal 6 karakter")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	o.PasswordHash = string(hash)
	return s.owners.Update(o)
}

// ---- Product / Category / Stock ----

type ProductService struct {
	products   *repository.ProductRepo
	categories *repository.CategoryRepo
	stock      *repository.StockRepo
	settings   *repository.SettingRepo
	db         *gorm.DB
}

func NewProductService(p *repository.ProductRepo, c *repository.CategoryRepo, s *repository.StockRepo, set *repository.SettingRepo, db *gorm.DB) *ProductService {
	return &ProductService{products: p, categories: c, stock: s, settings: set, db: db}
}

func (s *ProductService) ListProducts(f repository.ProductFilter) ([]model.Product, int64, error) {
	return s.products.List(f)
}

func (s *ProductService) GetProduct(id uint) (*model.Product, error) {
	return s.products.FindByID(id)
}

func (s *ProductService) SearchProducts(q string) ([]model.Product, error) {
	return s.products.Search(q, 20)
}

func (s *ProductService) LookupProduct(code string) (*model.Product, error) {
	return s.products.FindByBarcodeOrSKU(code)
}

func (s *ProductService) CreateProduct(p *model.Product) error {
	if p.Name == "" || p.SKU == "" {
		return errors.New("nama dan SKU wajib diisi")
	}
	if p.SellPrice < 0 || p.BuyPrice < 0 {
		return errors.New("harga tidak boleh negatif")
	}
	if p.MinStock == 0 {
		if set, err := s.settings.Get(); err == nil {
			p.MinStock = set.DefaultMinStock
		} else {
			p.MinStock = 5
		}
	}
	return s.products.Create(p)
}

func (s *ProductService) UpdateProduct(id uint, upd *model.Product) (*model.Product, error) {
	p, err := s.products.FindByID(id)
	if err != nil {
		return nil, errors.New("produk tidak ditemukan")
	}
	p.CategoryID = upd.CategoryID
	p.SKU = upd.SKU
	p.Barcode = upd.Barcode
	p.Name = upd.Name
	p.Description = upd.Description
	p.BuyPrice = upd.BuyPrice
	p.SellPrice = upd.SellPrice
	p.MinStock = upd.MinStock
	p.Unit = upd.Unit
	p.IsActive = upd.IsActive
	if upd.ImagePath != "" {
		p.ImagePath = upd.ImagePath
	}
	if err := s.products.Update(p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProductService) DeleteProduct(id uint) error {
	return s.products.Delete(id)
}

func (s *ProductService) AdjustStock(productID uint, typ string, qty int, notes string, ownerID uint) error {
	if qty <= 0 {
		return errors.New("jumlah harus lebih dari 0")
	}
	if typ != "in" && typ != "out" && typ != "adjust" {
		return errors.New("tipe stok tidak valid (in/out/adjust)")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var p model.Product
		if err := tx.First(&p, productID).Error; err != nil {
			return errors.New("produk tidak ditemukan")
		}
		before := p.Stock
		after := before
		switch typ {
		case "in":
			after = before + qty
		case "out":
			if before < qty {
				return errors.New("stok tidak mencukupi")
			}
			after = before - qty
		case "adjust":
			after = qty
			qty = after - before
			if qty >= 0 {
				typ = "in"
			} else {
				typ = "out"
				qty = -qty
			}
		}
		if err := tx.Model(&p).Update("stock", after).Error; err != nil {
			return err
		}
		mov := model.StockMovement{
			ProductID: productID, Type: typ, Quantity: qty,
			StockBefore: before, StockAfter: after,
			ReferenceType: "manual", Notes: notes, CreatedBy: ownerID,
		}
		return tx.Create(&mov).Error
	})
}

func (s *ProductService) ListStockMovements(productID *uint, from, to *time.Time, page, limit int) ([]model.StockMovement, int64, error) {
	return s.stock.List(productID, from, to, page, limit)
}

func (s *ProductService) LowStock() ([]model.Product, error) {
	return s.products.LowStock(50)
}

func (s *ProductService) ListCategories() ([]model.Category, error) {
	return s.categories.List()
}

func (s *ProductService) CreateCategory(c *model.Category) error {
	if c.Name == "" {
		return errors.New("nama kategori wajib")
	}
	return s.categories.Create(c)
}

func (s *ProductService) UpdateCategory(id uint, name, desc string) (*model.Category, error) {
	c, err := s.categories.FindByID(id)
	if err != nil {
		return nil, errors.New("kategori tidak ditemukan")
	}
	c.Name = name
	c.Description = desc
	if err := s.categories.Update(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ProductService) DeleteCategory(id uint) error {
	return s.categories.Delete(id)
}

func (s *ProductService) ImportProducts(rows []model.Product) (int, []string) {
	ok := 0
	var errs []string
	for i, p := range rows {
		if p.SKU == "" || p.Name == "" {
			errs = append(errs, fmt.Sprintf("baris %d: SKU/nama kosong", i+1))
			continue
		}
		if existing, err := s.products.FindBySKU(p.SKU); err == nil && existing != nil {
			existing.Name = p.Name
			existing.BuyPrice = p.BuyPrice
			existing.SellPrice = p.SellPrice
			existing.Stock = p.Stock
			existing.Unit = p.Unit
			existing.Barcode = p.Barcode
			existing.CategoryID = p.CategoryID
			if err := s.products.Update(existing); err != nil {
				errs = append(errs, fmt.Sprintf("baris %d: %v", i+1, err))
				continue
			}
			ok++
			continue
		}
		if err := s.products.Create(&p); err != nil {
			errs = append(errs, fmt.Sprintf("baris %d: %v", i+1, err))
			continue
		}
		ok++
	}
	return ok, errs
}
