package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InsertOne inserts a single document
func (c *Client) InsertOne(ctx context.Context, collection string, document interface{}) (*mongo.InsertOneResult, error) {
	return c.Collection(collection).InsertOne(ctx, document)
}

// InsertMany inserts multiple documents
func (c *Client) InsertMany(ctx context.Context, collection string, documents []interface{}) (*mongo.InsertManyResult, error) {
	return c.Collection(collection).InsertMany(ctx, documents)
}

// FindOne finds a single document
func (c *Client) FindOne(ctx context.Context, collection string, filter interface{}, result interface{}) error {
	return c.Collection(collection).FindOne(ctx, filter).Decode(result)
}

// Find finds multiple documents
func (c *Client) Find(ctx context.Context, collection string, filter interface{}, results interface{}, opts ...*options.FindOptions) error {
	cursor, err := c.Collection(collection).Find(ctx, filter, opts...)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	return cursor.All(ctx, results)
}

// FindByID finds a document by ID
func (c *Client) FindByID(ctx context.Context, collection string, id interface{}, result interface{}) error {
	filter := bson.M{"_id": id}
	return c.FindOne(ctx, collection, filter, result)
}

// UpdateOne updates a single document
func (c *Client) UpdateOne(ctx context.Context, collection string, filter, update interface{}) (*mongo.UpdateResult, error) {
	return c.Collection(collection).UpdateOne(ctx, filter, update)
}

// UpdateMany updates multiple documents
func (c *Client) UpdateMany(ctx context.Context, collection string, filter, update interface{}) (*mongo.UpdateResult, error) {
	return c.Collection(collection).UpdateMany(ctx, filter, update)
}

// UpdateByID updates a document by ID
func (c *Client) UpdateByID(ctx context.Context, collection string, id interface{}, update interface{}) (*mongo.UpdateResult, error) {
	filter := bson.M{"_id": id}
	return c.UpdateOne(ctx, collection, filter, update)
}

// ReplaceOne replaces a single document
func (c *Client) ReplaceOne(ctx context.Context, collection string, filter, replacement interface{}) (*mongo.UpdateResult, error) {
	return c.Collection(collection).ReplaceOne(ctx, filter, replacement)
}

// DeleteOne deletes a single document
func (c *Client) DeleteOne(ctx context.Context, collection string, filter interface{}) (*mongo.DeleteResult, error) {
	return c.Collection(collection).DeleteOne(ctx, filter)
}

// DeleteMany deletes multiple documents
func (c *Client) DeleteMany(ctx context.Context, collection string, filter interface{}) (*mongo.DeleteResult, error) {
	return c.Collection(collection).DeleteMany(ctx, filter)
}

// DeleteByID deletes a document by ID
func (c *Client) DeleteByID(ctx context.Context, collection string, id interface{}) (*mongo.DeleteResult, error) {
	filter := bson.M{"_id": id}
	return c.DeleteOne(ctx, collection, filter)
}

// CountDocuments counts documents matching filter
func (c *Client) CountDocuments(ctx context.Context, collection string, filter interface{}) (int64, error) {
	return c.Collection(collection).CountDocuments(ctx, filter)
}

// Aggregate performs aggregation
func (c *Client) Aggregate(ctx context.Context, collection string, pipeline interface{}, results interface{}) error {
	cursor, err := c.Collection(collection).Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	return cursor.All(ctx, results)
}

// CreateIndex creates an index
func (c *Client) CreateIndex(ctx context.Context, collection string, model mongo.IndexModel) (string, error) {
	return c.Collection(collection).Indexes().CreateOne(ctx, model)
}

// CreateIndexes creates multiple indexes
func (c *Client) CreateIndexes(ctx context.Context, collection string, models []mongo.IndexModel) ([]string, error) {
	return c.Collection(collection).Indexes().CreateMany(ctx, models)
}

// DropIndex drops an index
func (c *Client) DropIndex(ctx context.Context, collection, name string) error {
	_, err := c.Collection(collection).Indexes().DropOne(ctx, name)
	return err
}

// ListIndexes lists all indexes
func (c *Client) ListIndexes(ctx context.Context, collection string) ([]bson.M, error) {
	cursor, err := c.Collection(collection).Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var indexes []bson.M
	if err := cursor.All(ctx, &indexes); err != nil {
		return nil, err
	}

	return indexes, nil
}

// BulkWrite performs bulk write operations
func (c *Client) BulkWrite(ctx context.Context, collection string, models []mongo.WriteModel) (*mongo.BulkWriteResult, error) {
	return c.Collection(collection).BulkWrite(ctx, models)
}

// Distinct finds distinct values for a field
func (c *Client) Distinct(ctx context.Context, collection, field string, filter interface{}) ([]interface{}, error) {
	return c.Collection(collection).Distinct(ctx, field, filter)
}
