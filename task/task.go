package task

import (
	"context"
	"ctl/settings"
	"ctl/task/db"
	"errors"
	"fmt"
)

type Type string

const (
	dbInit      Type = "db_init"
	dbMigrate   Type = "db_migrate"
	dbDrop      Type = "db_drop"
	dbDropAll   Type = "db_drop_all"
	dbDropTable Type = "db_drop_table"
)

var (
	ErrParamDatabaseName      = errors.New("set \"Database name\" value as 2-d param")
	ErrParamDatabaseTableName = errors.New("set \"Table name\" value as 3-d param")
)

type Runnable interface {
	Run(ctx context.Context) error
}

func GetTaskByType(taskType Type, settings settings.Settings, args ...string) (Runnable, error) {
	switch taskType {
	case dbInit:
		return db.NewTaskInit(settings), nil
	case dbMigrate:
		return db.NewTaskApplyMigrations(settings), nil
	case dbDrop:
		if len(args) < 3 {
			return nil, ErrParamDatabaseName
		}
		return db.NewTaskDrop(settings, args[2]), nil
	case dbDropAll:
		return db.NewTaskDropAll(settings), nil
	case dbDropTable:
		if len(args) < 3 {
			return nil, ErrParamDatabaseName
		}
		if len(args) < 4 {
			return nil, ErrParamDatabaseTableName
		}
		return db.NewTaskDropTable(settings, args[2], args[3]), nil
	default:
		return nil, fmt.Errorf("unknown task \"%s\"", taskType)
	}
}
