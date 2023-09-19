package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/weizi-era/opc"
)

// App contains the opc connection and the API routes
type App struct {
	Conn   opc.Connection
	Router *mux.Router
	Config Config
}

//Config determines what services shall be exposed
//through the App
type Config struct {
	WriteTag  bool `toml:"allow_write"`
	AddTag    bool `toml:"allow_add"`
	DeleteTag bool `toml:"allow_remove"`
}

// Initialize sets OPC connection and creates routes
func (a *App) Initialize(conn opc.Connection) {
	a.Conn = conn
	a.Router = mux.NewRouter()
	a.Router.HandleFunc("/tags", a.getTags).Methods("GET")          // Read
	a.Router.HandleFunc("/tag", a.createTag).Methods("POST")        // Add(...)
	a.Router.HandleFunc("/tag/{id}", a.getTag).Methods("GET")       // ReadItem(id)
	a.Router.HandleFunc("/tag/{id}", a.deleteTag).Methods("DELETE") // Remove(id)
	a.Router.HandleFunc("/tag/{id}", a.updateTag).Methods("PUT")    // Write(id, value)
}

// Run starts serving the API
func (a *App) Run(addr string) {
	log.Fatal(http.ListenAndServe(addr, handlers.CORS(handlers.AllowedOrigins([]string{"*"}))(a.Router)))
}

// getTags returns all tags in the current opc connection, route: /tags
func (a *App) getTags(w http.ResponseWriter, r *http.Request) {
	respondWithJSON(w, http.StatusOK, a.Conn.Read())
}

// createTag creates the tags in the opc connection, route: /tag
func (a *App) createTag(w http.ResponseWriter, r *http.Request) {
	if a.Config.AddTag {
		var tags []string
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&tags); err != nil {
			fmt.Println("tags received:", tags)
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}
		defer r.Body.Close()

		err := a.Conn.Add(tags...)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Did not add tags")
			return
		}
		respondWithJSON(w, http.StatusCreated, map[string]interface{}{"result": "created"})
	} else {
		respondWithError(w, http.StatusBadRequest, "no additions allowed")
	}
}

// getTag returns the opc.Item for the given tag id, route: /tag/{id}
func (a *App) getTag(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	item := a.Conn.ReadItem(vars["id"])
	empty := opc.Item{}
	if item == empty {
		respondWithError(w, http.StatusNotFound, "tag not found")
		return
	}
	respondWithJSON(w, http.StatusOK, item)
}

// deleteTag removes the tag in the opc connection
func (a *App) deleteTag(w http.ResponseWriter, r *http.Request) {
	if a.Config.DeleteTag {
		vars := mux.Vars(r)
		a.Conn.Remove(vars["id"])
		respondWithJSON(w, http.StatusOK, map[string]interface{}{"result": "removed"})
	} else {
		respondWithError(w, http.StatusBadRequest, "deletions not allowed")
	}
}

// updateTag write value the opc.Item for the given tag id, route: /tag/{id}
func (a *App) updateTag(w http.ResponseWriter, r *http.Request) {
	if a.Config.WriteTag {
		vars := mux.Vars(r)

		var value interface{}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&value); err != nil {
			fmt.Println("value received:", value)
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}
		defer r.Body.Close()

		err := a.Conn.Write(vars["id"], value)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "value could not be written to tag")
			return
		}
		respondWithJSON(w, http.StatusOK, map[string]interface{}{"result": "updated"})
	} else {
		respondWithError(w, http.StatusBadRequest, "read-only")
	}
}

// responsWithError is a helper function to return a JSON error
func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

// responsWithJSON is helper function to return the data in JSON encoding
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
