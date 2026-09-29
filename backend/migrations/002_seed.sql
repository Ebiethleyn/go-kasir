-- Seed data awal
-- Password default owner: admin123 (bcrypt hash)
USE kasir_db;

INSERT INTO owners (username, password_hash, full_name, email, phone, created_at, updated_at)
SELECT 'admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Pemilik Toko', 'admin@toko.local', '081234567890', NOW(), NOW()
WHERE NOT EXISTS (SELECT 1 FROM owners WHERE username = 'admin');
-- Note: hash di atas adalah placeholder; seed aktual dijalankan via Go agar bcrypt konsisten.

INSERT INTO settings (store_name, store_address, store_phone, invoice_prefix, tax_percent, default_min_stock, receipt_width_mm, currency_symbol, updated_at)
SELECT 'Toko Maju Jaya', 'Jl. Merdeka No. 10, Jakarta', '021-1234567', 'INV', 11.00, 5, 80, 'Rp', NOW()
WHERE NOT EXISTS (SELECT 1 FROM settings LIMIT 1);

INSERT INTO categories (name, description, created_at, updated_at)
SELECT * FROM (
  SELECT 'Makanan' AS name, 'Produk makanan' AS description, NOW() AS created_at, NOW() AS updated_at
  UNION ALL SELECT 'Minuman', 'Produk minuman', NOW(), NOW()
  UNION ALL SELECT 'Sembako', 'Kebutuhan pokok', NOW(), NOW()
  UNION ALL SELECT 'ATK', 'Alat tulis kantor', NOW(), NOW()
) AS tmp
WHERE NOT EXISTS (SELECT 1 FROM categories LIMIT 1);
