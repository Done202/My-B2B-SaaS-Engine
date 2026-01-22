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
	db.Exec("CREATE TABLE IF NOT EXISTS customers (id INTEGER PRIMARY KEY, name TEXT, phone TEXT, email TEXT, remarks TEXT, deleted INTEGER DEFAULT 0)")

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if r.FormValue("user") == AdminUser && r.FormValue("pass") == AdminPass {
				cookie := &http.Cookie{Name: "session", Value: "active", Path: "/"}
				http.SetCookie(w, cookie)
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
		fmt.Fprintf(w, `<html><body style="font-family:sans-serif; background:#f0f2f5; display:flex; justify-content:center; align-items:center; height:100vh;">
			<form method="POST" style="background:white; padding:40px; border-radius:12px; box-shadow:0 4px 15px rgba(0,0,0,0.1); width:320px; text-align:center;">
				<h2 style="color:#006400;">B2B Admin Login</h2>
				<input type="text" name="user" placeholder="Username" style="width:100%%; padding:12px; margin:10px 0; border:1px solid #ddd; border-radius:6px;" required><br>
				<input type="password" name="pass" placeholder="Password" style="width:100%%; padding:12px; margin:10px 0; border:1px solid #ddd; border-radius:6px;" required><br>
				<button type="submit" style="width:100%%; padding:12px; background:#1a73e8; color:white; border:none; border-radius:6px; cursor:pointer; font-weight:bold;">Login</button>
			</form></body></html>`)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil || cookie.Value != "active" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		action := r.URL.Query().Get("action")
		id := r.URL.Query().Get("id")
		selection := r.URL.Query().Get("selection")

		if action == "logout" {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if action == "export_excel" || action == "export_pdf" {
			w.Header().Set("Content-Disposition", "attachment; filename=Report.txt")
			fmt.Fprintf(w, "Selection: %s\nActual Excel/PDF generation starting...", selection)
			return
		}

		if r.Method == "POST" {
			name, phone, email, remarks := r.FormValue("customerName"), r.FormValue("customerPhone"), r.FormValue("customerEmail"), r.FormValue("customerRemarks")
			editID := r.FormValue("editID")
			if editID != "" {
				db.Exec("UPDATE customers SET name=?, phone=?, email=?, remarks=? WHERE id=?", name, phone, email, remarks, editID)
			} else if name != "" {
				db.Exec("INSERT INTO customers (name, phone, email, remarks) VALUES (?, ?, ?, ?)", name, phone, email, remarks)
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		} else if action == "delete" {
			db.Exec("UPDATE customers SET deleted = 1 WHERE id = ?", id)
		} else if action == "recover" {
			db.Exec("UPDATE customers SET deleted = 0 WHERE id = ?", id)
		}

		rows, _ := db.Query("SELECT id, name, phone, email, remarks FROM customers WHERE deleted = 0")
		
		fmt.Fprintf(w, `<html><head><style>
				body { font-family: 'Segoe UI', sans-serif; background: #f0f2f5; padding: 20px; }
				.box { background: white; padding: 25px; border-radius: 12px; max-width: 1000px; margin: auto; box-shadow: 0 4px 20px rgba(0,0,0,0.08); }
				.title-green { color: #006400; text-align: center; font-weight: bold; font-size: 28px; }
				.export-bar { background: #e8f5e9; padding: 15px; border-radius: 8px; margin-bottom: 20px; display: flex; gap: 10px; align-items: center; border: 1px solid #c8e6c9; }
				table { width: 100%%; border-collapse: collapse; }
				th, td { text-align: left; padding: 12px; border-bottom: 1px solid #eee; }
				.btn { padding: 10px 15px; border-radius: 6px; border: none; cursor: pointer; color: white; font-weight: bold; }
			</style></head><body><div class="box">
				<div style="text-align:right;"><a href="/?action=logout" style="color:red; text-decoration:none;">Logout</a></div>
				<h1 class="title-green">B2B Customer Pro</h1>
				<div class="export-bar">
					<strong>Report:</strong>
					<input type="text" id="sel" placeholder="Range (3-50) or Custom (3,7,10)" style="flex:1; padding:10px; border-radius:5px; border:1px solid #ddd;">
					<button onclick="window.location.href='/?action=export_excel&selection='+document.getElementById('sel').value" class="btn" style="background:#2ecc71;">Excel</button>
					<button onclick="window.location.href='/?action=export_pdf&selection='+document.getElementById('sel').value" class="btn" style="background:#e74c3c;">PDF</button>
				</div>
				<form method="POST" style="display:flex; gap:10px; margin-bottom:20px;">
					<input type="text" name="customerName" id="n" placeholder="Name" required style="flex:1; padding:10px;">
					<input type="text" name="customerPhone" id="p" placeholder="Phone" style="flex:1; padding:10px;">
					<input type="email" name="customerEmail" id="e" placeholder="Email" style="flex:1; padding:10px;">
					<input type="text" name="customerRemarks" id="r" placeholder="Remarks" style="flex:1; padding:10px;">
					<input type="hidden" name="editID" id="eid">
					<button type="submit" id="mb" class="btn" style="background:#1a73e8;">Save</button>
				</form>
				<table><tr><th>SL</th><th>Name</th><th>Phone</th><th>Email</th><th>Remarks</th><th>Actions</th></tr>`)
		sl := 1
		for rows.Next() {
			var mid int; var name, phone, email, remarks string
			rows.Scan(&mid, &name, &phone, &email, &remarks)
			fmt.Fprintf(w, `<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td>
				<td><a href="#" onclick="document.getElementById('n').value='%s'; document.getElementById('p').value='%s'; document.getElementById('e').value='%s'; document.getElementById('r').value='%s'; document.getElementById('eid').value='%d'; document.getElementById('mb').innerText='Update'; return false;">Edit</a> | 
				<a href="/?action=delete&id=%d" style="color:red;">Delete</a></td></tr>`, sl, name, phone, email, remarks, name, phone, email, remarks, mid, mid)
			sl++
		}
		fmt.Fprintf(w, `</table></div></body></html>`)
	})
	http.ListenAndServe(":8080", nil)
}
