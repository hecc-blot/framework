package cache

type Config struct {
	Local Local
	Redis Redis
}

type Local struct {
	ClearInterval int `mapstructure:"clear_interval"`
}

type Redis struct {
	// redis服务器地址，ip:port格式，比如：192.168.1.100:6379
	// 默认为 :6379
	Addr string
	// 当redis服务器版本在6.0以上时，作为ACL认证信息配合密码一起使用，
	// 当redis服务器版本在6.0以下时，仅作为密码认证。
	// ACL是redis 6.0以上版本提供的认证功能，6.0以下版本仅支持密码认证。
	// 默认为空，不进行认证。
	Password string
	// 连接池最大连接数量，注意：这里不包括 pub/sub，pub/sub 将使用独立的网络连接
	// 默认为 10 * runtime.GOMAXPROCS
	PoolSize int
	// redis DB 数据库，默认为0
	DB int
}
