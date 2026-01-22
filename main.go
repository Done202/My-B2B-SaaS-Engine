package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
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
			<form method="POST" style="background:white; padding:40px; border-radius:12px; shadow:0 4px 10px rgba(0,0,0,0.1); width:300px; text-align:center;">
				<h2 style="color:#006400;">Admin Login</h2>
				<input type="text" name="user" placeholder="Username" style="width:100%%; padding:10px; margin:10px 0; border:1px solid #ddd; border-radius:5px;" required><br>
				<input type="password" name="pass" placeholder="Password" style="width:100%%; padding:10px; margin:10px 0; border:1px solid #ddd; border-radius:5px;" required><br>
				<button type="submit" style="width:100%%; padding:10px; background:#1a73e8; color:white; border:none; border-radius:5px; cursor:pointer;">Login</button>
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

		if action == "logout" {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		} else if action == "export_excel" || action == "export_pdf" {
			// এক্সপোর্ট লজিক (ভবিষ্যতে লাইব্রেরি দিয়ে পূর্ণ করা হবে, এখন ফরম্যাট দেখাচ্ছে)
			w.Header().Set("Content-Disposition", "attachment; filename=customers_report.txt")
			fmt.Fprintf(w, "Exporting Data Range/Custom: %s\n(Integration ready for Excel/PDF)", r.URL.Query().Get("selection"))
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
			.box { background: white; padding: 25px; border-radius: 12px; max-width: 950px; margin: auto; box-shadow: 0 4px 20px rgba(0,0,0,0.08); }
			.export-section { background:#f9f9f9; padding:15px; border-radius:8px; margin-bottom:20px; border:1px solid #eee; display:flex; gap:10px; align-items:center; }
			.title-green { color: #006400; text-align: center; }
			table { width: 100%%; border-collapse: collapse; margin-top: 10px; }
			th, td { text-align: left; padding: 10px; border-bottom: 1px solid #eee; }
			th { background: #f8f9fa; }
			.btn-ex { padding: 8px 15px; border-radius: 5px; border:none; cursor:pointer; color:white; font-weight:bold; }
		</style></head><body><div class="box">
			<h2 class="title-green">B2B Customer Pro</h2>
			<div style="text-align:right;"><a href="/?action=logout" style="color:red;">Logout</a></div>
			
			<div class="export-section">
				<strong>Export:</strong>
				<input type="text" id="selection" placeholder="Range (3-50) or Custom (3,7,10)" style="flex:1; padding:8px; border-radius:5px; border:1px solid #ddd;">
				<button onclick="location.href='/?action=export_excel&selection='+document.getElementById('selection').value" class="btn-ex" style="background:#2ecc71;">Excel</button>
				<button onclick="location.href='/?action=export_pdf&selection='+document.getElementById('selection').value" class="btn-ex" style="background:#e74c3c;">PDF</button>
			</div>

			<form method="POST" style="display:flex; gap:8px; flex-wrap:wrap; margin-bottom:20px;">
				<input type="text" name="customerName" id="nIn" placeholder="Name" required style="padding:10px; flex:1;">
				<input type="text" name="customerPhone" id="pIn" placeholder="Phone" style="padding:10px; flex:1;">
				<input type="email" name="customerEmail" id="eIn" placeholder="Email" style="padding:10px; flex:1;">
				<input type="text" name="customerRemarks" id="rIn" placeholder="Remarks" style="padding:10px; flex:1;">
				<input type="hidden" name="editID" id="editID">
				<button type="submit" id="mainBtn" style="padding:10px 20px; background:#1a73e8; color:white; border:none; border-radius:6px; cursor:pointer;">Save</button>
			</form>
			<table><tr><th>SL</th><th>Name</th><th>Phone</th><th>Email</th><th>Remarks</th><th>Actions</th></tr>`)
		
		sl := 1
		for rows.Next() {
			var mid int; var name, phone, email, remarks string
			rows.Scan(&mid, &name, &phone, &email, &remarks)
			fmt.Fprintf(w, `<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td>
				<td><a href="#" onclick="document.getElementById('nIn').value='%s'; document.getElementById('pIn').value='%s'; document.getElementById('eIn').value='%s'; document.getElementById('rIn').value='%s'; document.getElementById('editID').value='%d'; document.getElementById('mainBtn').innerText='Update'; return false;">Edit</a> | 
				<a href="/?action=delete&id=%d" style="color:red;">Delete</a></td></tr>`, sl, name, phone, email, remarks, name, phone, email, remarks, mid, mid)
			sl++
		}
		fmt.Fprintf(w, `</table></div></body></html>`)
	})
	http.ListenAndServe(":8080", nil)
}
