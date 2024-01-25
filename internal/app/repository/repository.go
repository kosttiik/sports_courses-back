package repository

import (
	"time"

	"github.com/google/uuid"
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

	err := r.db.Find(group, "title = ?", title).Error
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (r *Repository) GetGroupByID(id int) (*ds.Group, error) {
	group := &ds.Group{}

	err := r.db.Find(group, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (r *Repository) GetUserByID(id uuid.UUID) (*ds.User, error) {
	user := &ds.User{}

	err := r.db.Find(user, "UUID = ?", id).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *Repository) GetUserByLogin(login string) (*ds.User, error) {
	user := &ds.User{}

	err := r.db.Find(user, "name = ?", login).Error
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *Repository) GetUserID(name string) (uuid.UUID, error) {
	user := &ds.User{}

	err := r.db.Find(user, "name = ?", name).Error
	if err != nil {
		return uuid.Nil, err
	}

	return user.UUID, nil
}

func (r *Repository) GetGroupID(title string) (int, error) {
	group := &ds.Group{}

	err := r.db.Find(group, "title = ?", title).Error
	if err != nil {
		return -1, err
	}

	return int(group.ID), nil
}

func (r *Repository) GetGroupStatus(title string) (string, error) {
	group := &ds.Group{}

	err := r.db.Find(group, "title = ?", title).Error
	if err != nil {
		return "", err
	}

	return group.Status, nil
}

func (r *Repository) GetUserRole(name string) (role.Role, error) {
	user := &ds.User{}

	err := r.db.Find(user, "name = ?", name).Error
	if err != nil {
		return role.Undefined, err
	}

	return user.Role, nil
}

func (r *Repository) GetGroups(title_pattern string, course string, status string) ([]ds.Group, error) {
	groups := []ds.Group{}

	var tx *gorm.DB = r.db

	if title_pattern != "" {
		tx = tx.Where("title like ?", "%"+title_pattern+"%")

	}

	if course != "" {
		tx = tx.Where("course = ?", course)
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

func (r *Repository) GetEnrollments(status string, startDate string, endDate string, roleNumber role.Role, userUUID uuid.UUID) ([]ds.Enrollment, error) {
	enrollments := []ds.Enrollment{}

	var tx *gorm.DB = r.db
	if status != "" {
		tx = tx.Where("status = ?", status)
	}

	if startDate != "" {
		tx = tx.Where("date_created >= ?", startDate)
	}

	if endDate != "" {
		tx = tx.Where("date_created <= ?", endDate)
	}

	if roleNumber == role.User {
		tx = tx.Where("user_refer = ?", userUUID)
	}

	err := tx.Find(&enrollments).Error

	if err != nil {
		return nil, err
	}

	for i := range enrollments {
		if enrollments[i].ModeratorRefer != nil {
			moderator, _ := r.GetUserByID(*enrollments[i].ModeratorRefer)
			enrollments[i].Moderator = *moderator
		}
		user, _ := r.GetUserByID(*enrollments[i].UserRefer)
		enrollments[i].User = *user
	}

	return enrollments, nil
}

func (r *Repository) GetDraftEnrollment(user uuid.UUID) (ds.Enrollment, error) {
	enrollment := ds.Enrollment{}

	err := r.db.Where("user_refer = ?", user).Where("status = ?", "Черновик").Find(&enrollment).Error

	return enrollment, err
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

func (r *Repository) LogicalDeleteGroup(group_title string) error {
	tx := r.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Exec(`UPDATE public.groups SET status = ? WHERE title = ?`, "Недоступен", group_title).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *Repository) LogicalDeleteEnrollment(enrollment_id int) error {
	tx := r.db.Begin()

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Exec(`UPDATE public.enrollments SET status = ? WHERE id = ?`, "Удалён", enrollment_id).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *Repository) ModeratorConfirmEnrollment(uuid uuid.UUID, enrollment_id int, confirm bool) error {
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	new_status := "Отклонён"
	if confirm {
		new_status = "Завершён"
	}

	if err := tx.Exec(`UPDATE public.enrollments SET status = ?, moderator_refer = ?, date_processed = ? WHERE id = ?`, new_status, uuid, time.Now(), enrollment_id).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *Repository) UserConfirmEnrollment(uuid uuid.UUID, enrollment_id int) error {
	tx := r.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Exec(`UPDATE public.enrollments SET status = ?, user_refer = ? WHERE id = ?`, "Сформирован", uuid, enrollment_id).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *Repository) FindGroup(group ds.Group) (ds.Group, error) {
	var result ds.Group
	err := r.db.Where(&group).Find(&result).Error
	if err != nil {
		return ds.Group{}, err
	} else {
		return result, nil
	}
}

func (r *Repository) FindEnrollment(enrollment *ds.Enrollment) (ds.Enrollment, error) {
	var result ds.Enrollment
	err := r.db.Where(&enrollment).Find(&result).Error
	if err != nil {
		return ds.Enrollment{}, err
	}

	var user ds.User
	r.db.Where("uuid = ?", result.UserRefer).Find(&user)

	result.User = user

	var moderator ds.User
	r.db.Where("uuid = ?", result.ModeratorRefer).Find(&user)

	result.Moderator = moderator

	return result, nil
}

func (r *Repository) EditGroup(group *ds.Group) error {
	return r.db.Model(&ds.Group{}).Where("title = ?", group.Title).Updates(group).Error
}

func (r *Repository) EditEnrollment(enrollment *ds.Enrollment) error {
	return r.db.Model(&ds.Enrollment{}).Where("id = ?", enrollment.ID).Updates(enrollment).Error
}

func (r *Repository) SetGroupImage(id int, image string) error {
	return r.db.Model(&ds.Group{}).Where("id = ?", id).Update("image_name", image).Error
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

	enrollment := ds.Enrollment{}
	enrollment.UserRefer = &userUUID
	enrollment.DateCreated = time.Now()
	enrollment.Status = requestBody.Status

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
	err := r.db.Where("id = ?", id).Find(&result).Error

	if err != nil {
		return "", err
	}

	return result.Status, nil
}

func (r *Repository) GetEnrollmentToGroupAvailability(id int) (string, error) {
	var result ds.EnrollmentToGroup
	err := r.db.Where("id = ?", id).Find(&result).Error
	if err != nil {
		return "", err
	}

	return result.Availability, nil
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

func (r *Repository) DeleteEnrollmentToGroup(enrollment_id int, group_id int) error {
	return r.db.Where("enrollment_refer = ?", enrollment_id).Where("group_refer = ?", group_id).Delete(&ds.EnrollmentToGroup{}).Error
}

func (r *Repository) ChangeEnrollmentToGroupAvailability(enrollment_to_group *ds.EnrollmentToGroup) error {
	return r.db.Model(&ds.EnrollmentToGroup{}).Where("id = ?", enrollment_to_group.ID).Updates(enrollment_to_group).Error
}

func (r *Repository) Register(user *ds.User) error {
	if user.UUID == uuid.Nil {
		user.UUID = uuid.New()
	}

	return r.db.Create(user).Error
}
