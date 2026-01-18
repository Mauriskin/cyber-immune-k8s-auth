package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	pb_refmon "token-enforcer/refmon" // gRPC-клиент Reference Monitor
	pb_sec "token-enforcer/security"  // gRPC-клиент Security Controller

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	refmonAddr  = "refmon-svc.domain3-tcb.svc.cluster.local:50051"
	secCtrlAddr = "security-controller-svc.domain3-tcb.svc.cluster.local:50052"
)

var refmonClient pb_refmon.ReferenceMonitorClient
var securityClient pb_sec.SecurityControllerClient

// Инициализация клиентов при старте
func initClients() {
	// Клиент к Reference Monitor (TCB)
	connRef, err := grpc.Dial(refmonAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Failed to connect to Reference Monitor: %v", err)
	}
	refmonClient = pb_refmon.NewReferenceMonitorClient(connRef)
	log.Println("Connected to Reference Monitor")

	// Клиент к Security Controller
	connSec, err := grpc.Dial(secCtrlAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Failed to connect to Security Controller: %v", err)
	}
	securityClient = pb_sec.NewSecurityControllerClient(connSec)
	log.Println("Connected to Security Controller")
}

// Проверка политики
func checkPolicy(sourceDomain, targetDomain, action string) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := securityClient.CheckInteraction(ctx, &pb_sec.CheckRequest{
		SourceDomain: sourceDomain,
		TargetDomain: targetDomain,
		Action:       action,
	})
	if err != nil {
		log.Printf("Security Controller unreachable: %v", err)
		return false, "controller error"
	}
	if !resp.Allowed {
		log.Printf("Policy violation: %s (action %s from %s to %s)", resp.Reason, action, sourceDomain, targetDomain)
	}
	return resp.Allowed, resp.Reason
}

// === /issue ===
type IssueRequest struct {
	Subject string `json:"subject"`
}

func issueHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Проверка политики перед вызовом TCB
	allowed, reason := checkPolicy("domain3_token_enforcer", "domain3_reference_monitor", "issue_token")
	if !allowed {
		http.Error(w, fmt.Sprintf("Policy violation: %s", reason), http.StatusForbidden)
		return
	}

	var req IssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Subject == "" {
		http.Error(w, "Missing subject", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := refmonClient.Issue(ctx, &pb_refmon.IssueRequest{
		Subject:    req.Subject,
		TtlSeconds: 60,
	})
	if err != nil {
		log.Printf("Reference Monitor Issue error: %v", err)
		http.Error(w, "Failed to issue token", http.StatusInternalServerError)
		return
	}

	// Возвращаем чистый токен как text/plain (как в вашей текущей реализации)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(resp.Token)
}

// === /validate ===
func validateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Проверка политики перед вызовом reference monitor
	allowed, reason := checkPolicy("domain3_token_enforcer", "domain3_reference_monitor", "validate_token")
	if !allowed {
		http.Error(w, fmt.Sprintf("Policy violation: %s", reason), http.StatusForbidden)
		return
	}

	token := r.Header.Get("Authorization")
	if len(token) < 8 || token[:7] != "Bearer " {
		http.Error(w, "Missing or invalid Authorization header", http.StatusUnauthorized)
		return
	}
	token = token[7:]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := refmonClient.Validate(ctx, &pb_refmon.ValidateRequest{
		Token: []byte(token),
	})
	if err != nil {
		log.Printf("Reference Monitor Validate error: %v", err)
		http.Error(w, "Validation failed", http.StatusInternalServerError)
		return
	}

	result := struct {
		Valid bool   `json:"valid"`
		Error string `json:"error,omitempty"`
	}{
		Valid: resp.Valid,
		Error: resp.Error,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func main() {
	// Инициализация обоих gRPC-клиентов
	initClients()

	// HTTP-обработчики
	http.HandleFunc("/issue", issueHandler)
	http.HandleFunc("/validate", validateHandler)

	log.Println("Token Enforcer (outside TCB) listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
