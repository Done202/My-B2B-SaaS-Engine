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
	"net/smtp"
)

const AdminUser = "admin"
const AdminPass = "12345"
const AdminEmail = "admin@example.com"

// --- ইমেইল কনফিগারেশন ---
const senderEmail = "your-email@gmail.com" 
const senderPass = "your-app-password" 

func sendWelcomeEmail(to string, name string) {
	if to == "" { return }
	auth := smtp.PlainAuth("", senderEmail, senderPass, "smtp.gmail.com")
	msg := []byte("Subject: Welcome to B2B Customer Pro\r\n" +
		"\r\n" +
		"Hi " + name + ",\r\nRegistration successful.")
	smtp.SendMail("smtp.gmail.com:587", auth, senderEmail, []string{to}, msg)
}

func main() {
	db, _ := sql.Open("sqlite3", "saas_data.db")
	db.Exec("CREATE TABLE IF NOT EXISTS customers (id INTEGER PRIMARY KEY, name TEXT, phone TEXT, email TEXT, remarks TEXT, deleted INTEGER DEFAULT 0)")
	
	// লগইন পেজ (রিকভারি লিঙ্কসহ ফিরে এসেছে)
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if r.FormValue("user") == AdminUser && r.FormValue("pass") == AdminPass {
				http.SetCookie(w, &http.Cookie{Name: "session", Value: "active", Path: "/"})
				http.Redirect(w, r, "/", http.StatusSeeOther); return
			}
		}
		fmt.Fprintf(w, `<html><body style="font-family:sans-serif; background:#f0f2f5; display:flex; justify-content:center; align-items:center; height:100vh;">
			<form method="POST" style="background:white; padding:40px; border-radius:15px; box-shadow: 0 10px 30px rgba(0,0,0,0.2); width:320px; text-align:center;">
				<h2 style="color:#1a73e8;">Admin Login</h2>
				<input type="text" name="user" placeholder="Username" style="width:100%%; padding:12px; margin:10px 0; border:1px solid #ddd; border-radius:5px;"><br>
				<input type="password" name="pass" placeholder="Password" style="width:100%%; padding:12px; margin:10px 0; border:1px solid #ddd; border-radius:5px;"><br>
				<button type="submit" style="width:100%%; padding:12px; background:#1a73e8; color:white; border:none; border-radius:5px; cursor:pointer; font-weight:bold;">Login</button>
				<p style="margin-top:15px; font-size:13px;"><a href="/forgot" style="color:#d93025; text-decoration:none;">Forgot Username or Password?</a></p>
			</form></body></html>`)
	})

	// রিকভারি পেজ লজিক
	http.HandleFunc("/forgot", func(w http.ResponseWriter, r *http.Request) {
		msg := ""
		if r.Method == "POST" {
			if r.FormValue("email") == AdminEmail {
				msg = fmt.Sprintf("<div style='background:#e8f5e9; padding:10px; margin-bottom:10px;'>User: <b>%s</b> | Pass: <b>%s</b></div>", AdminUser, AdminPass)
			} else { msg = "<div style='color:red; margin-bottom:10px;'>Email not found!</div>" }
		}
		fmt.Fprintf(w, `<html><body style="font-family:sans-serif; background:#f0f2f5; display:flex; justify-content:center; align-items:center; height:100vh;">
			<div style="background:white; padding:30px; border-radius:15px; width:350px; text-align:center; box-shadow: 0 10px 30px rgba(0,0,0,0.2);">
				<h3>Account Recovery</h3> %s
				<form method="POST"><input type="email" name="email" placeholder="Admin Email" style="width:100%%; padding:12px; margin-bottom:10px;" required><br>
				<button type="submit" style="width:100%%; padding:12px; background:#d93025; color:white; border:none; border-radius:5px; cursor:pointer;">Recover Now</button></form>
				<a href="/login" style="font-size:13px; color:#1a73e8;">Back to Login</a>
			</div></body></html>`, msg)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cookie, _ := r.Cookie("session")
		if cookie == nil || cookie.Value != "active" { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

		action := r.URL.Query().Get("action")
		id := r.URL.Query().Get("id")
		selection := r.URL.Query().Get("selection")

		if action == "logout" {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
			http.Redirect(w, r, "/login", http.StatusSeeOther); return
		}
		if action == "delete" { db.Exec("UPDATE customers SET deleted = 1 WHERE id = ?", id); http.Redirect(w, r, "/", http.StatusSeeOther); return }
		if action == "recover" { db.Exec("UPDATE customers SET deleted = 0 WHERE id = ?", id); http.Redirect(w, r, "/", http.StatusSeeOther); return }

		// রিপোর্টিং লজিক (অক্ষুণ্ণ আছে)
		if action == "export_excel" || action == "export_pdf" {
			rows, _ := db.Query("SELECT name, phone, email, remarks FROM customers WHERE deleted = 0")
			var data [][]string
			sl := 1
			for rows.Next() {
				var n, p, e, r string; rows.Scan(&n, &p, &e, &r)
				match := false
				if selection == "" { match = true } else if strings.Contains(selection, "-") {
					parts := strings.Split(selection, "-"); s, _ := strconv.Atoi(parts[0]); end, _ := strconv.Atoi(parts[1])
					if sl >= s && sl <= end { match = true }
				} else {
					for _, ns := range strings.Split(selection, ",") {
						num, _ := strconv.Atoi(strings.TrimSpace(ns))
						if sl == num { match = true }
					}
				}
				if match { data = append(data, []string{strconv.Itoa(sl), n, p, e, r}) }
				sl++
			}
			if action == "export_excel" {
				f := excelize.NewFile(); f.SetSheetRow("Sheet1", "A1", &[]string{"SL", "Name", "Phone", "Email", "Remarks"})
				for i, row := range data { f.SetSheetRow("Sheet1", fmt.Sprintf("A%d", i+2), &row) }
				w.Header().Set("Content-Disposition", "attachment; filename=Report.xlsx"); f.Write(w)
			} else {
				pdf := gofpdf.New("L", "mm", "A4", ""); pdf.AddPage(); pdf.SetFont("Arial", "B", 14)
				pdf.Cell(280, 10, "B2B Customer Report"); pdf.Ln(12)
				for _, row := range data {
					for _, col := range row { pdf.CellFormat(40, 10, col, "1", 0, "L", false, 0, "") }
					pdf.Ln(-1)
				}
				w.Header().Set("Content-Type", "application/pdf"); pdf.Output(w)
			}
			return
		}

		if r.Method == "POST" {
			name, phone, email, remarks := r.FormValue("customerName"), r.FormValue("customerPhone"), r.FormValue("customerEmail"), r.FormValue("customerRemarks")
			if name != "" { 
				db.Exec("INSERT INTO customers (name, phone, email, remarks) VALUES (?, ?, ?, ?)", name, phone, email, remarks)
				go sendWelcomeEmail(email, name)
			}
			http.Redirect(w, r, "/", http.StatusSeeOther); return
		}

		expiryDate := time.Now().AddDate(0, 0, 30).Format("02 Jan, 2026")
		rows, _ := db.Query("SELECT id, name, phone, email, remarks FROM customers WHERE deleted = 0")
		deletedRows, _ := db.Query("SELECT id, name, phone FROM customers WHERE deleted = 1")

		fmt.Fprintf(w, `<html><head><style>
				body { font-family: sans-serif; background: #f0f2f5; padding: 20px; }
				.box { background: white; padding: 30px; border-radius: 15px; max-width: 1000px; margin: auto; box-shadow: 0 10px 30px rgba(0,0,0,0.1); }
				.header { position: relative; text-align: center; margin-bottom: 25px; }
				.word-art { font-size: 35px; font-weight: bold; background: linear-gradient(45deg, #006400, #1a73e8, #d93025); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
				.logout-btn { position: absolute; right: 0; top: 10px; color: #A52A2A; font-weight: bold; text-decoration: none; border: 2px solid #A52A2A; padding: 5px 15px; border-radius: 8px; }
				.status-bar { background: #e8f5e9; padding: 15px; border-radius: 8px; margin-bottom: 20px; border: 1px solid #c8e6c9; }
				#searchInp { width: 100%%; padding: 12px; border: 2px solid #1a73e8; border-radius: 8px; margin-bottom: 20px; outline: none; }
				.export-bar { background: #f8f9fa; padding: 15px; border-radius: 8px; margin-bottom: 20px; display: flex; gap: 10px; align-items: center; border: 1px solid #ddd; }
				table { width: 100%%; border-collapse: collapse; }
				th, td { text-align: left; padding: 12px; border-bottom: 1px solid #eee; }
			</style></head><body><div class="box">
				<div class="header">
					<div class="word-art">B2B Customer Pro</div>
					<a href="/?action=logout" class="logout-btn">Logout</a>
				</div>
				<div class="status-bar">
					<div style="text-align:center;">Status: <span style="color:green;">● Active</span> | <span onclick="this.innerText='Expiry: %s'" style="cursor:pointer; color:red; text-decoration:underline;">Check Expiry</span></div>
					<div style="margin-top:10px;">
						<strong>Packages:</strong><br>
						<input type="radio" disabled> Basic: $10/mo (50 Customers)<br>
						<input type="radio" checked> Standard: $25/mo (250 Customers)<br>
						<input type="radio" disabled> Premium: $50/mo (Unlimited)
					</div>
				</div>
				
				<input type="text" id="searchInp" onkeyup="searchTable()" placeholder="Search by Name, Phone or Email...">

				<div class="export-bar">
					<input type="text" id="sel" placeholder="Range (1-10) or Custom (1,3)" style="flex:1; padding:10px;">
					<button onclick="window.location.href='/?action=export_excel&selection='+document.getElementById('sel').value" style="background:#2ecc71; color:white; padding:10px; border:none; border-radius:5px; cursor:pointer;">Excel Download</button>
					<button onclick="window.location.href='/?action=export_pdf&selection='+document.getElementById('sel').value" style="background:#e74c3c; color:white; padding:10px; border:none; border-radius:5px; cursor:pointer;">PDF Download</button>
				</div>

				<form method="POST" style="display:flex; gap:10px; margin-bottom:20px;">
					<input type="text" name="customerName" placeholder="Name" required style="flex:1; padding:10px; border: 1px solid #ddd;">
					<input type="text" name="customerPhone" placeholder="Phone" style="flex:1; padding:10px; border: 1px solid #ddd;">
					<input type="email" name="customerEmail" placeholder="Email" style="flex:1; padding:10px; border: 1px solid #ddd;">
					<input type="text" name="customerRemarks" placeholder="Remarks" style="flex:1; padding:10px; border: 1px solid #ddd;">
					<button type="submit" style="background:#1a73e8; color:white; padding:10px 20px; border:none; border-radius:5px; cursor:pointer; font-weight:bold;">Save Customer</button>
				</form>

				<table id="custTable">
					<tr style="background:#f8f9fa;"><th>SL</th><th>Name</th><th>Phone</th><th>Email</th><th>Remarks</th><th>Actions</th></tr>`, expiryDate)
		sl := 1
		for rows.Next() {
			var mid int; var n, p, e, r string; rows.Scan(&mid, &n, &p, &e, &r)
			fmt.Fprintf(w, `<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td>
				<td><a href="/?action=delete&id=%d" style="color:red; font-weight:bold; text-decoration:none;">Delete</a></td></tr>`, sl, n, p, e, r, mid)
			sl++
		}
		fmt.Fprintf(w, `</table>
			<div style="margin-top:30px; border-top: 1px solid #ddd; padding-top:10px;"><h3 style="color:#d93025;">Recycle Bin</h3>`)
		for deletedRows.Next() {
			var mid int; var n, p string; deletedRows.Scan(&mid, &n, &p)
			fmt.Fprintf(w, `<s>%s (%s)</s> <a href="/?action=recover&id=%d" style="color:green; font-weight:bold; margin-left:10px; text-decoration:none;">[ Recover ]</a><br>`, n, p, mid)
		}
		fmt.Fprintf(w, `</div></div>
			<script>
			function searchTable() {
				var input, filter, table, tr, td, i, j, txtValue;
				input = document.getElementById("searchInp");
				filter = input.value.toUpperCase();
				table = document.getElementById("custTable");
				tr = table.getElementsByTagName("tr");
				for (i = 1; i < tr.length; i++) {
					tr[i].style.display = "none";
					td = tr[i].getElementsByTagName("td");
					for (j = 0; j < td.length; j++) {
						if (td[j] && (td[j].textContent || td[j].innerText).toUpperCase().indexOf(filter) > -1) {
							tr[i].style.display = ""; break;
						}
					}
				}
			}
			</script>
		</body></html>`)
	})
	http.ListenAndServe(":8080", nil)
}
