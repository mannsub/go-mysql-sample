package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-sql-driver/mysql"
)

type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var (
	db *sql.DB
	mu sync.Mutex
)

func main() {
	var err error

	dsn := mysql.Config{
		// DBName: {dbname},
		User: "root",
		// Passwd: {password},
		Addr: "localhost:3306",
		Net:  "tcp",
	}
	db, err = sql.Open("mysql", dsn.FormatDSN())
	if err != nil {
		panic(err)
	}

	if err = db.Ping(); err != nil {
		panic(err)
	}

	fmt.Println("Connected to MySQL!")

	http.HandleFunc("/items", handleItems)
	http.HandleFunc("/items/", handleItem)

	http.ListenAndServe(":8080", nil)
}

func handleItems(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		getAllItems(w)
	} else if r.Method == http.MethodPost {
		createItem(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleItem(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Path[len("/items/"):])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		getItem(w, id)
	case http.MethodPut:
		updateItem(w, r, id)
	case http.MethodDelete:
		deleteItem(w, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getAllItems(w http.ResponseWriter) {
	mu.Lock()
	defer mu.Unlock()

	rows, err := db.Query("SELECT id, name FROM items")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}

	json.NewEncoder(w).Encode(items)
}

func createItem(w http.ResponseWriter, r *http.Request) {
	var item Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	res, err := db.Exec("INSERT INTO items (name) VALUES (?)", item.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	item.ID = int(id)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func getItem(w http.ResponseWriter, id int) {
	mu.Lock()
	defer mu.Unlock()

	var item Item
	err := db.QueryRow("SELECT id, name FROM items WHERE id = ?", id).Scan(&item.ID, &item.Name)
	if err == sql.ErrNoRows {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(item)
}

func updateItem(w http.ResponseWriter, r *http.Request, id int) {
	var item Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	_, err := db.Exec("UPDATE items SET name = ? WHERE id = ?", item.Name, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	item.ID = id
	json.NewEncoder(w).Encode(item)
}

func deleteItem(w http.ResponseWriter, id int) {
	mu.Lock()
	defer mu.Unlock()

	_, err := db.Exec("DELETE FROM items WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
