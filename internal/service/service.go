package service


import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/jmoiron/sqlx"
	"ex8ed/pullreq-assigner/internal/entity"
)


var (
	ErrPRMerged      = errors.New("canot edit merged PR")
	ErrNotAssigned   = errors.New("user is not a reviewer")
	ErrNoCandidates  = errors.New("no candidates")
	ErrReviewerFound = errors.New("reviewer already assigned")
)


type Repository interface {
	BeginTx(ctx context.Context) (*sqlx.Tx, error)

	CreateTeam(ctx context.Context, team entity.Team) error
	GetTeam(ctx context.Context, name string) (*entity.Team, error)
	GetTeamMembers(ctx context.Context, teamName string) ([]entity.User, error)
	GetUser(ctx context.Context, userID string) (*entity.User, error)
	SetUserActive(ctx context.Context, userID string, isActive bool) error
	GetUserReviews(ctx context.Context, userID string) ([]entity.PullRequest, error)

	SavePR(ctx context.Context, tx *sqlx.Tx, pr entity.PullRequest) error
	SaveReviewers(ctx context.Context, tx *sqlx.Tx, prID string, reviewerIDs []string) error
	GetPR(ctx context.Context, prID string) (*entity.PullRequest, error)
	MergePR(ctx context.Context, prID string) (*entity.PullRequest, error)
	
	RemoveReviewer(ctx context.Context, tx *sqlx.Tx, prID, userID string) error
	AddReviewer(ctx context.Context, tx *sqlx.Tx, prID, userID string) error
}


type Service struct {
	repo Repository
}


func New(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateTeam(ctx context.Context, team entity.Team) error {
	return s.repo.CreateTeam(ctx, team)
}

func (s *Service) SetUserActive(ctx context.Context, userID string, isActive bool) error {
	return s.repo.SetUserActive(ctx, userID, isActive)
}

func (s *Service) GetTeam(ctx context.Context, name string) (*entity.Team, error) {
	return s.repo.GetTeam(ctx, name)
}

func (s *Service) GetUserReviews(ctx context.Context, userID string) ([]entity.PullRequest, error) {
	return s.repo.GetUserReviews(ctx, userID)
}

func (s *Service) MergePR(ctx context.Context, prID string) (*entity.PullRequest, error) {
	return s.repo.MergePR(ctx, prID)
}


func (s *Service) GetUser(ctx context.Context, userID string) (*entity.User, error) {
	return s.repo.GetUser(ctx, userID)
}


func (s *Service) CreatePR(ctx context.Context, reqID, name, authorID string) (*entity.PullRequest, error) {
	author, err := s.repo.GetUser(ctx, authorID)
	if err != nil {
		return nil, err
	}

	teamMembers, err := s.repo.GetTeamMembers(ctx, author.TeamName)
	if err != nil {
		return nil, err
	}

	candidates := make([]entity.User, 0)
	for _, u := range teamMembers {
		if u.IsActive && u.ID != author.ID {
			candidates = append(candidates, u)
		}
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(candidates), func(i, j int) { 
		candidates[i], candidates[j] = candidates[j], candidates[i] 
	})

	limit := 2
	if len(candidates) < 2 {
		limit = len(candidates)
	}
	
	chosenReviewers := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		chosenReviewers = append(chosenReviewers, candidates[i].ID)
	}

	pr := entity.PullRequest{
		ID:        reqID,
		Name:      name,
		AuthorID:  authorID,
		Status:    "OPEN",
		CreatedAt: time.Now(),
		Reviewers: nil,
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	if err := s.repo.SavePR(ctx, tx, pr); err != nil {
		return nil, err
	}

	if len(chosenReviewers) > 0 {
		if err := s.repo.SaveReviewers(ctx, tx, pr.ID, chosenReviewers); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &pr, nil
}


func (s *Service) ReassignReviewer(ctx context.Context, prID, oldUserID string) (*entity.PullRequest, string, error) {
	pr, err := s.repo.GetPR(ctx, prID)
	if err != nil {
		return nil, "", err
	}

	if pr.Status == "MERGED" {
		return nil, "", ErrPRMerged
	}

	busyMap := make(map[string]bool)
	isAssigned := false

	for _, u := range pr.Reviewers {
		busyMap[u.ID] = true

		if u.ID == oldUserID {
			isAssigned = true
		}
	}
	if !isAssigned {
		return nil, "", ErrNotAssigned
	}

	oldUser, err := s.repo.GetUser(ctx, oldUserID)
	if err != nil {
		return nil, "", err
	}

	teamMembers, err := s.repo.GetTeamMembers(ctx, oldUser.TeamName)
	if err != nil {
		return nil, "", err
	}

	candidates := make([]entity.User, 0)
	for _, u := range teamMembers {
		if !u.IsActive { continue }
		if u.ID == pr.AuthorID { continue }
		if busyMap[u.ID] { continue }

		candidates = append(candidates, u)
	}

	if len(candidates) == 0 {
		return nil, "", ErrNoCandidates
	}


	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	newReviewer := candidates[r.Intn(len(candidates))]

	tx, err := s.repo.BeginTx(ctx)
	if err != nil { 
		return nil, "", err 
	}

	defer tx.Rollback()

	if err := s.repo.RemoveReviewer(ctx, tx, prID, oldUserID); err != nil {
		return nil, "", err
	}

	if err := s.repo.AddReviewer(ctx, tx, prID, newReviewer.ID); err != nil {
		return nil, "", err
	}

	if err := tx.Commit(); err != nil {
		return nil, "", err
	}

	updatedPR, err := s.repo.GetPR(ctx, prID)
	return updatedPR, newReviewer.ID, err
}