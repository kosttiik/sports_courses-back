package app

import (
	"log"
	"net/http"

	"sports_courses/internal/app/dsn"
	"sports_courses/internal/app/repository"

	"github.com/gin-gonic/gin"
)

type Application struct {
	repo repository.Repository
	r    *gin.Engine
}

func New() Application {
	app := Application{}

	repo, _ := repository.New(dsn.FromEnv())

	app.repo = *repo

	return app

}

func (a *Application) StartServer() {
	log.Println("Server is starting up...")

	a.r = gin.Default()

	a.r.LoadHTMLGlob("templates/*.html")
	a.r.Static("/css", "templates/css")
	a.r.Static("/js", "templates/js")
	a.r.Static("/font", "resources/font")

	a.r.GET("/", a.loadCourses)
	a.r.GET("/:course_title", a.loadCourse)
	a.r.POST("/delete_course/:course_title", a.loadCourseChangeVisibility)

	a.r.Run()

	log.Println("Server shutdown.")
}

func (a *Application) loadCourses(c *gin.Context) {
	course_title := c.Query("course_title")

	if course_title == "" {
		all_courses, err := a.repo.GetAllCourses()

		if err != nil {
			log.Println(err)
			c.Error(err)
		}

		c.HTML(http.StatusOK, "courses.html", gin.H{
			"courses": all_courses,
		})
	} else {
		found_courses, err := a.repo.SearchCourses(course_title)

		if err != nil {
			c.Error(err)
			return
		}

		c.HTML(http.StatusOK, "courses.html", gin.H{
			"courses":     found_courses,
			"Search_text": course_title,
		})
	}
}

func (a *Application) loadCourse(c *gin.Context) {
	course_title := c.Param("course_title")

	if course_title == "favicon.ico" {
		return
	}

	course, err := a.repo.GetCourseByName(course_title)

	if err != nil {
		c.Error(err)
		return
	}

	c.HTML(http.StatusOK, "course.html", gin.H{
		"Title":         course.Title,
		"Image":         course.Image,
		"Location":      course.Location,
		"Description":   course.Description,
		"CoachName":     course.CoachName,
		"CoachEmail":    course.CoachEmail,
		"CoachPhone":    course.CoachPhone,
		"Capacity":      course.Capacity,
		"Enrolled":      course.Enrolled,
		"Course_status": course.Status,
	})
}

func (a *Application) loadCourseChangeVisibility(c *gin.Context) {
	course_title := c.Param("course_title")
	err := a.repo.ChangeCourseVisibility(course_title)

	if err != nil {
		c.Error(err)
	}

	c.Redirect(http.StatusFound, "/")
}
