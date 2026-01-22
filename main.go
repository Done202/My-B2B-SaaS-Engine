package main

import (
	"database/sql"
	"fmt"
	"net/http"
	_ "github.com/mattn/go-sqlite3"
)

const AdminUser = "admin"
const AdminPass = "12345"

func main() {
	db, _ := sql.Open("sqlite3", "saas_data.db")
	db.Exec("CREATE TABLE IF NOT EXISTS customers (id INTEGER PRIMARY KEY, name TEXT, deleted INTEGER DEFAULT 0)")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// লগআউট লজিক: ভুল ইউজার পাঠিয়ে ব্রাউজার সেশন ব্রেক করা
		if r.URL.Query().Get("action") == "logout" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "Logged out. <a href='/'>Login again</a>")
			return
		}

		// সিকিউর লগইন চেক
		user, pass, ok := r.BasicAuth()
		if !ok || user != AdminUser || pass != AdminPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		action := r.URL.Query().Get("action")
		id := r.URL.Query().Get("id")

		// CRUD অপারেশন (Add, Edit, Delete, Recover)
		if r.Method == "POST" {
			name := r.FormValue("customerName")
			editID := r.FormValue("editID")
			if editID != "" {
				db.Exec("UPDATE customers SET name = ? WHERE id = ?", name, editID)
			} else if name != "" {
				db.Exec("INSERT INTO customers (name) VALUES (?)", name)
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
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
				body { font-family: 'Segoe UI', sans-serif; background: #f0f2f5; padding: 20px; }
				.box { background: white; padding: 25px; border-radius: 12px; box-shadow: 0 4px 20px rgba(0,0,0,0.08); max-width: 600px; margin: auto; }
				.header { display: flex; justify-content: space-between; align-items: center; border-bottom: 2px solid #eee; margin-bottom: 20px; }
				input { padding: 12px; width: 65%%; border-radius: 6px; border: 1px solid #ddd; }
				button { padding: 12px 20px; background: #1a73e8; color: white; border: none; border-radius: 6px; cursor: pointer; }
				.item { padding: 12px; border-bottom: 1px solid #f0f0f0; display: flex; justify-content: space-between; align-items: center; }
				.edit-btn { color: #1a73e8; text-decoration: none; margin-right: 15px; font-weight: bold; }
				.del-btn { color: #d93025; text-decoration: none; font-weight: bold; }
				.logout-btn { background: #5f6368; color: white; padding: 8px 15px; text-decoration: none; border-radius: 5px; font-size: 0.9em; }
				.recycle-bin { margin-top: 30px; background: #fff8f8; padding: 15px; border-radius: 8px; border: 1px dashed #d93025; }
			</style></head>
			<body>
				<div class="box">
					<div class="header">
						<h2>Secure Admin Panel</h2>
						<a href="/?action=logout" class="logout-btn">Logout</a>
					</div>
					<form method="POST">
						<input type="text" name="customerName" id="customerInput" placeholder="Enter Name" required>
						<input type="hidden" name="editID" id="editID">
						<button type="submit" id="mainBtn">Add</button>
					</form>
					<div style="text-align:left; margin-top:20px;">
						<h3>Active Customers</h3>`)
		
		for rows.Next() {
			var mid int; var name string
			rows.Scan(&mid, &name)
			fmt.Fprintf(w, `
				<div class="item">
					<span>%s</span> 
					<span>
						<a href="#" class="edit-btn" onclick="document.getElementById('customerInput').value='%s'; document.getElementById('editID').value='%d'; document.getElementById('mainBtn').innerText='Update'; return false;">Edit</a>
						<a href="/?action=delete&id=%d" class="del-btn">Delete</a>
					</span>
				</div>`, name, name, mid, mid)
		}
		fmt.Fprintf(w, `
					</div>
					<div class="recycle-bin">
						<h3 style="color:#d93025;">Recycle Bin</h3>`)
		for deletedRows.Next() {
			var mid int; var name string
			deletedRows.Scan(&mid, &name)
			fmt.Fprintf(w, `<div class="item"><s>%s</s> <a href="/?action=recover&id=%d" style="color:green; text-decoration:none; font-weight:bold;">Recover</a></div>`, name, mid)
		}
		fmt.Fprintf(w, `</div></div></body></html>`)
	})

	http.ListenAndServe(":8080", nil)
}
