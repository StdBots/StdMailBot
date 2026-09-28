/*
 * Copyright (C) 2024-2026 STD DEEPANSHU <https://deepanshu.in>
 * STD BOTS - Telegram: @STD_DEEPANSHU, @STDBOTS
 *
 * This file is part of StdMailBot.
 * Licensed under AGPL-3.0.
 */

package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB wraps database connection and collections
type MongoDB struct {
	client   *mongo.Client
	db       *mongo.Database
	Users    *mongo.Collection
	MailLogs *mongo.Collection
}

// NewMongoDB initializes connection and creates necessary indexes
func NewMongoDB(uri, dbName string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri).SetMaxPoolSize(100).SetMinPoolSize(10)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(dbName)
	m := &MongoDB{
		client:   client,
		db:       db,
		Users:    db.Collection("users"),
		MailLogs: db.Collection("mail_logs"),
	}

	m.initIndexes()
	log.Printf("Connected to MongoDB successfully: db=%s", dbName)
	return m, nil
}

func (m *MongoDB) initIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, _ = m.Users.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "active_email", Value: 1}}},
		{Keys: bson.D{{Key: "joined_at", Value: -1}}},
	})

	_, _ = m.MailLogs.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
		{Keys: bson.D{{Key: "created_at", Value: -1}}},
	})
}

// Close gracefully closes the MongoDB connection
func (m *MongoDB) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return m.client.Disconnect(ctx)
}
