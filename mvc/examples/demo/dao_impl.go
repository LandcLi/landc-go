package demo

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"strings"
	"time"
)

// MemoryUserDAO 内存实现的用户 DAO（仅用于演示）
type MemoryUserDAO struct {
	mu      sync.RWMutex
	users   map[uint]*User
	nextID  atomic.Uint64
}

func NewMemoryUserDAO() *MemoryUserDAO {
	dao := &MemoryUserDAO{
		users: make(map[uint]*User),
	}
	// 预置一个管理员账号
	dao.nextID.Store(1)
	dao.users[1] = &User{
		ID:        1,
		Username:  "admin",
		Password:  "admin123",
		Email:     "admin@example.com",
		Role:      "admin",
		CreatedAt: time.Now().Unix(),
	}
	return dao
}

func (d *MemoryUserDAO) Create(ctx context.Context, user *User) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 检查用户名唯一性
	for _, u := range d.users {
		if u.Username == user.Username {
			return fmt.Errorf("username '%s' already exists", user.Username)
		}
	}

	id := uint(d.nextID.Add(1))
	user.ID = id
	user.CreatedAt = time.Now().Unix()
	d.users[id] = user
	return nil
}

func (d *MemoryUserDAO) GetByID(ctx context.Context, id uint) (*User, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	user, ok := d.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found: %d", id)
	}
	return user, nil
}

func (d *MemoryUserDAO) GetByUsername(ctx context.Context, username string) (*User, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, u := range d.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user not found: %s", username)
}

func (d *MemoryUserDAO) Update(ctx context.Context, user *User) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, ok := d.users[user.ID]; !ok {
		return fmt.Errorf("user not found: %d", user.ID)
	}
	d.users[user.ID] = user
	return nil
}

func (d *MemoryUserDAO) Delete(ctx context.Context, id uint) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, ok := d.users[id]; !ok {
		return fmt.Errorf("user not found: %d", id)
	}
	delete(d.users, id)
	return nil
}

func (d *MemoryUserDAO) List(ctx context.Context, offset, limit int, keyword string) ([]*User, int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var filtered []*User
	for _, u := range d.users {
		if keyword != "" {
			if !strings.Contains(u.Username, keyword) && !strings.Contains(u.Email, keyword) {
				continue
			}
		}
		filtered = append(filtered, u)
	}

	total := int64(len(filtered))

	// 分页
	if offset >= len(filtered) {
		return []*User{}, total, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[offset:end], total, nil
}
