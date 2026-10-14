package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	iocSvc "github.com/hecc-blot/core/service/ioc"
	serverConfig "github.com/hecc-blot/framework/config/http"
	envEnum "github.com/hecc-blot/framework/enum/env"
)

// BenchmarkApiRequest 度量一个请求经过 inject → 参数绑定 → Call → 响应包装的完整开销，
// 即框架在裸 Gin 之上的额外成本。
func BenchmarkApiRequest(b *testing.B) {
	container := iocSvc.New()
	container.Set(new(greeter), &echoGreeter{})

	handle := NewApiSvc(
		&serverConfig.Config{Env: envEnum.TestMode},
		container,
	).(*ApiHandle)
	handle.SetResponseSvc(NewResponseSvc())
	handle.Get("/hello", &injectApi{})

	req := httptest.NewRequest(http.MethodGet, "/hello?name=bench", nil)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		w := httptest.NewRecorder()
		handle.Engine().ServeHTTP(w, req)
	}
}
