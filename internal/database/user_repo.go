package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UserRepo manages user registrations and broadcasting
type UserRepo struct {
	coll *mongo.Collection
}

// NewUserRepo creates a new User repository
func NewUserRepo(db *MongoDB) *UserRepo {
	return &UserRepo{coll: db.Users}
}

// RegisterOrUpdate saves or updates user profile upon interaction
func (r *UserRepo) RegisterOrUpdate(id int64, username, firstName, lastName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"username":    username,
			"first_name":  firstName,
			"last_name":   lastName,
			"last_active": time.Now(),
		},
		"$setOnInsert": bson.M{
			"_id":                    id,
			"joined_at":              time.Now(),
			"active_email":           "",
			"active_token":           "",
			"active_server":          "",
			"total_emails_generated": 0,
			"total_emails_deleted":   0,
		},
	}
	opts := options.Update().SetUpsert(true)

	_, err := r.coll.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetAllUserIDs returns list of all user IDs for broadcasting
func (r *UserRepo) GetAllUserIDs() ([]int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := options.Find().SetProjection(bson.M{"_id": 1})
	cursor, err := r.coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var ids []int64
	for cursor.Next(ctx) {
		var doc struct {
			ID int64 `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err == nil {
			ids = append(ids, doc.ID)
		}
	}
	return ids, nil
}

// CountUsers returns total user count
func (r *UserRepo) CountUsers() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return r.coll.CountDocuments(ctx, bson.M{})
}
