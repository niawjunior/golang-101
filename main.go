package main

import (
	"database/sql"
	"html/template"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

var (
	tmplIndex  = loadTemplate("template/index.html", "template/layout.html")
	tmplSignUp = loadTemplate("template/signup.html", "template/layout.html")
	tmplSignIn = loadTemplate("template/signin.html", "template/layout.html")
)

func loadTemplate(filename ...string) *template.Template {
	templateName := filename[0]
	t := template.New("")

	t.Funcs(template.FuncMap{
		"templateName": func() string {
			return templateName
		},
		"dateTime": func(t time.Time) string {
			return t.Format("02/01/2006 15:04:05")
		},
		"toUpper": func(s string) string {
			return strings.ToUpper(s)
		},
	})

	t = template.Must(t.ParseFiles(filename...))
	t = t.Lookup("layout")
	return t
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("mysql", "root:niaw2362537@tcp(localhost:3306)/web1?parseTime=true")
	if err != nil {
		panic(err)
	}
	r := gin.Default()
	store := sessions.NewCookieStore([]byte("secret"))
	r.Use(sessions.Sessions("s", store))
	r.GET("/", index)
	r.GET("signup", signUp)
	r.GET("signin", signIn)
	r.GET("signout", signOut)
	r.POST("/signup", postSignUp)
	r.POST("/signin", postSignIn)
	r.POST("/post", postPost)
	r.POST("/post", postPost, allowUser)
	r.Run(":4000")
}

func allowUser(c *gin.Context) {
	sess := sessions.Default(c)
	userID, _ := sess.Get("userId").(int)
	if userID <= 0 {
		c.Redirect(http.StatusSeeOther, "/")
		return
	}
	c.Next()
}

// User type
type User struct {
	ID       int    `json:"id"`
	Username string `json:"user"`
}

func getUser(id int) (*User, error) {
	var u User
	err := db.QueryRow(`
		select
			id, username
		from users
		where id = ?
	
	`, id).Scan(&u.ID, &u.Username)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func getUsers() ([]*User, error) {
	rows, err := db.Query(`
		select
			id, username
		from users
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	us := make([]*User, 0)
	for rows.Next() {
		var u User
		err = rows.Scan(&u.ID, &u.Username)
		if err != nil {
			return nil, err
		}
		us = append(us, &u)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return us, nil
}

// Post struct
type Post struct {
	Username  string
	Msg       string
	CreatedAt time.Time
}

func createPost(userID int, msg string) error {
	_, err := db.Exec(`
		insert into posts (
			user_id, msg
			) values (
				?, ?
			)
		`, userID, msg)
	return err
}

func getPosts() ([]*Post, error) {
	rows, err := db.Query(`
		select 
			u.username, p.msg, p.created_at
		from posts as p
			left join users as u on u.id = p.user_id
		order by p.created_at desc
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	xs := make([]*Post, 0)
	for rows.Next() {
		var x Post
		err = rows.Scan(&x.Username, &x.Msg, &x.CreatedAt)
		if err != nil {
			return nil, err
		}
		xs = append(xs, &x)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return xs, nil
}

func index(c *gin.Context) {
	sess := sessions.Default(c)

	userID := sess.Get("userId").(int)
	u, _ := getUser(userID)
	posts, err := getPosts()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	data := map[string]interface{}{
		"User":  u,
		"Posts": posts,
	}
	tmplIndex.Execute(c.Writer, data)
}
func signUp(c *gin.Context) {
	tmplSignUp.Execute(c.Writer, nil)
}

func signIn(c *gin.Context) {
	tmplSignIn.Execute(c.Writer, nil)
}
func postSignUp(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	if utf8.RuneCountInString(username) < 4 {
		c.String(http.StatusBadRequest, "username required")
		return
	}

	if utf8.RuneCountInString(password) < 4 {
		c.String(http.StatusBadRequest, "username required")
		return
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	_, err = db.Exec(`
		insert into users (
			username, password
		) values (
			?, ?
		)
		`, username, string(hashedPass))
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/")
	//exec ไม่ return
	//query เอาทุก row
	//query row คือ เอาเฉพาะ
}

func postSignIn(c *gin.Context) {

	username := c.PostForm("username")
	password := c.PostForm("password")
	if utf8.RuneCountInString(username) < 4 {
		c.String(http.StatusBadRequest, "username required")
		return
	}

	if utf8.RuneCountInString(password) < 4 {
		c.String(http.StatusBadRequest, "username required")
		return
	}
	var (
		id       int
		hasePass string
	)
	err := db.QueryRow(`
		select
			id, password
		from users
		where username = ?	
	`, username).Scan(
		&id,
		&hasePass,
	)
	if err == sql.ErrNoRows {
		c.String(http.StatusBadRequest, "wrong username or password")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(hasePass),
		[]byte(password),
	)

	if err != nil {
		c.String(http.StatusBadRequest, "wrong username or password")
		return
	}

	sess := sessions.Default(c)
	sess.Set("userId", id)
	sess.Save()
	c.Redirect(http.StatusSeeOther, "/")
	// fmt.Println("user id:", id)

}

func signOut(c *gin.Context) {
	sess := sessions.Default(c)
	sess.Clear()
	sess.Save()
	c.Redirect(http.StatusFound, "/")
}

func postPost(c *gin.Context) {
	msg := strings.TrimSpace(c.PostForm("msg"))
	if len(msg) == 0 {
		c.String(http.StatusBadRequest, "msg required")
		return
	}
	sess := sessions.Default(c)
	userID, _ := sess.Get("userId").(int)
	if userID <= 0 {
		c.Redirect(http.StatusForbidden, "/")
		return
	}

	err := createPost(userID, msg)

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.Redirect(http.StatusSeeOther, "/")
}
