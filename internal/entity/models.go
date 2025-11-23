package entity

import "time"


type User struct {
	ID 			string 		`json:"user_id" db:"id"`
	Username 	string 		`json:"username" db:"username"`
	IsActive 	bool 		`json:"is_active" db:"is_active"`
	TeamName 	string 		`json:"team_name" db:"team_name"`
}


type Team struct {
	Name		string 		`json:"team_name" db:"name"`
	Members 	[]User 		`json:"members" db:"-"`
}


type PullRequest struct {
	ID			string 		`json:"pull_request_id" db:"id"`
	Name 		string 		`json:"pull_request_name" db:"name"`
	AuthorID 	string 		`json:"author_id" db:"author_id"`
	Status		string 		`json:"status" db:"status"`

	CreatedAt 	time.Time 	`json:"created_at" db:"created_at"`
	MergedAt	*time.Time 	`json:"merged_at,omitempty" db:"merged_at"`

	Reviewers	[]User 		`json:"assigned_reviewers" db:"-"`
}


type PRReviewerPair struct {
	PRID		string 		`db:"pull_request_id"`
	UserID		string 		`db:"user_id"`
}
