package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"personal-web/connection"
	"personal-web/middleware"

	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"golang.org/x/crypto/bcrypt"

	"github.com/labstack/echo/v4"
)

type Project struct {
	ID           int
	Name         string
	StartDate    *time.Time
	EndDate      *time.Time
	Description  string
	Technologies []string
	Image        string
}

type User struct {
	ID       int
	Uname    string
	Email    string
	Password string
}

var (
	store = sessions.NewCookieStore([]byte("secret"))
)

func main() {

	connection.DatabaseConnect()

	// create new echo instance
	e := echo.New()

	// serve static files from public directory
	e.Static("/assets", "assets")
	e.Static("/upload", "upload")

	// initialize to use session
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("session"))))

	// Routing
	e.GET("/", home)
	e.GET("/project", project)
	e.GET("/contact", contact)
	e.GET("/project-detail/:id", projectDetail)
	e.GET("/delete-project/:id", deleteProject)
	e.GET("/form-register", formRegister)
	e.GET("/form-login", formLogin)
	e.GET("/logout", logout)

	e.POST("/add-project", middleware.UploadFile(addProject))
	e.POST("/register", register)
	e.POST("/login", login)

	e.Logger.Fatal(e.Start("localhost:5000"))

}

func home(c echo.Context) error {
	tmpl, err := template.New("index.html").Funcs(template.FuncMap{
		"Contains": strings.Contains,
	}).ParseFiles("index.html")

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	data, _ := connection.Conn.Query(context.Background(), "SELECT id, name, COALESCE(start_date, '1970-01-01') AS start_date, COALESCE(end_date, '1970-01-01') AS end_date, description, technologies, image FROM tb_project")

	var results []Project
	for data.Next() {
		var each = Project{}

		err = data.Scan(&each.ID, &each.Name, &each.StartDate, &each.EndDate, &each.Description, &each.Technologies, &each.Image)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
		}

		results = append(results, each)
	}

	sess, _ := store.Get(c.Request(), "session")
	flash := map[string]interface{}{
		"FlashStatus":  sess.Values["alertStatus"],
		"FlashMessage": sess.Values["message"],
		"FlashName":    sess.Values["name"],
	}
	delete(sess.Values, "message")
	delete(sess.Values, "alertStatus")

	projects := map[string]interface{}{
		"Project":      results,
		"FlashStatus":  flash["FlashStatus"],
		"FlashMessage": flash["FlashMessage"],
		"FlashName":    flash["FlashName"],
	}

	// Jika method adalah POST, artinya ada data yang dikirimkan
	if c.Request().Method == "POST" {
		var project Project
		if err := c.Bind(&project); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"message": err.Error()})
		}

		// Inisialisasi slice kosong untuk menampung teknologi yang dipilih
		var technologies []string

		// Iterasi setiap teknologi yang dipilih pada project
		for _, tech := range project.Technologies {
			switch tech {
			case "angular":
				// Tambahkan icon Angular pada slice
				technologies = append(technologies, `<i class="fab fa-angular"></i>`)
			case "vultr":
				// Tambahkan icon Vultr pada slice
				technologies = append(technologies, `<i class="fas fa-server"></i>`)
			case "reactjs":
				// Tambahkan icon ReactJS pada slice
				technologies = append(technologies, `<i class="fab fa-react"></i>`)
			}
		}

		// Update slice technologies pada project yang bersangkutan
		for i := range results {
			if results[i].ID == project.ID {
				results[i].Technologies = technologies
			}
		}

		sess.Values["message"] = "Project berhasil diupdate!"
		sess.Values["alertStatus"] = "success"
		sess.Values["name"] = project.Name

		return c.Redirect(http.StatusSeeOther, "/")
	}

	return tmpl.Execute(c.Response().Writer, projects)
}

func project(c echo.Context) error {
	// get session
	sess, err := session.Get("session", c)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message ": err.Error()})
	}

	// create struct for template data
	type ProjectData struct {
		IsLogin bool
		Uname   string
	}

	// populate struct with session data
	projectData := ProjectData{}
	if sess.Values["isLogin"] != true {
		projectData.IsLogin = false
	} else {
		projectData.IsLogin = true
		projectData.Uname = sess.Values["name"].(string)
	}

	// execute template with data
	tmpl, err := template.ParseFiles("project.html")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message ": err.Error()})
	}

	return tmpl.Execute(c.Response(), projectData)
}

func projectDetail(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	tmpl, err := template.ParseFiles("project-detail.html")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message ": err.Error()})
	}

	var ProjectDetail = Project{}

	err = connection.Conn.QueryRow(context.Background(), "SELECT id, name, start_date, end_date, description FROM tb_project WHERE ID = $1", id).Scan(&ProjectDetail.ID, &ProjectDetail.Name, &ProjectDetail.StartDate, &ProjectDetail.EndDate, &ProjectDetail.Description)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message ": err.Error()})
	}

	data := map[string]interface{}{
		"Project": ProjectDetail,
	}

	return tmpl.Execute(c.Response(), data)
}

func addProject(c echo.Context) error {
	// Get form data
	name := c.FormValue("inputName")
	startDateStr := c.FormValue("start-date")
	endDateStr := c.FormValue("end-date")
	description := c.FormValue("description")
	technologies := c.Request().Form["technologies[]"] // mengambil nilai checkbox
	image := c.Get("dataFile").(string)

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message ": err.Error()})
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message ": err.Error()})
	}

	newProject := Project{
		Name:         name,
		StartDate:    &startDate,
		EndDate:      &endDate,
		Description:  description,
		Technologies: technologies,
		Image:        image,
	}

	// Insert data to database
	_, err = connection.Conn.Exec(context.Background(), "INSERT INTO public.tb_project(name, start_date, end_date, description, image, technologies) VALUES ($1, $2, $3, $4, $5, (SELECT ARRAY(SELECT unnest($6::text[]))))", newProject.Name, *newProject.StartDate, *newProject.EndDate, newProject.Description, newProject.Image, newProject.Technologies)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message ": err.Error()})
	}

	return c.Redirect(http.StatusMovedPermanently, "/")
}

func contact(c echo.Context) error {
	tmpl, err := template.ParseFiles("contact.html")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message ": err.Error()})
	}

	return tmpl.Execute(c.Response(), "/")
}

func deleteProject(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	_, err := connection.Conn.Exec(context.Background(), "DELETE FROM public.tb_project WHERE id =$1", id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	return c.Redirect(http.StatusMovedPermanently, "/")
}

func formRegister(c echo.Context) error {
	tmpl, err := template.ParseFiles("register.html")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	return tmpl.Execute(c.Response(), nil)
}

func formLogin(c echo.Context) error {
	sess, _ := session.Get("session", c)
	flash := map[string]interface{}{
		"FlashStatus":  sess.Values["alertStatus"],
		"FlashMessage": sess.Values["message"],
	}

	delete(sess.Values, "message")
	delete(sess.Values, "alertStatus")

	tmpl, err := template.ParseFiles("login.html")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": err.Error()})
	}

	return tmpl.Execute(c.Response(), flash)
}

func login(c echo.Context) error {
	err := c.Request().ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	email := c.FormValue("email")
	password := c.FormValue("password")

	user := User{}
	err = connection.Conn.QueryRow(context.Background(), "SELECT * FROM public.tb_user WHERE email=$1", email).Scan(&user.ID, &user.Uname, &user.Email, &user.Password)
	if err != nil {
		return redirectWithMessage(c, "Email Salah", false, "/form-login")
	}

	fmt.Println(user)

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return redirectWithMessage(c, "Password Salah", false, "/form-login")
	}

	sess, _ := session.Get("session", c)
	sess.Options.MaxAge = 10800 // 3 jam
	sess.Values["message"] = "Login Success"
	sess.Values["alertStatus"] = true // show alert
	sess.Values["name"] = user.Uname
	sess.Values["id"] = user.ID
	sess.Values["isLogin"] = true // acces login
	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusMovedPermanently, "/")
}

func register(c echo.Context) error {
	err := c.Request().ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	name := c.FormValue("name")
	email := c.FormValue("email")
	password := c.FormValue("password")

	// generate password
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)

	_, err = connection.Conn.Exec(context.Background(), "INSERT INTO public.tb_user (name, email, password) VALUES ($1, $2, $3)", name, email, passwordHash)

	if err != nil {
		redirectWithMessage(c, "Register Failed", false, "/form-register")
	}

	return redirectWithMessage(c, "Register Success", true, "/form-login")
}

func logout(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options.MaxAge = -1
	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusMovedPermanently, "/")
}

func redirectWithMessage(c echo.Context, message string, status bool, path string) error {
	sess, _ := session.Get("session", c)
	sess.Values["message"] = message
	sess.Values["alertStatus"] = status
	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusMovedPermanently, path)
}
