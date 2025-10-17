package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cobalto/noppera/internal/config"
	"github.com/cobalto/noppera/internal/middleware"
	"github.com/cobalto/noppera/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// RegisterAuth sets up authentication routes.
func RegisterAuth(r *chi.Mux, db *pgxpool.Pool, cfg config.Config) {
	r.Post("/auth/register", register(db))
	r.Post("/auth/login", login(db, cfg))
	r.With(middleware.Auth(cfg), middleware.AdminOnly).Post("/auth/register/admin", registerAdmin(db))
}

// register handles POST /auth/register, creating a new user.
// @Summary Register a new user
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param user body models.User true "User registration data"
// @Success 201 {object} map[string]interface{} "User created successfully"
// @Failure 400 {string} string "Invalid request body"
// @Failure 500 {string} string "Failed to create user"
// @Router /auth/register [post]
func register(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if user.Username == "" || user.Password == "" {
			http.Error(w, "Username and password required", http.StatusBadRequest)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}
		user.Password = string(hashedPassword)
		user.IsAdmin = false

		ctx := r.Context()
		if err := models.CreateUser(ctx, db, &user); err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
		})
	}
}

// login handles POST /auth/login, issuing a JWT.
// @Summary Login user
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body object{username=string,password=string} true "Login credentials"
// @Success 200 {object} map[string]string "Login successful"
// @Failure 400 {string} string "Invalid request body"
// @Failure 401 {string} string "Invalid credentials"
// @Failure 500 {string} string "Failed to generate token"
// @Router /auth/login [post]
func login(db *pgxpool.Pool, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		user, err := models.GetUserByUsername(ctx, db, creds.Username)
		if err != nil || user == nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(creds.Password)); err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Parse JWTExpiry duration
		expiry, err := time.ParseDuration(cfg.JWTExpiry)
		if err != nil {
			http.Error(w, "Invalid JWT expiry configuration", http.StatusInternalServerError)
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, middleware.User{
			ID:       user.ID,
			Username: user.Username,
			IsAdmin:  user.IsAdmin,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		})

		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
	}
}

// registerAdmin handles POST /auth/register/admin, creating an admin user.
// @Summary Register admin user
// @Description Create a new admin user account (admin only)
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user body models.User true "Admin user registration data"
// @Success 201 {object} map[string]interface{} "Admin user created successfully"
// @Failure 400 {string} string "Invalid request body"
// @Failure 401 {string} string "Unauthorized"
// @Failure 403 {string} string "Admin access required"
// @Failure 500 {string} string "Failed to create admin"
// @Router /auth/register/admin [post]
func registerAdmin(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user models.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if user.Username == "" || user.Password == "" {
			http.Error(w, "Username and password required", http.StatusBadRequest)
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}
		user.Password = string(hashedPassword)
		user.IsAdmin = true

		ctx := r.Context()
		if err := models.CreateUser(ctx, db, &user); err != nil {
			http.Error(w, "Failed to create admin", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"is_admin": user.IsAdmin,
		})
	}
}
