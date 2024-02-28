package ftpusers

import (
	"errors"
	"fmt"
	"github.com/telebroad/ftpserver/ftp"
	"net/netip"
	"strings"
	"sync"
)

type User struct {
	Username string
	Password string
	IPs      map[string]*netip.Prefix
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
func (u *User) AddIP(ip string) (err error) {

	var prefix netip.Prefix
	if !strings.Contains(ip, "/") {

		addr, err := netip.ParseAddr(ip)
		if err != nil {
			return fmt.Errorf("error parsing IP: %w", err)
		}
		if addr.Is4() {
			prefix, _ = addr.Prefix(32)
		} else {
			prefix, _ = addr.Prefix(128)
		}
	} else {
		prefix, err = netip.ParsePrefix(ip)
		if err != nil {
			return fmt.Errorf("error parsing IP: %w", err)
		}
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

var localUserMaxID int64 = 0
var _ ftp.Users = &LocalUsers{}

type LocalUsers struct {
	users map[string]*User
	wg    sync.RWMutex
}

// List returns all users
func (u *LocalUsers) List() (map[string]*User, error) {
	u.wg.RLock()
	defer u.wg.RUnlock()
	return u.users, nil
}

// Get returns a user by username, if the user is not found it returns an error
func (u *LocalUsers) Get(username string) (*User, error) {
	u.wg.RLock()
	defer u.wg.RUnlock()
	user, ok := u.users[username]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}

// Find returns a user by username and password, if the user is not found it returns an error
func (u *LocalUsers) Find(username, password, ipaddr string) (any, error) {
	userInfo, err := u.Get(username)
	if err != nil {
		return nil, err
	}
	if userInfo.Password != password {
		return nil, fmt.Errorf("password is incorrect")
	}
	if !userInfo.FindIP(ipaddr) {
		return nil, fmt.Errorf("ip origin %s is not allowed", ipaddr)
	}
	return userInfo, nil
}

// Add adds a new user
func (u *LocalUsers) Add(user, pass string, customerID int64) *User {
	u.wg.Lock()
	defer u.wg.Unlock()

	newUser := &User{
		Username: user,
		Password: pass,
		IPs:      make(map[string]*netip.Prefix),
	}

	u.users[newUser.Username] = newUser
	return newUser
}

// Remove removes a user
func (u *LocalUsers) Remove(user string) *User {
	u.wg.Lock()
	defer u.wg.Unlock()
	oldUser := u.users[user]
	delete(u.users, user)
	return oldUser
}

// NewLocalUsers creates a new LocalUsers
func NewLocalUsers() *LocalUsers {
	return &LocalUsers{
		users: make(map[string]*User),
	}
}
