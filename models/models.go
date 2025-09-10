package models

import (
	"github.com/cidekar/adele-framework"
	upper "github.com/upper/db/v4"
)

var DB upper.Session

// The Models struct is the single place to register and access all database models throughout
// your application. The Models struct acts as a container that holds all your application's
// data models— automatic database session setup.
type Models struct{}

// A constructor that initializes and returns a Models struct for use throughout the appication.
func New(a *adele.Adele) *Models {

	// Sets Up Database Session
	DB = a.DB.NewSession()

	// Returns any initialized Models
	return &Models{}
}
