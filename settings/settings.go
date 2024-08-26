package settings

import (
	"encoding/json"
	"fmt"
	"os"
)

type Settings struct {
	DbMigrationsPath string `json:"db_migrations_path"`
	DbSettingsPath   string `json:"db_settings_path"`
}

func (s *Settings) Read(filePath string) error {
	dat, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(dat, s)
	if err != nil {
		return err
	}

	return nil
}

type DBSpec struct {
	Diver    string `json:"diver"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type DBSettings struct {
	DBs map[string]DBSpec `json:"dbs"`
}

func (s *DBSpec) ConnStr() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", s.Username, s.Password, s.Host, s.Port, s.Name)
}

func (s *DBSettings) Read(filePath string) error {
	dat, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(dat, s)
	if err != nil {
		return err
	}

	return nil
}
