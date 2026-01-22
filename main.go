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
	// ডাটাবেজে নতুন কলাম: phone এবং email যুক্ত করা হয়েছে
	db.Exec("CREATE TABLE IF NOT EXISTS customers (id INTEGER PRIMARY KEY, name TEXT, phone TEXT, email TEXT, deleted INTEGER DEFAULT 0)")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") == "logout" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, "Logged out. <a href='/'>Login again</a>")
			return
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != AdminUser || pass != AdminPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		action := r.URL.Query().Get("action")
		id := r.URL.Query().Get("id")

		if r.Method == "POST" {
			name := r.FormValue("customerName")
			phone := r.FormValue("customerPhone")
			email := r.FormValue("customerEmail")
			editID := r.FormValue("editID")
			if editID != "" {
				db.Exec("UPDATE customers SET name=?, phone=?, email=? WHERE id=?", name, phone, email, editID)
			} else if name != "" {
				db.Exec("INSERT INTO customers (name, phone, email) VALUES (?, ?, ?)", name, phone, email)
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		} else if action == "delete" {
			db.Exec("UPDATE customers SET deleted = 1 WHERE id = ?", id)
		} else if action == "recover" {
			db.Exec("UPDATE customers SET deleted = 0 WHERE id = ?", id)
		}

		rows, _ := db.Query("SELECT id, name, phone, email FROM customers WHERE deleted = 0")
		deletedRows, _ := db.Query("SELECT id, name FROM customers WHERE deleted = 1")

		fmt.Fprintf(w, `
			<html>
			<head><style>
				body { font-family: 'Segoe UI', sans-serif; background: #f0f2f5; padding: 20px; }
				.box { background: white; padding: 25px; border-radius: 12px; box-shadow: 0 4px 20px rgba(0,0,0,0.08); max-width: 800px; margin: auto; }
				.header { display: flex; justify-content: space-between; align-items: center; border-bottom: 2px solid #eee; margin-bottom: 20px; }
				.form-group { display: flex; gap: 10px; margin-bottom: 20px; flex-wrap: wrap; }
				input { padding: 12px; flex: 1; min-width: 150px; border-radius: 6px; border: 1px solid #ddd; }
				button { padding: 12px 25px; background: #1a73e8; color: white; border: none; border-radius: 6px; cursor: pointer; font-weight: bold; }
				table { width: 100%%; border-collapse: collapse; margin-top: 10px; }
				th, td { text-align: left; padding: 12px; border-bottom: 1px solid #eee; }
				th { background: #f8f9fa; color: #5f6368; }
				.edit-link { color: #1a73e8; text-decoration: none; margin-right: 10px; font-weight: bold; }
				.del-link { color: #d93025; text-decoration: none; font-weight: bold; }
				.logout-btn { background: #5f6368; color: white; padding: 8px 15px; text-decoration: none; border-radius: 5px; }
			</style></head>
			<body>
				<div class="box">
					<div class="header">
						<h2>B2B Customer Pro</h2>
						<a href="/?action=logout" class="logout-btn">Logout</a>
					</div>
					<form method="POST" class="form-group">
						<input type="text" name="customerName" id="nameIn" placeholder="Name" required>
						<input type="text" name="customerPhone" id="phoneIn" placeholder="Phone">
						<input type="email" name="customerEmail" id="emailIn" placeholder="Email">
						<input type="hidden" name="editID" id="editID">
						<button type="submit" id="mainBtn">Save Customer</button>
					</form>
					<h3>Active Directory</h3>
					<table>
						<tr><th>Name</th><th>Phone</th><th>Email</th><th>Actions</th></tr>`)
		for rows.Next() {
			var mid int; var name, phone, email string
			rows.Scan(&mid, &name, &phone, &email)
			fmt.Fprintf(w, `<tr>
				<td>%s</td><td>%s</td><td>%s</td>
				<td>
					<a href="#" class="edit-link" onclick="document.getElementById('nameIn').value='%s'; document.getElementById('phoneIn').value='%s'; document.getElementById('emailIn').value='%s'; document.getElementById('editID').value='%d'; document.getElementById('mainBtn').innerText='Update'; return false;">Edit</a>
					<a href="/?action=delete&id=%d" class="del-link">Delete</a>
				</td></tr>`, name, phone, email, name, phone, email, mid, mid)
		}
		fmt.Fprintf(w, `</table>
					<div style="margin-top:40px; color:#d93025;">
						<h4>Deleted Records (Recycle Bin)</h4>`)
		for deletedRows.Next() {
			var mid int; var name, phone, email string
			deletedRows.Scan(&mid, &name, &phone, &email)
			fmt.Fprintf(w, `<p style="font-size:0.9em;"><s>%s (%s)</s> <a href="/?action=recover&id=%d" style="color:green; text-decoration:none;">Recover</a></p>`, name, phone, mid)
		}
		fmt.Fprintf(w, `</div></div></body></html>`)
	})
	http.ListenAndServe(":8080", nil)
}
