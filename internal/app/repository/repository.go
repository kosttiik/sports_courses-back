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

func (r *Repository) GetGroupByTitle(title string) (*ds.Group, error) {
	group := &ds.Group{}

	err := r.db.First(group, "title = ?", title).Error
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (r *Repository) GetGroupByID(id int) (*ds.Group, error) {
	group := &ds.Group{}

	err := r.db.First(group, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return group, nil
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

func (r *Repository) GetGroupID(title string) (int, error) {
	group := &ds.Group{}

	err := r.db.First(group, "title = ?", title).Error
	if err != nil {
		return -1, err
	}

	return int(group.ID), nil
}

func (r *Repository) GetGroupStatus(title string) (string, error) {
	group := &ds.Group{}

	err := r.db.First(group, "title = ?", title).Error
	if err != nil {
		return "", err
	}

	return group.Status, nil
}

func (r *Repository) GetUserRole(name string) (role.Role, error) {
	user := &ds.User{}

	err := r.db.First(user, "name = ?", name).Error
	if err != nil {
		return role.Undefined, err
	}

	return user.Role, nil
}

func (r *Repository) GetAllGroups(title_pattern string, course string, location string, status string) ([]ds.Group, error) {
	groups := []ds.Group{}

	var tx *gorm.DB = r.db

	if title_pattern != "" {
		tx = tx.Where("title like ?", "%"+title_pattern+"%")

	}

	if course != "" {
		tx = tx.Where("course = ?", course)
	}

	if location != "" {
		tx = tx.Where("location = ?", location)
	}

	if status != "" {
		tx = tx.Where("status = ?", status)
	}

	err := tx.Find(&groups).Error

	if err != nil {
		return nil, err
	}

	return groups, nil
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

func (r *Repository) CreateGroup(group ds.Group) error {
	return r.db.Create(&group).Error
}

func (r *Repository) CreateUser(user ds.User) error {
	return r.db.Create(&user).Error
}

func (r *Repository) CreateEnrollment(enrollment ds.Enrollment) error {
	return r.db.Create(&enrollment).Error
}

func (r *Repository) CreateEnrollmentToGroup(enrollment_to_group ds.EnrollmentToGroup) error {
	return r.db.Create(&enrollment_to_group).Error
}

func (r *Repository) DeleteGroup(group_title string) error {
	return r.db.Delete(&ds.Group{}, "title = ?", group_title).Error
}

func (r *Repository) DeleteEnrollment(id int) error {
	return r.db.Delete(&ds.Enrollment{}, "id = ?", id).Error
}

func (r *Repository) DeleteEnrollmentToGroup(enrollment_id int, group_id int) error {
	return r.db.Where("enrollment_refer = ?", enrollment_id).Where("group_refer = ?", group_id).Delete(&ds.EnrollmentToGroup{}).Error
}

func (r *Repository) LogicalDeleteGroup(group_title string) error {
	return r.db.Model(&ds.Group{}).Where("title = ?", group_title).Update("status", "Недоступен").Error
}

func (r *Repository) LogicalDeleteEnrollment(enrollment_id int) error {
	return r.db.Model(&ds.Enrollment{}).Where("id = ?", enrollment_id).Update("status", "Удалён").Error
}

func (r *Repository) DeleteRestoreGroup(group_title string) error {
	var new_status string

	group_status, err := r.GetGroupStatus(group_title)

	if err != nil {
		return err
	}

	if group_status == "Действует" {
		new_status = "Недоступен"
	} else {
		new_status = "Действует"
	}

	return r.db.Model(&ds.Group{}).Where("title = ?", group_title).Update("status", new_status).Error
}

func (r *Repository) FindGroup(group ds.Group) (ds.Group, error) {
	var result ds.Group
	err := r.db.Where(&group).First(&result).Error
	if err != nil {
		return ds.Group{}, err
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
	r.db.Where("uuid = ?", result.UserRefer).First(&user)

	result.User = user

	var moderator ds.User
	r.db.Where("uuid = ?", result.ModeratorRefer).First(&user)

	result.Moderator = moderator

	return result, nil
}

func (r *Repository) EditGroup(group *ds.Group) error {
	return r.db.Model(&ds.Group{}).Where("title = ?", group.Title).Updates(group).Error
}

func (r *Repository) EditEnrollment(enrollment *ds.Enrollment, moderatorUUID uuid.UUID) error {
	enrollment.DateProcessed = datatypes.Date(time.Now())
	enrollment.ModeratorRefer = moderatorUUID
	return r.db.Model(&ds.Enrollment{}).Where("id = ?", enrollment.ID).Updates(enrollment).Error
}

func (r *Repository) Enroll(requestBody ds.EnrollRequestBody, userUUID uuid.UUID) error {
	var group_ids []int
	for _, groupTitle := range requestBody.Groups {
		group_id, err := r.GetGroupID(groupTitle)
		if err != nil {
			return err
		}
		group_ids = append(group_ids, group_id)
	}

	current_date := datatypes.Date(time.Now())

	enrollment := ds.Enrollment{}
	enrollment.UserRefer = userUUID
	enrollment.DateCreated = current_date
	enrollment.Status = "Черновик"

	err := r.db.Omit("moderator_refer", "date_processed", "date_finished").Create(&enrollment).Error
	if err != nil {
		return err
	}

	for _, group_id := range group_ids {
		enrollment_to_group := ds.EnrollmentToGroup{}
		enrollment_to_group.EnrollmentRefer = int(enrollment.ID)
		enrollment_to_group.GroupRefer = int(group_id)
		err = r.CreateEnrollmentToGroup(enrollment_to_group)

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) GetEnrollmentStatus(id int) (string, error) {
	var result ds.Enrollment
	err := r.db.Where("id = ?", id).First(&result).Error
	if err != nil {
		return "", err
	}

	return result.Status, nil
}

func (r *Repository) GetEnrollmentToGroupStatus(id int) (string, error) {
	var result ds.EnrollmentToGroup
	err := r.db.Where("id = ?", id).First(&result).Error
	if err != nil {
		return "", err
	}

	return result.Status, nil
}

func (r *Repository) GetEnrollmentGroups(id int) ([]ds.Group, error) {
	enrollment_to_groups := []ds.EnrollmentToGroup{}

	err := r.db.Model(&ds.EnrollmentToGroup{}).Where("enrollment_refer = ?", id).Find(&enrollment_to_groups).Error
	if err != nil {
		return []ds.Group{}, err
	}

	var groups []ds.Group
	for _, enrollment_to_group := range enrollment_to_groups {
		group, err := r.GetGroupByID(enrollment_to_group.GroupRefer)
		if err != nil {
			return []ds.Group{}, err
		}
		for _, ele := range groups {
			if ele == *group {
				continue
			}
		}
		groups = append(groups, *group)
	}

	return groups, nil
}

func (r *Repository) SetEnrollmentGroups(enrollmentID int, groups []string) error {
	var group_ids []int
	for _, group := range groups {
		group_id, err := r.GetGroupID(group)
		if err != nil {
			return err
		}

		for _, ele := range group_ids {
			if ele == group_id {
				continue
			}
		}
		group_ids = append(group_ids, group_id)
	}

	var existing_links []ds.EnrollmentToGroup
	err := r.db.Model(&ds.EnrollmentToGroup{}).Where("enrollment_refer = ?", enrollmentID).Find(&existing_links).Error
	if err != nil {
		return err
	}

	for _, link := range existing_links {
		groupFound := false
		groupIndex := -1
		for index, ele := range group_ids {
			if ele == link.GroupRefer {
				groupFound = true
				groupIndex = index
				break
			}
		}

		if groupFound {
			group_ids = append(group_ids[:groupIndex], group_ids[groupIndex+1:]...)
		} else {
			err := r.db.Model(&ds.EnrollmentToGroup{}).Delete(&link).Error
			if err != nil {
				return err
			}
		}
	}

	for _, group_id := range group_ids {
		newLink := ds.EnrollmentToGroup{
			EnrollmentRefer: enrollmentID,
			GroupRefer:      group_id,
		}

		err := r.db.Model(&ds.EnrollmentToGroup{}).Create(&newLink).Error
		if err != nil {
			return nil
		}
	}

	return nil
}

func (r *Repository) SetEnrollmentModerator(enrollmentID int, moderatorUUID uuid.UUID) error {
	return r.db.Model(&ds.Enrollment{}).Where("id = ?", enrollmentID).Update("moderator_refer", moderatorUUID).Error
}

func (r *Repository) ChangeEnrollmentStatusUser(id int, status string, userUUID uuid.UUID) error {
	return r.db.Model(&ds.Enrollment{}).Where("id = ?", id).Where("user_refer = ?", userUUID).Update("status", status).Error
}

func (r *Repository) ChangeEnrollmentStatus(id int, status string) error {
	return r.db.Model(&ds.Enrollment{}).Where("id = ?", id).Update("status", status).Error
}

func (r *Repository) ChangeEnrollmentToGroupStatus(id int, status string) error {
	return r.db.Model(&ds.EnrollmentToGroup{}).Where("id = ?", id).Update("status", status).Error
}

func (r *Repository) Register(user *ds.User) error {
	if user.UUID == uuid.Nil {
		user.UUID = uuid.New()
	}

	return r.db.Create(user).Error
}
