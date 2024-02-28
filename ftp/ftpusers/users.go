package ftpusers

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"sync"
)

type User struct {
	Username   string
	Password   string
	CustomerID int64
	IPs        map[string]*netip.Prefix
}

func UniqSlice[T comparable](s []T) []T {
	m := make(map[T]struct{})
	for _, v := range s {
		m[v] = struct{}{}
	}
	var result []T
	for k := range m {
		result = append(result, k)
	}
	return result
}

// FindIP finds an IP in the prefixes in the user
func (u *User) FindIP(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	for _, v := range u.IPs {
		if v.Contains(addr) {
			return true
		}
	}
	return false
}

// AddIP adds an IP prefix to the user
// if the ip is without the prefix, it will add /32
func (u *User) AddIP(ip string) error {
	if !strings.Contains(ip, "/") {
		ip = ip + "/32"
	}

	prefix, err := netip.ParsePrefix(ip)
	if err != nil {
		return fmt.Errorf("error parsing IP: %w", err)
	}

	u.IPs[ip] = &prefix
	return nil
}

// RemoveIP removes an IP prefix from the user
// if the ip is without the prefix, it will add /32
func (u *User) RemoveIP(ip string) {
	if !strings.Contains(ip, "/") {
		ip = ip + "/32"
	}

	delete(u.IPs, ip)
}

type Users interface {
	List() (map[string]*User, error)
	// Get finds a user by username
	// if the user is not found, don't it returns an error just a nil user
	Get(id string) (*User, error)
}

var localUserMaxID int64 = 0
var _ Users = &LocalUsers{}

type LocalUsers struct {
	users map[string]*User
	wg    sync.RWMutex
}

func (u *LocalUsers) List() (map[string]*User, error) {
	u.wg.RLock()
	defer u.wg.RUnlock()
	return u.users, nil
}

func (u *LocalUsers) Get(username string) (*User, error) {
	u.wg.RLock()
	defer u.wg.RUnlock()
	user, ok := u.users[username]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (u *LocalUsers) Add(user, pass string, customerID int64) *User {
	u.wg.Lock()
	defer u.wg.Unlock()

	newUser := &User{
		Username:   user,
		Password:   pass,
		CustomerID: customerID,
		IPs:        make(map[string]*netip.Prefix),
	}

	u.users[newUser.Username] = newUser
	return newUser
}

func (u *LocalUsers) Remove(user string) *User {
	u.wg.Lock()
	defer u.wg.Unlock()
	oldUser := u.users[user]
	delete(u.users, user)
	return oldUser
}

func NewLocalUsers() *LocalUsers {
	return &LocalUsers{
		users: make(map[string]*User),
	}
}
