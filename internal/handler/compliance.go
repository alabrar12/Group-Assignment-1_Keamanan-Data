package handler

import (
	"aaa_bekend/internal/db"
	"aaa_bekend/internal/lmodel"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func GetMyData(w http.ResponseWriter, r *http.Request) {
	user, err := lmodel.UserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	profile, err := db.GetUserByID(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch user data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func UpdateMyData(w http.ResponseWriter, r *http.Request) {
	user, err := lmodel.UserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Username) > 50 {
		http.Error(w, "Username is required (max 50 characters)", http.StatusBadRequest)
		return
	}

	if err := db.UpdateMyData(r.Context(), user.ID, req.Username); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			http.Error(w, "Username already taken", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to update user data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message":  "Data updated successfully",
		"username": req.Username,
	})
}

func GetMyAuditLog(w http.ResponseWriter, r *http.Request) {
	user, err := lmodel.UserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	logs, err := db.GetUserAuditLogs(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch audit logs", http.StatusInternalServerError)
		return
	}

	if logs == nil {
		logs = []lmodel.AuditLog{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user":  user.Username,
		"total": len(logs),
		"logs":  logs,
	})
}

func ExportMyData(w http.ResponseWriter, r *http.Request) {
	user, err := lmodel.UserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	profile, err := db.GetUserByID(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch user data", http.StatusInternalServerError)
		return
	}

	projects, err := db.GetUserProjects(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch projects", http.StatusInternalServerError)
		return
	}
	if projects == nil {
		projects = []lmodel.Project{}
	}

	grades := make([]lmodel.Grade, 0)
	for _, project := range projects {
		projectGrades, err := db.GetGradesForProject(r.Context(), project.ID)
		if err != nil {
			http.Error(w, "Failed to fetch project grades", http.StatusInternalServerError)
			return
		}
		grades = append(grades, projectGrades...)
	}

	logs, err := db.GetUserAuditLogs(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to fetch audit log", http.StatusInternalServerError)
		return
	}
	if logs == nil {
		logs = []lmodel.AuditLog{}
	}

	export := lmodel.DataExport{
		User:       *profile,
		Projects:   projects,
		Grades:     grades,
		AuditLog:   logs,
		ExportedAt: time.Now(),
	}

	w.Header().Set("Content-Disposition", "attachment; filename=my_data_export.json")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(export)
}
func DeleteMyAccount(w http.ResponseWriter, r *http.Request) {
	user, err := lmodel.UserFromContext(r.Context())
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tx, err := db.Pool.Begin(r.Context())
	if err != nil {
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(r.Context())

	if _, err := tx.Exec(r.Context(), `DELETE FROM audit_logs WHERE user_id = $1`, user.ID); err != nil {
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	result, err := tx.Exec(r.Context(), `DELETE FROM users WHERE id = $1`, user.ID)
	if err != nil {
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}
	if result.RowsAffected() == 0 {
		http.Error(w, "Account not found", http.StatusNotFound)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Your account and all data have been permanently deleted",
	})
}
