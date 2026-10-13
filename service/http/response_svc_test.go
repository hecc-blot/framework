package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/hecc-blot/core/enum/errorcode"
	"github.com/hecc-blot/framework/enum/response"
	errorSvc "github.com/hecc-blot/framework/service/error"
)

func TestResponseSvcRegular(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := NewResponseSvc()

	t.Run("成功响应", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		svc.Regular(NewContext(c), "hello", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp ResponseSvc
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, response.Success, resp.Code)
		assert.Equal(t, response.CodeMap[response.Success], resp.Message)
		assert.Equal(t, "hello", resp.Data)
	})

	t.Run("错误响应", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		svc.Regular(NewContext(c), "original", errorSvc.NewError(response.Fail, errors.New("db error")))

		assert.Equal(t, http.StatusOK, w.Code)
		var resp ResponseSvc
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, response.Fail, resp.Code)
		assert.Equal(t, response.CodeMap[response.Fail], resp.Message)
		assert.Nil(t, resp.Data)
	})
}

// TestMessageFallbackChain：文案三跳 —— 协议码文案 → core 段文案 → Fail 兜底。
func TestMessageFallbackChain(t *testing.T) {
	// 协议码：走 CodeMap
	assert.Equal(t, response.CodeMap[response.ValidateError], message(response.ValidateError))
	// core 段码：CodeMap 里没有，回退到 errorcode.Message
	assert.Equal(t, "request timeout", message(errorcode.CodeTimeout))
	// 业务自定义码（60000+）：框架无从知道文案，兜底到 Fail 的文案而不是空串
	assert.Equal(t, response.CodeMap[response.Fail], message(60001))
}

// TestRegularPassesThroughUnmappedCoreCode：未注册映射的 core 码原码透传，文案取 core 段。
// 这条钉住"默认直用"在**响应链路**上的表现（不只是 Resolve 一个函数的行为）。
func TestRegularPassesThroughUnmappedCoreCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := NewResponseSvc()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sentinel := errorcode.New(errorcode.CodeRateLimited, "util: limit full")
	svc.Regular(NewContext(c), nil, errorSvc.From(fmt.Errorf("handle msg: %w", sentinel)))

	var resp ResponseSvc
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, errorcode.CodeRateLimited, resp.Code)
	assert.Equal(t, "too many requests", resp.Message)
}

// TestRegularTranslatesMappedCoreCode：注册映射后，同一个 core 码在响应里被翻译成协议码，
// 文案随之取协议码的文案 —— 逐码、可选，而不是"翻译模式"开关。
func TestRegularTranslatesMappedCoreCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	errorSvc.MapCode(errorcode.CodeServerBusy, response.Busy)

	svc := NewResponseSvc()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	sentinel := errorcode.New(errorcode.CodeServerBusy, "circuitbreaker: circuit open")
	svc.Regular(NewContext(c), nil, errorSvc.From(sentinel))

	var resp ResponseSvc
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, response.Busy, resp.Code)
	assert.Equal(t, response.CodeMap[response.Busy], resp.Message)
}
