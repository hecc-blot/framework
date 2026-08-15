package db

import (
	"context"

	dbEnum "github.com/hecc-blot/hecc-blot-core/enum/db"
)

type IDbFactory interface {
	Build(ctx context.Context, value ...dbEnum.Value) IDb
	SetDefault(dbEnum.Value)
}
