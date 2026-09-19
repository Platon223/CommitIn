// Package score stores per-commit quality scores submitted by the CLI.
package score

import (
	"context"
	"errors"
	"math"
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

// PassingScore is the minimum score the CLI's commit-msg gate accepts.
//
// This mirrors cli/internal/claude.PassingScore -- they're separate Go
// modules with no shared package yet, so the two constants have to be kept
// in sync by hand. It's only used here to classify a stored score as
// "rejected" for stats; the gate itself lives in the CLI.
const PassingScore = 6

// StatsWindow is the length of one stats period. Stats compares the most
// recent window against the window immediately before it.
const StatsWindow = 30 * 24 * time.Hour

// PeriodStats summarizes one window of a user's judged attempts.
type PeriodStats struct {
	Attempts     int     `json:"attempts"`
	AverageScore float64 `json:"average_score"`
	// Rejected counts attempts scoring below PassingScore -- the ones the
	// gate blocked. Divide by Attempts for a rejection rate.
	Rejected int `json:"rejected"`
}

// DayStats is one UTC calendar day inside the current window. Days with no
// attempts are omitted.
type DayStats struct {
	Date         string  `json:"date"` // YYYY-MM-DD, UTC
	Attempts     int     `json:"attempts"`
	AverageScore float64 `json:"average_score"`
}

// Stats is one user's history: the current window, the equal-length window
// before it (for a trend), and per-day buckets for the current window.
type Stats struct {
	WindowDays   int         `json:"window_days"`
	PassingScore int         `json:"passing_score"`
	Current      PeriodStats `json:"current"`
	Previous     PeriodStats `json:"previous"`
	Daily        []DayStats  `json:"daily"`
}

type periodRow struct {
	Attempts int     `bson:"attempts"`
	Avg      float64 `bson:"avg"`
	Rejected int     `bson:"rejected"`
}

type dayRow struct {
	Date     string  `bson:"_id"`
	Attempts int     `bson:"attempts"`
	Avg      float64 `bson:"avg"`
}

func round2(x float64) float64 { return math.Round(x*100) / 100 }

func toPeriod(rows []periodRow) PeriodStats {
	if len(rows) == 0 {
		return PeriodStats{}
	}
	r := rows[0]
	return PeriodStats{Attempts: r.Attempts, AverageScore: round2(r.Avg), Rejected: r.Rejected}
}

// Stats returns userID's history for the last window and the window before
// it, in a single aggregation round trip ($facet). Day buckets are UTC.
func (s *Store) Stats(ctx context.Context, userID bson.ObjectID, window time.Duration) (*Stats, error) {
	now := time.Now().UTC()
	curStart := now.Add(-window)
	prevStart := now.Add(-2 * window)

	inCurrent := bson.D{{Key: "$match", Value: bson.D{{Key: "created_at", Value: bson.D{{Key: "$gte", Value: curStart}}}}}}
	inPrevious := bson.D{{Key: "$match", Value: bson.D{{Key: "created_at", Value: bson.D{{Key: "$lt", Value: curStart}}}}}}
	summarize := bson.D{{Key: "$group", Value: bson.D{
		{Key: "_id", Value: nil},
		{Key: "attempts", Value: bson.D{{Key: "$sum", Value: 1}}},
		{Key: "avg", Value: bson.D{{Key: "$avg", Value: "$score"}}},
		{Key: "rejected", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: bson.A{
			bson.D{{Key: "$lt", Value: bson.A{"$score", PassingScore}}}, 1, 0,
		}}}}}},
	}}}
	byDay := bson.D{{Key: "$group", Value: bson.D{
		{Key: "_id", Value: bson.D{{Key: "$dateToString", Value: bson.D{
			{Key: "format", Value: "%Y-%m-%d"},
			{Key: "date", Value: "$created_at"},
			{Key: "timezone", Value: "UTC"},
		}}}},
		{Key: "attempts", Value: bson.D{{Key: "$sum", Value: 1}}},
		{Key: "avg", Value: bson.D{{Key: "$avg", Value: "$score"}}},
	}}}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "user_id", Value: userID},
			{Key: "created_at", Value: bson.D{{Key: "$gte", Value: prevStart}}},
		}}},
		{{Key: "$facet", Value: bson.D{
			{Key: "current", Value: bson.A{inCurrent, summarize}},
			{Key: "previous", Value: bson.A{inPrevious, summarize}},
			{Key: "daily", Value: bson.A{inCurrent, byDay, bson.D{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}}}},
		}}},
	}

	cur, err := s.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var res []struct {
		Current  []periodRow `bson:"current"`
		Previous []periodRow `bson:"previous"`
		Daily    []dayRow    `bson:"daily"`
	}
	if err := cur.All(ctx, &res); err != nil {
		return nil, err
	}

	out := &Stats{
		WindowDays:   int(window.Hours() / 24),
		PassingScore: PassingScore,
		Daily:        []DayStats{},
	}
	if len(res) == 0 { // $facet always yields one doc, but don't index blindly
		return out, nil
	}
	out.Current = toPeriod(res[0].Current)
	out.Previous = toPeriod(res[0].Previous)
	for _, d := range res[0].Daily {
		out.Daily = append(out.Daily, DayStats{Date: d.Date, Attempts: d.Attempts, AverageScore: round2(d.Avg)})
	}
	return out, nil
}
