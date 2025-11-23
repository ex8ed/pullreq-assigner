package storage


import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"ex8ed/pullreq-assigner/internal/entity"
)


var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
)


type Storage struct {
	db *sqlx.DB
}


func New(db *sqlx.DB) *Storage {
	return &Storage{db: db}
}


func (s *Storage) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return s.db.BeginTxx(ctx, nil)
}

// =====================================================================
// TEAMS & USERS
// =====================================================================

func (s *Storage) CreateTeam(ctx context.Context, team entity.Team) error {
	_, err := s.db.NamedExecContext(ctx, `INSERT INTO teams (name) VALUES (:name)`, team)

	if err != nil {
		return err
	}

	query := `
		INSERT INTO users (id, username, is_active, team_name)
		VALUES (:id, :username, :is_active, :team_name)
		ON CONFLICT (id) DO UPDATE SET
			username = EXCLUDED.username,
			is_active = EXCLUDED.is_active,
			team_name = EXCLUDED.team_name;
	`
	for _, member := range team.Members {
		member.TeamName = team.Name
		
		if _, err := s.db.NamedExecContext(ctx, query, member); err != nil {
			return err
		}
	}

	return nil
}


func (s *Storage) GetTeam(ctx context.Context, name string) (*entity.Team, error) {
	var team entity.Team
	err := s.db.GetContext(ctx, &team, "SELECT name FROM teams WHERE name = $1", name)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	err = s.db.SelectContext(ctx, &team.Members, "SELECT * FROM users WHERE team_name = $1", name)
	if err != nil {
		return nil, err
	}

	return &team, nil
}


func (s *Storage) GetUser(ctx context.Context, userID string) (*entity.User, error) {
	var user entity.User
	err := s.db.GetContext(ctx, &user, "SELECT * FROM users WHERE id = $1", userID)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &user, err
}


func (s *Storage) SetUserActive(ctx context.Context, userID string, isActive bool) error {
	res, err := s.db.ExecContext(ctx, "UPDATE users SET is_active = $1 WHERE id = $2", isActive, userID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}


func (s *Storage) GetTeamMembers(ctx context.Context, teamName string) ([]entity.User, error) {
	var users []entity.User
	err := s.db.SelectContext(ctx, &users, "SELECT * FROM users WHERE team_name = $1", teamName)
	return users, err
}

// =====================================================================
// PULL REQUESTS
// =====================================================================


func (s *Storage) SavePR(ctx context.Context, tx *sqlx.Tx, pr entity.PullRequest) error {
	query := `
		INSERT INTO pull_requests (id, name, author_id, status, created_at)
		VALUES (:id, :name, :author_id, :status, :created_at)
	`
	_, err := tx.NamedExecContext(ctx, query, pr)
	return err
}


func (s *Storage) SaveReviewers(ctx context.Context, tx *sqlx.Tx, prID string, reviewerIDs []string) error {
	for _, uid := range reviewerIDs {
		_, err := tx.ExecContext(ctx, 
			"INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)", 
			prID, uid)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) GetPR(ctx context.Context, prID string) (*entity.PullRequest, error) {
	var pr entity.PullRequest

	err := s.db.GetContext(ctx, &pr, "SELECT * FROM pull_requests WHERE id = $1", prID)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	var reviewerIDs []string
	err = s.db.SelectContext(ctx, &reviewerIDs, "SELECT user_id FROM pr_reviewers WHERE pull_request_id = $1", prID)
	
	if err != nil {
		return nil, err
	}
	
	for _, rid := range reviewerIDs {
		pr.Reviewers = append(pr.Reviewers, entity.User{ID: rid})
	}

	return &pr, nil
}

func (s *Storage) MergePR(ctx context.Context, prID string) (*entity.PullRequest, error) {
	query := `
		UPDATE pull_requests 
		SET status = 'MERGED', merged_at = NOW() 
		WHERE id = $1 AND status = 'OPEN'
	`
	_, err := s.db.ExecContext(ctx, query, prID)
	if err != nil {
		return nil, err
	}
	
	return s.GetPR(ctx, prID)
}


// =====================================================================
// REASSIGN (Сложная логика)
// =====================================================================


func (s *Storage) RemoveReviewer(ctx context.Context, tx *sqlx.Tx, prID, userID string) error {
	_, err := tx.ExecContext(ctx, "DELETE FROM pr_reviewers WHERE pull_request_id=$1 AND user_id=$2", prID, userID)
	return err
}


func (s *Storage) AddReviewer(ctx context.Context, tx *sqlx.Tx, prID, userID string) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO pr_reviewers (pull_request_id, user_id) VALUES ($1, $2)", prID, userID)
	return err
}


func (s *Storage) GetUserReviews(ctx context.Context, userID string) ([]entity.PullRequest, error) {
	var prs []entity.PullRequest
	query := `
		SELECT p.* 
		FROM pull_requests p
		JOIN pr_reviewers r ON p.id = r.pull_request_id
		WHERE r.user_id = $1
	`
	err := s.db.SelectContext(ctx, &prs, query, userID)
	return prs, err
}