package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"json-server-go/db"
	"json-server-go/handlers"

	"github.com/gorilla/mux"
)

type Config struct {
	Port      int
	Host      string
	ReadOnly  bool
	NoCors    bool
	Delay     int
	StaticDir string
	Routes    string
	IdField   string
	Quiet     bool
}

type Server struct {
	DB       *db.Database
	Handler  *handlers.Handler
	Router   *mux.Router
	Config   *Config
	Rewriter *Rewriter
	mu       sync.RWMutex
}

func New(d *db.Database, config *Config) *Server {
	rewriter, err := NewRewriter(config.Routes)
	if err != nil {
		log.Printf("Error loading routes: %v", err)
	}

	return &Server{
		DB:       d,
		Handler:  handlers.New(d, config.IdField),
		Router:   mux.NewRouter(),
		Config:   config,
		Rewriter: rewriter,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	router := s.Router
	rewriter := s.Rewriter
	s.mu.RUnlock()

	var handler http.Handler = router
	if rewriter != nil {
		handler = rewriter.Middleware(router)
	}
	handler.ServeHTTP(w, r)
}

func (s *Server) InitRoutes() {
	s.mu.Lock()
	defer s.mu.Unlock()

	router := mux.NewRouter()

	// 1. Middlewares
	if !s.Config.Quiet {
		router.Use(s.loggingMiddleware)
	}
	router.Use(s.delayMiddleware)

	if s.Config.ReadOnly {
		router.Use(s.readOnlyMiddleware)
	}

	// 2. Default Routes
	router.HandleFunc("/db", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s.DB.GetData())
	}).Methods("GET")

	// 3. Dynamic Routes based on DB content
	data := s.DB.GetData()
	for key, value := range data {
		resourceName := key
		if db.IsPlural(value) {
			log.Printf("Registering plural resource: /%s", resourceName)
			s.registerPluralRoutes(router, resourceName)
		} else {
			log.Printf("Registering singular resource: /%s", resourceName)
			s.registerSingularRoutes(router, resourceName)
		}
	}

	// 4. Nested Routes (Generic Handler)
	// GET /:parent/:parentId/:child -> /:child?:parentId=:parentId
	router.HandleFunc("/{parent}/{parentId}/{child}", s.nestedRouteHandler).Methods("GET", "POST")

	// 5. Static Files or Homepage
	if s.Config.StaticDir != "" {
		// Serve static files
		fs := http.FileServer(http.Dir(s.Config.StaticDir))
		router.PathPrefix("/").Handler(fs)
	} else {
		// Default homepage
		router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<h1>JSON Server is running</h1>")
		}).Methods("GET")
	}

	s.Router = router
}

func (s *Server) registerPluralRoutes(router *mux.Router, name string) {
	// GET /name
	router.HandleFunc(fmt.Sprintf("/%s", name), func(w http.ResponseWriter, r *http.Request) {
		s.Handler.PluralList(w, r, name)
	}).Methods("GET")

	// GET /name/:id
	router.HandleFunc(fmt.Sprintf("/%s/{id}", name), func(w http.ResponseWriter, r *http.Request) {
		s.Handler.PluralGet(w, r, name)
	}).Methods("GET")

	// POST /name
	router.HandleFunc(fmt.Sprintf("/%s", name), func(w http.ResponseWriter, r *http.Request) {
		s.Handler.PluralCreate(w, r, name)
	}).Methods("POST")

	// PUT /name/:id
	router.HandleFunc(fmt.Sprintf("/%s/{id}", name), func(w http.ResponseWriter, r *http.Request) {
		s.Handler.PluralUpdate(w, r, name)
	}).Methods("PUT")

	// PATCH /name/:id
	router.HandleFunc(fmt.Sprintf("/%s/{id}", name), func(w http.ResponseWriter, r *http.Request) {
		s.Handler.PluralPatch(w, r, name)
	}).Methods("PATCH")

	// DELETE /name/:id
	router.HandleFunc(fmt.Sprintf("/%s/{id}", name), func(w http.ResponseWriter, r *http.Request) {
		s.Handler.PluralDelete(w, r, name)
	}).Methods("DELETE")
}

func (s *Server) registerSingularRoutes(router *mux.Router, name string) {
	// GET /name
	router.HandleFunc(fmt.Sprintf("/%s", name), func(w http.ResponseWriter, r *http.Request) {
		s.Handler.SingularGet(w, r, name)
	}).Methods("GET")

	// POST /name, PUT /name, PATCH /name
	router.HandleFunc(fmt.Sprintf("/%s", name), func(w http.ResponseWriter, r *http.Request) {
		s.Handler.SingularUpdate(w, r, name)
	}).Methods("POST", "PUT", "PATCH")
}

func (s *Server) nestedRouteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	parent := vars["parent"]
	parentId := vars["parentId"]
	child := vars["child"]

	// Check if child resource exists
	if _, ok := s.DB.Get(child); !ok {
		http.NotFound(w, r)
		return
	}

	// Rewrite query: ?parentId=parentId
	// We need to singularize parent name to guess the foreign key
	// e.g. posts -> postId
	singularParent := strings.TrimSuffix(parent, "s")
	foreignKey := singularParent + "Id"

	if r.Method == "POST" {
		// Read body, add foreign key, reset body
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body[foreignKey] = parentId

		// Re-encode body
		jsonValue, _ := json.Marshal(body)
		r.Body = io.NopCloser(bytes.NewBuffer(jsonValue))
		r.ContentLength = int64(len(jsonValue)) // Update content length

		// Delegate to PluralCreate
		s.Handler.PluralCreate(w, r, child)
		return
	}

	// Add to query
	q := r.URL.Query()
	q.Add(foreignKey, parentId)
	r.URL.RawQuery = q.Encode()

	// Delegate to PluralList
	s.Handler.PluralList(w, r, child)
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *Server) delayMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		delay := s.Config.Delay
		if d := r.URL.Query().Get("_delay"); d != "" {
			if val, err := strconv.Atoi(d); err == nil {
				delay = val
			}
		}
		if delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) readOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
		} else {
			w.WriteHeader(http.StatusForbidden)
		}
	})
}
