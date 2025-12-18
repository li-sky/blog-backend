package services

import (
	"blog-backend/models"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func permUser(id uint, perms ...string) *models.User {
	ps := make([]models.Permission, 0, len(perms))
	for _, p := range perms {
		ps = append(ps, models.Permission{Name: p})
	}
	return &models.User{Model: gorm.Model{ID: id}, Roles: []models.Role{{Permissions: ps}}}
}

func TestUserService_ListUsers_PermissionDeniedAndAllowed(t *testing.T) {
	allowed := permUser(1, "assign_roles")
	denied := permUser(2)

	svc := NewUserService(userRepoStub{
		list: func(offset, limit int) ([]models.User, int64, error) {
			return []models.User{{Model: gorm.Model{ID: 3}, Username: "x"}}, 1, nil
		},
	}, roleRepoStub{})

	_, _, err := svc.ListUsers(denied, 0, 10)
	require.Error(t, err)

	items, total, err := svc.ListUsers(allowed, 0, 10)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
}

func TestUserService_GetUser_ValidationAndPermissions(t *testing.T) {
	svc := NewUserService(userRepoStub{
		getByID: func(id uint) (*models.User, error) {
			return &models.User{Model: gorm.Model{ID: id}, Username: "u"}, nil
		},
	}, roleRepoStub{})

	_, err := svc.GetUser(permUser(1), 0)
	require.Error(t, err)

	_, err = svc.GetUser(permUser(1), 2)
	require.Error(t, err)

	self, err := svc.GetUser(permUser(2), 2)
	require.NoError(t, err)
	require.Equal(t, uint(2), self.ID)

	admin, err := svc.GetUser(permUser(1, "edit_user"), 2)
	require.NoError(t, err)
	require.Equal(t, uint(2), admin.ID)
}

func TestUserService_UpdateUser_Success(t *testing.T) {
	operator := permUser(1, "edit_user")

	stored := &models.User{Model: gorm.Model{ID: 2}, Username: "old", Email: "old@example.com", Password: "p"}
	svc := NewUserService(userRepoStub{
		getByID: func(id uint) (*models.User, error) { return stored, nil },
		update: func(user *models.User) error {
			require.Equal(t, "new", user.Username)
			require.Equal(t, "new@example.com", user.Email)
			require.Equal(t, "newpass", user.Password)
			return nil
		},
	}, roleRepoStub{})

	err := svc.UpdateUser(operator, &models.User{Model: gorm.Model{ID: 2}, Username: "new", Email: "new@example.com", Password: "newpass"})
	require.NoError(t, err)
}

func TestUserService_SetUserRoles_MismatchAndSuccess(t *testing.T) {
	operator := permUser(1, "assign_roles")

	svcMismatch := NewUserService(userRepoStub{
		getByID: func(id uint) (*models.User, error) { return &models.User{Model: gorm.Model{ID: id}}, nil },
	}, roleRepoStub{
		getByIDs: func(ids []uint) ([]models.Role, error) { return []models.Role{{Model: gorm.Model{ID: 1}}}, nil },
	})

	_, err := svcMismatch.SetUserRoles(operator, 2, []uint{1, 2})
	require.Error(t, err)

	svcOK := NewUserService(userRepoStub{
		getByID: func(id uint) (*models.User, error) { return &models.User{Model: gorm.Model{ID: id}}, nil },
		update: func(user *models.User) error {
			require.Len(t, user.Roles, 2)
			return nil
		},
	}, roleRepoStub{
		getByIDs: func(ids []uint) ([]models.Role, error) {
			return []models.Role{{Model: gorm.Model{ID: 1}}, {Model: gorm.Model{ID: 2}}}, nil
		},
	})

	user, err := svcOK.SetUserRoles(operator, 2, []uint{1, 2})
	require.NoError(t, err)
	require.Len(t, user.Roles, 2)
}

func TestUserService_CreateAdminUser_InitialAndNonInitial(t *testing.T) {
	// Initial: admin exists
	svcInit := NewUserService(userRepoStub{
		hasUserWithRole: func(roleName string) (bool, error) { return true, nil },
		create:         func(user *models.User) error { return nil },
	}, roleRepoStub{})

	err := svcInit.CreateAdminUser(nil, &models.User{Username: "a", Email: "a@e.com", Password: "veryverysecure"}, true)
	require.Error(t, err)

	// Non-initial: permission denied
	svcNonInit := NewUserService(userRepoStub{}, roleRepoStub{})
	err = svcNonInit.CreateAdminUser(permUser(1), &models.User{Username: "a", Email: "a@e.com", Password: "veryverysecure"}, false)
	require.Error(t, err)

	// Non-initial: allowed
	svcAllowed := NewUserService(userRepoStub{create: func(user *models.User) error { return nil }}, roleRepoStub{})
	err = svcAllowed.CreateAdminUser(permUser(1, "create_admin_user"), &models.User{Username: "a", Email: "a@e.com", Password: "veryverysecure"}, false)
	require.NoError(t, err)
}

func TestUserService_EnsureAdminRole_Error(t *testing.T) {
	svc := NewUserService(userRepoStub{}, roleRepoStub{ensureRole: func(name string) (*models.Role, error) {
		return nil, errors.New("boom")
	}})

	_, err := svc.ensureAdminRole()
	require.Error(t, err)
}

func TestUserService_ListUserRoles_GetByID(t *testing.T) {
	svc := NewUserService(userRepoStub{
		getByID: func(id uint) (*models.User, error) {
			return &models.User{Model: gorm.Model{ID: id}, Username: "u"}, nil
		},
	}, roleRepoStub{})

	roles, err := svc.ListUserRoles(&models.User{Roles: []models.Role{{Name: "r"}}})
	require.NoError(t, err)
	require.Len(t, roles, 1)

	_, err = svc.GetByID(0)
	require.Error(t, err)

	user, err := svc.GetByID(9)
	require.NoError(t, err)
	require.Equal(t, uint(9), user.ID)
}

func TestUserService_DeleteUser_PermissionDenied(t *testing.T) {
	svc := NewUserService(userRepoStub{}, roleRepoStub{})
	err := svc.DeleteUser(permUser(1), 2)
	require.Error(t, err)
}

func TestUserService_Register_EmailTaken(t *testing.T) {
	svc := NewUserService(userRepoStub{
		getByEmail: func(email string) (*models.User, error) { return &models.User{Model: gorm.Model{ID: 1}}, nil },
	}, roleRepoStub{})

	err := svc.Register(&models.User{Username: "alice", Email: "a@example.com", Password: "veryverysecure"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "email")
}

func TestUserService_Register_NilUser(t *testing.T) {
	svc := NewUserService(userRepoStub{}, roleRepoStub{})
	err := svc.Register(nil)
	require.Error(t, err)
}
