package storage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStorageFromEnv_Filesystem(t *testing.T) {
	// 保存原始环境变量
	oldBackend := os.Getenv("STORAGE_BACKEND")
	oldBasePath := os.Getenv("STORAGE_FS_BASE_PATH")
	oldBaseURL := os.Getenv("STORAGE_FS_BASE_URL")
	defer func() {
		os.Setenv("STORAGE_BACKEND", oldBackend)
		os.Setenv("STORAGE_FS_BASE_PATH", oldBasePath)
		os.Setenv("STORAGE_FS_BASE_URL", oldBaseURL)
	}()

	t.Run("uses default filesystem backend", func(t *testing.T) {
		os.Unsetenv("STORAGE_BACKEND")
		tmpDir := t.TempDir()
		os.Setenv("STORAGE_FS_BASE_PATH", tmpDir)
		os.Setenv("STORAGE_FS_BASE_URL", "http://localhost/test")

		backend, err := NewStorageFromEnv()

		require.NoError(t, err)
		assert.NotNil(t, backend)

		fs, ok := backend.(*FilesystemStorage)
		require.True(t, ok)
		assert.Equal(t, tmpDir, fs.BasePath)
		assert.Equal(t, "http://localhost/test", fs.BaseURL)
	})

	t.Run("uses explicit filesystem backend", func(t *testing.T) {
		os.Setenv("STORAGE_BACKEND", "filesystem")
		tmpDir := t.TempDir()
		os.Setenv("STORAGE_FS_BASE_PATH", tmpDir)
		os.Setenv("STORAGE_FS_BASE_URL", "http://example.com/uploads")

		backend, err := NewStorageFromEnv()

		require.NoError(t, err)
		assert.NotNil(t, backend)

		fs, ok := backend.(*FilesystemStorage)
		require.True(t, ok)
		assert.Equal(t, tmpDir, fs.BasePath)
		assert.Equal(t, "http://example.com/uploads", fs.BaseURL)
	})

	t.Run("returns error for unsupported backend", func(t *testing.T) {
		os.Setenv("STORAGE_BACKEND", "unsupported")

		backend, err := NewStorageFromEnv()

		assert.Error(t, err)
		assert.Nil(t, backend)
		assert.Contains(t, err.Error(), "unsupported storage backend")
	})

	t.Run("uses default values when env vars not set", func(t *testing.T) {
		os.Setenv("STORAGE_BACKEND", "filesystem")
		os.Unsetenv("STORAGE_FS_BASE_PATH")
		os.Unsetenv("STORAGE_FS_BASE_URL")

		backend, err := NewStorageFromEnv()

		require.NoError(t, err)
		fs, ok := backend.(*FilesystemStorage)
		require.True(t, ok)
		assert.Equal(t, "./uploads", fs.BasePath)
		assert.Equal(t, "http://localhost:8080/uploads", fs.BaseURL)
	})
}

func TestGetEnv(t *testing.T) {
	t.Run("returns env value when set", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR", "test_value")
		defer os.Unsetenv("TEST_ENV_VAR")

		value := getEnv("TEST_ENV_VAR", "default")

		assert.Equal(t, "test_value", value)
	})

	t.Run("returns default when env not set", func(t *testing.T) {
		os.Unsetenv("TEST_ENV_VAR_NOT_SET")

		value := getEnv("TEST_ENV_VAR_NOT_SET", "default_value")

		assert.Equal(t, "default_value", value)
	})

	t.Run("returns default when env is empty string", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR_EMPTY", "")
		defer os.Unsetenv("TEST_ENV_VAR_EMPTY")

		value := getEnv("TEST_ENV_VAR_EMPTY", "default")

		assert.Equal(t, "default", value)
	})
}
