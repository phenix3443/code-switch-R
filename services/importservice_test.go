package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCcSwitchImportPathSupportsDirectory(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "cc-switch.db")
	if err := os.WriteFile(dbPath, []byte("SQLite format 3\x00test"), 0644); err != nil {
		t.Fatalf("写入测试数据库失败: %v", err)
	}

	got, err := resolveCcSwitchImportPath(dir)
	if err != nil {
		t.Fatalf("resolveCcSwitchImportPath() 返回错误: %v", err)
	}
	if got != dbPath {
		t.Fatalf("期望解析到 %s，得到 %s", dbPath, got)
	}
}

func TestResolveCcSwitchImportPathExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dbPath := filepath.Join(home, ".cc-switch", "cc-switch.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	if err := os.WriteFile(dbPath, []byte("SQLite format 3\x00test"), 0644); err != nil {
		t.Fatalf("写入测试数据库失败: %v", err)
	}

	got, err := resolveCcSwitchImportPath("~/.cc-switch")
	if err != nil {
		t.Fatalf("resolveCcSwitchImportPath() 返回错误: %v", err)
	}
	if got != dbPath {
		t.Fatalf("期望解析到 %s，得到 %s", dbPath, got)
	}
}

func TestCcSwitchConfigPathFallsBackToDefaultDatabasePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := ccSwitchConfigPath()
	if err != nil {
		t.Fatalf("ccSwitchConfigPath() 返回错误: %v", err)
	}

	want := filepath.Join(home, ".cc-switch", "cc-switch.db")
	if got != want {
		t.Fatalf("期望默认路径为 %s，得到 %s", want, got)
	}
}
