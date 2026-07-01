package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/medisave/app/internal/application/service"
	"github.com/medisave/app/internal/infrastructure/database/migrations"
	repo "github.com/medisave/app/internal/infrastructure/repository"
	"github.com/medisave/app/internal/presentation/http/handler"
	pkgjwt "github.com/medisave/app/pkg/jwt"
	"github.com/medisave/app/pkg/logger"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	logger.Init("error") // suppress logs during tests
	m.Run()
}

// newTestDB opens an in-memory SQLite DB and runs all migrations.
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	db.Exec("PRAGMA foreign_keys=ON")
	require.NoError(t, migrations.Run(db))
	return db
}

// newAuthRouter wires up only the auth routes for testing.
func newAuthRouter(t *testing.T) *gin.Engine {
	t.Helper()
	db := newTestDB(t)

	jwtManager := pkgjwt.NewManager("test-access-secret", "test-refresh-secret", 1, 7)

	userRepo    := repo.NewGORMUserRepository(db)
	patientRepo := repo.NewGORMPatientRepository(db)
	doctorRepo  := repo.NewGORMDoctorRepository(db)
	walletRepo  := repo.NewGORMWalletRepository(db)

	authSvc := service.NewAuthService(userRepo, patientRepo, doctorRepo, walletRepo, jwtManager)
	authHandler := handler.NewAuthHandler(authSvc)

	r := gin.New()
	auth := r.Group("/api/v1/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)

	return r
}

func post(t *testing.T, router *gin.Engine, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

func TestRegister_Success(t *testing.T) {
	router := newAuthRouter(t)

	w := post(t, router, "/api/v1/auth/register", map[string]interface{}{
		"first_name": "Test",
		"last_name":  "User",
		"email":      "test@example.com",
		"password":   "Password@123",
		"phone":      "+2348012345679",
		"role":       "patient",
	})

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["success"])

	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok, "data must be an object")
	tokens, ok := data["tokens"].(map[string]interface{})
	require.True(t, ok, "data.tokens must be an object")
	assert.NotEmpty(t, tokens["access_token"])
	assert.NotEmpty(t, tokens["refresh_token"])
}

func TestRegister_DuplicateEmail(t *testing.T) {
	router := newAuthRouter(t)

	body := map[string]interface{}{
		"first_name": "Test",
		"last_name":  "User",
		"email":      "dup@example.com",
		"password":   "Password@123",
		"phone":      "+2348099999991",
		"role":       "patient",
	}

	w1 := post(t, router, "/api/v1/auth/register", body)
	assert.Equal(t, http.StatusCreated, w1.Code)

	// Second registration with same email
	body["phone"] = "+2348099999992"
	w2 := post(t, router, "/api/v1/auth/register", body)
	assert.Equal(t, http.StatusConflict, w2.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["success"])
}

func TestRegister_InvalidBody(t *testing.T) {
	router := newAuthRouter(t)

	// Missing required fields
	w := post(t, router, "/api/v1/auth/register", map[string]interface{}{
		"email": "bad@example.com",
	})
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestLogin_Success(t *testing.T) {
	router := newAuthRouter(t)

	// Register first
	post(t, router, "/api/v1/auth/register", map[string]interface{}{
		"first_name": "Login",
		"last_name":  "Test",
		"email":      "login@example.com",
		"password":   "Password@123",
		"phone":      "+2348011111111",
		"role":       "patient",
	})

	// Login
	w := post(t, router, "/api/v1/auth/login", map[string]interface{}{
		"email":    "login@example.com",
		"password": "Password@123",
	})

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["success"])

	data := resp["data"].(map[string]interface{})
	tokens := data["tokens"].(map[string]interface{})
	assert.NotEmpty(t, tokens["access_token"])
}

func TestLogin_WrongPassword(t *testing.T) {
	router := newAuthRouter(t)

	post(t, router, "/api/v1/auth/register", map[string]interface{}{
		"first_name": "Wrong",
		"last_name":  "Pass",
		"email":      "wrongpass@example.com",
		"password":   "Password@123",
		"phone":      "+2348022222222",
		"role":       "patient",
	})

	w := post(t, router, "/api/v1/auth/login", map[string]interface{}{
		"email":    "wrongpass@example.com",
		"password": "WrongPassword!",
	})

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["success"])
}

func TestLogin_UnknownEmail(t *testing.T) {
	router := newAuthRouter(t)

	w := post(t, router, "/api/v1/auth/login", map[string]interface{}{
		"email":    "nobody@example.com",
		"password": "Password@123",
	})

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
