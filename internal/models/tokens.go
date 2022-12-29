package models

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"log"
	"time"
)

const (
	ScopeAuthentication = "authentication"
)

// Token is the type for authentication tokens
type Token struct {
	PlainText string    `json:"token"`
	UserID    int64     `json:"-"`
	Hash      []byte    `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

// GenerateToken generates a token that lasts for ttl, and returns it
func GenerateToken(userID int, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: int64(userID),
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.PlainText = base32.
		StdEncoding.
		WithPadding(base32.NoPadding).
		EncodeToString(randomBytes)
	hash := sha256.Sum256(([]byte(token.PlainText)))
	token.Hash = hash[:]
	return token, nil
}

const deleteTokenByUserIDSql = `
delete from tokens where user_id = ?
`

const insertTokenSql = `
insert into tokens (user_id, name, email, token_hash, expiry, created_at, updated_at)
values (?,?,?,?,?,?,?)
`

func (m *DBModel) InsertToken(t *Token, u User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// delete existing tokens by that user
	_, err := m.DB.ExecContext(ctx, deleteTokenByUserIDSql, u.ID)
	if err != nil {
		return err
	}

	_, err = m.DB.ExecContext(ctx, insertTokenSql,
		u.ID,
		u.LastName,
		u.Email,
		t.Hash,
		t.Expiry,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		return err
	}

	return nil
}

const getUserByTokenSql = `
select
	u.id, u.first_name, u.last_name, u.email
from
	users u
	inner join tokens t on (u.id = t.user_id)
where
	t.token_hash = ?
	and t.expiry > ?
`

func (m *DBModel) GetUserByToken(token string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tokenHash := sha256.Sum256([]byte(token))

	var user User
	err := m.DB.QueryRowContext(ctx, getUserByTokenSql, tokenHash[:], time.Now()).
		Scan(
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Email,
		)

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &user, nil
}
