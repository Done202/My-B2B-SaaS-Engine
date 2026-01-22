package main

import (
	"database/sql"
	"fmt"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, _ := sql.Open("sqlite3", "saas_data.db")
	db.Exec("CREATE TABLE IF NOT EXISTS customers (id INTEGER PRIMARY KEY, name TEXT)")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			name := r.FormValue("customerName")
			if name != "" {
				db.Exec("INSERT INTO customers (name) VALUES (?)", name)
			}
		}

		// ডাটাবেজ থেকে কাস্টমার লিস্ট নিয়ে আসা
		rows, _ := db.Query("SELECT name FROM customers ORDER BY id DESC")
		defer rows.Close()

		var customerListHTML string
		var count int
		for rows.Next() {
			var name string
			rows.Scan(&name)
			customerListHTML += fmt.Sprintf("<li>%s</li>", name)
			count++
		}

		fmt.Fprintf(w, `
			<html>
				<head>
					<title>Enterprise CRM</title>
					<style>
						body { font-family: 'Segoe UI', sans-serif; background: #f0f2f5; padding: 40px; color: #333; }
						.container { max-width: 500px; margin: auto; background: white; padding: 30px; border-radius: 15px; box-shadow: 0 5px 15px rgba(0,0,0,0.1); }
						h2 { color: #1a73e8; border-bottom: 2px solid #f0f2f5; padding-bottom: 10px; }
						input { width: 70%%; padding: 12px; border: 1px solid #ddd; border-radius: 8px; margin-bottom: 10px; }
						button { background: #1a73e8; color: white; border: none; padding: 12px 20px; border-radius: 8px; cursor: pointer; font-weight: bold; }
						.list-section { text-align: left; margin-top: 30px; }
						ul { list-style: none; padding: 0; }
						li { background: #f8f9fa; padding: 10px; border-bottom: 1px solid #eee; margin-bottom: 5px; border-radius: 5px; }
						.stats { font-size: 0.9em; color: #666; margin-top: 10px; }
					</style>
				</head>
				<body>
					<div class="container">
						<h2>B2B Customer Manager</h2>
						<form method="POST">
							<input type="text" name="customerName" placeholder="Customer Name" required>
							<button type="submit">Add</button>
						</form>
						
						<div class="list-section">
							<h3>Customer List (%d)</h3>
							<ul>%s</ul>
						</div>
						<p class="stats">Database: <b>SQLite Active</b></p>
					</div>
				</body>
			</html>
		`, count, customerListHTML)
	})

	fmt.Println("Server running on port 8080...")
	http.ListenAndServe(":8080", nil)
}
