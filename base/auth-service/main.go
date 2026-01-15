package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"

	pb "auth-service/security" // сгенерированный пакет из security.proto

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Hardcoded credentials для теста
const (
	validUser     = "admin"
	validPassword = "secret123"
	validMFA      = "123456"
)

// Адрес Token Enforcer (прокси к Reference Monitor)
const tokenEndpoint = "http://token-enforcer-svc.domain3-tcb.svc.cluster.local:8080/issue"

// Адрес Security Controller
const securityControllerAddr = "security-controller-svc.domain3-tcb.svc.cluster.local:50052"

// Глобальный gRPC-клиент к Security Controller
var securityClient pb.SecurityControllerClient

// Инициализация подключения к Security Controller при старте
func initSecurityClient() {
	conn, err := grpc.Dial(securityControllerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Security Controller: %v", err)
	}
	securityClient = pb.NewSecurityControllerClient(conn)
	log.Println("Connected to Security Controller")
}

// Rate limiting (in-memory)
type RateLimiter struct {
	mu    sync.Mutex
	count map[string]int
	last  map[string]time.Time
}

var limiter = RateLimiter{
	count: make(map[string]int),
	last:  make(map[string]time.Time),
}

// Allow проверяет лимит 10 запросов в минуту на IP
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if last, ok := rl.last[ip]; ok && now.Sub(last) < time.Minute {
		if rl.count[ip] >= 10 {
			return false
		}
		rl.count[ip]++
	} else {
		rl.count[ip] = 1
		rl.last[ip] = now
	}
	return true
}

// getClientIP извлекает реальный IP клиента (учитывает прокси/Ingress)
func getClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if comma := strings.Index(forwarded, ","); comma != -1 {
			return strings.TrimSpace(forwarded[:comma])
		}
		return strings.TrimSpace(forwarded)
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	MFA      string `json:"mfa"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)

	// 1. Rate limiting
	if !limiter.Allow(clientIP) {
		log.Printf("Rate limit exceeded for IP: %s", clientIP)
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// 2. Чтение и парсинг тела
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Cannot read request body", http.StatusBadRequest)
		return
	}

	var req LoginRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 3. Проверка учётных данных
	if req.Username != validUser || req.Password != validPassword || req.MFA != validMFA {
		log.Printf("Failed login attempt from IP: %s (user: %s)", clientIP, req.Username)
		http.Error(w, "Invalid credentials or MFA", http.StatusUnauthorized)
		return
	}

	log.Printf("Successful authentication for user: %s from IP: %s", req.Username, clientIP)

	// 4. Проверка политики через Security Controller
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	policyResp, err := securityClient.CheckInteraction(ctx, &pb.CheckRequest{
		SourceDomain: "domain2_authentication",
		TargetDomain: "domain3_token_policy",
		Action:       "issue_token",
		Context: map[string]string{
			"user": req.Username,
			"ip":   clientIP,
		},
	})
	if err != nil {
		log.Printf("Error checking policy: %v", err)
		http.Error(w, "Policy check failed", http.StatusInternalServerError)
		return
	}
	if !policyResp.Allowed {
		log.Printf("Policy violation: %s (user: %s, IP: %s)", policyResp.Reason, req.Username, clientIP)
		http.Error(w, fmt.Sprintf("Policy violation: %s", policyResp.Reason), http.StatusForbidden)
		return
	}

	// 5. Если политика разрешает — forward к Token Enforcer
	issuePayload := map[string]string{"subject": req.Username}
	issueBody, _ := json.Marshal(issuePayload)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(tokenEndpoint, "application/json", bytes.NewReader(issueBody))
	if err != nil {
		log.Printf("Error connecting to Token Enforcer: %v", err)
		http.Error(w, "Failed to contact token service", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Token Enforcer returned status: %d", resp.StatusCode)
		http.Error(w, "Failed to obtain token", http.StatusInternalServerError)
		return
	}

	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read token response", http.StatusInternalServerError)
		return
	}

	response := TokenResponse{Token: string(bytes.TrimSpace(tokenBytes))}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

func main() {
	// Инициализация подключения к Security Controller
	initSecurityClient()

	r := mux.NewRouter()
	r.HandleFunc("/login", loginHandler).Methods("POST")
	r.HandleFunc("/health", healthHandler).Methods("GET")

	log.Println("Auth Service starting on :8080")
	log.Printf("Forwarding token requests to: %s", tokenEndpoint)

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
