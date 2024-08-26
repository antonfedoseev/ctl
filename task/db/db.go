package db

import (
	"context"
	"ctl/settings"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
)

const (
	defaultMigrationsTable           = "schema_migrations"
	defaultMigrationsTableColumnName = "migration"
	shardsDBAlias                    = "shards"
)

var (
	ErrWrongPath              = errors.New("wrong migrations directory path")
	ErrWrongDbSettingsPath    = errors.New("failed to read database settings")
	ErrReadPath               = errors.New("error of migrations directory path")
	ErrEmptyDirectory         = errors.New("migrations directory is empty")
	ErrDataBaseNotExists      = errors.New("database not exists in settings")
	ErrShardDataBasesNotExist = errors.New("shard databases not exist in settings")
	ErrDatabasesCreation      = errors.New("can't create databases")
)

var dbMigrationsFileExp = regexp.MustCompile("^*.sql$")

type TaskDb struct {
	settings   settings.Settings
	dbSettings settings.DBSettings
}

func (t *TaskDb) Run(ctx context.Context) error {
	s, err := initSettings(t.settings.DbSettingsPath)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrWrongDbSettingsPath, err)
	}

	t.dbSettings = s
	return nil
}

func initSettings(path string) (settings.DBSettings, error) {
	s := settings.DBSettings{}
	err := s.Read(path)
	return s, err
}

func getMigrationsFilesPaths(files []os.DirEntry) []string {
	fileNames := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		path := file.Name()
		if !dbMigrationsFileExp.MatchString(path) {
			continue
		}

		fileNames = append(fileNames, path)
	}

	return fileNames
}

func applyDatabasesMigrations(ctx context.Context, migrationsPath string, dbSettings settings.DBSettings) error {
	if len(migrationsPath) == 0 {
		return ErrWrongPath
	}

	dirs, err := os.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("%v: %w", ErrReadPath, err)
	}

	if len(dirs) == 0 {
		return ErrEmptyDirectory
	}

	for _, v := range dirs {
		if !v.IsDir() {
			continue
		}

		directoryInfo, err := v.Info()
		if err != nil {
			log.Default().Printf("Skip database directory \"%s\" reading. Error: %v\n", directoryInfo.Name(), err)
			continue
		}

		files, err := os.ReadDir(v.Name())
		if err != nil {
			log.Default().Printf("Skip database directory \"%s\" reading. Read migrations error: %v\n", directoryInfo.Name(), err)
			continue
		}

		filePaths := getMigrationsFilesPaths(files)
		if len(filePaths) == 0 {
			log.Default().Printf("Skip database directory \"%s\". There are no migrations here!\n", directoryInfo.Name())
			continue
		}

		err = apply(ctx, dbSettings, directoryInfo.Name(), filePaths)
		if err != nil {
			return err
		}
	}

	return nil
}

func apply(ctx context.Context, dbSettings settings.DBSettings, dbAlias string, files []string) error {
	if dbAlias == shardsDBAlias {
		return applyMigrationsToShardDBs(ctx, dbSettings, files)
	}

	dbSpec, ok := dbSettings.DBs[dbAlias]
	if !ok {
		return fmt.Errorf("\"%s\" %w: ", dbAlias, ErrDataBaseNotExists)
	}

	return applyMigrationsToDB(ctx, dbSpec, files)
}

func createDatabases(ctx context.Context, dbSettings settings.DBSettings) error {
	for _, spec := range dbSettings.DBs {
		if err := createDatabase(ctx, spec); err != nil {
			return fmt.Errorf("%v: %w", ErrDatabasesCreation, err)
		}
	}

	return nil
}

func createDatabase(ctx context.Context, dbSpec settings.DBSpec) error {
	db, err := sql.Open(dbSpec.Diver, dbSpec.ConnStr())
	if err != nil {
		return err
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbSpec.Name)); err != nil {
		return err
	}

	err = createMigrationsTable(ctx, conn)
	if err != nil {
		return err
	}

	return nil
}

func dropDatabase(ctx context.Context, dbSpec settings.DBSpec) error {
	db, err := sql.Open(dbSpec.Diver, dbSpec.ConnStr())
	if err != nil {
		return err
	}

	log.Default().Printf("Dropping database: \"%s\"...", dbSpec.Name)

	if _, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbSpec.Name)); err != nil {
		log.Default().Printf(" FAIL!")
		return err
	}

	log.Default().Println(" DONE!")

	return nil
}

func dropAllDatabases(ctx context.Context, dbSettings settings.DBSettings) error {
	for _, spec := range dbSettings.DBs {
		err := dropDatabase(ctx, spec)
		if err != nil {
			return err
		}
	}

	return nil
}

func createMigrationsTable(ctx context.Context, conn *sql.Conn) error {
	sql := fmt.Sprintf(
		"CREATE TABLE IF NOT EXISTS %s (`%s` String) ENGINE = MergeTree() ORDER BY %s",
		defaultMigrationsTable,
		defaultMigrationsTableColumnName,
		defaultMigrationsTableColumnName,
	)

	if _, err := conn.ExecContext(ctx, sql); err != nil {
		return err
	}

	return nil
}

func dropTable(ctx context.Context, dbSpec settings.DBSpec, tableName string) error {
	db, err := sql.Open(dbSpec.Diver, dbSpec.ConnStr())
	if err != nil {
		return err
	}

	log.Default().Printf("Dropping database: \"%s\"...", dbSpec.Name)

	sql := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)

	if _, err := db.ExecContext(ctx, sql); err != nil {
		return err
	}

	return nil
}
