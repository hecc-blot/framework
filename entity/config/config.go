package config

import (
	"github.com/hecc-blot/hecc-blot-core/entity/config/cache"
	"github.com/hecc-blot/hecc-blot-core/entity/config/db"
	"github.com/hecc-blot/hecc-blot-core/entity/config/log"
	"github.com/hecc-blot/hecc-blot-core/entity/config/server"
	"github.com/hecc-blot/hecc-blot-core/entity/config/trace"
)

type Config struct {
	Cache  cache.Config
	Db     db.Config
	Log    log.Config
	Server server.Config
	Trace  trace.Config
}
