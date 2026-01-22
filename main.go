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
		// ডাটা সেভ করার লজিক
		if r.Method == "POST" {
			name := r.FormValue("customerName")
			db.Exec("INSERT INTO customers (name) VALUES (?)", name)
		}

		// ড্যাশবোর্ড ইন্টারফেস
		var customerCount int
		db.QueryRow("SELECT COUNT(*) FROM customers").Scan(&customerCount)

		fmt.Fprintf(w, `
			<html>
				<head>
					<title>Customer Manager</title>
					<style>
						body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: #eef2f3; text-align: center; padding: 40px; }
						.card { background: white; padding: 30px; border-radius: 15px; box-shadow: 0 10px 25px rgba(0,0,0,0.1); display: inline-block; width: 350px; }
						input { padding: 10px; width: 80%%; margin-bottom: 10px; border: 1px solid #ddd; border-radius: 5px; }
						button { background: #3498db; color: white; border: none; padding: 10px 20px; border-radius: 5px; cursor: pointer; }
						.counter { font-size: 24px; color: #2ecc71; font-weight: bold; margin-top: 20px; }
					</style>
				</head>
				<body>
					<div class="card">
						<h2>B2B Customer Manager</h2>
						<form method="POST">
							<input type="text" name="customerName" placeholder="Enter Customer Name" required>
							<button type="submit">Add Customer</button>
						</form>
						<div class="counter">Total Customers: %d</div>
						<p>Status: <span style="color:green;">Connected to SQLite</span></p>
					</div>
				</body>
			</html>
		`, customerCount)
	})

	fmt.Println("Server starting on port 8080...")
	http.ListenAndServe(":8080", nil)
}
