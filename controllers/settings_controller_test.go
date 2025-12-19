package controllers

import (
	"blog-backend/models"
	"blog-backend/services"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type settingRepoCtrlStub struct {
	list     func() ([]models.Setting, error)
	upsert   func(key, value, desc string) (*models.Setting, error)
	getByKey func(key string) (*models.Setting, error)
}

func (s settingRepoCtrlStub) List() ([]models.Setting, error) {
	if s.list != nil {
		return s.list()
	}
	return nil, nil
}
func (s settingRepoCtrlStub) GetByKey(key string) (*models.Setting, error) {
	if s.getByKey != nil {
		return s.getByKey(key)
	}
	return &models.Setting{Key: key, Value: ""}, nil
}
func (s settingRepoCtrlStub) Upsert(key, value, description string) (*models.Setting, error) {
	if s.upsert != nil {
		return s.upsert(key, value, description)
	}
	return &models.Setting{Key: key, Value: value, Desc: description}, nil
}

func TestSettingController_List_And_Update(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := services.NewSettingService(settingRepoCtrlStub{
		list: func() ([]models.Setting, error) { return []models.Setting{{Key: "k", Value: "v"}}, nil },
		upsert: func(key, value, desc string) (*models.Setting, error) {
			return &models.Setting{Key: key, Value: value, Desc: desc}, nil
		},
	})
	sc := NewSettingController(svc)

	r := gin.New()
	r.GET("/settings", sc.List)
	r.PUT("/settings/:key", func(c *gin.Context) {
		c.Set("currentUser", controllerUser(1, "manage_settings"))
		sc.Update(c)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// unauthorized
	r2 := gin.New()
	r2.PUT("/settings/:key", sc.Update)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/settings/k", bytes.NewBufferString(`{"value":"v"}`))
	req.Header.Set("Content-Type", "application/json")
	r2.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	// empty key
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/settings/%20", bytes.NewBufferString(`{"value":"v"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// bind error
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPut, "/settings/k", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)

	// success
	w = httptest.NewRecorder()
	body, _ := json.Marshal(updateSettingRequest{Value: "v", Description: "d"})
	req = httptest.NewRequest(http.MethodPut, "/settings/k", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestSettingController_List_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := services.NewSettingService(settingRepoCtrlStub{list: func() ([]models.Setting, error) { return nil, errors.New("boom") }})
	sc := NewSettingController(svc)

	r := gin.New()
	r.GET("/settings", sc.List)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/settings", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestSettingController_Update_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := services.NewSettingService(settingRepoCtrlStub{upsert: func(key, value, desc string) (*models.Setting, error) {
		return nil, errors.New("boom")
	}})
	sc := NewSettingController(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("currentUser", controllerUser(1, "manage_settings")) })
	r.PUT("/settings/:key", sc.Update)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/settings/k", bytes.NewBufferString(`{"value":"v"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
