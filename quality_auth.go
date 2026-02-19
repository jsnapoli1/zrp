package main

import (
	"net/http"
)

// getUserRole returns the role of the authenticated user from the request
func getUserRole(r *http.Request) string {
	cookie, err := r.Cookie("zrp_session")
	if err != nil {
		return "user" // default role for unauthenticated users
	}
	
	var role string
	err = db.QueryRow(`
		SELECT u.role 
		FROM users u 
		JOIN sessions s ON u.id = s.user_id 
		WHERE s.token = ?
	`, cookie.Value).Scan(&role)
	
	if err != nil {
		return "user" // default role if query fails
	}
	
	return role
}

// getUserID returns the ID of the authenticated user from the request
func getUserID(r *http.Request) (int, error) {
	cookie, err := r.Cookie("zrp_session")
	if err != nil {
		return 0, err
	}
	
	var userID int
	err = db.QueryRow(`
		SELECT u.id 
		FROM users u 
		JOIN sessions s ON u.id = s.user_id 
		WHERE s.token = ?
	`, cookie.Value).Scan(&userID)
	
	return userID, err
}

// canApproveCAPA checks if the user has the appropriate role to approve CAPAs
func canApproveCAPA(r *http.Request, approvalType string) bool {
	role := getUserRole(r)
	
	switch approvalType {
	case "qe":
		return role == "qe" || role == "admin"
	case "manager":
		return role == "manager" || role == "admin"
	default:
		return false
	}
}