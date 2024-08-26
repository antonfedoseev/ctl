package db

import (
	"context"
	"ctl/settings"
)

type TaskApplyMigrations struct {
	TaskDb
}

func NewTaskApplyMigrations(settings settings.Settings) *TaskApplyMigrations {
	return &TaskApplyMigrations{TaskDb{settings: settings}}
}

func (t *TaskApplyMigrations) Run(ctx context.Context) error {
	err := t.TaskDb.Run(ctx)
	if err != nil {
		return err
	}

	err = applyDatabasesMigrations(ctx, t.settings.DbMigrationsPath, t.dbSettings)
	if err != nil {
		return err
	}

	return nil
}
