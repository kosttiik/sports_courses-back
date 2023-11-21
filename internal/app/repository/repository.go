package repository

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"sports_courses/internal/app/ds"
)

type Repository struct {
	db *gorm.DB
}

func New(dsn string) (*Repository, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &Repository{
		db: db,
	}, nil
}

func (r *Repository) GetCourseByTitle(title string) (*ds.Course, error) {
	course := &ds.Course{}

	err := r.db.First(course, "title = ?", title).Error
	if err != nil {
		return nil, err
	}

	return course, nil
}

func (r *Repository) GetCourseByID(id int) (*ds.Course, error) {
	course := &ds.Course{}

	err := r.db.First(course, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return course, nil
}

func (r *Repository) GetUserByID(id int) (*ds.User, error) {
	user := &ds.User{}

	err := r.db.First(user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *Repository) GetUserID(name string) (int, error) {
	user := &ds.User{}

	err := r.db.First(user, "name = ?", name).Error
	if err != nil {
		return -1, err
	}

	return int(user.ID), nil
}

func (r *Repository) GetCourseID(title string) (int, error) {
	course := &ds.Course{}

	err := r.db.First(course, "title = ?", title).Error
	if err != nil {
		return -1, err
	}

	return int(course.ID), nil
}

func (r *Repository) GetUserRole(name string) (string, error) {
	user := &ds.User{}

	err := r.db.First(user, "name = ?", name).Error
	if err != nil {
		return "", err
	}

	return user.Role, nil
}

func (r *Repository) GetAllCourses(requestBody ds.GetCoursesRequestBody) ([]ds.Course, error) {
	courses := []ds.Course{}

	var tx *gorm.DB = r.db
	if requestBody.Location != "" {
		tx = tx.Where("location = ?", requestBody.Location)
	}
	if requestBody.Status != "" {
		tx = tx.Where("status = ?", requestBody.Status)
	}

	err := tx.Find(&courses).Error

	if err != nil {
		return nil, err
	}

	return courses, nil
}

func (r *Repository) GetAllEnrollments(requestBody ds.GetEnrollmentsRequestBody) ([]ds.Enrollment, error) {
	enrollments := []ds.Enrollment{}

	var tx *gorm.DB = r.db
	if requestBody.Status != "" {
		tx = tx.Where("status = ?", requestBody.Status)
	}

	err := tx.Find(&enrollments).Error

	if err != nil {
		return nil, err
	}

	for i := range enrollments {
		if enrollments[i].ModeratorRefer != 0 {
			moderator, _ := r.GetUserByID(enrollments[i].ModeratorRefer)
			enrollments[i].Moderator = *moderator
		}
		user, _ := r.GetUserByID(enrollments[i].UserRefer)
		enrollments[i].User = *user
	}

	return enrollments, nil
}

func (r *Repository) CreateCourse(course ds.Course) error {
	return r.db.Create(&course).Error
}

func (r *Repository) CreateUser(user ds.User) error {
	return r.db.Create(&user).Error
}

func (r *Repository) CreateEnrollment(enrollment ds.Enrollment) error {
	return r.db.Create(&enrollment).Error
}

func (r *Repository) CreateEnrollmentToCourse(enrollment_to_course ds.EnrollmentToCourse) error {
	return r.db.Create(&enrollment_to_course).Error
}

func (r *Repository) DeleteCourse(course_title string) error {
	return r.db.Delete(&ds.Course{}, "title = ?", course_title).Error
}

func (r *Repository) DeleteEnrollment(id int) error {
	return r.db.Delete(&ds.Enrollment{}, "id = ?", id).Error
}

func (r *Repository) DeleteEnrollmentToCourse(enrollment_id int, course_id int) error {
	return r.db.Where("enrollment_refer = ?", enrollment_id).Where("course_refer = ?", course_id).Delete(&ds.EnrollmentToCourse{}).Error
}

func (r *Repository) LogicalDeleteCourse(course_title string) error {
	return r.db.Model(&ds.Course{}).Where("title = ?", course_title).Update("status", "Недоступен").Error
}

func (r *Repository) LogicalDeleteEnrollment(enrollment_id int) error {
	return r.db.Model(&ds.Enrollment{}).Where("id = ?", enrollment_id).Update("status", "Удалён").Error
}

func (r *Repository) FindCourse(course ds.Course) (ds.Course, error) {
	var result ds.Course
	err := r.db.Where(&course).First(&result).Error
	if err != nil {
		return ds.Course{}, err
	} else {
		return result, nil
	}
}

func (r *Repository) FindEnrollment(enrollment ds.Enrollment) (ds.Enrollment, error) {
	var result ds.Enrollment
	err := r.db.Where(&enrollment).First(&result).Error
	if err != nil {
		return ds.Enrollment{}, err
	}

	var user ds.User
	r.db.Where("id = ?", result.UserRefer).First(&user)

	result.User = user

	var moderator ds.User
	r.db.Where("id = ?", result.ModeratorRefer).First(&user)

	result.Moderator = moderator

	return result, nil
}

func (r *Repository) EditCourse(course ds.Course) error {
	return r.db.Model(&ds.Course{}).Where("title = ?", course.Title).Updates(course).Error
}

func (r *Repository) EditEnrollment(enrollment ds.Enrollment) error {
	return r.db.Model(&ds.Enrollment{}).Where("id = ?", enrollment.ID).Updates(enrollment).Error
}

func (r *Repository) EnrollCourse(requestBody ds.EnrollCourseRequestBody) error {
	user_id, err := r.GetUserID(requestBody.UserName)

	if err != nil {
		return err
	}

	var course_id int
	course_id, err = r.GetCourseID(requestBody.CourseName)
	if err != nil {
		return err
	}

	current_date := datatypes.Date(time.Now())
	start_date, err := time.Parse(time.RFC3339, requestBody.StartDate+"T00:00:00Z")
	if err != nil {
		return err
	}
	end_date, err := time.Parse(time.RFC3339, requestBody.EndDate+"T00:00:00Z")
	if err != nil {
		return err
	}

	enrollment := ds.Enrollment{}
	enrollment.StartDate = datatypes.Date(start_date)
	enrollment.EndDate = datatypes.Date(end_date)
	enrollment.UserRefer = user_id
	enrollment.DateCreated = current_date

	err = r.db.Omit("moderator_refer", "date_processed", "date_finished").Create(&enrollment).Error

	if err != nil {
		return err
	}

	enrollment_to_course := ds.EnrollmentToCourse{}
	enrollment_to_course.EnrollmentRefer = int(enrollment.ID)
	enrollment_to_course.CourseRefer = int(course_id)
	err = r.CreateEnrollmentToCourse(enrollment_to_course)

	return err

}

func (r *Repository) ChangeEnrollmentStatus(id int, status string) error {
	return r.db.Model(&ds.Enrollment{}).Where("id = ?", id).Update("status", status).Error
}
