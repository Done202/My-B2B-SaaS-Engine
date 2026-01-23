package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/xuri/excelize/v2"
	"github.com/jung-kurt/gofpdf"
)

// ৩০ ডিজিটের একটি গোপন চাবি (Security Key)
const secretKey = "a-very-secret-key-32-characters!!" 

const (
	AdminUser  = "admin"
	AdminPass  = "12345"
	AdminEmail = "admin@example.com"
	senderEmail = "your-email@gmail.com"
	senderPass  = "your-app-password"
)

// ---------------- ENCRYPTION LOGIC ----------------
func encrypt(text string) string {
	block, _ := aes.NewCipher([]byte(secretKey))
	ciphertext := make([]byte, aes.BlockSize+len(text))
	iv := ciphertext[:aes.BlockSize]
	io.ReadFull(rand.Reader, iv)
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], []byte(text))
	return base64.URLEncoding.EncodeToString(ciphertext)
}

func decrypt(cryptoText string) string {
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)
	block, _ := aes.NewCipher([]byte(secretKey))
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(ciphertext, ciphertext)
	return string(ciphertext)
}

// ---------------- EMAIL ----------------
func sendWelcomeEmail(to, name string) {
	if to == "" { return }
	auth := smtp.PlainAuth("", senderEmail, senderPass, "smtp.gmail.com")
	msg := []byte("Subject: Welcome to B2B Pro\r\n\r\nHi " + name + ",\nRegistration successful.")
	smtp.SendMail("smtp.gmail.com:587", auth, senderEmail, []string{to}, msg)
}

func main() {
	db, _ := sql.Open("sqlite3", "saas_data.db")
	db.Exec(`CREATE TABLE IF NOT EXISTS customers (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, phone TEXT, email TEXT, remarks TEXT, deleted INTEGER DEFAULT 0)`)

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if r.FormValue("user") == AdminUser && r.FormValue("pass") == AdminPass {
				http.SetCookie(w, &http.Cookie{Name: "session", Value: "active", Path: "/"})
				http.Redirect(w, r, "/", http.StatusSeeOther); return
			}
		}
		fmt.Fprint(w, `<html><body style="font-family:sans-serif; background:#f0f2f5; display:flex; justify-content:center; align-items:center; height:100vh;">
			<form method="POST" style="background:white; padding:40px; border-radius:15px; width:320px; text-align:center; box-shadow:0 10px 30px rgba(0,0,0,0.1);">
				<h2 style="color:#1a73e8;">Admin Login</h2>
				<input name="user" placeholder="User" style="width:100%; padding:10px; margin:10px 0;"><br>
				<input type="password" name="pass" placeholder="Pass" style="width:100%; padding:10px; margin:10px 0;"><br>
				<button style="width:100%; padding:10px; background:#1a73e8; color:white; border:none; border-radius:5px; cursor:pointer;">Login</button>
				<p style="margin-top:15px; font-size:13px;"><a href="/forgot" style="color:#d93025; text-decoration:none;">Forgot Username or Password?</a></p>
			</form></body></html>`)
	})

	http.HandleFunc("/forgot", func(w http.ResponseWriter, r *http.Request) {
		msg := ""
		if r.Method == "POST" && r.FormValue("email") == AdminEmail {
			msg = fmt.Sprintf("<div style='color:green; margin-bottom:10px;'>User: %s | Pass: %s</div>", AdminUser, AdminPass)
		}
		fmt.Fprintf(w, `<html><body style="font-family:sans-serif; background:#f0f2f5; display:flex; justify-content:center; align-items:center; height:100vh;">
			<div style="background:white; padding:30px; border-radius:15px; text-align:center;">
				<h3>Recovery</h3>%s
				<form method="POST"><input name="email" placeholder="Admin Email" style="padding:10px; margin:10px 0; width:100%%;"><br>
				<button style="background:#d93025; color:white; border:none; padding:10px; width:100%%; cursor:pointer;">Recover</button></form>
				<a href="/login">Back</a>
			</div></body></html>`, msg)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		c, _ := r.Cookie("session")
		if c == nil || c.Value != "active" { http.Redirect(w, r, "/login", http.StatusSeeOther); return }

		action, id, selection := r.URL.Query().Get("action"), r.URL.Query().Get("id"), r.URL.Query().Get("selection")

		if action == "logout" { http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1}); http.Redirect(w, r, "/login", http.StatusSeeOther); return }
		if action == "delete" { db.Exec("UPDATE customers SET deleted=1 WHERE id=?", id); http.Redirect(w, r, "/", http.StatusSeeOther); return }
		if action == "recover" { db.Exec("UPDATE customers SET deleted=0 WHERE id=?", id); http.Redirect(w, r, "/", http.StatusSeeOther); return }

		if action == "edit" {
			var n, p, e, rmk string
			db.QueryRow("SELECT name,phone,email,remarks FROM customers WHERE id=?", id).Scan(&n, &p, &e, &rmk)
			// ডিক্রিপশন করে এডিট বক্সে দেখানো
			fmt.Fprintf(w, `<html><body style="font-family:sans-serif; background:#f0f2f5; padding:40px;">
				<div style="background:white; padding:30px; border-radius:15px; max-width:500px; margin:auto;">
					<h2>Edit Customer</h2>
					<form method="POST" action="/?action=update&id=%s">
						<input name="customerName" value="%s" style="width:100%%; padding:10px; margin:5px 0;"><br>
						<input name="customerPhone" value="%s" style="width:100%%; padding:10px; margin:5px 0;"><br>
						<input name="customerEmail" value="%s" style="width:100%%; padding:10px; margin:5px 0;"><br>
						<input name="customerRemarks" value="%s" style="width:100%%; padding:10px; margin:5px 0;"><br>
						<button style="background:#1a73e8; color:white; padding:10px; border:none; border-radius:5px;">Update</button>
						<a href="/" style="margin-left:10px;">Cancel</a>
					</form></div></body></html>`, id, decrypt(n), decrypt(p), e, rmk)
			return
		}

		if action == "update" && r.Method == "POST" {
			db.Exec("UPDATE customers SET name=?, phone=?, email=?, remarks=? WHERE id=?", encrypt(r.FormValue("customerName")), encrypt(r.FormValue("customerPhone")), r.FormValue("customerEmail"), r.FormValue("customerRemarks"), id)
			http.Redirect(w, r, "/", http.StatusSeeOther); return
		}

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
					s, _ := strconv.Atoi(parts[0]); en, _ := strconv.Atoi(parts[1])
					match = sl >= s && sl <= en
				}
				if match { data = append(data, []string{strconv.Itoa(sl), decrypt(n), decrypt(p), e, r}) }
				sl++
			}
			if action == "export_excel" {
				f := excelize.NewFile(); f.SetSheetRow("Sheet1", "A1", &[]string{"SL", "Name", "Phone", "Email", "Remarks"})
				for i, rd := range data { f.SetSheetRow("Sheet1", fmt.Sprintf("A%d", i+2), &rd) }
				w.Header().Set("Content-Disposition", "attachment; filename=report.xlsx"); f.Write(w)
			} else {
				pdf := gofpdf.New("L", "mm", "A4", ""); pdf.AddPage(); pdf.SetFont("Arial", "B", 12)
				for _, rd := range data {
					for _, col := range rd { pdf.Cell(40, 10, col) }
					pdf.Ln(10)
				}
				w.Header().Set("Content-Type", "application/pdf"); pdf.Output(w)
			}
			return
		}

		if r.Method == "POST" {
			name := r.FormValue("customerName")
			if name != "" {
				db.Exec("INSERT INTO customers (name,phone,email,remarks) VALUES (?,?,?,?)", encrypt(name), encrypt(r.FormValue("customerPhone")), r.FormValue("customerEmail"), r.FormValue("customerRemarks"))
				go sendWelcomeEmail(r.FormValue("customerEmail"), name)
			}
			http.Redirect(w, r, "/", http.StatusSeeOther); return
		}

		expiryDate := time.Now().AddDate(0, 0, 30).Format("02 Jan, 2026")
		fmt.Fprintf(w, `<html><head><style>
			body { font-family: sans-serif; background: #f0f2f5; padding: 20px; }
			.box { background: white; padding: 30px; border-radius: 15px; max-width: 1000px; margin: auto; box-shadow: 0 10px 30px rgba(0,0,0,0.1); }
			.header { position: relative; text-align: center; margin-bottom: 25px; }
			.word-art { font-size: 32px; font-weight: bold; color: #1a73e8; }
			.logout-btn { position: absolute; right: 0; top: 0; color: #d93025; border: 1px solid #d93025; padding: 5px 15px; border-radius: 5px; text-decoration: none; }
			.status-bar { background: #e8f5e9; padding: 15px; border-radius: 8px; margin-bottom: 20px; border: 1px solid #c8e6c9; }
			.report-bar { background: #f8f9fa; padding: 15px; border-radius: 8px; margin-bottom: 20px; display: flex; gap: 10px; border: 1px solid #ddd; }
			#searchInp { width: 100%%; padding: 12px; border: 2px solid #1a73e8; border-radius: 8px; margin-bottom: 20px; }
			table { width: 100%%; border-collapse: collapse; }
			th, td { text-align: left; padding: 12px; border-bottom: 1px solid #eee; }
		</style></head><body><div class="box">
			<div class="header">
				<div class="word-art">B2B Customer Pro</div>
				<a href="/?action=logout" class="logout-btn">Logout</a>
			</div>
			<div class="status-bar">
				<div style="text-align:center;">Status: <span style="color:green;">● Active</span> | <span onclick="this.innerText='Expiry: %s'" style="cursor:pointer; color:red; text-decoration:underline;">Check Expiry</span></div>
			</div>
			<input type="text" id="searchInp" onkeyup="searchTable()" placeholder="Search...">
			<div class="report-bar">
				<input type="text" id="sel" placeholder="Range (1-10)" style="flex:1; padding:10px;">
				<button onclick="window.location.href='/?action=export_excel&selection='+document.getElementById('sel').value" style="background:#2ecc71; color:white; padding:10px; border:none; border-radius:5px; cursor:pointer;">Excel</button>
				<button onclick="window.location.href='/?action=export_pdf&selection='+document.getElementById('sel').value" style="background:#e74c3c; color:white; padding:10px; border:none; border-radius:5px; cursor:pointer;">PDF</button>
			</div>
			<form method="POST" style="display:flex; gap:10px; margin-bottom:20px;">
				<input name="customerName" placeholder="Name" required style="flex:1; padding:10px;">
				<input name="customerPhone" placeholder="Phone" style="flex:1; padding:10px;">
				<input name="customerEmail" placeholder="Email" style="flex:1; padding:10px;">
				<input name="customerRemarks" placeholder="Remarks" style="flex:1; padding:10px;">
				<button style="background:#1a73e8; color:white; border:none; padding:10px 20px; border-radius:5px; cursor:pointer;">Save</button>
			</form>
			<table id="custTable">
			<tr style="background:#f8f9fa;"><th>SL</th><th>Name</th><th>Phone</th><th>Email</th><th>Remarks</th><th>Actions</th></tr>`, expiryDate)
		
		rows, _ := db.Query("SELECT id,name,phone,email,remarks FROM customers WHERE deleted=0")
		sl := 1
		for rows.Next() {
			var id int; var n, p, e, r string
			rows.Scan(&id, &n, &p, &e, &r)
			fmt.Fprintf(w, `<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td>
				<td><a href="/?action=edit&id=%d" style="color:#1a73e8;">Edit</a> | 
				<a href="/?action=delete&id=%d" style="color:#d93025;">Delete</a></td></tr>`, sl, decrypt(n), decrypt(p), e, r, id, id)
			sl++
		}
		fmt.Fprint(w, `</table>
			<div style="margin-top:30px; border-top: 1px solid #ddd; padding-top:10px;"><h3 style="color:#d93025;">Recycle Bin</h3>`)
		delRows, _ := db.Query("SELECT id,name,phone FROM customers WHERE deleted=1")
		for delRows.Next() {
			var did int; var dn, dp string
			delRows.Scan(&did, &dn, &dp)
			fmt.Fprintf(w, `<div><s>%s (%s)</s> <a href="/?action=recover&id=%d" style="color:green;">[ Recover ]</a></div>`, decrypt(dn), decrypt(dp), did)
		}
		fmt.Fprint(w, `</div></div>
			<script>
			function searchTable() {
				var input, filter, table, tr, td, i, j;
				input = document.getElementById("searchInp"); filter = input.value.toUpperCase();
				table = document.getElementById("custTable"); tr = table.getElementsByTagName("tr");
				for (i = 1; i < tr.length; i++) {
					tr[i].style.display = "none"; td = tr[i].getElementsByTagName("td");
					for (j = 0; j < td.length; j++) {
						if (td[j] && (td[j].textContent || td[j].innerText).toUpperCase().indexOf(filter) > -1) {
							tr[i].style.display = ""; break;
						}
					}
				}
			}
			</script></body></html>`)
	})

	fmt.Println("Server running on :8080")
	http.ListenAndServe(":8080", nil)
}
