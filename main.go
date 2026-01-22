package main

import (
	"database/sql"
	"fmt"
	"net/http"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, _ := sql.Open("sqlite3", "saas_data.db")
	// ডাটাবেজে 'deleted' কলাম যুক্ত করা হয়েছে রিকভারি করার জন্য
	db.Exec("CREATE TABLE IF NOT EXISTS customers (id INTEGER PRIMARY KEY, name TEXT, deleted INTEGER DEFAULT 0)")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// অ্যাকশন হ্যান্ডলিং: Add, Edit, Delete, Recover
		action := r.URL.Query().Get("action")
		id := r.URL.Query().Get("id")

		if r.Method == "POST" {
			name := r.FormValue("customerName")
			editID := r.FormValue("editID")
			if editID != "" {
				db.Exec("UPDATE customers SET name = ? WHERE id = ?", name, editID) // Edit Logic
			} else if name != "" {
				db.Exec("INSERT INTO customers (name) VALUES (?)", name) // Add Logic
			}
		} else if action == "delete" {
			db.Exec("UPDATE customers SET deleted = 1 WHERE id = ?", id) // Soft Delete
		} else if action == "recover" {
			db.Exec("UPDATE customers SET deleted = 0 WHERE id = ?", id) // Recovery Logic
		}

		// একটিভ এবং ডিলিটেড কাস্টমার লিস্ট সংগ্রহ
		rows, _ := db.Query("SELECT id, name FROM customers WHERE deleted = 0")
		deletedRows, _ := db.Query("SELECT id, name FROM customers WHERE deleted = 1")

		fmt.Fprintf(w, `
			<html>
			<head><style>
				body { font-family: sans-serif; background: #f4f7f6; padding: 30px; }
				.box { background: white; padding: 20px; border-radius: 10px; box-shadow: 0 5px 15px rgba(0,0,0,0.1); max-width: 600px; margin: auto; }
				input { padding: 10px; width: 60%%; border-radius: 5px; border: 1px solid #ddd; }
				button { padding: 10px 20px; background: #1a73e8; color: white; border: none; border-radius: 5px; cursor: pointer; }
				.list { margin-top: 20px; text-align: left; }
				.item { padding: 10px; border-bottom: 1px solid #eee; display: flex; justify-content: space-between; }
				.del-btn { color: red; text-decoration: none; font-size: 0.8em; }
				.edit-btn { color: blue; text-decoration: none; font-size: 0.8em; margin-right: 10px; }
				.rec-btn { color: green; text-decoration: none; font-size: 0.8em; }
				.deleted-section { margin-top: 30px; background: #fff1f1; padding: 10px; border-radius: 5px; }
			</style></head>
			<body>
				<div class="box">
					<h2>B2B Enterprise Manager</h2>
					<form method="POST">
						<input type="text" name="customerName" placeholder="Customer Name" required>
						<input type="hidden" name="editID" id="editID">
						<button type="submit" id="mainBtn">Add Customer</button>
					</form>

					<div class="list">
						<h3>Active Customers</h3>`)
		for rows.Next() {
			var mid int; var name string
			rows.Scan(&mid, &name)
			fmt.Fprintf(w, `<div class="item"><span>%s</span> 
				<span>
					<a href="#" class="edit-btn" onclick="document.getElementsByName('customerName')[0].value='%s'; document.getElementById('editID').value='%d'; document.getElementById('mainBtn').innerText='Update'; return false;">Edit</a>
					<a href="/?action=delete&id=%d" class="del-btn">Delete</a>
				</span></div>`, name, name, mid, mid)
		}
		fmt.Fprintf(w, `</div><div class="deleted-section"><h3>Recycle Bin (Recovery)</h3>`)
		for deletedRows.Next() {
			var mid int; var name string
			deletedRows.Scan(&mid, &name)
			fmt.Fprintf(w, `<div class="item"><s>%s</s> <a href="/?action=recover&id=%d" class="rec-btn">Recover</a></div>`, name, mid)
		}
		fmt.Fprintf(w, `</div></div></body></html>`)
	})
	http.ListenAndServe(":8080", nil)
}
