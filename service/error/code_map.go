package service

import (
	"sync"

	"github.com/hecc-blot/core/enum/errorcode"
	"github.com/hecc-blot/framework/enum/response"
)

// 映射表：core 段码 → 对外协议码。**空表 = 直用**（core 码原样对外）。
//
// 设计意图是「core 定义码、framework 决定暴露哪些」：默认一个码都不翻译，
// 想让 core 段不出现在响应里的业务，按需把关心的那几个码注册进来 —— 逐码、增量，
// 而不是给一个「翻译模式」开关。
//
// 为什么不做运行时开关：对外码表是对客户端的契约。开关一开一关，同一个错误在不同
// 部署/环境会返回不同的码，客户端的码表、文档与回归用例都得覆盖两条路径。
// 把决定放在代码里（表是否为空），它才随二进制走、可 review、可测试。
var (
	codeMapMu sync.RWMutex
	codeMap   = map[errorcode.Code]response.Value{}
)

// MapCode 注册一个 core 码的对外翻译，如 errorcode.CodeTimeout → response.Busy。
// 未注册的码保持透传。只需在启动阶段调用；运行时并发调用是安全的，但语义上只该配一次。
func MapCode(from errorcode.Code, to response.Value) {
	codeMapMu.Lock()
	defer codeMapMu.Unlock()
	codeMap[from] = to
}

// Resolve 查映射表：命中则返回翻译后的对外码与 true；
// 未命中返回入参本身与 false —— 调用方据此「原码透传」。
func Resolve(code errorcode.Code) (response.Value, bool) {
	codeMapMu.RLock()
	defer codeMapMu.RUnlock()
	to, ok := codeMap[code]
	if !ok {
		return code, false
	}
	return to, true
}
