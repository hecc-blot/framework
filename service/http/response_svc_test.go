package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

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
