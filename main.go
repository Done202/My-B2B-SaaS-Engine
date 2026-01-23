package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"net/smtp"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xuri/excelize/v2"
	"github.com/jung-kurt/gofpdf"
)

const (
	AdminUser  = "admin"
	AdminPass  = "12345"
	AdminEmail = "admin@example.com"

	senderEmail = "your-email@gmail.com"
	senderPass  = "your-app-password"
)

// ---------------- EMAIL ----------------
func sendWelcomeEmail(to, name string) {
	if to == "" {
		return
	}
	auth := smtp.PlainAuth("", senderEmail, senderPass, "smtp.gmail.com")
	msg := []byte("Subject: Welcome\r\n\r\nHi " + name + ",\nRegistration successful.")
	smtp.SendMail("smtp.gmail.com:587", auth, senderEmail, []string{to}, msg)
}

// ---------------- MAIN ----------------
func main() {

	db, _ := sql.Open("sqlite3", "saas_data.db")
	db.Exec(`CREATE TABLE IF NOT EXISTS customers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		phone TEXT,
		email TEXT,
		remarks TEXT,
		deleted INTEGER DEFAULT 0
	)`)

	// ---------------- LOGIN ----------------
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if r.FormValue("user") == AdminUser && r.FormValue("pass") == AdminPass {
				http.SetCookie(w, &http.Cookie{Name: "session", Value: "active", Path: "/"})
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
		fmt.Fprint(w, `<form method="POST">
			User: <input name="user"><br>
			Pass: <input type="password" name="pass"><br>
			<button>Login</button>
		</form>`)
	})

	// ---------------- ROOT ----------------
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		c, _ := r.Cookie("session")
		if c == nil || c.Value != "active" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		action := r.URL.Query().Get("action")
		id := r.URL.Query().Get("id")
		selection := r.URL.Query().Get("selection")

		// LOGOUT
		if action == "logout" {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// DELETE / RECOVER
		if action == "delete" {
			db.Exec("UPDATE customers SET deleted=1 WHERE id=?", id)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if action == "recover" {
			db.Exec("UPDATE customers SET deleted=0 WHERE id=?", id)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// ---------------- EDIT PAGE ----------------
		if action == "edit" {
			row := db.QueryRow("SELECT name,phone,email,remarks FROM customers WHERE id=?", id)
			var n, p, e, rmk string
			row.Scan(&n, &p, &e, &rmk)

			fmt.Fprintf(w, `
			<h2>Edit Customer</h2>
			<form method="POST" action="/?action=update&id=%s">
				Name: <input name="customerName" value="%s"><br>
				Phone: <input name="customerPhone" value="%s"><br>
				Email: <input name="customerEmail" value="%s"><br>
				Remarks: <input name="customerRemarks" value="%s"><br>
				<button>Update</button>
				<a href="/">Cancel</a>
			</form>`, id, n, p, e, rmk)
			return
		}

		// ---------------- UPDATE ----------------
		if action == "update" && r.Method == "POST" {
			db.Exec(`UPDATE customers 
				SET name=?, phone=?, email=?, remarks=? 
				WHERE id=?`,
				r.FormValue("customerName"),
				r.FormValue("customerPhone"),
				r.FormValue("customerEmail"),
				r.FormValue("customerRemarks"),
				id,
			)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// ---------------- EXPORT ----------------
		if action == "export_excel" || action == "export_pdf" {
			rows, _ := db.Query("SELECT name,phone,email,remarks FROM customers WHERE deleted=0")
			var data [][]string
			sl := 1
			for rows.Next() {
				var n, p, e, r string
				rows.Scan(&n, &p, &e, &r)

				match := selection == ""
				if strings.Contains(selection, "-") {
					parts := strings.Split(selection, "-")
					s, _ := strconv.Atoi(parts[0])
					en, _ := strconv.Atoi(parts[1])
					match = sl >= s && sl <= en
				}

				if match {
					data = append(data, []string{strconv.Itoa(sl), n, p, e, r})
				}
				sl++
			}

			if action == "export_excel" {
				f := excelize.NewFile()
				f.SetSheetRow("Sheet1", "A1", &[]string{"SL", "Name", "Phone", "Email", "Remarks"})
				for i, r := range data {
					f.SetSheetRow("Sheet1", fmt.Sprintf("A%d", i+2), &r)
				}
				w.Header().Set("Content-Disposition", "attachment; filename=report.xlsx")
				f.Write(w)
			} else {
				pdf := gofpdf.New("L", "mm", "A4", "")
				pdf.AddPage()
				pdf.SetFont("Arial", "", 10)
				for _, r := range data {
					for _, c := range r {
						pdf.Cell(40, 10, c)
					}
					pdf.Ln(10)
				}
				w.Header().Set("Content-Type", "application/pdf")
				pdf.Output(w)
			}
			return
		}

		// ---------------- ADD CUSTOMER ----------------
		if r.Method == "POST" {
			name := r.FormValue("customerName")
			if name != "" {
				db.Exec("INSERT INTO customers (name,phone,email,remarks) VALUES (?,?,?,?)",
					name,
					r.FormValue("customerPhone"),
					r.FormValue("customerEmail"),
					r.FormValue("customerRemarks"),
				)
				go sendWelcomeEmail(r.FormValue("customerEmail"), name)
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// ---------------- LIST ----------------
		fmt.Fprintf(w, `
		<h2>B2B Customer Pro</h2>
		<a href="/?action=logout">Logout</a>

		<form method="POST">
			<input name="customerName" placeholder="Name">
			<input name="customerPhone" placeholder="Phone">
			<input name="customerEmail" placeholder="Email">
			<input name="customerRemarks" placeholder="Remarks">
			<button>Add</button>
		</form>

		<table border="1" cellpadding="5">
		<tr><th>SL</th><th>Name</th><th>Phone</th><th>Email</th><th>Remarks</th><th>Actions</th></tr>
		`)

		rows, _ := db.Query("SELECT id,name,phone,email,remarks FROM customers WHERE deleted=0")
		sl := 1
		for rows.Next() {
			var id int
			var n, p, e, r string
			rows.Scan(&id, &n, &p, &e, &r)
			fmt.Fprintf(w, `
			<tr>
				<td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td>
				<td>
					<a href="/?action=edit&id=%d">Edit</a> |
					<a href="/?action=delete&id=%d">Delete</a>
				</td>
			</tr>`, sl, n, p, e, r, id, id)
			sl++
		}
		fmt.Fprint(w, "</table>")
	})

	fmt.Println("Server running on :8080")
	http.ListenAndServe(":8080", nil)
}
