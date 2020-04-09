package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const DEFAULT_MIGRATIONS_DIR = "../migrations"

func GenerateMigrationName(name string) string {
	today := time.Now().UTC()
	return fmt.Sprintf("%d_%s.sql", today.Unix(), name)
}

func GenerateMigration(dir, name string) error {
	migration_name := GenerateMigrationName(name)

	content := fmt.Sprintf("-- Migration name: %s", migration_name)
	byte_content := []byte(content)
	file_name := fmt.Sprintf("%s/%s", dir, migration_name)
	if err := ioutil.WriteFile(file_name, byte_content, 0644); err != nil {
		return err
	}

	return nil
}

func ReadMigration(migration_path string) ([]byte, error) {
	return ioutil.ReadFile(migration_path)
}

func GetMigrationName(migration_path string) string {
	tmp := strings.Split(migration_path, "/")
	return tmp[len(tmp)-1]
}

func ExistsDir(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}

	return true
}

func SetupMigrationsDir(dir string) (string, error) {
	if dir != "" {
		abs_dir, err := filepath.Abs(dir)
		if err != nil {
			return "", err
		}

		if !ExistsDir(abs_dir) {
			return CreateDir(abs_dir)
		}
		return abs_dir, nil
	} else {
		if !ExistsDir(dir) {
			return CreateDir(DEFAULT_MIGRATIONS_DIR)
		}
		return DEFAULT_MIGRATIONS_DIR, nil
	}
}

func CreateDir(dir string) (string, error) {
	if err := os.Mkdir(dir, 0755); err != nil {
		return "", err
	}

	return dir, nil
}
