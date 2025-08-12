package main

import (
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"forum-mvp/internal/app"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	addr := flag.String("addr", ":8080", "http listen address")
	dataDir := flag.String("data", "./data", "data directory for sqlite")
	tplDir := flag.String("templates", "./internal/web/templates", "templates dir")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatal(err)
	}

	dbPath := filepath.Join(*dataDir, "forum.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	schema, err := os.ReadFile("internal/db/schema.sql")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		log.Fatal(err)
	}

	tpls, err := loadTemplates(*tplDir)
	if err != nil {
		log.Fatal(err)
	}

	a := &app.App{DB: db, Templates: tpls, CookieName: "forum_session", SessionTTL: 7 * 24 * time.Hour}

	// Routes
	http.HandleFunc("/", a.HandleIndex)
	http.HandleFunc("/register", a.HandleRegister)
	http.HandleFunc("/login", a.HandleLogin)
	http.HandleFunc("/logout", a.HandleLogout)
	http.HandleFunc("/post", a.HandleShowPost)
	http.HandleFunc("/post/new", a.RequireAuth(a.HandleNewPost))
	http.HandleFunc("/comment/new", a.RequireAuth(a.HandleNewComment))
	http.HandleFunc("/like", a.RequireAuth(a.HandleLike))

	log.Printf("listening on %s", *addr)
	// Wrap the mux to handle 404/500
	wrappedMux := withCustomErrors(http.DefaultServeMux, a)

	log.Printf("listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, logRequest(wrappedMux)))
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start))
	})
}

func loadTemplates(dir string) (map[string]*template.Template, error) {
	layout := filepath.Join(dir, "layout.html")
	pages, err := filepath.Glob(filepath.Join(dir, "*.html"))
	if err != nil {
		return nil, err
	}
	m := make(map[string]*template.Template)
	for _, page := range pages {
		if filepath.Base(page) == "layout.html" {
			continue
		}
		t, err := template.ParseFiles(layout, page)
		if err != nil {
			return nil, err
		}
		m[filepath.Base(page)] = t
	}
	return m, nil
}

func withCustomErrors(next *http.ServeMux, app *app.App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-store")
				w.WriteHeader(http.StatusInternalServerError)
				if tpl, ok := app.Templates["500.html"]; ok {
					tpl.ExecuteTemplate(w, "500.html", appTemplateData(r, app))
				} else {
					http.Error(w, "Internal Server Error", 500)
				}
			}
		}()

		rw := &responseWriter{ResponseWriter: w, statusCode: 200}
		next.ServeHTTP(rw, r)

		if rw.statusCode == http.StatusNotFound {
			if tpl, ok := app.Templates["404.html"]; ok {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-store")
				tpl.ExecuteTemplate(w, "404.html", appTemplateData(r, app))
			} else {
				http.NotFound(w, r)
			}
		}
	})
}

// Capture status codes
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// appTemplateData builds the data map like your other render calls
func appTemplateData(r *http.Request, app *app.App) map[string]any {
	uid, uname, logged := app.CurrentUser(r)
	cats, _ := app.AllCategories()
	return map[string]any{
		"LoggedIn":   logged,
		"UserID":     uid,
		"Username":   uname,
		"Categories": cats,
	}
}
