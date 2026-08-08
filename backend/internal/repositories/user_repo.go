package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

type UserRepository interface {
	FindByID(id uuid.UUID) (*models.User, error)
	FindByUsername(username string) (*models.User, error)
	FindByEmail(email string) (*models.User, error)
	FindAll(page, pageSize int, search string) ([]models.User, int64, error)
	Create(user *models.User) error
	Update(user *models.User) error
	Delete(id uuid.UUID) error
	UpdateLastLogin(id uuid.UUID) error
	IncrementFailedAttempts(id uuid.UUID) error
	LockUser(id uuid.UUID, minutes int) error
	ResetFailedAttempts(id uuid.UUID) error
	AssignRole(userID, roleID uuid.UUID) error
	RemoveRole(userID, roleID uuid.UUID) error
}

type userRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) UserRepository { return &userRepository{db: db} }

func (r *userRepository) FindByID(id uuid.UUID) (*models.User, error) {
	var u models.User
	err := r.db.Preload("Roles.Permissions").First(&u, "id = ?", id).Error
	return &u, err
}

func (r *userRepository) FindByUsername(username string) (*models.User, error) {
	var u models.User
	err := r.db.Preload("Roles").First(&u, "username = ?", username).Error
	return &u, err
}

func (r *userRepository) FindByEmail(email string) (*models.User, error) {
	var u models.User
	err := r.db.Preload("Roles").First(&u, "email = ?", email).Error
	return &u, err
}

func (r *userRepository) FindAll(page, pageSize int, search string) ([]models.User, int64, error) {
	var users []models.User
	var total int64
	q := r.db.Model(&models.User{}).Preload("Roles")
	if search != "" {
		q = q.Where("full_name ILIKE ? OR username ILIKE ? OR email ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	q.Count(&total)
	err := q.Offset((page - 1) * pageSize).Limit(pageSize).Find(&users).Error
	return users, total, err
}

func (r *userRepository) Create(u *models.User) error { return r.db.Create(u).Error }
func (r *userRepository) Update(u *models.User) error { return r.db.Save(u).Error }
func (r *userRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.User{}, "id = ?", id).Error
}
func (r *userRepository) UpdateLastLogin(id uuid.UUID) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("last_login", gorm.Expr("NOW()")).Error
}
func (r *userRepository) IncrementFailedAttempts(id uuid.UUID) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Update("failed_attempts", gorm.Expr("failed_attempts + 1")).Error
}
func (r *userRepository) LockUser(id uuid.UUID, minutes int) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"locked_until": gorm.Expr(fmt.Sprintf("NOW() + INTERVAL '%d minutes'", minutes)),
	}).Error
}
func (r *userRepository) ResetFailedAttempts(id uuid.UUID) error {
	return r.db.Model(&models.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"failed_attempts": 0,
		"locked_until":    nil,
	}).Error
}
func (r *userRepository) AssignRole(userID, roleID uuid.UUID) error {
	return r.db.FirstOrCreate(&models.UserRole{}, models.UserRole{UserID: userID, RoleID: roleID}).Error
}
func (r *userRepository) RemoveRole(userID, roleID uuid.UUID) error {
	return r.db.Delete(&models.UserRole{}, "user_id = ? AND role_id = ?", userID, roleID).Error
}
