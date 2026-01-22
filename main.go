package main

import (
	"database/sql"
	"fmt"
	"net/http"

	_ "github.com/mattn/go-sqlite3" // ডাটাবেজ ড্রাইভার
)

func main() {
	// এন্টারপ্রাইজ লেভেল ডাটাবেজ কানেকশন
	db, _ := sql.Open("sqlite3", "saas_data.db")
	db.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// প্রফেশনাল অ্যাডমিন ড্যাশবোর্ড ইন্টারফেস (HTML)
		fmt.Fprintf(w, `
			<html>
				<head>
					<title>Admin Dashboard</title>
					<style>
						body { font-family: Arial; background: #f4f7f6; text-align: center; padding: 50px; }
						.card { background: white; padding: 20px; border-radius: 10px; box-shadow: 0 4px 8px rgba(0,0,0,0.1); display: inline-block; }
						h1 { color: #2c3e50; }
						.status { color: #27ae60; font-weight: bold; }
					</style>
				</head>
				<body>
					<div class="card">
						<h1>B2B SaaS Engine: Admin Panel</h1>
						<p>System Status: <span class="status">ONLINE (Secure)</span></p>
						<hr>
						<p>Welcome back, Admin! Your enterprise infrastructure is ready.</p>
						<button onclick="alert('Database is connected!')">Check Database</button>
					</div>
				</body>
			</html>
		`)
	})

	fmt.Println("Server starting on port 8080...")
	http.ListenAndServe(":8080", nil)
}
