package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"sports_courses/internal/app/ds"
	"sports_courses/internal/app/role"
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

func (r *Repository) GetUserByID(id uuid.UUID) (*ds.User, error) {
	user := &ds.User{}

	err := r.db.First(user, "UUID = ?", id).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *Repository) GetUserByLogin(login string) (*ds.User, error) {
	user := &ds.User{}

	err := r.db.First(user, "name = ?", login).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *Repository) GetUserID(name string) (uuid.UUID, error) {
	user := &ds.User{}

	err := r.db.First(user, "name = ?", name).Error
	if err != nil {
		return uuid.Nil, err
	}

	return user.UUID, nil
}

func (r *Repository) GetCourseID(title string) (int, error) {
	course := &ds.Course{}

	err := r.db.First(course, "title = ?", title).Error
	if err != nil {
		return -1, err
	}

	return int(course.ID), nil
}

func (r *Repository) GetCourseStatus(title string) (string, error) {
	course := &ds.Course{}

	err := r.db.First(course, "title = ?", title).Error
	if err != nil {
		return "", err
	}

	return course.Status, nil
}

func (r *Repository) GetUserRole(name string) (role.Role, error) {
	user := &ds.User{}

	err := r.db.First(user, "name = ?", name).Error
	if err != nil {
		return role.Undefined, err
	}

	return user.Role, nil
}

func (r *Repository) GetAllCourses(title_pattern string, location string, status string) ([]ds.Course, error) {
	courses := []ds.Course{}

	var tx *gorm.DB = r.db

	if title_pattern != "" {
		tx = tx.Where("title like ?", "%"+title_pattern+"%")

	}

	if location != "" {
		tx = tx.Where("location = ?", location)
	}

	if status != "" {
		tx = tx.Where("status = ?", status)
	}

	err := tx.Find(&courses).Error

	if err != nil {
		return nil, err
	}

	return courses, nil
}

func (r *Repository) GetAllEnrollments(status string, roleNumber role.Role, userUUID uuid.UUID) ([]ds.Enrollment, error) {
	enrollments := []ds.Enrollment{}

	var tx *gorm.DB = r.db
	if status != "" {
		tx = tx.Where("status = ?", status)
	}

	if roleNumber == role.User {
		tx = tx.Where("user_refer = ?", userUUID)
	}

	err := tx.Find(&enrollments).Error

	if err != nil {
		return nil, err
	}

	for i := range enrollments {
		if enrollments[i].ModeratorRefer != uuid.Nil {
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

func (r *Repository) DeleteRestoreCourse(course_title string) error {
	var new_status string

	course_status, err := r.GetCourseStatus(course_title)

	if err != nil {
		return err
	}

	if course_status == "Действует" {
		new_status = "Недоступен"
	} else {
		new_status = "Действует"
	}

	return r.db.Model(&ds.Course{}).Where("title = ?", course_title).Update("status", new_status).Error

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

func (r *Repository) FindEnrollment(enrollment *ds.Enrollment) (ds.Enrollment, error) {
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

func (r *Repository) EditCourse(course *ds.Course) error {
	return r.db.Model(&ds.Course{}).Where("title = ?", course.Title).Updates(course).Error
}

func (r *Repository) EditEnrollment(enrollment *ds.Enrollment) error {
	return r.db.Model(&ds.Enrollment{}).Where("id = ?", enrollment.ID).Updates(enrollment).Error
}

func (r *Repository) EnrollCourse(requestBody ds.EnrollCourseRequestBody, userUUID uuid.UUID) error {
	course_id, err := r.GetCourseID(requestBody.CourseName)
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
	enrollment.UserRefer = userUUID
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

func (r *Repository) GetEnrollmentStatus(id int) (string, error) {
	var result ds.Enrollment
	err := r.db.Where("id = ?", id).First(&result).Error
	if err != nil {
		return "", err
	}

	return result.Status, nil
}

func (r *Repository) ChangeEnrollmentStatusUser(id int, status string, userUUID uuid.UUID) error {
	return r.db.Model(&ds.Enrollment{}).Where("id = ?", id).Where("user_refer = ?", userUUID).Update("status", status).Error
}

func (r *Repository) ChangeEnrollmentStatus(id int, status string) error {
	return r.db.Model(&ds.Enrollment{}).Where("id = ?", id).Update("status", status).Error
}

func (r *Repository) Register(user *ds.User) error {
	if user.UUID == uuid.Nil {
		user.UUID = uuid.New()
	}

	return r.db.Create(user).Error
}
