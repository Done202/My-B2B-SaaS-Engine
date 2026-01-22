package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	_ "github.com/mattn/go-sqlite3"
)

const AdminUser = "admin"
const AdminPass = "12345"

func main() {
	db, _ := sql.Open("sqlite3", "saas_data.db")
	db.Exec("CREATE TABLE IF NOT EXISTS customers (id INTEGER PRIMARY KEY, name TEXT, phone TEXT, email TEXT, remarks TEXT, deleted INTEGER DEFAULT 0)")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Logout ও Login চেক (আগের মতোই)
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

		// CSV/Excel ডাউনলোড লজিক
		if r.URL.Query().Get("action") == "download" {
			downloadType := r.FormValue("downloadType") // 'range' or 'custom'
			w.Header().Set("Content-Type", "text/csv")
			w.Header().Set("Content-Disposition", "attachment;filename=customer_report.csv")
			fmt.Fprintln(w, "SL,Name,Phone,Email,Remarks")

			rows, _ := db.Query("SELECT name, phone, email, remarks FROM customers WHERE deleted = 0")
			var allData [][]string
			for rows.Next() {
				var n, p, e, r string
				rows.Scan(&n, &p, &e, &r)
				allData = append(allData, []string{n, p, e, r})
			}

			if downloadType == "range" {
				start, _ := strconv.Atoi(r.FormValue("rangeStart"))
				end, _ := strconv.Atoi(r.FormValue("rangeEnd"))
				for i := start - 1; i < end && i < len(allData); i++ {
					if i >= 0 {
						fmt.Fprintf(w, "%d,%s,%s,%s,%s\n", i+1, allData[i][0], allData[i][1], allData[i][2], allData[i][3])
					}
				}
			} else {
				customSls := strings.Split(r.FormValue("customSls"), ",")
				for _, s := range customSls {
					idx, _ := strconv.Atoi(strings.TrimSpace(s))
					if idx > 0 && idx <= len(allData) {
						fmt.Fprintf(w, "%d,%s,%s,%s,%s\n", idx, allData[idx-1][0], allData[idx-1][1], allData[idx-1][2], allData[idx-1][3])
					}
				}
			}
			return
		}

		// CRUD অপারেশন (আগের মতো)
		action := r.URL.Query().Get("action")
		id := r.URL.Query().Get("id")
		if r.Method == "POST" && r.URL.Query().Get("action") != "download" {
			name := r.FormValue("customerName"); phone := r.FormValue("customerPhone")
			email := r.FormValue("customerEmail"); remarks := r.FormValue("customerRemarks")
			editID := r.FormValue("editID")
			if editID != "" {
				db.Exec("UPDATE customers SET name=?, phone=?, email=?, remarks=? WHERE id=?", name, phone, email, remarks, editID)
			} else {
				db.Exec("INSERT INTO customers (name, phone, email, remarks) VALUES (?, ?, ?, ?)", name, phone, email, remarks)
			}
			http.Redirect(w, r, "/", http.StatusSeeOther); return
		} else if action == "delete" {
			db.Exec("UPDATE customers SET deleted = 1 WHERE id = ?", id)
			http.Redirect(w, r, "/", http.StatusSeeOther); return
		} else if action == "recover" {
			db.Exec("UPDATE customers SET deleted = 0 WHERE id = ?", id)
			http.Redirect(w, r, "/", http.StatusSeeOther); return
		}

		rows, _ := db.Query("SELECT id, name, phone, email, remarks FROM customers WHERE deleted = 0")
		deletedRows, _ := db.Query("SELECT id, name, phone, email FROM customers WHERE deleted = 1")

		fmt.Fprintf(w, `
			<html>
			<head><style>
				body { font-family: 'Segoe UI', sans-serif; background: #f0f2f5; padding: 20px; }
				.box { background: white; padding: 25px; border-radius: 12px; box-shadow: 0 4px 20px rgba(0,0,0,0.08); max-width: 1000px; margin: auto; }
				.title-green { color: #006400; text-align: center; font-weight: bold; margin: 0; }
				.header { display: flex; justify-content: space-between; align-items: center; border-bottom: 2px solid #eee; margin-bottom: 20px; padding-bottom: 10px; }
				.form-group { display: flex; gap: 8px; margin-bottom: 20px; flex-wrap: wrap; }
				input, select { padding: 10px; border-radius: 6px; border: 1px solid #ddd; }
				button { padding: 10px 20px; background: #1a73e8; color: white; border: none; border-radius: 6px; cursor: pointer; font-weight: bold; }
				table { width: 100%%; border-collapse: collapse; margin-top: 10px; }
				th, td { text-align: left; padding: 12px; border-bottom: 1px solid #eee; }
				th { background: #f8f9fa; color: #5f6368; }
				.download-box { background: #e8f0fe; padding: 15px; border-radius: 8px; margin-bottom: 20px; border: 1px solid #1a73e8; }
				.logout-btn { background: #5f6368; color: white; padding: 6px 12px; text-decoration: none; border-radius: 5px; font-size: 0.85em; }
			</style></head>
			<body>
				<div class="box">
					<div class="header">
						<div style="flex:1"></div>
						<h2 class="title-green">B2B Customer Pro</h2>
						<div style="flex:1; text-align:right;"><a href="/?action=logout" class="logout-btn">Logout</a></div>
					</div>

					<div class="download-box">
						<h4 style="margin-top:0; color:#1a73e8;">Download Excel Report</h4>
						<form method="POST" action="/?action=download">
							<div style="display:flex; gap:20px; align-items:center; flex-wrap:wrap;">
								<label><input type="radio" name="downloadType" value="range" checked> Range (SL to SL)</label>
								<input type="number" name="rangeStart" placeholder="Start SL" style="width:80px">
								<input type="number" name="rangeEnd" placeholder="End SL" style="width:80px">
								
								<label style="margin-left:20px;"><input type="radio" name="downloadType" value="custom"> Custom (e.g. 3,7,10)</label>
								<input type="text" name="customSls" placeholder="3, 7, 10..." style="width:150px">
								
								<button type="submit" style="background: #188038;">Download Excel (CSV)</button>
							</div>
						</form>
					</div>

					<form method="POST" class="form-group">
						<input type="text" name="customerName" id="nameIn" placeholder="Name" required>
						<input type="text" name="customerPhone" id="phoneIn" placeholder="Phone">
						<input type="email" name="customerEmail" id="emailIn" placeholder="Email">
						<input type="text" name="customerRemarks" id="remarksIn" placeholder="Remarks">
						<input type="hidden" name="editID" id="editID">
						<button type="submit" id="mainBtn">Save Customer</button>
					</form>

					<h3>Active Directory</h3>
					<table>
						<tr><th>SL</th><th>Name</th><th>Phone</th><th>Email</th><th>Remarks</th><th>Actions</th></tr>`)
		
		sl := 1
		for rows.Next() {
			var mid int; var name, phone, email, remarks string
			rows.Scan(&mid, &name, &phone, &email, &remarks)
			fmt.Fprintf(w, `<tr>
				<td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td>
				<td>
					<a href="#" style="color:#1a73e8; font-weight:bold; text-decoration:none; margin-right:10px;" onclick="document.getElementById('nameIn').value='%s'; document.getElementById('phoneIn').value='%s'; document.getElementById('emailIn').value='%s'; document.getElementById('remarksIn').value='%s'; document.getElementById('editID').value='%d'; document.getElementById('mainBtn').innerText='Update'; return false;">Edit</a>
					<a href="/?action=delete&id=%d" style="color:red; font-weight:bold; text-decoration:none;">Delete</a>
				</td></tr>`, sl, name, phone, email, remarks, name, phone, email, remarks, mid, mid)
			sl++
		}
		fmt.Fprintf(w, `</table>
					<div style="margin-top:40px; color:#d93025; border-top:1px solid #eee; padding-top:10px;">
						<h4>Recycle Bin (Recovery)</h4>`)
		for deletedRows.Next() {
			var mid int; var name, phone, email string
			deletedRows.Scan(&mid, &name, &phone, &email)
			fmt.Fprintf(w, `<p style="font-size:0.85em;"><s>%s (%s)</s> <a href="/?action=recover&id=%d" style="color:green; text-decoration:none; font-weight:bold; margin-left:10px;">Recover</a></p>`, name, phone, mid)
		}
		fmt.Fprintf(w, `</div></div></body></html>`)
	})
	http.ListenAndServe(":8080", nil)
}
