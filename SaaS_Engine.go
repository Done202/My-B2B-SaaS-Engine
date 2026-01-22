package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// এন্টারপ্রাইজ লেভেল ডাটাবেজ কানেকশন
	db, _ := sql.Open("sqlite3", "enterprise.db")
	db.Exec("CREATE TABLE IF NOT EXISTS logs (id INTEGER PRIMARY KEY, timestamp TEXT, data TEXT)")

	// অ্যাডমিন ড্যাশবোর্ড (পাসওয়ার্ড প্রটেক্টেড)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		pass := r.URL.Query().Get("pass")
		if pass != "admin786" { // আপনার মাস্টার পাসওয়ার্ড
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<h2 style='color:red;'>Access Denied!</h2><p>Please use correct password.</p>")
			return
		}

		rows, _ := db.Query("SELECT * FROM logs ORDER BY id DESC")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<h1>B2B SaaS Enterprise Dashboard</h1><table border='1'><tr><th>ID</th><th>Time</th><th>Data</th></tr>")
		for rows.Next() {
			var id int
			var ts, data string
			rows.Scan(&id, &ts, &data)
			fmt.Fprintf(w, "<tr><td>%d</td><td>%s</td><td>%s</td></tr>", id, ts, data)
		}
		fmt.Fprint(w, "</table>")
	})

	// ডাটা রিসিভার এন্ডপয়েন্ট
	http.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) {
		msg := r.URL.Query().Get("msg")
		ts := time.Now().Format("2006-01-02 15:04:05")
		db.Exec("INSERT INTO logs (timestamp, data) VALUES (?, ?)", ts, msg)
		fmt.Fprint(w, "Status: 200 OK | Data Secured")
	})

	port := os.Getenv("PORT")
	if port == "" { port = "10000" }
	fmt.Println("Enterprise Engine LIVE on port:", port)
	http.ListenAndServe(":"+port, nil)
}
