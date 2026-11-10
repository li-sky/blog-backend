package router

import (
	"blog-backend/models"
	"blog-backend/services"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/require"
)

type noopUserRepo struct{}

func (noopUserRepo) Create(user *models.User) error                       { return nil }
func (noopUserRepo) GetByID(id uint) (*models.User, error)                { return &models.User{}, nil }
func (noopUserRepo) GetByUsername(username string) (*models.User, error)  { return &models.User{}, nil }
func (noopUserRepo) GetByEmail(email string) (*models.User, error)        { return &models.User{}, nil }
func (noopUserRepo) List(offset, limit int) ([]models.User, int64, error) { return nil, 0, nil }
func (noopUserRepo) Update(user *models.User) error                       { return nil }
func (noopUserRepo) Delete(id uint) error                                 { return nil }
func (noopUserRepo) HasUserWithRole(roleName string) (bool, error)        { return false, nil }

type noopRoleRepo struct{}

func (noopRoleRepo) EnsureRole(name string) (*models.Role, error) {
	return &models.Role{Name: name}, nil
}
func (noopRoleRepo) EnsurePermission(name, description string) (*models.Permission, error) {
	return &models.Permission{Name: name, Description: description}, nil
}
func (noopRoleRepo) ReplacePermissions(role *models.Role, permissions []models.Permission) error {
	role.Permissions = permissions
	return nil
}
func (noopRoleRepo) List(offset, limit int) ([]models.Role, int64, error) { return nil, 0, nil }
func (noopRoleRepo) GetByID(id uint) (*models.Role, error)                { return &models.Role{}, nil }
func (noopRoleRepo) GetByName(name string) (*models.Role, error)          { return &models.Role{}, nil }
func (noopRoleRepo) GetByIDs(ids []uint) ([]models.Role, error)           { return nil, nil }
func (noopRoleRepo) Create(role *models.Role) error                       { return nil }
func (noopRoleRepo) Update(role *models.Role) error                       { return nil }
func (noopRoleRepo) Delete(id uint) error                                 { return nil }

type noopPostRepo struct{}

func (noopPostRepo) Create(post *models.Post) error        { return nil }
func (noopPostRepo) GetByID(id uint) (*models.Post, error) { return &models.Post{}, nil }
func (noopPostRepo) GetByIDAndStatus(id uint, status string) (*models.Post, error) {
	return &models.Post{}, nil
}
func (noopPostRepo) List(offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	return nil, 0, nil
}
func (noopPostRepo) ListByStatus(status string, offset, limit int, sortBy string, desc bool) ([]models.Post, int64, error) {
	return nil, 0, nil
}
func (noopPostRepo) Update(post *models.Post) error { return nil }
func (noopPostRepo) Delete(id uint) error           { return nil }

type noopCommentRepo struct{}

func (noopCommentRepo) Create(comment *models.Comment) error { return nil }
func (noopCommentRepo) ListByPost(postID uint, offset, limit int) ([]models.Comment, int64, error) {
	return nil, 0, nil
}
func (noopCommentRepo) GetByID(id uint) (*models.Comment, error) { return &models.Comment{}, nil }
func (noopCommentRepo) Delete(id uint) error                     { return nil }
func (noopCommentRepo) Update(comment *models.Comment) error     { return nil }

type noopSettingRepo struct{}

func (noopSettingRepo) List() ([]models.Setting, error) { return nil, nil }
func (noopSettingRepo) GetByKey(key string) (*models.Setting, error) {
	return &models.Setting{Key: key, Value: ""}, nil
}
func (noopSettingRepo) Upsert(key, value, description string) (*models.Setting, error) {
	return &models.Setting{Key: key, Value: value, Desc: description}, nil
}

type noopImageRepo struct{}

func (noopImageRepo) Create(image *models.Image) error                      { return nil }
func (noopImageRepo) GetByID(id uint) (*models.Image, error)                { return &models.Image{}, nil }
func (noopImageRepo) List(offset, limit int) ([]models.Image, int64, error) { return nil, 0, nil }
func (noopImageRepo) Update(image *models.Image) error                      { return nil }
func (noopImageRepo) Delete(id uint) error                                  { return nil }
func (noopImageRepo) GetByFilename(filename string) (*models.Image, error) {
	return &models.Image{}, nil
}

type noopStorageBackend struct{}

func (noopStorageBackend) Save(file *multipart.FileHeader) (string, string, error) {
	return "", "", nil
}
func (noopStorageBackend) Delete(filename string) error                             { return nil }
func (noopStorageBackend) GetImageDimensions(file multipart.File) (int, int, error) { return 0, 0, nil }

func TestSetupRouter_BuildsEngine(t *testing.T) {
	userSvc := services.NewUserService(noopUserRepo{}, noopRoleRepo{})
	roleSvc := services.NewRoleService(noopRoleRepo{})
	postSvc := services.NewPostService(noopPostRepo{})
	commentSvc := services.NewCommentService(noopCommentRepo{}, noopPostRepo{})
	settingSvc := services.NewSettingService(noopSettingRepo{})
	imageSvc := services.NewImageService(noopImageRepo{}, noopStorageBackend{})

	engine := SetupRouter(userSvc, roleSvc, postSvc, commentSvc, settingSvc, imageSvc, nil)
	require.NotNil(t, engine)
}
