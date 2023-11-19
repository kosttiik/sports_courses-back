package app

import (
	"log"

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

	a.r.GET("courses", a.get_courses)
	a.r.GET("course", a.get_course)
	a.r.GET("enrollments", a.get_enrollments)
	a.r.GET("enrollment", a.get_enrollment)

	a.r.PUT("enroll", a.enroll_course)

	a.r.PUT("course/add", a.add_course)
	a.r.PUT("course/edit", a.edit_course)
	a.r.PUT("enrollment/edit", a.edit_enrollment)
	a.r.PUT("enrollment/status_change/moderator", a.enrollment_mod_status_change)
	a.r.PUT("enrollment/status_change/user", a.enrollment_user_status_change)

	a.r.PUT("course/delete/:course_title", a.delete_course)
	a.r.PUT("enrollment/delete/:enrollment_id", a.delete_enrollment)

	a.r.DELETE("enrollment_to_course/delete", a.delete_enrollment_to_course)

	a.r.Run()

	log.Println("Server shutdown.")
}

func (a *Application) get_courses(c *gin.Context) {

}

func (a *Application) add_course(c *gin.Context) {

}

func (a *Application) get_course(c *gin.Context) {

}

func (a *Application) edit_course(c *gin.Context) {

}

func (a *Application) delete_course(c *gin.Context) {

}

func (a *Application) enroll_course(c *gin.Context) {

}

func (a *Application) get_enrollments(c *gin.Context) {

}

func (a *Application) get_enrollment(c *gin.Context) {

}

func (a *Application) edit_enrollment(c *gin.Context) {

}

func (a *Application) enrollment_mod_status_change(c *gin.Context) {

}

func (a *Application) enrollment_user_status_change(c *gin.Context) {

}

func (a *Application) delete_enrollment(c *gin.Context) {

}

func (a *Application) delete_enrollment_to_course(c *gin.Context) {

}
