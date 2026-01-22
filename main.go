package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xuri/excelize/v2"
	"github.com/jung-kurt/gofpdf"
)

const AdminUser = "admin"
const AdminPass = "12345"

func main() {
	db, _ := sql.Open("sqlite3", "saas_data.db")
	db.Exec("CREATE TABLE IF NOT EXISTS customers (id INTEGER PRIMARY KEY, name TEXT, phone TEXT, email TEXT, remarks TEXT, deleted INTEGER DEFAULT 0)")
	
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if r.FormValue("user") == AdminUser && r.FormValue("pass") == AdminPass {
				http.SetCookie(w, &http.Cookie{Name: "session", Value: "active", Path: "/"})
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
		fmt.Fprintf(w, `<html><body style="font-family:sans-serif; background:#f4f7f6; display:flex; justify-content:center; align-items:center; height:100vh;">
			<form method="POST" style="background:white; padding:40px; border-radius:12px; box-shadow: 0 5px 20px rgba(0,0,0,0.1); width:320px; text-align:center; border-top: 5px solid #1a73e8;">
				<h2 style="color:#333;">Admin Access</h2>
				<input type="text" name="user" placeholder="Username" style="width:100%%; padding:12px; margin:10px 0; border:1px solid #ddd; border-radius:4px;" required><br>
				<input type="password" name="pass" placeholder="Password" style="width:100%%; padding:12px; margin:10px 0; border:1px solid #ddd; border-radius:4px;" required><br>
				<button type="submit" style="width:100%%; padding:12px; background:#1a73e8; color:white; border:none; border-radius:4px; cursor:pointer; font-weight:bold;">LOGIN</button>
			</form></body></html>`)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cookie, _ := r.Cookie("session")
		if cookie == nil || cookie.Value != "active" { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

		action := r.URL.Query().Get("action")
		id := r.URL.Query().Get("id")

		if action == "logout" {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
			http.Redirect(w, r, "/login", http.StatusSeeOther); return
		}
		if action == "delete" { db.Exec("UPDATE customers SET deleted = 1 WHERE id = ?", id); http.Redirect(w, r, "/", http.StatusSeeOther); return }
		if action == "recover" { db.Exec("UPDATE customers SET deleted = 0 WHERE id = ?", id); http.Redirect(w, r, "/", http.StatusSeeOther); return }

		// Excel & PDF logic remained same as before
		if action == "export_excel" || action == "export_pdf" {
			// (আপনার আগের কোডের এক্সপোর্ট লজিক এখানে থাকবে)
			return
		}

		if r.Method == "POST" {
			name, phone, email, remarks := r.FormValue("customerName"), r.FormValue("customerPhone"), r.FormValue("customerEmail"), r.FormValue("customerRemarks")
			editID := r.FormValue("editID")
			if editID != "" { db.Exec("UPDATE customers SET name=?, phone=?, email=?, remarks=? WHERE id=?", name, phone, email, remarks, editID)
			} else if name != "" { db.Exec("INSERT INTO customers (name, phone, email, remarks) VALUES (?, ?, ?, ?)", name, phone, email, remarks) }
			http.Redirect(w, r, "/", http.StatusSeeOther); return
		}

		expiryDate := time.Now().AddDate(0, 0, 30).Format("02 Jan, 2026")
		rows, _ := db.Query("SELECT id, name, phone, email, remarks FROM customers WHERE deleted = 0")
		deletedRows, _ := db.Query("SELECT id, name, phone FROM customers WHERE deleted = 1")

		fmt.Fprintf(w, `<html><head><style>
				body { font-family: 'Segoe UI', sans-serif; background: #f0f2f5; padding: 20px; }
				.box { background: white; padding: 30px; border-radius: 15px; max-width: 1000px; margin: auto; box-shadow: 0 10px 30px rgba(0,0,0,0.1); }
				.header { display: flex; justify-content: space-between; align-items: center; border-bottom: 2px solid #eee; padding-bottom: 10px; margin-bottom: 20px; }
				.word-art { background: linear-gradient(45deg, #006400, #1a73e8); -webkit-background-clip: text; -webkit-text-fill-color: transparent; font-size: 30px; font-weight: bold; }
				.logout-btn { color: #A52A2A; font-size: 22px; font-weight: bold; text-decoration: none; border: 2px solid #A52A2A; padding: 5px 15px; border-radius: 8px; }
				.status-bar { background: #f9f9f9; padding: 10px; border-radius: 8px; margin-bottom: 20px; font-weight: bold; font-size: 15px; color: #444; }
				table { width: 100%%; border-collapse: collapse; margin-top: 20px; }
				th, td { text-align: left; padding: 12px; border-bottom: 1px solid #ddd; }
				.btn { padding: 10px 20px; border-radius: 5px; border: none; cursor: pointer; color: white; font-weight: bold; }
			</style></head><body><div class="box">
				<div class="header">
					<div class="word-art">B2B Customer Pro</div>
					<a href="/?action=logout" class="logout-btn">Logout</a>
				</div>
				<div class="status-bar">
					Package: <select style="padding:5px; border-radius:5px; border:1px solid #1a73e8; font-weight:bold;">
						<option>Basic $10 (50 Customers)</option>
						<option selected>Standard $25 (250 Customers)</option>
						<option>Premium $50 (Unlimited)</option>
					</select> | Status: <span style="color:green;">● Active</span> | 
					<span id="exp" style="cursor:pointer; color:#d93025; text-decoration:underline;" onclick="this.innerText='Expiry: %s'">Check Expiry</span>
				</div>
				<form method="POST" style="display:flex; gap:10px; flex-wrap:wrap;">
					<input type="text" name="customerName" id="n" placeholder="Name" required style="flex:1; padding:12px; border:1px solid #ddd;">
					<input type="text" name="customerPhone" id="p" placeholder="Phone" style="flex:1; padding:12px; border:1px solid #ddd;">
					<input type="email" name="customerEmail" id="e" placeholder="Email" style="flex:1; padding:12px; border:1px solid #ddd;">
					<input type="text" name="customerRemarks" id="r" placeholder="Remarks" style="flex:1; padding:12px; border:1px solid #ddd;">
					<input type="hidden" name="editID" id="eid">
					<button type="submit" id="mb" class="btn" style="background:#1a73e8;">Save</button>
				</form>
				<table><tr><th>SL</th><th>Name</th><th>Phone</th><th>Email</th><th>Remarks</th><th>Action</th></tr>`, expiryDate)
		sl := 1
		for rows.Next() {
			var mid int; var n, p, e, r string; rows.Scan(&mid, &n, &p, &e, &r)
			fmt.Fprintf(w, `<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td>
				<td><a href="#" style="color:#1a73e8;" onclick="document.getElementById('n').value='%s'; document.getElementById('p').value='%s'; document.getElementById('e').value='%s'; document.getElementById('r').value='%s'; document.getElementById('eid').value='%d'; document.getElementById('mb').innerText='Update'; return false;">Edit</a> | 
				<a href="/?action=delete&id=%d" style="color:red;">Delete</a></td></tr>`, sl, n, p, e, r, n, p, e, r, mid, mid)
			sl++
		}
		fmt.Fprintf(w, `</table><div style="margin-top:40px; border-top:2px solid #d93025; padding-top:10px;"><h3 style="color:#d93025;">Recycle Bin</h3>`)
		for deletedRows.Next() {
			var mid int; var n, p string; deletedRows.Scan(&mid, &n, &p)
			fmt.Fprintf(w, `<p><s>%s (%s)</s> <a href="/?action=recover&id=%d" style="color:green; font-weight:bold; margin-left:15px;">[ RECOVER ]</a></p>`, n, p, mid)
		}
		fmt.Fprintf(w, `</div></div></body></html>`)
	})
	http.ListenAndServe(":8080", nil)
}
