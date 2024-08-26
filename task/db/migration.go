package db

import (
	"context"
	"ctl/settings"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"regexp"
	"sort"
)

type Migration struct {
	Name    string
	Content string
}

var dbShardNameExp = regexp.MustCompile("^*_shard_\\d{2}$")

func applyMigrations(ctx context.Context, sqlDb *sql.DB, dbName string, migrations []Migration) error {
	log.Default().Printf("Apply migrations for \"%s\" database...", dbName)

	conn, err := sqlDb.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	for _, migration := range migrations {
		if err := applyMigration(ctx, conn, migration); err != nil {
			return err
		}
	}

	log.Default().Printf("Migrations applied for \"%s\" database!", dbName)
	return nil
}

func applyMigration(ctx context.Context, conn *sql.Conn, migration Migration) (err error) {
	log.Default().Printf("Apply migration: %s, content: %s", migration.Name, migration.Content)

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	if _, err := tx.ExecContext(ctx, migration.Content); err != nil {
		return err
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?)", defaultMigrationsTable, defaultMigrationsTableColumnName)

	if _, err = tx.ExecContext(ctx, sql, migration.Name); err != nil {
		return err
	}

	log.Default().Printf("Migration %s applied!", migration.Name)
	return nil
}

func applyMigrationsToShardDBs(ctx context.Context, dbSettings settings.DBSettings, files []string) error {
	shardSpecs := getShardsSpecs(dbSettings)
	if len(shardSpecs) == 0 {
		return ErrShardDataBasesNotExist
	}

	for _, spec := range shardSpecs {
		err := applyMigrationsToDB(ctx, spec, files)
		if err != nil {
			return err
		}
	}

	return nil
}

func applyMigrationsToDB(ctx context.Context, dbSpec settings.DBSpec, files []string) error {

	sqlDb, err := sql.Open(dbSpec.Diver, dbSpec.ConnStr())
	if err != nil {
		return err
	}
	defer sqlDb.Close()

	appliedMigrations, err := loadAppliedMigrations(ctx, sqlDb)
	if err != nil {
		return err
	}

	notAppliedMigrationsFiles := getNotAppliedMigrations(files, appliedMigrations)

	migrations, err := loadMigrationsFromFiles(dbSpec.Name, notAppliedMigrationsFiles)
	if err != nil {
		return err
	}

	err = applyMigrations(ctx, sqlDb, dbSpec.Name, migrations)
	if err != nil {
		return err
	}

	return nil
}

func getShardsSpecs(dbSettings settings.DBSettings) map[string]settings.DBSpec {
	m := make(map[string]settings.DBSpec, len(dbSettings.DBs))

	for dbName, spec := range dbSettings.DBs {
		if dbShardNameExp.MatchString(dbName) {
			m[dbName] = spec
		}
	}

	return m
}

func loadAppliedMigrations(ctx context.Context, sqlDb *sql.DB) (map[string]struct{}, error) {
	log.Printf("Loading migrations from \"%s\" table", defaultMigrationsTable)

	conn, err := sqlDb.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rows, err := conn.QueryContext(ctx, "SELECT * FROM ?", defaultMigrationsTable)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := rows.Close() //Обработка ошибок во время закрытия строк
		if err != nil {
			log.Printf("failed to close rows: %v\n", err)
		}
	}()

	applied := make(map[string]struct{}, 0)
	for rows.Next() {
		var migrationName string
		if err := rows.Scan(&migrationName); err != nil {
			return nil, err
		}

		log.Printf("Loaded migration: %s", migrationName)

		applied[migrationName] = struct{}{}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	log.Printf("Loading migrations from \"%s\" table done!", defaultMigrationsTable)

	return applied, nil
}

func getNotAppliedMigrations(files []string, appliedMigrations map[string]struct{}) []string {
	notAppliedMigrations := make([]string, 0, len(files))
	for _, name := range files {
		if _, ok := appliedMigrations[name]; !ok {
			notAppliedMigrations = append(notAppliedMigrations, name)
		}
	}

	return notAppliedMigrations
}

func loadMigrationsFromFiles(dbName string, files []string) ([]Migration, error) {
	log.Default().Printf("Loading not applied migrations for \"%s\"", dbName)
	migrations := make([]Migration, 0, len(files))

	for _, notAppliedMigration := range files {
		p := path.Join(dbName, notAppliedMigration)
		migrationContent, err := ioutil.ReadFile(p)
		if err != nil {
			return nil, err
		}

		log.Default().Printf("Load migration: %s; content: %s", p, migrationContent)

		m := Migration{
			Name:    notAppliedMigration,
			Content: string(migrationContent),
		}
		migrations = append(migrations, m)
	}

	sort.SliceStable(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})
	log.Default().Printf("Loading not applied migrations for \"%s\" done!", dbName)
	return migrations, nil
}
