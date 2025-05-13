package user

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Exemple de test pour le handler UserHandler
func TestUserHandler(t *testing.T) {
	// Crée une requête HTTP GET vers "/user"
	req := httptest.NewRequest(http.MethodGet, "/user/whoami", nil)

	// Crée un enregistreur de réponse pour capturer la réponse du handler
	rr := httptest.NewRecorder()

	userHandler := NewUserHandler(30)

	// Appelle le handler avec la requête et l'enregistreur de réponse
	userHandler.ServeHTTP(rr, req)

	// Vérifie que le code de statut HTTP est 403 Forbidden
	if status := rr.Code; status != http.StatusForbidden {
		t.Errorf("Code de statut incorrect : obtenu %v, attendu %v", status, http.StatusForbidden)
	}

	// Vérifie que le corps de la réponse contient le texte attendu
	expected := "You need to be authenticated"
	if rr.Body.String() != expected {
		t.Errorf("Corps de la réponse incorrect : obtenu %v, attendu %v", rr.Body.String(), expected)
	}
}
