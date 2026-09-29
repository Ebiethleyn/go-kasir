package backup

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-kasir/backend/internal/config"
)

func findMySQLBin(name string) string {
	candidates := []string{name}
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			`C:\laragon\bin\mysql\mysql-8.4.3-winx64\bin\`+name+".exe",
			`C:\xampp\mysql\bin\`+name+".exe",
			`C:\Program Files\MySQL\MySQL Server 8.0\bin\`+name+".exe",
		)
	}
	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil {
			return path
		}
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return name
}

// Dump generates SQL dump using mysqldump if available.
func Dump(cfg *config.Config) ([]byte, string, error) {
	bin := findMySQLBin("mysqldump")
	args := []string{
		"-h", cfg.DBHost,
		"-P", cfg.DBPort,
		"-u", cfg.DBUser,
		"--single-transaction",
		"--routines",
		"--triggers",
		cfg.DBName,
	}
	cmd := exec.Command(bin, args...)
	if cfg.DBPassword != "" {
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+cfg.DBPassword)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, "", fmt.Errorf("mysqldump gagal: %v (%s)", err, stderr.String())
	}
	fname := fmt.Sprintf("backup_%s_%s.sql", cfg.DBName, time.Now().Format("20060102_150405"))
	return stdout.Bytes(), fname, nil
}

// Restore runs uploaded SQL against the database.
func Restore(cfg *config.Config, sqlReader io.Reader) error {
	bin := findMySQLBin("mysql")
	args := []string{
		"-h", cfg.DBHost,
		"-P", cfg.DBPort,
		"-u", cfg.DBUser,
		cfg.DBName,
	}
	cmd := exec.Command(bin, args...)
	if cfg.DBPassword != "" {
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+cfg.DBPassword)
	}
	cmd.Stdin = sqlReader
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("restore gagal: %v (%s)", err, stderr.String())
	}
	return nil
}

// SaveLocal writes dump to a backups folder.
func SaveLocal(dir string, data []byte, filename string) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return path, nil
}

func SanitizeSQLFilename(name string) string {
	name = filepath.Base(name)
	if !strings.HasSuffix(strings.ToLower(name), ".sql") {
		name += ".sql"
	}
	return name
}
