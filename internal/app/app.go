package app

import (
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type App struct {
	DB         *sql.DB
	Templates  map[string]*template.Template
	CookieName string
	SessionTTL time.Duration
}

// Utilities
func (a *App) CurrentUser(r *http.Request) (id int64, username string, ok bool) {
	c, err := r.Cookie(a.CookieName)
	if err != nil {
		return 0, "", false
	}
	var userID int64
	var expires time.Time
	err = a.DB.QueryRow(`SELECT user_id, expires_at FROM sessions WHERE id=?`, c.Value).Scan(&userID, &expires)
	if err != nil {
		return 0, "", false
	}
	if time.Now().After(expires) {
		a.DB.Exec(`DELETE FROM sessions WHERE id=?`, c.Value)
		return 0, "", false
	}
	err = a.DB.QueryRow(`SELECT username FROM users WHERE id=?`, userID).Scan(&username)
	if err != nil {
		return 0, "", false
	}
	return userID, username, true
}

func (a *App) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := a.CurrentUser(r); !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (a *App) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Helpful in dev to avoid stale pages:
	w.Header().Set("Cache-Control", "no-store")
	tpl, ok := a.Templates[name]
	if !ok {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	if err := tpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Handlers
func (a *App) HandleIndex(w http.ResponseWriter, r *http.Request) {
	// Filters: category, mine=posts, liked
	q := `SELECT p.id, p.title, p.body, p.created_at, u.username,
                 (SELECT COUNT(*) FROM likes l WHERE l.target_type='post' AND l.target_id=p.id AND l.value=1) AS likes,
                 (SELECT COUNT(*) FROM likes l WHERE l.target_type='post' AND l.target_id=p.id AND l.value=-1) AS dislikes
          FROM posts p
          JOIN users u ON u.id = p.user_id
          WHERE 1=1`
	args := []any{}

	if cat := r.URL.Query().Get("category"); cat != "" {
		q += ` AND EXISTS (SELECT 1 FROM post_categories pc JOIN categories c ON c.id=pc.category_id WHERE pc.post_id=p.id AND c.name=?)`
		args = append(args, cat)
	}

	if r.URL.Query().Get("mine") == "1" {
		if uid, _, ok := a.CurrentUser(r); ok {
			q += ` AND p.user_id = ?`
			args = append(args, uid)
		} else {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
	}

	if r.URL.Query().Get("liked") == "1" {
		if uid, _, ok := a.CurrentUser(r); ok {
			q += ` AND p.id IN (SELECT target_id FROM likes WHERE target_type='post' AND user_id=? AND value=1)`
			args = append(args, uid)
		} else {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
	}

	q += ` ORDER BY p.created_at DESC LIMIT 50`

	rows, err := a.DB.Query(q, args...)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type Row struct {
		ID                    int64
		Title, Body, Username string
		CreatedAt             string
		Likes, Dislikes       int
	}
	var posts []Row
	for rows.Next() {
		var rrow Row
		var ts time.Time
		if err := rows.Scan(&rrow.ID, &rrow.Title, &rrow.Body, &ts, &rrow.Username, &rrow.Likes, &rrow.Dislikes); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		rrow.CreatedAt = ts.Format("2006-01-02 15:04")
		posts = append(posts, rrow)
	}

	// Fetch categories for sidebar/filter
	cats, _ := a.AllCategories()

	uid, uname, logged := a.CurrentUser(r)
	a.render(w, "index.html", map[string]any{
		"View":       "index",
		"Posts":      posts,
		"Categories": cats,
		"LoggedIn":   logged,
		"UserID":     uid,
		"Username":   uname,
		"Filter": map[string]string{
			"category": r.URL.Query().Get("category"),
			"mine":     r.URL.Query().Get("mine"),
			"liked":    r.URL.Query().Get("liked"),
		},
	})
}

func (a *App) AllCategories() ([]string, error) {
	rows, err := a.DB.Query(`SELECT name FROM categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		out = append(out, n)
	}
	return out, nil
}

func (a *App) HandleRegister(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.render(w, "register.html", map[string]any{"View": "register"})
	case http.MethodPost:
		email := r.FormValue("email")
		username := r.FormValue("username")
		pass := r.FormValue("password")
		if email == "" || username == "" || pass == "" {
			http.Error(w, "missing fields", http.StatusBadRequest)
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_, err = a.DB.Exec(`INSERT INTO users(email, username, password_hash) VALUES(?,?,?)`, email, username, string(hash))
		if err != nil {
			if isUniqueErr(err) {
				// http.Error(w, "email or username already taken", http.StatusConflict)
				w.WriteHeader(http.StatusUnauthorized)
				a.render(w, "register.html", map[string]any{
					"View":  "register",
					"Error": "email or username already taken",
				})
				return
			}
			http.Error(w, err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func isUniqueErr(err error) bool {
	if err == nil {
		return false
	}
	// Simple detection based on SQLite error string for MVP
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func (a *App) HandleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.render(w, "login.html", map[string]any{"View": "login"})
	case http.MethodPost:
		email := r.FormValue("email")
		pass := r.FormValue("password")
		var id int64
		var hash string
		var username string
		err := a.DB.QueryRow(`SELECT id, username, password_hash FROM users WHERE email=?`, email).Scan(&id, &username, &hash)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(pass)) != nil {
			w.WriteHeader(http.StatusUnauthorized)
			a.render(w, "login.html", map[string]any{
				"View":  "login",
				"Error": "invalid credentials",
			})
			return
		}
		sid := uuid.NewString()
		_, err = a.DB.Exec(`INSERT INTO sessions(id, user_id, expires_at) VALUES(?,?,?)`, sid, id, time.Now().Add(a.SessionTTL))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: a.CookieName, Value: sid, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(a.SessionTTL)})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *App) HandleLogout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(a.CookieName)
	if err == nil {
		a.DB.Exec(`DELETE FROM sessions WHERE id=?`, c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: a.CookieName, Value: "", Path: "/", Expires: time.Unix(0, 0), MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) HandleNewPost(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cats, _ := a.AllCategories()
		a.render(w, "post_new.html", map[string]any{"View": "post_new", "Categories": cats})
	case http.MethodPost:
		uid, _, ok := a.CurrentUser(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		title := r.FormValue("title")
		body := r.FormValue("body")
		if title == "" || body == "" {
			http.Error(w, "title/body required", 400)
			return
		}
		res, err := a.DB.Exec(`INSERT INTO posts(user_id, title, body) VALUES(?,?,?)`, uid, title, body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		pid, _ := res.LastInsertId()
		if err := r.ParseForm(); err == nil {
			seen := map[string]bool{}
			for _, cname := range r.Form["category"] {
				for _, part := range strings.Split(cname, ",") {
					c := strings.TrimSpace(part)
					if c == "" || seen[c] {
						continue
					}
					seen[c] = true
					var cid int64
					err := a.DB.QueryRow(`SELECT id FROM categories WHERE name=?`, c).Scan(&cid)
					if errors.Is(err, sql.ErrNoRows) {
						res, err := a.DB.Exec(`INSERT INTO categories(name) VALUES(?)`, c)
						if err == nil {
							cid, _ = res.LastInsertId()
						}
					}
					if cid != 0 {
						a.DB.Exec(`INSERT OR IGNORE INTO post_categories(post_id, category_id) VALUES(?,?)`, pid, cid)
					}
				}
			}
		}
		http.Redirect(w, r, "/post?id="+strconv.FormatInt(pid, 10), http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *App) HandleShowPost(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	type Post struct {
		ID                    int64
		Title, Body, Username string
		CreatedAt             string
		Likes, Dislikes       int
		Categories            []string
	}
	var p Post
	var ts time.Time
	err := a.DB.QueryRow(`SELECT p.id, p.title, p.body, p.created_at, u.username,
        (SELECT COUNT(*) FROM likes l WHERE l.target_type='post' AND l.target_id=p.id AND l.value=1) AS likes,
        (SELECT COUNT(*) FROM likes l WHERE l.target_type='post' AND l.target_id=p.id AND l.value=-1) AS dislikes
        FROM posts p
        JOIN users u ON u.id=p.user_id
        WHERE p.id=?`, id).Scan(&p.ID, &p.Title, &p.Body, &ts, &p.Username, &p.Likes, &p.Dislikes)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	p.CreatedAt = ts.Format("2006-01-02 15:04")
	// categories
	rows, _ := a.DB.Query(`SELECT c.name FROM categories c JOIN post_categories pc ON pc.category_id=c.id WHERE pc.post_id=? ORDER BY c.name`, id)
	defer rows.Close()
	for rows.Next() {
		var n string
		rows.Scan(&n)
		p.Categories = append(p.Categories, n)
	}

	// comments
	crows, err := a.DB.Query(`SELECT c.id, c.body, c.created_at, u.username,
        (SELECT COUNT(*) FROM likes l WHERE l.target_type='comment' AND l.target_id=c.id AND l.value=1) AS likes,
        (SELECT COUNT(*) FROM likes l WHERE l.target_type='comment' AND l.target_id=c.id AND l.value=-1) AS dislikes
        FROM comments c
        JOIN users u ON u.id=c.user_id
        WHERE c.post_id=?
        ORDER BY c.created_at ASC`, id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	type Comment struct {
		ID                        int64
		Body, Username, CreatedAt string
		Likes, Dislikes           int
	}
	var comments []Comment
	for crows.Next() {
		var cmt Comment
		var t time.Time
		if err := crows.Scan(&cmt.ID, &cmt.Body, &t, &cmt.Username, &cmt.Likes, &cmt.Dislikes); err == nil {
			cmt.CreatedAt = t.Format("2006-01-02 15:04")
			comments = append(comments, cmt)
		}
	}
	uid, uname, logged := a.CurrentUser(r)
	a.render(w, "post_show.html", map[string]any{
		"View":     "post_show",
		"Post":     p,
		"Comments": comments,
		"LoggedIn": logged,
		"UserID":   uid,
		"Username": uname,
	})
}

func (a *App) HandleNewComment(w http.ResponseWriter, r *http.Request) {
	uid, _, ok := a.CurrentUser(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	postID := r.FormValue("post_id")
	body := r.FormValue("body")
	if body == "" {
		http.Error(w, "empty comment", 400)
		return
	}
	if _, err := a.DB.Exec(`INSERT INTO comments(post_id, user_id, body) VALUES(?,?,?)`, postID, uid, body); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/post?id="+postID, http.StatusSeeOther)
}

func (a *App) HandleLike(w http.ResponseWriter, r *http.Request) {
	uid, _, ok := a.CurrentUser(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	typ := r.FormValue("type") // post/comment
	tid := r.FormValue("id")
	val := r.FormValue("value") // 1 or -1
	_, err := a.DB.Exec(`INSERT INTO likes(user_id, target_type, target_id, value) VALUES(?,?,?,?)
                         ON CONFLICT(user_id, target_type, target_id) DO UPDATE SET value=excluded.value`, uid, typ, tid, val)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if typ == "post" {
		http.Redirect(w, r, "/post?id="+tid, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
	}
}
