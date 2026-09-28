package database

import (
	"context"
	"time"

	"github.com/StdBots/StdMailBot/internal/mail"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MailRepo manages user email sessions and logs
type MailRepo struct {
	users    *mongo.Collection
	mailLogs *mongo.Collection
}

// NewMailRepo creates a new Mail repository
func NewMailRepo(db *MongoDB) *MailRepo {
	return &MailRepo{
		users:    db.Users,
		mailLogs: db.MailLogs,
	}
}

// SaveActiveMail updates user's active email session and logs the creation
func (r *MailRepo) SaveActiveMail(userID int64, session *mail.MailSession) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Update user record
	_, err := r.users.UpdateOne(ctx,
		bson.M{"_id": userID},
		bson.M{
			"$set": bson.M{
				"active_email":  session.Email,
				"active_token":  session.Token,
				"active_server": session.Server,
				"last_active":   time.Now(),
			},
			"$inc": bson.M{"total_emails_generated": 1},
		},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return err
	}

	// Insert audit log
	_, _ = r.mailLogs.InsertOne(ctx, MailRecord{
		UserID:    userID,
		Email:     session.Email,
		Token:     session.Token,
		Server:    session.Server,
		IsActive:  true,
		CreatedAt: time.Now(),
	})

	return nil
}

// GetActiveMail fetches user's currently active mailbox session
func (r *MailRepo) GetActiveMail(userID int64) (*mail.MailSession, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	err := r.users.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}

	if user.ActiveEmail == "" {
		return nil, nil
	}

	return &mail.MailSession{
		Email:  user.ActiveEmail,
		Token:  user.ActiveToken,
		Server: user.ActiveServer,
	}, nil
}

// DeleteActiveMail clears active mailbox and returns previous session for cleanup
func (r *MailRepo) DeleteActiveMail(userID int64) (*mail.MailSession, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := r.GetActiveMail(userID)
	if err != nil || session == nil {
		return nil, err
	}

	_, err = r.users.UpdateOne(ctx,
		bson.M{"_id": userID},
		bson.M{
			"$set": bson.M{
				"active_email":  "",
				"active_token":  "",
				"active_server": "",
				"last_active":   time.Now(),
			},
			"$inc": bson.M{"total_emails_deleted": 1},
		},
	)
	if err != nil {
		return nil, err
	}

	// Mark log inactive
	_, _ = r.mailLogs.UpdateMany(ctx,
		bson.M{"user_id": userID, "email": session.Email},
		bson.M{"$set": bson.M{"is_active": false}},
	)

	return session, nil
}

// CountActiveEmails returns number of users currently holding an active email
func (r *MailRepo) CountActiveEmails() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.users.CountDocuments(ctx, bson.M{"active_email": bson.M{"$ne": ""}})
}

// CountTotalGenerated returns total emails generated across history
func (r *MailRepo) CountTotalGenerated() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.mailLogs.CountDocuments(ctx, bson.M{})
}

// CountTotalDeleted aggregates total deleted count from users
func (r *MailRepo) CountTotalDeleted() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$total_emails_deleted"}}},
		}}},
	}

	cursor, err := r.users.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var results []struct {
		Total int64 `bson:"total"`
	}
	if err = cursor.All(ctx, &results); err != nil || len(results) == 0 {
		return 0, nil
	}
	return results[0].Total, nil
}
