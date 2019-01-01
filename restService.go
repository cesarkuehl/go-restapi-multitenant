package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"

	_ "github.com/jinzhu/gorm/dialects/postgres"
)

// The JWT Key, change this to your own key
const jwtKey string = "c0e18b6a-a204-4197-93f4-c11f6cd5bad8"

// Database connection pool
var dbPool *gorm.DB

// Person defines the basic struct of a person
type Person struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ServiceClaims defines the claims that should be received by the JWT token
type ServiceClaims struct {
	UserName string `json:"username"`
	Tenant   string `json:"tenant"`
	jwt.StandardClaims
}

// Defines the handler signature
type handler func(w http.ResponseWriter, r *http.Request, db *gorm.DB) error

// Gets a environment variable value, otherwise returns the defined default value
func loadEnv(envName string, defaultValue string) string {
	value, ok := os.LookupEnv(envName)

	if !ok {
		value = defaultValue
		log.Printf("Could not find the env %s, using it's default value (%s) instead", envName, defaultValue)
	}

	return value
}

// Validates if the Authorization header is preset and has a valid value
func isValidAuthorizationHeader(authorizationHeader string) bool {
	if authorizationHeader == "" {
		return false
	}

	// The header value must be in the format: Bearer <jwtToken>
	auth := strings.SplitN(authorizationHeader, " ", 2)

	if len(auth) == 2 {
		return auth[0] == "Bearer" && auth[1] != ""
	}

	return false
}

// Gets the function name of a interface
func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

// Prepares the execution of the serviceHandler by setting the schema for the tenant
// and migrating the database if necessary
func (serviceHandler handler) prepare(w http.ResponseWriter, r *http.Request, tenant string) {
	log.Printf("Executing %v for tenant %v", getFunctionName(serviceHandler), tenant)

	// Starts the transaction and change the schema for the tenant
	db := dbPool.Begin()

	// Creates the schema for the tenant if not exists
	db.Exec("create schema if not exists " + tenant)

	// Modify the search path in order to use a different schema per tenant
	db.Exec("set search_path to " + tenant)

	// Migrates the database
	db.AutoMigrate(&Person{})

	// Executes the handler
	err := serviceHandler(w, r, db)

	// If any error is found the transaction will be rolled back, otherwise commits
	if err == nil {
		db.Commit()
	} else {
		db.Rollback()
	}
}

// Checks if the JWT is informed and valid, then prepares the handler execution
func (serviceHandler handler) authorize(w http.ResponseWriter, r *http.Request) {
	authorizationHeader := r.Header.Get("Authorization")

	if isValidAuthorizationHeader(authorizationHeader) {
		tokenString := strings.SplitN(authorizationHeader, " ", 2)[1]

		token, err := jwt.ParseWithClaims(tokenString, &ServiceClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtKey), nil
		})

		if err != nil {
			http.Error(w, fmt.Sprintf("Token is not valid: %v", err), 401)
			return
		}

		claims := token.Claims.(*ServiceClaims)

		if token.Valid {
			// Prepare the execution of the serviceHandler
			serviceHandler.prepare(w, r, claims.Tenant)
		} else {
			http.Error(w, "Token is not valid or is expired", 401)
			return
		}
	} else {
		http.Error(w, fmt.Sprintf("Invalid authorization header: %v", authorizationHeader), 401)
		return
	}
}

// ListPeople is a handler that returns a list of Person
func ListPeople(w http.ResponseWriter, r *http.Request, db *gorm.DB) error {
	var people []Person

	db.Find(&people)

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(people)

	return nil
}

// GetPerson is a handler that returns a single Person based on the informed ID
func GetPerson(w http.ResponseWriter, r *http.Request, db *gorm.DB) error {
	params := mux.Vars(r)
	var person Person
	id, err := strconv.Atoi(params["id"])

	if err == nil {
		db.Where("id = ?", id).Find(&person)

		w.Header().Add("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(person)
	}

	return err
}

// CreatePerson is a handler that creates a new Person with the informed ID
func CreatePerson(w http.ResponseWriter, r *http.Request, db *gorm.DB) error {
	params := mux.Vars(r)
	var person Person
	var personList []Person

	_ = json.NewDecoder(r.Body).Decode(&person)

	id, err := strconv.Atoi(params["id"])

	if err == nil {
		person.ID = id

		db.Create(&person)
		db.Find(&personList)

		w.Header().Add("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(personList)
	}

	return err
}

// DeletePerson is a handler that deletes the person with the informed ID
func DeletePerson(w http.ResponseWriter, r *http.Request, db *gorm.DB) error {
	params := mux.Vars(r)
	var person Person
	var personList []Person

	id, err := strconv.Atoi(params["id"])

	if err == nil {
		person.ID = id

		db.Delete(&person)
		db.Find(&personList)

		w.Header().Add("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(personList)
	}

	return err
}

// UpdatePerson updates the person with the informed ID
func UpdatePerson(w http.ResponseWriter, r *http.Request, db *gorm.DB) error {
	params := mux.Vars(r)
	var person Person
	var personList []Person

	_ = json.NewDecoder(r.Body).Decode(&person)

	id, err := strconv.Atoi(params["id"])

	if err == nil {
		person.ID = id

		db.Save(&person)
		db.Find(&personList)

		w.Header().Add("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(personList)
	}

	return err
}

func main() {
	var err error

	log.Println("Creating the database connection...")
	dbPool, err = gorm.Open("postgres", "host=localhost port=5432 user=postgres dbname=postgres password='password' sslmode=disable")

	if err != nil {
		panic(err)
	}

	defer dbPool.Close()

	log.Println("Success! Waiting for requests...")

	router := mux.NewRouter()

	var listPeopleHandler handler = ListPeople
	var getPersonHandler handler = GetPerson
	var createPersonHandler handler = CreatePerson
	var deletePersonHandler handler = DeletePerson
	var updatePersonHandler handler = UpdatePerson

	router.HandleFunc("/people", listPeopleHandler.authorize).Methods("GET")
	router.HandleFunc("/person/{id}", getPersonHandler.authorize).Methods("GET")
	router.HandleFunc("/person/{id}", createPersonHandler.authorize).Methods("POST")
	router.HandleFunc("/person/{id}", deletePersonHandler.authorize).Methods("DELETE")
	router.HandleFunc("/person/{id}", updatePersonHandler.authorize).Methods("PUT")

	log.Fatal(http.ListenAndServe(":8000", router))
}
