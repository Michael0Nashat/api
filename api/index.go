package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/tbxark/g4vercel" // Importing the g4vercel package
)

type Post struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func Handler() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Retrieve the DB connection string from environment variable
	connStr := os.Getenv("DB_CONNECTION_STRING")
	if connStr == "" {
		log.Fatal("DB_CONNECTION_STRING is not set in the .env file")
	}

	// Connect to the database
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	// Test the connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping the database: %v", err)
	}
	fmt.Println("Successfully connected to the database!")

	// Handle the homepage
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema='public'")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to query database: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		fmt.Fprintf(w, "<h1>Public Tables in the Database</h1><ul>")
		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				http.Error(w, fmt.Sprintf("Failed to read row: %v", err), http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, "<li>%s</li>", tableName)
		}
		fmt.Fprintf(w, "</ul>")
	})

	// Handle GET request for posts
	http.HandleFunc("/api/posts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			rows, err := db.Query("SELECT id, title, content FROM posts")
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to query posts: %v", err), http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			posts := []Post{}
			for rows.Next() {
				var post Post
				if err := rows.Scan(&post.ID, &post.Title, &post.Content); err != nil {
					http.Error(w, fmt.Sprintf("Failed to read post: %v", err), http.StatusInternalServerError)
					return
				}
				posts = append(posts, post)
			}

			// Respond with JSON
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(posts); err != nil {
				http.Error(w, fmt.Sprintf("Failed to encode posts: %v", err), http.StatusInternalServerError)
			}
		} else {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		}
	})

	// Handle POST request to create a new post
	http.HandleFunc("/api/posts/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var post Post
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&post); err != nil {
				http.Error(w, fmt.Sprintf("Failed to decode JSON: %v", err), http.StatusBadRequest)
				return
			}

			// Insert the new post into the database
			query := "INSERT INTO posts (title, content) VALUES ($1, $2) RETURNING id"
			err := db.QueryRow(query, post.Title, post.Content).Scan(&post.ID)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to insert post: %v", err), http.StatusInternalServerError)
				return
			}

			// Respond with the created post in JSON format
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			if err := json.NewEncoder(w).Encode(post); err != nil {
				http.Error(w, fmt.Sprintf("Failed to encode post: %v", err), http.StatusInternalServerError)
			}
		} else {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		}
	})

	// Start the server
	fmt.Println("Server is running at http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
