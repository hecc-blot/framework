package response

type Value int

const (
	Success          Value = 10000 // 成功
	Processing       Value = 10001 // 处理中
	Fail             Value = 40000 // 失败
	Busy             Value = 40001 // 业务繁忙
	ValidateError    Value = 40002 // 参数验证失败
	TokenInvalid     Value = 40003 // 无效token
	AccessDenied     Value = 40004 // 禁止访问
	NoDataPermission Value = 40005 // 无数据处理权限
	Illegal          Value = 50000 // 非法请求
	Panic            Value = 50001 // 服务器内部错误
)

var CodeMap = map[Value]string{
	Success:          "请求成功",
	Processing:       "处理中",
	Fail:             "请求失败",
	Busy:             "业务繁忙",
	ValidateError:    "参数校验失败",
	TokenInvalid:     "token失效",
	AccessDenied:     "无权访问",
	NoDataPermission: "无权处理",
	Illegal:          "非法请求",
	Panic:            "程序异常",
}
