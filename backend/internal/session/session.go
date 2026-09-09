// Package session stores opaque auth tokens as server-side sessions.
package session

import (
	"context"
	"errors"
	"time"

	"github.com/Platon223/commitin/backend/internal/auth"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// TTL is how long a token stays valid after it is issued.
const TTL = 30 * 24 * time.Hour

// Session is one issued token. Only the token's hash is persisted.
type Session struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	UserID    bson.ObjectID `bson:"user_id"`
	TokenHash string        `bson:"token_hash"`
	CreatedAt time.Time     `bson:"created_at"`
	ExpiresAt time.Time     `bson:"expires_at"`
}

// ErrNotFound means the token is unknown or expired.
var ErrNotFound = errors.New("session: not found or expired")

// Store is the sessions collection.
type Store struct {
	col *mongo.Collection
}

// NewStore wraps the "sessions" collection of db.
func NewStore(db *mongo.Database) *Store {
	return &Store{col: db.Collection("sessions")}
}

// EnsureIndexes creates the token lookup, user lookup, and TTL indexes.
// Safe to call on every startup.
func (s *Store) EnsureIndexes(ctx context.Context) error {
	_, err := s.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token_hash", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_token_hash"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}},
			Options: options.Index().SetName("by_user"),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0).SetName("ttl_expires_at"),
		},
	})
	return err
}

// Create issues a new session for userID and returns the plaintext token.
// The plaintext is only available here; the store keeps just its hash.
func (s *Store) Create(ctx context.Context, userID bson.ObjectID) (string, *Session, error) {
	token, err := auth.GenerateToken()
	if err != nil {
		return "", nil, err
	}
	now := time.Now().UTC()
	sess := &Session{
		ID:        bson.NewObjectID(),
		UserID:    userID,
		TokenHash: auth.HashToken(token),
		CreatedAt: now,
		ExpiresAt: now.Add(TTL),
	}
	if _, err := s.col.InsertOne(ctx, sess); err != nil {
		return "", nil, err
	}
	return token, sess, nil
}

// Lookup returns the session for a plaintext token, or ErrNotFound if it is
// unknown or expired. (Mongo's TTL sweep is not instant, so expiry is also
// checked here.)
func (s *Store) Lookup(ctx context.Context, token string) (*Session, error) {
	var sess Session
	err := s.col.FindOne(ctx, bson.M{"token_hash": auth.HashToken(token)}).Decode(&sess)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, ErrNotFound
	}
	return &sess, nil
}

// Delete removes the session for a plaintext token (logout). It is not an
// error if no such session exists.
func (s *Store) Delete(ctx context.Context, token string) error {
	_, err := s.col.DeleteOne(ctx, bson.M{"token_hash": auth.HashToken(token)})
	return err
}
