package database

import "time"

// User stores user information and usage statistics
type User struct {
	ID                   int64     `bson:"_id" json:"id"`
	Username             string    `bson:"username" json:"username"`
	FirstName            string    `bson:"first_name" json:"first_name"`
	LastName             string    `bson:"last_name" json:"last_name"`
	ActiveEmail          string    `bson:"active_email" json:"active_email"`
	ActiveToken          string    `bson:"active_token" json:"active_token"`
	ActiveServer         string    `bson:"active_server" json:"active_server"`
	TotalEmailsGenerated int       `bson:"total_emails_generated" json:"total_emails_generated"`
	TotalEmailsDeleted   int       `bson:"total_emails_deleted" json:"total_emails_deleted"`
	JoinedAt             time.Time `bson:"joined_at" json:"joined_at"`
	LastActive           time.Time `bson:"last_active" json:"last_active"`
}

// MailRecord preserves audit history of generated disposable emails
type MailRecord struct {
	UserID    int64     `bson:"user_id" json:"user_id"`
	Email     string    `bson:"email" json:"email"`
	Token     string    `bson:"token" json:"token"`
	Server    string    `bson:"server" json:"server"`
	IsActive  bool      `bson:"is_active" json:"is_active"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}
