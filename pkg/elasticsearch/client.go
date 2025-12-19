package elasticsearch

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/olivere/elastic/v7"
)

// ErrNotFound is returned when document is not found
var ErrNotFound = fmt.Errorf("document not found")

// Client wraps elastic.Client with additional methods
type Client struct {
	client *elastic.Client
}

// NewClient creates a new Client wrapper
func NewClient(ec *elastic.Client) *Client {
	return &Client{client: ec}
}

// Index indexes a document
func (c *Client) Index(ctx context.Context, index, id string, doc interface{}) error {
	_, err := c.client.Index().
		Index(index).
		Id(id).
		BodyJson(doc).
		Do(ctx)
	return err
}

// Get retrieves a document by ID
func (c *Client) Get(ctx context.Context, index, id string, result interface{}) error {
	res, err := c.client.Get().
		Index(index).
		Id(id).
		Do(ctx)

	if err != nil {
		if elastic.IsNotFound(err) {
			return ErrNotFound
		}
		return err
	}

	if !res.Found {
		return ErrNotFound
	}

	return json.Unmarshal(res.Source, result)
}

// Search performs a search query
func (c *Client) Search(ctx context.Context, index string, query map[string]interface{}) ([]map[string]interface{}, error) {
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	res, err := c.client.Search().
		Index(index).
		Source(string(queryJSON)).
		Do(ctx)

	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for _, hit := range res.Hits.Hits {
		var doc map[string]interface{}
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			continue
		}
		doc["_id"] = hit.Id
		results = append(results, doc)
	}

	return results, nil
}

// Update updates a document
func (c *Client) Update(ctx context.Context, index, id string, doc interface{}) error {
	_, err := c.client.Update().
		Index(index).
		Id(id).
		Doc(doc).
		Do(ctx)
	return err
}

// Delete deletes a document
func (c *Client) Delete(ctx context.Context, index, id string) error {
	_, err := c.client.Delete().
		Index(index).
		Id(id).
		Do(ctx)
	return err
}

// BulkIndex indexes multiple documents
func (c *Client) BulkIndex(ctx context.Context, index string, docs map[string]interface{}) error {
	bulk := c.client.Bulk()

	for id, doc := range docs {
		req := elastic.NewBulkIndexRequest().
			Index(index).
			Id(id).
			Doc(doc)
		bulk.Add(req)
	}

	_, err := bulk.Do(ctx)
	return err
}

// CreateIndex creates an index with mappings
func (c *Client) CreateIndex(ctx context.Context, index string, mapping string) error {
	_, err := c.client.CreateIndex(index).
		Body(mapping).
		Do(ctx)
	return err
}

// DeleteIndex deletes an index
func (c *Client) DeleteIndex(ctx context.Context, index string) error {
	_, err := c.client.DeleteIndex(index).Do(ctx)
	return err
}

// IndexExists checks if index exists
func (c *Client) IndexExists(ctx context.Context, index string) (bool, error) {
	return c.client.IndexExists(index).Do(ctx)
}

// Refresh refreshes an index
func (c *Client) Refresh(ctx context.Context, index string) error {
	_, err := c.client.Refresh(index).Do(ctx)
	return err
}

// Count returns document count in index
func (c *Client) Count(ctx context.Context, index string) (int64, error) {
	count, err := c.client.Count(index).Do(ctx)
	if err != nil {
		return 0, err
	}
	return count, nil
}
