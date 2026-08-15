package db

import (
	"context"
)

type IDb interface {
	Add(entry IDbModel) error
	Count() (int64, error)
	Find(dst interface{}) error
	Limit(v int) IDb
	Offset(v int) IDb
	Order(fields ...string) IDb
	Query(entry IDbModel) IDb
	Remove(entry IDbModel) error
	Save(entry IDbModel) error
	Select(args ...interface{}) IDb
	Take(dst interface{}) error
	Where(args ...interface{}) IDb
	WithContext(ctx context.Context)
	Begin() IDb
	Commit() error
	Rollback()
	GetInstance() any
}
