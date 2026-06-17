package testhelpers

import (
	"os"
	"path/filepath"
	"testing"
)

// SetIsolatedHome 将 HOME/USERPROFILE 指向临时目录，避免污染真实用户数据。
func SetIsolatedHome(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

// MustWriteFile 写入测试文件并自动创建父目录。
func MustWriteFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("创建父目录失败 %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入测试文件失败 %s: %v", path, err)
	}
}

