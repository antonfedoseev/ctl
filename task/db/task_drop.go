package db

import (
	"context"
	"ctl/settings"
	"fmt"
)

type TaskDropAll struct {
	TaskDb
}

func NewTaskDropAll(settings settings.Settings) *TaskDropAll {
	return &TaskDropAll{TaskDb{settings: settings}}
}

func (t *TaskDropAll) Run(ctx context.Context) error {
	err := t.TaskDb.Run(ctx)
	if err != nil {
		return err
	}

	err = dropAllDatabases(ctx, t.dbSettings)
	if err != nil {
		return err
	}

	return nil
}

type TaskDrop struct {
	TaskDb
	dbName string
}

func NewTaskDrop(settings settings.Settings, dbName string) *TaskDrop {
	return &TaskDrop{TaskDb{settings: settings}, dbName}
}

func (t *TaskDrop) Run(ctx context.Context) error {
	err := t.TaskDb.Run(ctx)
	if err != nil {
		return err
	}

	if t.dbName == shardsDbAlias {
		shards := getShardsSpecs(t.dbSettings)
		for _, dbSpec := range shards {
			err = dropDatabase(ctx, dbSpec)
			if err != nil {
				return err
			}
		}
	} else {
		_, dbSpec, ok := findDBSpecByName(t.dbSettings, t.dbName)
		if !ok {
			return fmt.Errorf("\"%s\" %w: ", t.dbName, ErrDataBaseNotExists)
		}

		err = dropDatabase(ctx, dbSpec)
		if err != nil {
			return err
		}
	}

	return nil
}

func findDBSpecByName(dbSettings settings.DBSettings, dbName string) (
	dbAlias string, dbSpec settings.DBSpec, found bool) {

	for alias, spec := range dbSettings.DBs {
		if spec.Name == dbName {
			return alias, spec, true
		}
	}

	return "", settings.DBSpec{}, false
}

type TaskDropTable struct {
	TaskDb
	dbName    string
	tableName string
}

func NewTaskDropTable(settings settings.Settings, dbName string, tableName string) *TaskDropTable {
	return &TaskDropTable{TaskDb{settings: settings}, dbName, tableName}
}

func (t *TaskDropTable) Run(ctx context.Context) error {
	err := t.TaskDb.Run(ctx)
	if err != nil {
		return err
	}

	if t.dbName == shardsDbAlias {
		shards := getShardsSpecs(t.dbSettings)
		for _, dbSpec := range shards {
			err = dropTable(ctx, dbSpec, t.tableName)
			if err != nil {
				return err
			}
		}
	} else {
		_, dbSpec, ok := findDBSpecByName(t.dbSettings, t.dbName)
		if !ok {
			return fmt.Errorf("\"%s\" %w: ", t.dbName, ErrDataBaseNotExists)
		}

		err = dropTable(ctx, dbSpec, t.tableName)
		if err != nil {
			return err
		}
	}

	return nil
}
