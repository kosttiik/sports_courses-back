package repository

import (
	"log"
	"strings"

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

func (r *Repository) GetCourseByID(id int) (*ds.Course, error) {
	course := &ds.Course{}

	err := r.db.First(course, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return course, nil
}

func (r *Repository) GetCourseByName(title string) (*ds.Course, error) {
	course := &ds.Course{}

	err := r.db.First(course, "title = ?", title).Error
	if err != nil {
		return nil, err
	}

	return course, nil
}

func (r *Repository) SearchCourses(course_title string) ([]ds.Course, error) {
	courses := []ds.Course{}

	all_courses, all_courses_err := r.GetAllCourses()

	if all_courses_err != nil {
		return nil, all_courses_err
	}

	for i := range all_courses {
		if strings.Contains(strings.ToLower(all_courses[i].Title), strings.ToLower(course_title)) {
			courses = append(courses, all_courses[i])
		}
	}

	return courses, nil
}

func (r *Repository) GetAllCourses() ([]ds.Course, error) {
	courses := []ds.Course{}

	err := r.db.Find(&courses).Error

	if err != nil {
		return nil, err
	}

	return courses, nil
}

func (r *Repository) FilterActiveCourses(courses []ds.Course) []ds.Course {
	var new_courses = []ds.Course{}

	for i := range courses {
		if courses[i].Status == "Действует" {
			new_courses = append(new_courses, courses[i])
		}
	}

	return new_courses

}

func (r *Repository) LogicalDeleteCourse(course_title string) error {
	return r.db.Model(&ds.Course{}).Where("title = ?", course_title).Update("status", "Недоступен").Error
}

func (r *Repository) ChangeCourseVisibility(course_title string) error {
	course, err := r.GetCourseByName(course_title)

	if err != nil {
		log.Println(err)
		return err
	}

	new_status := ""

	if course.Status == "Действует" {
		new_status = "Недоступен"
	} else {
		new_status = "Действует"
	}

	return r.db.Model(&ds.Course{}).Where("title = ?", course_title).Update("status", new_status).Error
}

func (r *Repository) DeleteCourse(course_title string) error {
	return r.db.Delete(&ds.Course{}, "title = ?", course_title).Error
}

func (r *Repository) CreateCourse(course ds.Course) error {
	return r.db.Create(course).Error
}

func (r *Repository) CreateUser(user ds.User) error {
	return r.db.Create(user).Error
}

func (r *Repository) CreateEnrollment(enrollment ds.Enrollment) error {
	return r.db.Create(enrollment).Error
}

func (r *Repository) CreateEnrollmentToCourse(enrollment_to_course ds.EnrollmentToCourse) error {
	return r.db.Create(enrollment_to_course).Error
}
