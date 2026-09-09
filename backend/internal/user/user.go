// Package user defines the account model and its MongoDB storage layer.
package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Plan is a user's subscription tier.
type Plan string

const (
	PlanFree Plan = "free"
	PlanPro  Plan = "pro"
)

// User is a CommitIn account.
//
// Email and Username are always stored normalized (trimmed + lowercased); the
// Username charset is already lowercase-only, so no case-folding index is
// needed. PasswordHash is a bcrypt hash and is never serialized to JSON.
type User struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string        `bson:"email" json:"email"`
	Username     string        `bson:"username" json:"username"`
	PasswordHash string        `bson:"password_hash" json:"-"`
	Plan         Plan          `bson:"plan" json:"plan"`
	CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time     `bson:"updated_at" json:"updated_at"`
}

// Storage errors returned by Store methods.
var (
	ErrNotFound  = errors.New("user: not found")
	ErrDuplicate = errors.New("user: email or username already taken")
)

// Store is the users collection.
type Store struct {
	col *mongo.Collection
}

// NewStore wraps the "users" collection of db.
func NewStore(db *mongo.Database) *Store {
	return &Store{col: db.Collection("users")}
}

// EnsureIndexes creates the unique indexes the users collection relies on.
// It is safe to call on every startup.
func (s *Store) EnsureIndexes(ctx context.Context) error {
	_, err := s.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_email"),
		},
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_username"),
		},
	})
	return err
}

// Create inserts a new user. Email and Username are normalized here; Plan
// defaults to free. Returns ErrDuplicate if the email or username is taken.
func (s *Store) Create(ctx context.Context, u *User) error {
	now := time.Now().UTC()
	u.ID = bson.NewObjectID()
	u.Email = normalize(u.Email)
	u.Username = normalize(u.Username)
	if u.Plan == "" {
		u.Plan = PlanFree
	}
	u.CreatedAt = now
	u.UpdatedAt = now

	if _, err := s.col.InsertOne(ctx, u); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

// GetByEmail looks up a user by (normalized) email. Returns ErrNotFound if
// there is no match.
func (s *Store) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.findOne(ctx, bson.M{"email": normalize(email)})
}

// GetByUsername looks up a user by (normalized) username. Returns ErrNotFound
// if there is no match.
func (s *Store) GetByUsername(ctx context.Context, username string) (*User, error) {
	return s.findOne(ctx, bson.M{"username": normalize(username)})
}

func (s *Store) findOne(ctx context.Context, filter bson.M) (*User, error) {
	var u User
	err := s.col.FindOne(ctx, filter).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
