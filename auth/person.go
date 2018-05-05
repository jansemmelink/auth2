package auth

import (
	"net/http"

	"github.com/gorilla/pat"
)

//Person is stored do describe a real person
type Person struct {
}

//AddPersonRoutes add the API to the router
func AddPersonRoutes(r *pat.Router) http.Handler {
	//auth operations
	/*r.Post("/person", newHandler)
	r.Get("/person", lstHandler)
	r.Get("/person/{id}", getHandler)
	r.Put("/person", updHandler)
	r.Delete("/person/{id}", delHandler)*/
	return r
}
