package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-kasir/backend/internal/config"
	"github.com/go-kasir/backend/internal/middleware"
	"github.com/go-kasir/backend/internal/model"
	"github.com/go-kasir/backend/internal/pkg/excel"
	"github.com/go-kasir/backend/internal/pkg/response"
	"github.com/go-kasir/backend/internal/repository"
	"github.com/go-kasir/backend/internal/service"
)

type Handlers struct {
	Auth     *service.AuthService
	Product  *service.ProductService
	Trx      *service.TransactionService
	Customer *service.CustomerService
	Setting  *service.SettingService
	Dash     *service.DashboardService
	Report   *service.ReportService
	Cfg      *config.Config
	Products *repository.ProductRepo
	Cats     *repository.CategoryRepo
	Custs    *repository.CustomerRepo
	Trxs     *repository.TransactionRepo
	Logs     *repository.ActivityRepo
}

func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func clientIP(r *http.Request) string {
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		return strings.Split(x, ",")[0]
	}
	return r.RemoteAddr
}

// ---- Auth ----

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	res, err := h.Auth.Login(req.Username, req.Password, clientIP(r))
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}
	response.OK(w, "login berhasil", res)
}

func (h *Handlers) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := decodeJSON(r, &req); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	res, err := h.Auth.Refresh(req.RefreshToken)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}
	response.OK(w, "token diperbarui", res)
}

func (h *Handlers) Profile(w http.ResponseWriter, r *http.Request) {
	o, err := h.Auth.Profile(middleware.GetOwnerID(r))
	if err != nil {
		response.NotFound(w, "profil tidak ditemukan")
		return
	}
	response.OK(w, "ok", o)
}

func (h *Handlers) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
	}
	if err := decodeJSON(r, &req); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	o, err := h.Auth.UpdateProfile(middleware.GetOwnerID(r), req.FullName, req.Email, req.Phone)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, "profil diperbarui", o)
}

func (h *Handlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	if err := h.Auth.ChangePassword(middleware.GetOwnerID(r), req.OldPassword, req.NewPassword); err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, "password berhasil diganti", nil)
}

// ---- Dashboard ----

func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	data, err := h.Dash.Summary()
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", data)
}

// ---- Categories ----

func (h *Handlers) ListCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.Product.ListCategories()
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", list)
}

func (h *Handlers) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var c model.Category
	if err := decodeJSON(r, &c); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	if err := h.Product.CreateCategory(&c); err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.Created(w, "kategori dibuat", c)
}

func (h *Handlers) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSON(r, &req); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	c, err := h.Product.UpdateCategory(uint(id), req.Name, req.Description)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, "kategori diperbarui", c)
}

func (h *Handlers) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err := h.Product.DeleteCategory(uint(id)); err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, "kategori dihapus", nil)
}

// ---- Products ----

func (h *Handlers) ListProducts(w http.ResponseWriter, r *http.Request) {
	f := repository.ProductFilter{
		Search: r.URL.Query().Get("search"),
		Page:   atoiDefault(r.URL.Query().Get("page"), 1),
		Limit:  atoiDefault(r.URL.Query().Get("limit"), 20),
		LowStock: r.URL.Query().Get("low_stock") == "1",
	}
	if cid := r.URL.Query().Get("category_id"); cid != "" {
		v, _ := strconv.ParseUint(cid, 10, 64)
		u := uint(v)
		f.CategoryID = &u
	}
	list, total, err := h.Product.ListProducts(f)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", map[string]interface{}{"items": list, "total": total, "page": f.Page, "limit": f.Limit})
}

func (h *Handlers) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	p, err := h.Product.GetProduct(uint(id))
	if err != nil {
		response.NotFound(w, "produk tidak ditemukan")
		return
	}
	response.OK(w, "ok", p)
}

func (h *Handlers) SearchProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		response.OK(w, "ok", []model.Product{})
		return
	}
	list, err := h.Product.SearchProducts(q)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", list)
}

func (h *Handlers) LookupProduct(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	p, err := h.Product.LookupProduct(code)
	if err != nil {
		response.NotFound(w, "produk tidak ditemukan")
		return
	}
	response.OK(w, "ok", p)
}

func (h *Handlers) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var p model.Product
	if err := decodeJSON(r, &p); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	p.IsActive = true
	if err := h.Product.CreateProduct(&p); err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.Created(w, "produk dibuat", p)
}

func (h *Handlers) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	var p model.Product
	if err := decodeJSON(r, &p); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	out, err := h.Product.UpdateProduct(uint(id), &p)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, "produk diperbarui", out)
}

func (h *Handlers) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err := h.Product.DeleteProduct(uint(id)); err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, "produk dihapus", nil)
}

func (h *Handlers) AdjustStock(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	var req struct {
		Type     string `json:"type"`
		Quantity int    `json:"quantity"`
		Notes    string `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		response.BadRequest(w, "body tidak valid")
		return
	}
	if err := h.Product.AdjustStock(uint(id), req.Type, req.Quantity, req.Notes, middleware.GetOwnerID(r)); err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	response.OK(w, "stok diperbarui", nil)
}

func (h *Handlers) ListStockMovements(w http.ResponseWriter, r *http.Request) {
	var pid *uint
	if v := r.URL.Query().Get("product_id"); v != "" {
		n, _ := strconv.ParseUint(v, 10, 64)
		u := uint(n)
		pid = &u
	}
	from, to, _ := service.ParseDateRange(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	list, total, err := h.Product.ListStockMovements(pid, &from, &to, atoiDefault(r.URL.Query().Get("page"), 1), atoiDefault(r.URL.Query().Get("limit"), 50))
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", map[string]interface{}{"items": list, "total": total})
}

func (h *Handlers) LowStock(w http.ResponseWriter, r *http.Request) {
	list, err := h.Product.LowStock()
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "ok", list)
}

func (h *Handlers) ImportProducts(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		response.BadRequest(w, "upload gagal")
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "file wajib (field: file)")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		response.Internal(w, "gagal membaca file")
		return
	}
	rows, err := excel.ParseProductImport(data)
	if err != nil {
		response.BadRequest(w, err.Error())
		return
	}
	ok, errs := h.Product.ImportProducts(rows)
	response.OK(w, "import selesai", map[string]interface{}{"imported": ok, "errors": errs})
}

func (h *Handlers) UploadProductImage(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err := r.ParseMultipartForm(int64(h.Cfg.MaxUploadMB) << 20); err != nil {
		response.BadRequest(w, "upload gagal")
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		response.BadRequest(w, "field image wajib")
		return
	}
	defer file.Close()
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		response.BadRequest(w, "format gambar harus jpg/png/webp")
		return
	}
	_ = os.MkdirAll(h.Cfg.UploadDir, 0755)
	name := fmtFilename(uint(id), ext)
	path := filepath.Join(h.Cfg.UploadDir, name)
	out, err := os.Create(path)
	if err != nil {
		response.Internal(w, "gagal menyimpan file")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		response.Internal(w, "gagal menulis file")
		return
	}
	p, err := h.Product.GetProduct(uint(id))
	if err != nil {
		response.NotFound(w, "produk tidak ditemukan")
		return
	}
	p.ImagePath = "/uploads/" + name
	_, err = h.Product.UpdateProduct(uint(id), p)
	if err != nil {
		response.Internal(w, err.Error())
		return
	}
	response.OK(w, "gambar diunggah", map[string]string{"image_path": p.ImagePath})
}

func fmtFilename(id uint, ext string) string {
	return "product_" + strconv.FormatUint(uint64(id), 10) + "_" + strconv.FormatInt(time.Now().Unix(), 10) + ext
}

func atoiDefault(s string, d int) int {
	if s == "" {
		return d
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return d
	}
	return n
}
