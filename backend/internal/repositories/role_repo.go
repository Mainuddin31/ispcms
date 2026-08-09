package repositories

import (
	"github.com/google/uuid"
	"github.com/ispcms/backend/internal/models"
	"gorm.io/gorm"
)

type RoleRepository interface {
	FindAll() ([]models.Role, error)
	FindByID(id uuid.UUID) (*models.Role, error)
	FindByName(name string) (*models.Role, error)
	Create(role *models.Role) error
	Update(role *models.Role) error
	Delete(id uuid.UUID) error
	FindAllPermissions() ([]models.Permission, error)
	AssignPermission(roleID, permID uuid.UUID) error
	RemovePermission(roleID, permID uuid.UUID) error
	GetUserPermissions(userID uuid.UUID) ([]models.Permission, error)
	// GetUserAccountPrefixes returns the merged account_prefixes from all roles
	// the user has. Super-admin returns nil (meaning unrestricted).
	GetUserAccountPrefixes(userID uuid.UUID) ([]string, bool, error)
	UpdateAccountPrefixes(roleID uuid.UUID, prefixes []string) error
}

type roleRepository struct{ db *gorm.DB }

func NewRoleRepository(db *gorm.DB) RoleRepository { return &roleRepository{db: db} }

func (r *roleRepository) FindAll() ([]models.Role, error) {
	var roles []models.Role
	err := r.db.Preload("Permissions").Find(&roles).Error
	return roles, err
}
func (r *roleRepository) FindByID(id uuid.UUID) (*models.Role, error) {
	var role models.Role
	err := r.db.Preload("Permissions").First(&role, "id = ?", id).Error
	return &role, err
}
func (r *roleRepository) FindByName(name string) (*models.Role, error) {
	var role models.Role
	err := r.db.First(&role, "name = ?", name).Error
	return &role, err
}
func (r *roleRepository) Create(role *models.Role) error { return r.db.Create(role).Error }
func (r *roleRepository) Update(role *models.Role) error { return r.db.Save(role).Error }
func (r *roleRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Role{}, "id = ?", id).Error
}
func (r *roleRepository) FindAllPermissions() ([]models.Permission, error) {
	var perms []models.Permission
	err := r.db.Find(&perms).Error
	return perms, err
}
func (r *roleRepository) AssignPermission(roleID, permID uuid.UUID) error {
	return r.db.FirstOrCreate(&models.RolePermission{}, models.RolePermission{RoleID: roleID, PermissionID: permID}).Error
}
func (r *roleRepository) RemovePermission(roleID, permID uuid.UUID) error {
	return r.db.Delete(&models.RolePermission{}, "role_id = ? AND permission_id = ?", roleID, permID).Error
}
func (r *roleRepository) GetUserPermissions(userID uuid.UUID) ([]models.Permission, error) {
	var perms []models.Permission
	err := r.db.Raw(`
		SELECT DISTINCT p.* FROM permissions p
		JOIN role_permissions rp ON rp.permission_id = p.id
		JOIN user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_id = ?`, userID).Scan(&perms).Error
	return perms, err
}

// GetUserAccountPrefixes collects account_prefixes from all roles the user belongs to.
// Returns (prefixes, isSuperAdmin, error). If isSuperAdmin is true, caller should skip
// prefix filtering entirely. Returns an empty slice (not nil) when no prefixes are set.
func (r *roleRepository) GetUserAccountPrefixes(userID uuid.UUID) ([]string, bool, error) {
	var roles []models.Role
	// Use Find (not Raw/Scan) so GORM properly invokes sql.Scanner on StringSlice fields.
	err := r.db.
		Joins("JOIN user_roles ur ON ur.role_id = roles.id").
		Where("ur.user_id = ? AND roles.deleted_at IS NULL", userID).
		Find(&roles).Error
	if err != nil {
		return nil, false, err
	}
	merged := []string{}
	for _, role := range roles {
		if role.Name == "super_admin" || role.Name == "admin" {
			return nil, true, nil // unrestricted
		}
		for _, p := range role.AccountPrefixes {
			if p != "" {
				merged = append(merged, p)
			}
		}
	}
	return merged, false, nil
}

func (r *roleRepository) UpdateAccountPrefixes(roleID uuid.UUID, prefixes []string) error {
	sp := models.StringSlice(prefixes)
	v, err := sp.Value()
	if err != nil {
		return err
	}
	return r.db.Exec(`UPDATE roles SET account_prefixes = ?, updated_at = NOW() WHERE id = ?`, v, roleID).Error
}
