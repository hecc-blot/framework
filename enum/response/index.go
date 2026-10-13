package response

import "github.com/hecc-blot/core/enum/errorcode"

// Value 对外响应码，别名到 core 的 errorcode.Code。
//
// 用别名（不是新类型）的两个理由：
//  1. framework 段（10000 / 40000 / 50000…）与 core 段（20000–29999）从此是**同一个类型**，
//     边界可以直接把 core 码原样发出去（默认透传），不需要任何类型转换；
//  2. 现有所有 response.Value 的引用与常量声明一字不用改。
//
// 于是「对外发哪个码」这件事由渲染层的映射表决定（见 framework/service/error 的 MapCode），
// 而不是由类型系统决定。
type Value = errorcode.Code

const (
	Success          Value = 10000 // 成功
	Processing       Value = 10001 // 处理中
	Fail             Value = 40000 // 失败
	Busy             Value = 40001 // 业务繁忙
	ValidateError    Value = 40002 // 参数验证失败
	TokenInvalid     Value = 40003 // 无效token
	AccessDenied     Value = 40004 // 禁止访问
	NoDataPermission Value = 40005 // 无数据处理权限
	RateLimit        Value = 40006 // 请求过于频繁
	Illegal          Value = 50000 // 非法请求
	Panic            Value = 50001 // 服务器内部错误
)

// CodeMap 协议码 → 对外文案。
//
// 文案面向客户端，统一英文（与 core 段 errorcode.Message 同风格）；
// 注释与内部诊断消息不受此约束。已发布的文案视为对外契约的一部分，改动需评估客户端。
var CodeMap = map[Value]string{
	Success:          "success",
	Processing:       "processing",
	Fail:             "request failed",
	Busy:             "server busy",
	ValidateError:    "invalid parameters",
	TokenInvalid:     "invalid token",
	AccessDenied:     "access denied",
	NoDataPermission: "no permission to process data",
	RateLimit:        "too many requests",
	Illegal:          "illegal request",
	Panic:            "internal server error",
}
