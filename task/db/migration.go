package db

import (
	"context"
	"ctl/settings"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"sort"
)

type Migration struct {
	Name    string
	Content string
}

type Migrations struct {
	Directory string
	Files     []string
	DbAlias   string
}

var dbShardNameExp = regexp.MustCompile("^shard_\\d{2}$")

func applyMigrations(ctx context.Context, sqlDb *sql.DB, dbName string, migrations []Migration) error {
	log.Default().Printf("\nApply migrations for \"%s\" database...", dbName)

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

	log.Default().Printf("Migrations applied for \"%s\" database!\n", dbName)
	return nil
}

func applyMigration(ctx context.Context, conn *sql.Conn, migration Migration) (err error) {
	log.Default().Printf("\nApply migration: %s, content:\n%s\n", migration.Name, migration.Content)

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
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (?);", defaultMigrationsTable, defaultMigrationsTableColumnName)

	if _, err = tx.ExecContext(ctx, sql, migration.Name); err != nil {
		return err
	}

	log.Default().Printf("Migration %s applied!\n", migration.Name)
	return nil
}

func applyMigrationsToShardDBs(ctx context.Context, dbSettings settings.DBSettings, migrations Migrations) error {
	shardSpecs := getShardsSpecs(dbSettings)
	if len(shardSpecs) == 0 {
		return ErrShardDataBasesNotExist
	}

	for _, spec := range shardSpecs {
		err := applyMigrationsToDB(ctx, spec, migrations)
		if err != nil {
			return err
		}
	}

	return nil
}

func applyMigrationsToDB(ctx context.Context, dbSpec settings.DBSpec, m Migrations) error {

	sqlDb, err := sql.Open(dbSpec.Diver, dbSpec.ConnStr())
	if err != nil {
		return err
	}
	defer sqlDb.Close()

	appliedMigrations, err := loadAppliedMigrations(ctx, sqlDb)
	if err != nil {
		return err
	}

	notAppliedMigrationsFiles := getNotAppliedMigrations(m, appliedMigrations)

	migrations, err := loadMigrationsFromFiles(m.DbAlias, m.Directory, notAppliedMigrationsFiles)
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

	for dbAlias, spec := range dbSettings.DBs {
		if dbShardNameExp.MatchString(dbAlias) {
			m[dbAlias] = spec
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

	rows, err := conn.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %s;", defaultMigrationsTable))
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

func getNotAppliedMigrations(migrations Migrations, appliedMigrations map[string]struct{}) []string {
	notAppliedMigrations := make([]string, 0, len(migrations.Files))
	for _, name := range migrations.Files {
		if _, ok := appliedMigrations[name]; !ok {
			notAppliedMigrations = append(notAppliedMigrations, name)
		}
	}

	return notAppliedMigrations
}

func loadMigrationsFromFiles(dbAlias string, directory string, files []string) ([]Migration, error) {
	log.Default().Printf("Loading not applied migrations for \"%s\"", dbAlias)
	list := make([]Migration, 0, len(files))

	for _, notAppliedMigration := range files {
		p := path.Join(directory, notAppliedMigration)
		migrationContent, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}

		log.Default().Printf("Load migration: %s", p)

		m := Migration{
			Name:    notAppliedMigration,
			Content: string(migrationContent),
		}
		list = append(list, m)
	}

	sort.SliceStable(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	log.Default().Printf("Loading not applied migrations for \"%s\" done!", dbAlias)
	return list, nil
}
