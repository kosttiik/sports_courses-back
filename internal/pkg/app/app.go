package app

import (
	"log"
	"net/http"
	"strconv"

	"sports_courses/docs"
	"sports_courses/internal/app/ds"
	"sports_courses/internal/app/dsn"
	"sports_courses/internal/app/repository"

	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
)

// @BasePath /

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
	a.r.GET("course/:course", a.get_course)

	// swagger
	docs.SwaggerInfo.BasePath = "/"
	a.r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	a.r.GET("enrollments", a.get_enrollments)
	a.r.GET("enrollment", a.get_enrollment)

	a.r.PUT("enroll", a.enroll_course)

	a.r.PUT("course/add", a.add_course)
	a.r.PUT("course/edit", a.edit_course)
	a.r.PUT("enrollment/edit", a.edit_enrollment)
	a.r.PUT("enrollment/status_change/moderator", a.enrollment_mod_status_change)
	a.r.PUT("enrollment/status_change/user", a.enrollment_user_status_change)

	a.r.PUT("course/delete/:course_title", a.delete_course)
	a.r.PUT("course/delete_restore/:course_title", a.delete_restore_course)
	a.r.PUT("enrollment/delete/:enrollment_id", a.delete_enrollment)
	a.r.PUT("enrollment_to_course/delete", a.delete_enrollment_to_course)

	docs.SwaggerInfo.BasePath = "/"
	a.r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	a.r.Run()

	log.Println("Server shutdown.")
}

// @Summary Get all existing courses
// @Schemes
// @Description Returns all existing courses
// @Tags courses
// @Accept json
// @Produce json
// @Success 200 {} string
// @Router /courses [get]
func (a *Application) get_courses(c *gin.Context) {
	var title_pattern = c.Query("title_pattern")
	var location = c.Query("location")
	var status = c.Query("status")

	courses, err := a.repo.GetAllCourses(title_pattern, location, status)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, courses)
}

// @Summary      Adds courses to database
// @Description  Creates a new course with parameters, specified in json
// @Tags courses
// @Accept json
// @Produce      json
// @Success      302  {object}  string
// @Router       /course/add [put]
func (a *Application) add_course(c *gin.Context) {
	var course ds.Course

	if err := c.BindJSON(&course); err != nil {
		c.String(http.StatusBadRequest, "Can't parse course\n"+err.Error())
		return
	}

	err := a.repo.CreateCourse(course)

	if err != nil {
		c.String(http.StatusNotFound, "Can't create course\n"+err.Error())
		return
	}

	c.String(http.StatusCreated, "Course created successfully")

}

// @Summary      Get course
// @Description  Returns course with given name
// @Tags         courses
// @Produce      json
// @Success      200  {object}  string
// @Router       /course/:course [get]
func (a *Application) get_course(c *gin.Context) {
	var course = ds.Course{}

	course.Title = c.Param("course")

	found_course, err := a.repo.FindCourse(course)

	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusFound, found_course)

}

// @Summary      Edits course
// @Description  Finds course by name and updates its fields
// @Tags         courses
// @Accept json
// @Produce      json
// @Success      302  {object}  string
// @Router       /course/edit [put]
func (a *Application) edit_course(c *gin.Context) {
	var course ds.Course

	if err := c.BindJSON(&course); err != nil {
		c.Error(err)
		return
	}

	err := a.repo.EditCourse(course)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusCreated, "Course was successfuly edited")

}

// @Summary      Deletes course
// @Description  Finds course by name and changes its status to "Недоступен"
// @Tags         courses
// @Accept json
// @Produce      json
// @Success      302  {object}  string
// @Router       /course/delete/:course_title [put]
func (a *Application) delete_course(c *gin.Context) {
	course_title := c.Param("course_title")

	log.Println(course_title)

	err := a.repo.LogicalDeleteCourse(course_title)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusFound, "Course was successfully deleted")
}

// @Summary      Deletes or restores course
// @Description  Switches course status from "Действует" to "Недоступен" and back
// @Tags         courses
// @Produce      json
// @Success      200  {object}  string
// @Router       /course/delete_restore/:course_title [get]
func (a *Application) delete_restore_course(c *gin.Context) {
	course_title := c.Param("course_title")

	err := a.repo.DeleteRestoreCourse(course_title)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusFound, "Course status was successfully switched")
}

// @Summary      Enroll course
// @Description  Creates a new enrollment and adds current course in it
// @Tags general
// @Accept json
// @Produce      json
// @Success      302  {object}  string
// @Router       /enroll [put]
func (a *Application) enroll_course(c *gin.Context) {
	var request_body ds.EnrollCourseRequestBody

	if err := c.BindJSON(&request_body); err != nil {
		c.Error(err)
		c.String(http.StatusBadGateway, "Can't parse json")
		return
	}

	err := a.repo.EnrollCourse(request_body)

	if err != nil {
		c.Error(err)
		c.String(http.StatusNotFound, "Can't enroll course")
		return
	}

	c.String(http.StatusCreated, "Course was successfully enrolled")

}

// @Summary      Get enrollments
// @Description  Returns list of all available enrollments
// @Tags         enrollments
// @Produce      json
// @Success      302  {object}  string
// @Router       /enrollments [get]
func (a *Application) get_enrollments(c *gin.Context) {
	var requestBody ds.GetEnrollmentsRequestBody

	c.BindJSON(&requestBody)

	enrollments, err := a.repo.GetAllEnrollments(requestBody)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusFound, enrollments)
}

// a.r.GET("enrollment", a.get_enrollment)
// @Summary      Get enrollment
// @Description  Returns enrollment with given parameters
// @Tags         enrollments
// @Accept		 json
// @Produce      json
// @Success      302  {object}  string
// @Router       /enrollment [get]
func (a *Application) get_enrollment(c *gin.Context) {
	var enrollment ds.Enrollment

	if err := c.BindJSON(&enrollment); err != nil {
		c.Error(err)
		return
	}

	found_enrollment, err := a.repo.FindEnrollment(enrollment)

	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusFound, found_enrollment)
}

// @Summary      Edits enrollment
// @Description  Finds enrollment and updates it fields
// @Tags         enrollments
// @Accept json
// @Produce      json
// @Success      201  {object}  string
// @Router       /enrollment/edit [put]
func (a *Application) edit_enrollment(c *gin.Context) {
	var enrollment ds.Enrollment

	if err := c.BindJSON(&enrollment); err != nil {
		c.Error(err)
		return
	}

	err := a.repo.EditEnrollment(enrollment)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusCreated, "Enrollment was successfuly edited")
}

// @Summary      Changes enrollment status as moderator
// @Description  Changes enrollment status to any available status
// @Tags         enrollments
// @Accept json
// @Produce      json
// @Success      201  {object}  string
// @Router       /enrollment/status_change/moderator [put]
func (a *Application) enrollment_mod_status_change(c *gin.Context) {
	var requestBody ds.ChangeEnrollmentStatusRequestBody

	if err := c.BindJSON(&requestBody); err != nil {
		c.Error(err)
		return
	}

	user_role, err := a.repo.GetUserRole(requestBody.UserName)

	if err != nil {
		c.Error(err)
		return
	}

	if user_role != "Модератор" {
		c.String(http.StatusBadRequest, "у пользователя должна быть роль модератора")
		return
	}

	err = a.repo.ChangeEnrollmentStatus(requestBody.ID, requestBody.Status)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusCreated, "Enrollment status was successfully changed")
}

// Ping godoc
// @Summary      Changes enrollments status as user
// @Description  Changes enrollment status as allowed to user
// @Tags         enrollments
// @Accept json
// @Produce      json
// @Success      201  {object}  string
// @Router       /enrollment/status_change/user [put]
func (a *Application) enrollment_user_status_change(c *gin.Context) {
	var requestBody ds.ChangeEnrollmentStatusRequestBody

	if err := c.BindJSON(&requestBody); err != nil {
		c.Error(err)
		return
	}

	err := a.repo.ChangeEnrollmentStatus(requestBody.ID, requestBody.Status)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusCreated, "Enrollment status was successfully changed")
}

// @Summary      Deletes enrollment
// @Description  Changes enrollment status to "Удалён"
// @Tags         enrollments
// @Accept json
// @Produce      json
// @Success      302  {object}  string
// @Router       /enrollment/delete/:enrollment_id [put]
func (a *Application) delete_enrollment(c *gin.Context) {
	enrollment_id, _ := strconv.Atoi(c.Param("enrollment_id"))

	err := a.repo.LogicalDeleteEnrollment(enrollment_id)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusFound, "Enrollment was successfully deleted")
}

// @Summary      Deletes enrollment_to_course connection
// @Description  Deletes course from enrollment
// @Tags         enrollments
// @Accept json
// @Produce      json
// @Success      201  {object}  string
// @Router       /enrollment_to_course/delete [put]
func (a *Application) delete_enrollment_to_course(c *gin.Context) {
	var requestBody ds.DeleteEnrollmentToCourseRequestBody

	if err := c.BindJSON(&requestBody); err != nil {
		c.Error(err)
		return
	}

	err := a.repo.DeleteEnrollmentToCourse(requestBody.EnrollmentID, requestBody.CourseID)

	if err != nil {
		c.Error(err)
		return
	}

	c.String(http.StatusCreated, "Enrollment-to-course m-m was successfully deleted")
}
