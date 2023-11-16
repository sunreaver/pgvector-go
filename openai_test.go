package pgvector_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
)

type apiRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

func FetchEmbeddings(input []string, apiKey string) ([][]float64, error) {
	url := "https://api.openai.com/v1/embeddings"
	data := &apiRequest{
		Input: input,
		Model: "text-embedding-ada-002",
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Bad status code: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	var embeddings [][]float64
	for _, item := range result["data"].([]interface{}) {
		var embedding []float64
		for _, v := range item.(map[string]interface{})["embedding"].([]interface{}) {
			embedding = append(embedding, float64(v.(float64)))
		}
		embeddings = append(embeddings, embedding)
	}
	return embeddings, nil
}

func TestOpenAI(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping example")
	}

	ctx := context.Background()

	conn, err := pgx.Connect(ctx, "postgres://localhost/pgvector_example")
	if err != nil {
		panic(err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		panic(err)
	}

	_, err = conn.Exec(ctx, "DROP TABLE IF EXISTS documents")
	if err != nil {
		panic(err)
	}

	_, err = conn.Exec(ctx, "CREATE TABLE documents (id bigserial PRIMARY KEY, content text, embedding vector(1536))")
	if err != nil {
		panic(err)
	}

	input := []string{
		"The dog is barking",
		"The cat is purring",
		"The bear is growling",
	}
	embeddings, err := FetchEmbeddings(input, apiKey)
	if err != nil {
		panic(err)
	}

	for i, content := range input {
		_, err := conn.Exec(ctx, "INSERT INTO documents (content, embedding) VALUES ($1, $2)", content, pgvector.NewVector(embeddings[i]))
		if err != nil {
			panic(err)
		}
	}

	documentId := 1
	rows, err := conn.Query(ctx, "SELECT id, content FROM documents WHERE id != $1 ORDER BY embedding <=> (SELECT embedding FROM documents WHERE id = $1) LIMIT 5", documentId)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var content string
		err = rows.Scan(&id, &content)
		if err != nil {
			panic(err)
		}
		fmt.Println(id, content)
	}

	if rows.Err() != nil {
		panic(rows.Err())
	}
}
