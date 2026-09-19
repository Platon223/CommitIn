// Package score stores per-commit quality scores submitted by the CLI.
package score

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Score bounds.
const (
	MinScore = 0
	MaxScore = 10
)

// RateLimitWindow and RateLimitMax bound how many scores one user can submit
// in a rolling window -- a defense against a scripted /scores abuser, not a
// limit the CLI should ever hit (it submits at most one score per commit).
const (
	RateLimitWindow = time.Hour
	RateLimitMax    = 30
)

// Score is one judged commit-message attempt, tied to the user who made it.
//
// AttemptID is an opaque per-attempt identifier, not a git commit hash: the
// CLI submits from the commit-msg hook, before any commit object exists, so
// there's nothing to hash yet -- and a rejected (bad-verdict) attempt never
// becomes a commit at all, but still gets scored (see commitin-decisions).
type Score struct {
	ID        bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    bson.ObjectID `bson:"user_id" json:"user_id"`
	Score     int           `bson:"score" json:"score"`
	AttemptID string        `bson:"attempt_id" json:"attempt_id"`
	RepoName  string        `bson:"repo_name" json:"repo_name"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
}

// Storage errors returned by Store methods.
var (
	// ErrDuplicate means this user already submitted this exact attempt ID
	// -- protects against a retried or double-fired submission for the same
	// hook invocation.
	ErrDuplicate = errors.New("score: already submitted for this attempt")
	// ErrRateLimited means this user has hit RateLimitMax submissions within
	// RateLimitWindow.
	ErrRateLimited = errors.New("score: rate limit exceeded")
)

// Store is the scores collection.
type Store struct {
	col *mongo.Collection
}

// NewStore wraps the "scores" collection of db.
func NewStore(db *mongo.Database) *Store {
	return &Store{col: db.Collection("scores")}
}

// EnsureIndexes creates the indexes the scores collection relies on: a
// unique (user_id, attempt_id) pair for dedup, and (user_id, created_at) for
// both the rate-limit count and the future leaderboard window query. Safe to
// call on every startup.
func (s *Store) EnsureIndexes(ctx context.Context) error {
	_, err := s.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "attempt_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_user_attempt"),
		},
		{
			Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: 1}},
			Options: options.Index().SetName("by_user_created"),
		},
	})
	return err
}

// Create records a score for userID, enforcing the rate limit and the
// per-attempt dedup. sc must already be validated to [MinScore, MaxScore] by
// the caller.
func (s *Store) Create(ctx context.Context, userID bson.ObjectID, sc int, attemptID, repoName string) (*Score, error) {
	windowStart := time.Now().UTC().Add(-RateLimitWindow)
	count, err := s.col.CountDocuments(ctx, bson.M{
		"user_id":    userID,
		"created_at": bson.M{"$gte": windowStart},
	})
	if err != nil {
		return nil, err
	}
	if count >= RateLimitMax {
		return nil, ErrRateLimited
	}

	doc := &Score{
		ID:        bson.NewObjectID(),
		UserID:    userID,
		Score:     sc,
		AttemptID: attemptID,
		RepoName:  repoName,
		CreatedAt: time.Now().UTC(),
	}
	if _, err := s.col.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrDuplicate
		}
		return nil, err
	}
	return doc, nil
}

// Leaderboard ranking parameters. Rank by rolling AVERAGE score over a
// window, not sum/total, so volume-farming throwaway commits can't buy
// rank; MinCommits keeps one lucky score from landing someone at #1.
const (
	LeaderboardWindow     = 30 * 24 * time.Hour
	LeaderboardMinCommits = 10
	LeaderboardLimit      = 100
)

// LeaderboardEntry is one ranked row. Deliberately carries no PII beyond the
// username -- this is a public endpoint.
type LeaderboardEntry struct {
	Username     string  `bson:"username" json:"username"`
	AverageScore float64 `bson:"avg_score" json:"average_score"`
	CommitCount  int     `bson:"count" json:"commit_count"`
}

// Leaderboard ranks users by average score over the last window, requiring
// at least minCommits scores in that window to appear, highest average
// first, capped at limit rows.
func (s *Store) Leaderboard(ctx context.Context, window time.Duration, minCommits, limit int) ([]LeaderboardEntry, error) {
	windowStart := time.Now().UTC().Add(-window)
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "created_at", Value: bson.D{{Key: "$gte", Value: windowStart}}}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$user_id"},
			{Key: "avg_score", Value: bson.D{{Key: "$avg", Value: "$score"}}},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		{{Key: "$match", Value: bson.D{{Key: "count", Value: bson.D{{Key: "$gte", Value: minCommits}}}}}},
		{{Key: "$sort", Value: bson.D{{Key: "avg_score", Value: -1}}}},
		{{Key: "$limit", Value: int64(limit)}},
		// Join to users for the display name. Scores never store a username
		// directly, so this is the one place the leaderboard crosses into
		// the users collection.
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "user"},
		}}},
		{{Key: "$unwind", Value: "$user"}},
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "username", Value: "$user.username"},
			{Key: "avg_score", Value: bson.D{{Key: "$round", Value: bson.A{"$avg_score", 2}}}},
			{Key: "count", Value: 1},
		}}},
	}

	cur, err := s.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	entries := []LeaderboardEntry{}
	if err := cur.All(ctx, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
