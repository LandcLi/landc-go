package codemod

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateDBContext(t *testing.T) {
	dir := t.TempDir()

	daoFile := `package dao

type UserDao interface {
	Create(user *User) error
	Count() (int64, error)
}

func NewUserDao() *UserDao { return nil }
`
	if err := os.WriteFile(filepath.Join(dir, "user_dao.go"), []byte(daoFile), 0o644); err != nil {
		t.Fatal(err)
	}

	implFile := `package daoimpl

import (
	"context"

	"github.com/example/db"
)

type userDao struct{}

func NewUserDaoImpl() *userDao { return &userDao{} }

func (d *userDao) Create(user *User) error {
	err := db.GetDB().Create(user).Error
	_ = context.Background()
	return err
}

func (d *userDao) Count() (int64, error) {
	return db.GetDB().Model(&User{}).Count(), nil
}
`
	if err := os.WriteFile(filepath.Join(dir, "user_dao_impl.go"), []byte(implFile), 0o644); err != nil {
		t.Fatal(err)
	}

	modified, err := MigrateDBContext(dir)
	if err != nil {
		t.Fatalf("MigrateDBContext: %v", err)
	}
	if len(modified) != 2 {
		t.Fatalf("modified = %v, want 2 files", modified)
	}

	daoOut, _ := os.ReadFile(filepath.Join(dir, "user_dao.go"))
	daoStr := string(daoOut)
	if !strings.Contains(daoStr, "Create(ctx context.Context, user *User) error") {
		t.Errorf("interface method Create missing ctx:\n%s", daoStr)
	}
	if !strings.Contains(daoStr, "Count(ctx context.Context)") {
		t.Errorf("interface method Count missing ctx:\n%s", daoStr)
	}
	if strings.Contains(daoStr, "NewUserDao(ctx") {
		t.Error("constructor NewUserDao should not get ctx param")
	}

	implOut, _ := os.ReadFile(filepath.Join(dir, "user_dao_impl.go"))
	implStr := string(implOut)
	if !strings.Contains(implStr, "func (d *userDao) Create(ctx context.Context, user *User) error") {
		t.Errorf("impl method Create missing ctx:\n%s", implStr)
	}
	if !strings.Contains(implStr, "db.GetDBFrom(ctx).Create(user)") {
		t.Errorf("GetDB() not migrated:\n%s", implStr)
	}
	if strings.Contains(implStr, "context.Background()") {
		t.Errorf("context.Background() not migrated:\n%s", implStr)
	}
	if strings.Contains(implStr, "NewUserDaoImpl(ctx") {
		t.Error("constructor NewUserDaoImpl should not get ctx param")
	}
}

func TestMigrateDBContextSkipsConstructor(t *testing.T) {
	dir := t.TempDir()
	src := `package svc

import "github.com/example/db"

func NewService() *Service { return &Service{db: db.GetDB()} }

func (s *Service) Ping() error {
	return s.db.Exec("SELECT 1").Error
}
`
	if err := os.WriteFile(filepath.Join(dir, "service.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	// 构造函数内 GetDB() 不应被替换（构造函数无 ctx）
	if _, err := MigrateDBContext(dir); err != nil {
		t.Fatalf("MigrateDBContext: %v", err)
	}
	out, _ := os.ReadFile(filepath.Join(dir, "service.go"))
	s := string(out)
	if !strings.Contains(s, "NewService() *Service") {
		t.Errorf("constructor signature should be unchanged:\n%s", s)
	}
	if strings.Contains(s, "GetDBFrom(ctx)") {
		t.Errorf("constructor body GetDB should NOT be migrated (no ctx there):\n%s", s)
	}
	if !strings.Contains(s, "Ping(ctx context.Context)") {
		t.Errorf("method Ping should get ctx:\n%s", s)
	}
}
