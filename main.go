package main

import (
	"database/sql"
	"fmt"
	"net/http"
	_ "github.com/mattn/go-sqlite3"
)

// এখানে আপনি আপনার পছন্দমতো ইউজারনেম ও পাসওয়ার্ড পরিবর্তন করতে পারেন
const AdminUser = "admin"
const AdminPass = "12345"

func main() {
	db, _ := sql.Open("sqlite3", "saas_data.db")
	db.Exec("CREATE TABLE IF NOT EXISTS customers (id INTEGER PRIMARY KEY, name TEXT, deleted INTEGER DEFAULT 0)")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// লগইন চেক করার লজিক
		user, pass, ok := r.BasicAuth()
		if !ok || user != AdminUser || pass != AdminPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized. Please login with correct credentials.", http.StatusUnauthorized)
			return
		}

		// আপনার আগের কাস্টমার ম্যানেজমেন্ট লজিক
		action := r.URL.Query().Get("action")
		id := r.URL.Query().Get("id")

		if r.Method == "POST" {
			name := r.FormValue("customerName")
			editID := r.FormValue("editID")
			if editID != "" {
				db.Exec("UPDATE customers SET name = ? WHERE id = ?", name, editID)
			} else if name != "" {
				db.Exec("INSERT INTO customers (name) VALUES (?)", name)
			}
		} else if action == "delete" {
			db.Exec("UPDATE customers SET deleted = 1 WHERE id = ?", id)
		} else if action == "recover" {
			db.Exec("UPDATE customers SET deleted = 0 WHERE id = ?", id)
		}

		rows, _ := db.Query("SELECT id, name FROM customers WHERE deleted = 0")
		deletedRows, _ := db.Query("SELECT id, name FROM customers WHERE deleted = 1")

		fmt.Fprintf(w, `
			<html>
			<head><style>
				body { font-family: sans-serif; background: #f4f7f6; padding: 30px; }
				.box { background: white; padding: 20px; border-radius: 10px; box-shadow: 0 5px 15px rgba(0,0,0,0.1); max-width: 600px; margin: auto; text-align: center; }
				input { padding: 10px; width: 60%%; border-radius: 5px; border: 1px solid #ddd; }
				button { padding: 10px 20px; background: #1a73e8; color: white; border: none; border-radius: 5px; cursor: pointer; }
				.item { padding: 10px; border-bottom: 1px solid #eee; display: flex; justify-content: space-between; }
				.del-btn { color: red; text-decoration: none; }
				.rec-btn { color: green; text-decoration: none; }
			</style></head>
			<body>
				<div class="box">
					<h2>Secure Admin Panel</h2>
					<p>Logged in as: <b>%s</b></p>
					<form method="POST">
						<input type="text" name="customerName" placeholder="Customer Name" required>
						<input type="hidden" name="editID" id="editID">
						<button type="submit" id="mainBtn">Add Customer</button>
					</form>
					<div style="text-align:left; margin-top:20px;">
						<h3>Active Customers</h3>`, AdminUser)
		
		for rows.Next() {
			var mid int; var name string
			rows.Scan(&mid, &name)
			fmt.Fprintf(w, `<div class="item"><span>%s</span> 
				<span>
					<a href="/?action=delete&id=%d" class="del-btn">Delete</a>
				</span></div>`, name, mid)
		}
		fmt.Fprintf(w, `<h3>Recycle Bin</h3>`)
		for deletedRows.Next() {
			var mid int; var name string
			deletedRows.Scan(&mid, &name)
			fmt.Fprintf(w, `<div class="item"><s>%s</s> <a href="/?action=recover&id=%d" class="rec-btn">Recover</a></div>`, name, mid)
		}
		fmt.Fprintf(w, `</div></div></body></html>`)
	})

	fmt.Println("Server secured and starting on port 8080...")
	http.ListenAndServe(":8080", nil)
}
