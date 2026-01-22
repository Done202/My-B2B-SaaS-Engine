package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xuri/excelize/v2"
	"github.com/jung-kurt/gofpdf"
)

// বর্তমানে হার্ডকোড করা অ্যাডমিন (পরবর্তীতে এটি ডাটাবেজ থেকে আসবে)
const AdminUser = "admin"
const AdminPass = "12345"
const AdminEmail = "admin@example.com"

func main() {
	db, _ := sql.Open("sqlite3", "saas_data.db")
	// কাস্টমার টেবিল এবং সাবস্ক্রিপশন ইনফো (প্যাকেজ ও স্ট্যাটাস)
	db.Exec("CREATE TABLE IF NOT EXISTS customers (id INTEGER PRIMARY KEY, name TEXT, phone TEXT, email TEXT, remarks TEXT, deleted INTEGER DEFAULT 0)")
	
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if r.FormValue("user") == AdminUser && r.FormValue("pass") == AdminPass {
				http.SetCookie(w, &http.Cookie{Name: "session", Value: "active", Path: "/"})
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
		fmt.Fprintf(w, `<html><body style="font-family:sans-serif; background:#f0f2f5; display:flex; justify-content:center; align-items:center; height:100vh;">
			<form method="POST" style="background:white; padding:40px; border-radius:15px; box-shadow: 0 10px 30px rgba(0,0,0,0.2); width:320px; text-align:center; border: 2px solid #006400;">
				<h2 style="background: linear-gradient(45deg, #006400, #1a73e8); -webkit-background-clip: text; -webkit-text-fill-color: transparent; font-size: 30px;">Admin Login</h2>
				<input type="text" name="user" placeholder="Username" style="width:100%%; padding:12px; margin:10px 0; border:2px solid #ddd; border-radius:6px;" required><br>
				<input type="password" name="pass" placeholder="Password" style="width:100%%; padding:12px; margin:10px 0; border:2px solid #ddd; border-radius:6px;" required><br>
				<button type="submit" style="width:100%%; padding:12px; background:#1a73e8; color:white; border:none; border-radius:6px; cursor:pointer; font-weight:bold;">Login</button>
				<p style="margin-top:15px; font-size:13px;"><a href="/forgot" style="color:#d93025; text-decoration:none;">Forgot Username or Password?</a></p>
			</form></body></html>`)
	})

	http.HandleFunc("/forgot", func(w http.ResponseWriter, r *http.Request) {
		msg := ""
		if r.Method == "POST" {
			email := r.FormValue("email")
			if email == AdminEmail {
				msg = fmt.Sprintf("<div style='color:green; padding:10px;'>Recovery Sent! <br>Username: <b>%s</b> <br>Password: <b>%s</b></div>", AdminUser, AdminPass)
			} else {
				msg = "<div style='color:red;'>Email not found!</div>"
			}
		}
		fmt.Fprintf(w, `<html><body style="font-family:sans-serif; background:#f0f2f5; display:flex; justify-content:center; align-items:center; height:100vh;">
			<div style="background:white; padding:30px; border-radius:15px; box-shadow: 0 10px 30px rgba(0,0,0,0.2); width:350px; text-align:center; border: 2px solid #d93025;">
				<h3 style="color:#d93025;">Account Recovery</h3>
				%s
				<form method="POST">
					<p style="font-size:14px; color:#666;">Enter your registered email to recover account details.</p>
					<input type="email" name="email" placeholder="Your Email" style="width:100%%; padding:12px; margin:10px 0; border:2px solid #ddd; border-radius:6px;" required><br>
					<button type="submit" style="width:100%%; padding:12px; background:#d93025; color:white; border:none; border-radius:6px; cursor:pointer; font-weight:bold;">Recover Now</button>
				</form>
				<a href="/login" style="font-size:13px; color:#1a73e8; text-decoration:none;">Back to Login</a>
			</div></body></html>`, msg)
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
		} else if action == "delete" {
			db.Exec("UPDATE customers SET deleted = 1 WHERE id = ?", id)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		} else if action == "recover" {
			db.Exec("UPDATE customers SET deleted = 0 WHERE id = ?", id)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		if action == "export_excel" || action == "export_pdf" {
			rows, _ := db.Query("SELECT name, phone, email, remarks FROM customers WHERE deleted = 0")
			var data [][]string
			sl := 1
			for rows.Next() {
				var n, p, e, r string
				rows.Scan(&n, &p, &e, &r)
				match := false
				if selection == "" { match = true } else {
					if strings.Contains(selection, "-") {
						parts := strings.Split(selection, "-")
						start, _ := strconv.Atoi(parts[0]); end, _ := strconv.Atoi(parts[1])
						if sl >= start && sl <= end { match = true }
					} else {
						nums := strings.Split(selection, ",")
						for _, numStr := range nums {
							num, _ := strconv.Atoi(strings.TrimSpace(numStr))
							if sl == num { match = true }
						}
					}
				}
				if match { data = append(data, []string{strconv.Itoa(sl), n, p, e, r}) }
				sl++
			}
			if action == "export_excel" {
				f := excelize.NewFile()
				f.SetSheetRow("Sheet1", "A1", &[]string{"SL", "Name", "Phone", "Email", "Remarks"})
				for i, row := range data { f.SetSheetRow("Sheet1", fmt.Sprintf("A%d", i+2), &row) }
				w.Header().Set("Content-Disposition", "attachment; filename=Customers.xlsx")
				f.Write(w)
			} else {
				pdf := gofpdf.New("L", "mm", "A4", "")
				pdf.AddPage(); pdf.SetFont("Arial", "B", 16); pdf.Cell(280, 10, "B2B Customer Report"); pdf.Ln(12)
				headers := []string{"SL", "Name", "Phone", "Email", "Remarks"}; widths := []float64{15, 60, 45, 70, 80}
				pdf.SetFont("Arial", "B", 12)
				for i, h := range headers { pdf.CellFormat(widths[i], 10, h, "1", 0, "C", false, 0, "") }
				pdf.Ln(-1); pdf.SetFont("Arial", "", 11)
				for _, row := range data {
					for i, col := range row { pdf.CellFormat(widths[i], 10, col, "1", 0, "L", false, 0, "") }
					pdf.Ln(-1)
				}
				w.Header().Set("Content-Type", "application/pdf")
				pdf.Output(w)
			}
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
		}

		rows, _ := db.Query("SELECT id, name, phone, email, remarks FROM customers WHERE deleted = 0")
		deletedRows, _ := db.Query("SELECT id, name, phone FROM customers WHERE deleted = 1")
		
		fmt.Fprintf(w, `<html><head><style>
				body { font-family: 'Segoe UI', sans-serif; background: #f0f2f5; padding: 20px; }
				.box { background: white; padding: 30px; border-radius: 15px; max-width: 1050px; margin: auto; box-shadow: 0 15px 35px rgba(0,0,0,0.15); border: 1px solid #ddd; }
				.word-art { background: linear-gradient(45deg, #006400, #1a73e8, #d93025); -webkit-background-clip: text; -webkit-text-fill-color: transparent; text-align: center; font-weight: 900; font-size: 35px; margin-bottom: 5px; }
				.status-bar { text-align: center; margin-bottom: 20px; font-weight: bold; font-size: 14px; color: #555; }
				.export-bar { background: #e8f5e9; padding: 15px; border-radius: 8px; margin-bottom: 20px; display: flex; gap: 10px; align-items: center; border: 2px solid #006400; }
				.btn { padding: 10px 15px; border-radius: 6px; border: none; cursor: pointer; color: white; font-weight: bold; }
				table { width: 100%%; border-collapse: collapse; }
				th, td { text-align: left; padding: 12px; border-bottom: 2px solid #eee; }
				th { background: #f8f9fa; }
			</style></head><body><div class="box">
				<div style="text-align:right;"><a href="/?action=logout" style="color:red; font-weight:bold; text-decoration:none;">[ Logout ]</a></div>
				<h1 class="word-art">B2B Customer Pro</h1>
				<div class="status-bar">
					Package: <span style="color:#1a73e8;">Premium (Unlimited)</span> | Status: <span style="color:green;">● Active</span> | Expiry: <span style="color:#d93025;">Jan 23, 2027</span>
				</div>
				<div class="export-bar">
					<strong>Report:</strong>
					<input type="text" id="sel" placeholder="Range (3-50) or Custom (3,7,10)" style="flex:1; padding:10px; border:2px solid #ddd; border-radius:5px;">
					<button onclick="window.location.href='/?action=export_excel&selection='+document.getElementById('sel').value" class="btn" style="background:#2ecc71;">Excel</button>
					<button onclick="window.location.href='/?action=export_pdf&selection='+document.getElementById('sel').value" class="btn" style="background:#e74c3c;">PDF</button>
				</div>
				<form method="POST" style="display:flex; gap:10px; margin-bottom:20px; flex-wrap:wrap;">
					<input type="text" name="customerName" id="n" placeholder="Name" required style="flex:1; padding:12px; border:2px solid #ddd;">
					<input type="text" name="customerPhone" id="p" placeholder="Phone" style="flex:1; padding:12px; border:2px solid #ddd;">
					<input type="email" name="customerEmail" id="e" placeholder="Email" style="flex:1; padding:12px; border:2px solid #ddd;">
					<input type="text" name="customerRemarks" id="r" placeholder="Remarks" style="flex:1; padding:12px; border:2px solid #ddd;">
					<input type="hidden" name="editID" id="eid">
					<button type="submit" id="mb" class="btn" style="background:#1a73e8;">Save Customer</button>
				</form>
				<table><tr><th>SL</th><th>Name</th><th>Phone</th><th>Email</th><th>Remarks</th><th>Actions</th></tr>`)
		sl := 1
		for rows.Next() {
			var mid int; var name, phone, email, remarks string
			rows.Scan(&mid, &name, &phone, &email, &remarks)
			fmt.Fprintf(w, `<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td>
				<td><a href="#" style="color:#1a73e8; font-weight:bold;" onclick="document.getElementById('n').value='%s'; document.getElementById('p').value='%s'; document.getElementById('e').value='%s'; document.getElementById('r').value='%s'; document.getElementById('eid').value='%d'; document.getElementById('mb').innerText='Update'; return false;">Edit</a> | 
				<a href="/?action=delete&id=%d" style="color:red; font-weight:bold;">Delete</a></td></tr>`, sl, name, phone, email, remarks, name, phone, email, remarks, mid, mid)
			sl++
		}
		fmt.Fprintf(w, `</table>
				<div style="margin-top:30px; border-top:2px solid #d93025; padding-top:10px;">
					<h3 style="color:#d93025;">Recycle Bin</h3>`)
		for deletedRows.Next() {
			var mid int; var name, phone string
			deletedRows.Scan(&mid, &name, &phone)
			fmt.Fprintf(w, `<p><s>%s (%s)</s> <a href="/?action=recover&id=%d" style="color:green; font-weight:bold; margin-left:15px; text-decoration:none;">[ RECOVER ]</a></p>`, name, phone, mid)
		}
		fmt.Fprintf(w, `</div></div></body></html>`)
	})
	http.ListenAndServe(":8080", nil)
}
