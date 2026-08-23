package service

import (
	"fmt"

	logConfig "github.com/hecc-blot/framework/config/log"
	logContract "github.com/hecc-blot/framework/contract/log"
)

// NewLogger 创建本地日志服务（框架默认 logger）。
// SLS 等厂商后端见 github.com/hecc-blot/log-sls/service，由组装层按配置二选一。
func NewLogger(config *logConfig.LocalConfig) (logContract.ILog, error) {
	if !config.Enable {
		return nil, fmt.Errorf("未启用本地日志服务")
	}
	return newLogSvc(config)
}
