CREATE TABLE IF NOT EXISTS finance_entries (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  entry_type VARCHAR(30) NOT NULL,
  amount DECIMAL(15,2) NOT NULL,
  occurred_at DATETIME(3) NOT NULL,
  description VARCHAR(255) NULL,
  created_by BIGINT UNSIGNED NOT NULL,
  created_at DATETIME(3) NULL,
  KEY idx_finance_entries_type (entry_type),
  KEY idx_finance_entries_occurred (occurred_at),
  KEY idx_finance_entries_created_by (created_by),
  CONSTRAINT fk_finance_entries_owner FOREIGN KEY (created_by) REFERENCES owners(id)
) ENGINE=InnoDB;
