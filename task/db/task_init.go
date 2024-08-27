package db

import (
	"context"
	"ctl/settings"
)

type TaskInit struct {
	TaskDb
}

func NewTaskInit(settings settings.Settings) *TaskInit {
	return &TaskInit{TaskDb{settings: settings}}
}

func (t *TaskInit) Run(ctx context.Context) error {
	err := t.TaskDb.Run(ctx)
	if err != nil {
		return err
	}

	err = createDatabases(ctx, t.dbSettings)
	if err != nil {
		return err
	}

	err = createMigrationsTables(ctx, t.dbSettings)
	if err != nil {
		return err
	}

	err = applyDatabasesMigrations(ctx, t.settings.DbMigrationsPath, t.dbSettings)
	if err != nil {
		return err
	}

	return nil
}
