package storage

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFilesystemStorage(t *testing.T) {
	t.Run("creates storage directory if not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		uploadDir := filepath.Join(tmpDir, "uploads")

		fs, err := NewFilesystemStorage(uploadDir, "http://localhost:8080/uploads")

		require.NoError(t, err)
		assert.NotNil(t, fs)
		assert.Equal(t, uploadDir, fs.BasePath)
		assert.Equal(t, "http://localhost:8080/uploads", fs.BaseURL)

		// 验证目录已创建
		info, err := os.Stat(uploadDir)
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("removes trailing slash from baseURL", func(t *testing.T) {
		tmpDir := t.TempDir()

		fs, err := NewFilesystemStorage(tmpDir, "http://localhost:8080/uploads/")

		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080/uploads", fs.BaseURL)
	})
}

func TestFilesystemStorage_Save(t *testing.T) {
	t.Run("saves file successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		fs, err := NewFilesystemStorage(tmpDir, "http://localhost:8080/uploads")
		require.NoError(t, err)

		// 创建测试文件
		fileHeader := createTestFileHeader(t, "test.png", "image/png", []byte("fake image data"))

		filename, url, err := fs.Save(fileHeader)

		require.NoError(t, err)
		assert.NotEmpty(t, filename)
		assert.True(t, strings.HasSuffix(filename, ".png"))
		assert.Equal(t, "http://localhost:8080/uploads/"+filename, url)

		// 验证文件已保存
		savedPath := filepath.Join(tmpDir, filename)
		data, err := os.ReadFile(savedPath)
		require.NoError(t, err)
		assert.Equal(t, []byte("fake image data"), data)
	})

	t.Run("generates unique filenames", func(t *testing.T) {
		tmpDir := t.TempDir()
		fs, err := NewFilesystemStorage(tmpDir, "http://localhost:8080/uploads")
		require.NoError(t, err)

		fileHeader1 := createTestFileHeader(t, "test.png", "image/png", []byte("data1"))
		fileHeader2 := createTestFileHeader(t, "test.png", "image/png", []byte("data2"))

		filename1, _, err := fs.Save(fileHeader1)
		require.NoError(t, err)

		filename2, _, err := fs.Save(fileHeader2)
		require.NoError(t, err)

		assert.NotEqual(t, filename1, filename2)
	})
}

func TestFilesystemStorage_Delete(t *testing.T) {
	t.Run("deletes existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		fs, err := NewFilesystemStorage(tmpDir, "http://localhost:8080/uploads")
		require.NoError(t, err)

		// 创建并保存文件
		fileHeader := createTestFileHeader(t, "test.png", "image/png", []byte("test data"))
		filename, _, err := fs.Save(fileHeader)
		require.NoError(t, err)

		// 删除文件
		err = fs.Delete(filename)
		require.NoError(t, err)

		// 验证文件已删除
		_, err = os.Stat(filepath.Join(tmpDir, filename))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("does not error on non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		fs, err := NewFilesystemStorage(tmpDir, "http://localhost:8080/uploads")
		require.NoError(t, err)

		err = fs.Delete("nonexistent.png")
		assert.NoError(t, err)
	})
}

func TestFilesystemStorage_GetImageDimensions(t *testing.T) {
	t.Run("gets dimensions for PNG image", func(t *testing.T) {
		tmpDir := t.TempDir()
		fs, err := NewFilesystemStorage(tmpDir, "http://localhost:8080/uploads")
		require.NoError(t, err)

		// 创建真实的 PNG 图片
		img := createTestImage(100, 50)
		buf := new(bytes.Buffer)
		err = png.Encode(buf, img)
		require.NoError(t, err)

		// 创建 multipart.File
		file := &mockMultipartFile{
			Reader: bytes.NewReader(buf.Bytes()),
		}

		width, height, err := fs.GetImageDimensions(file)

		require.NoError(t, err)
		assert.Equal(t, 100, width)
		assert.Equal(t, 50, height)
	})

	t.Run("resets file pointer after reading", func(t *testing.T) {
		tmpDir := t.TempDir()
		fs, err := NewFilesystemStorage(tmpDir, "http://localhost:8080/uploads")
		require.NoError(t, err)

		img := createTestImage(100, 50)
		buf := new(bytes.Buffer)
		err = png.Encode(buf, img)
		require.NoError(t, err)

		file := &mockMultipartFile{
			Reader: bytes.NewReader(buf.Bytes()),
		}

		_, _, err = fs.GetImageDimensions(file)
		require.NoError(t, err)

		// 验证文件指针已重置
		pos, err := file.Seek(0, io.SeekCurrent)
		require.NoError(t, err)
		assert.Equal(t, int64(0), pos)
	})
}

// Helper functions

func createTestFileHeader(t *testing.T, filename, contentType string, data []byte) *multipart.FileHeader {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)

	part, err := writer.CreatePart(h)
	require.NoError(t, err)

	_, err = part.Write(data)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(10 << 20) // 10MB
	require.NoError(t, err)

	files := form.File["file"]
	require.Len(t, files, 1)

	return files[0]
}

func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	return img
}

type mockMultipartFile struct {
	*bytes.Reader
}

func (m *mockMultipartFile) Close() error {
	return nil
}
