package dexclient

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type DexUser interface {
	GetID() string
	GetEmail() string
	GetPassword() []byte
	GetUsername() string

	SetPassword(password string)
	SetRawPassword(password []byte)
}

type dexUser struct {
	id       string
	username string
	email    string
	password []byte
}

func (d *dexUser) GetID() string {
	return d.id
}

func (d *dexUser) GetEmail() string {
	return d.email
}

func (d *dexUser) GetPassword() []byte {
	return d.password
}

func (d *dexUser) GetUsername() string {
	return d.username
}

func (d *dexUser) SetPassword(password string) {
	pwdBytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		panic(err)
	}

	d.password = pwdBytes
}

func (d *dexUser) SetRawPassword(password []byte) {
	d.password = password
}

func (d *dexUser) String() string {
	builder := strings.Builder{}

	builder.WriteString("\nID: ")
	builder.WriteString(d.id)

	builder.WriteString("\nUsername: ")
	builder.WriteString(d.username)

	builder.WriteString("\nEmail: ")
	builder.WriteString(d.email)

	if len(d.password) > 0 {
		builder.WriteString("\nPassword: ")
		builder.Write(d.password)
	}

	builder.WriteRune('\n')

	return builder.String()
}

func NewDexUser(id, username, email, password string) DexUser {
	user := &dexUser{
		id:       id,
		email:    email,
		username: username,
	}

	user.SetPassword(password)

	return user
}

func NewSimpleDexUser(id, username, email string) DexUser {
	return &dexUser{
		id:       id,
		email:    email,
		username: username,
	}
}
