-- ============================================================
-- Kasir App — Schema MySQL
-- ============================================================

CREATE DATABASE IF NOT EXISTS kasir_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE kasir_db;

CREATE TABLE IF NOT EXISTS owners (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(50) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  full_name VARCHAR(100) NOT NULL,
  email VARCHAR(100) NULL,
  phone VARCHAR(30) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  deleted_at DATETIME(3) NULL,
  UNIQUE KEY uk_owners_username (username),
  KEY idx_owners_deleted (deleted_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS categories (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  description VARCHAR(255) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  deleted_at DATETIME(3) NULL,
  UNIQUE KEY uk_categories_name (name),
  KEY idx_categories_deleted (deleted_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS products (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  category_id BIGINT UNSIGNED NULL,
  sku VARCHAR(50) NOT NULL,
  barcode VARCHAR(50) NULL,
  name VARCHAR(150) NOT NULL,
  description TEXT NULL,
  buy_price DECIMAL(15,2) NOT NULL DEFAULT 0,
  sell_price DECIMAL(15,2) NOT NULL DEFAULT 0,
  stock INT NOT NULL DEFAULT 0,
  min_stock INT NOT NULL DEFAULT 5,
  unit VARCHAR(20) DEFAULT 'pcs',
  image_path VARCHAR(255) NULL,
  is_active TINYINT(1) DEFAULT 1,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  deleted_at DATETIME(3) NULL,
  UNIQUE KEY uk_products_sku (sku),
  KEY idx_products_barcode (barcode),
  KEY idx_products_name (name),
  KEY idx_products_category (category_id),
  KEY idx_products_deleted (deleted_at),
  CONSTRAINT fk_products_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS stock_movements (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  product_id BIGINT UNSIGNED NOT NULL,
  type VARCHAR(20) NOT NULL,
  quantity INT NOT NULL,
  stock_before INT NOT NULL,
  stock_after INT NOT NULL,
  reference_type VARCHAR(30) NULL,
  reference_id BIGINT UNSIGNED NULL,
  notes VARCHAR(255) NULL,
  created_by BIGINT UNSIGNED NULL,
  created_at DATETIME(3) NULL,
  KEY idx_sm_product (product_id),
  KEY idx_sm_type (type),
  KEY idx_sm_created (created_at),
  CONSTRAINT fk_sm_product FOREIGN KEY (product_id) REFERENCES products(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS customers (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  phone VARCHAR(30) NULL,
  email VARCHAR(100) NULL,
  address VARCHAR(255) NULL,
  notes VARCHAR(255) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  deleted_at DATETIME(3) NULL,
  KEY idx_customers_name (name),
  KEY idx_customers_deleted (deleted_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS transactions (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  invoice_no VARCHAR(50) NOT NULL,
  customer_id BIGINT UNSIGNED NULL,
  owner_id BIGINT UNSIGNED NOT NULL,
  subtotal DECIMAL(15,2) NOT NULL DEFAULT 0,
  discount_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
  discount_percent DECIMAL(5,2) NOT NULL DEFAULT 0,
  tax_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
  tax_percent DECIMAL(5,2) NOT NULL DEFAULT 0,
  total DECIMAL(15,2) NOT NULL DEFAULT 0,
  paid_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
  change_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
  payment_method VARCHAR(20) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'completed',
  void_reason VARCHAR(255) NULL,
  voided_at DATETIME(3) NULL,
  notes VARCHAR(255) NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  UNIQUE KEY uk_transactions_invoice (invoice_no),
  KEY idx_transactions_created (created_at),
  KEY idx_transactions_status (status),
  KEY idx_transactions_payment (payment_method),
  KEY idx_transactions_customer (customer_id),
  CONSTRAINT fk_trx_customer FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE SET NULL,
  CONSTRAINT fk_trx_owner FOREIGN KEY (owner_id) REFERENCES owners(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS transaction_items (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  transaction_id BIGINT UNSIGNED NOT NULL,
  product_id BIGINT UNSIGNED NOT NULL,
  product_name VARCHAR(150) NOT NULL,
  product_sku VARCHAR(50) NULL,
  quantity INT NOT NULL,
  buy_price DECIMAL(15,2) NOT NULL DEFAULT 0,
  sell_price DECIMAL(15,2) NOT NULL DEFAULT 0,
  discount_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
  subtotal DECIMAL(15,2) NOT NULL DEFAULT 0,
  created_at DATETIME(3) NULL,
  KEY idx_ti_trx (transaction_id),
  KEY idx_ti_product (product_id),
  CONSTRAINT fk_ti_trx FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
  CONSTRAINT fk_ti_product FOREIGN KEY (product_id) REFERENCES products(id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS settings (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  store_name VARCHAR(150) NOT NULL DEFAULT 'Toko Saya',
  store_address VARCHAR(255) NULL,
  store_phone VARCHAR(30) NULL,
  store_logo VARCHAR(255) NULL,
  invoice_prefix VARCHAR(20) DEFAULT 'INV',
  tax_percent DECIMAL(5,2) DEFAULT 0,
  default_min_stock INT DEFAULT 5,
  receipt_width_mm INT DEFAULT 80,
  currency_symbol VARCHAR(10) DEFAULT 'Rp',
  updated_at DATETIME(3) NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS activity_logs (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  owner_id BIGINT UNSIGNED NULL,
  action VARCHAR(50) NOT NULL,
  entity_type VARCHAR(50) NULL,
  entity_id BIGINT UNSIGNED NULL,
  details TEXT NULL,
  ip_address VARCHAR(45) NULL,
  created_at DATETIME(3) NULL,
  KEY idx_al_owner (owner_id),
  KEY idx_al_action (action),
  KEY idx_al_created (created_at)
) ENGINE=InnoDB;
