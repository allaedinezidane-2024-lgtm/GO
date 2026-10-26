package main

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"

	"latihan-fiber/config"
	"latihan-fiber/database"
)

func main() {
	config.LoadEnv()
	ctx := context.Background()

	db, err := database.NewPool(ctx)
	if err != nil {
		log.Fatal("Koneksi database gagal: ", err)
	}
	defer db.Close()

	// Akun admin khusus pengujian UTS.
	adminPassword, err := bcrypt.GenerateFromPassword(
		[]byte("AdminSiakad123!"), bcrypt.DefaultCost,
	)
	if err != nil {
		log.Fatal(err)
	}

	var adminID int
	err = db.QueryRow(ctx,
		`SELECT id FROM users WHERE LOWER(email)=LOWER($1)`,
		"admin@siakad.local",
	).Scan(&adminID)

	if err == nil {
		_, err = db.Exec(ctx,
			`UPDATE users SET password=$1, role='admin', is_active=TRUE
             WHERE id=$2`,
			string(adminPassword), adminID,
		)
		if err != nil {
			log.Fatal("Gagal memperbarui akun admin: ", err)
		}
	} else {
		err = db.QueryRow(ctx,
			`INSERT INTO users(username,email,password,is_active,role)
             VALUES($1,$2,$3,TRUE,'admin') RETURNING id`,
			"admin_siakad_uts", "admin@siakad.local",
			string(adminPassword),
		).Scan(&adminID)
		if err != nil {
			log.Fatal("Gagal membuat akun admin: ", err)
		}
	}

	fmt.Println("Akun admin siap.")

	courses := []struct {
		code, name           string
		sks, semester, quota int
	}{
		{"UTS101", "Algoritma dan Pemrograman", 3, 1, 40},
		{"UTS102", "Basis Data", 3, 2, 40},
		{"UTS103", "Struktur Data", 3, 2, 35},
		{"UTS104", "Pemrograman Web", 3, 3, 40},
		{"UTS105", "Jaringan Komputer", 3, 3, 35},
		{"UTS106", "Sistem Operasi", 3, 3, 35},
		{"UTS107", "Rekayasa Perangkat Lunak", 3, 4, 40},
		{"UTS108", "Pemrograman Mobile", 3, 5, 30},
		{"UTS109", "Keamanan Informasi", 3, 5, 30},
		{"UTS110", "Kecerdasan Buatan", 3, 6, 30},
	}

	for _, course := range courses {
		_, err := db.Exec(ctx,
			`INSERT INTO courses(kode_mk,nama_mk,sks,semester,kuota)
             VALUES($1,$2,$3,$4,$5)
             ON CONFLICT(kode_mk) DO NOTHING`,
			course.code, course.name, course.sks,
			course.semester, course.quota,
		)
		if err != nil {
			log.Fatalf("Gagal seed mata kuliah %s: %v", course.code, err)
		}
	}
	fmt.Println("10 kode mata kuliah sudah dipastikan tersedia.")

	created := 0
	skipped := 0

	for i := 1; i <= 20; i++ {
		nim := fmt.Sprintf("2026%08d", i)
		email := fmt.Sprintf("mhs%02d@siakad.local", i)
		name := fmt.Sprintf("Mahasiswa UTS %02d", i)
		prodi := "Teknik Informatika"
		angkatan := 2026
		ipk := 2.50 + float64(i%16)/10.0
		if ipk > 4 {
			ipk = 4
		}

		var existingID int
		err := db.QueryRow(ctx,
			`SELECT id FROM students WHERE nim=$1`, nim,
		).Scan(&existingID)
		if err == nil {
			skipped++
			continue
		}

		var emailExists bool
		err = db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email)=LOWER($1))`,
			email,
		).Scan(&emailExists)
		if err != nil {
			log.Fatal("Gagal memeriksa email: ", err)
		}
		if emailExists {
			log.Fatalf("Email %s sudah dipakai akun lain, seed dihentikan agar data tidak tertimpa.", email)
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(nim), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal(err)
		}

		tx, err := db.Begin(ctx)
		if err != nil {
			log.Fatal(err)
		}

		var userID int
		err = tx.QueryRow(ctx,
			`INSERT INTO users(username,email,password,is_active,role)
             VALUES($1,$2,$3,TRUE,'mahasiswa') RETURNING id`,
			nim, email, string(hash),
		).Scan(&userID)
		if err != nil {
			_ = tx.Rollback(ctx)
			log.Fatalf("Gagal membuat akun %s: %v", email, err)
		}

		_, err = tx.Exec(ctx,
			`INSERT INTO students(nim,nama,prodi,owner_id,user_id,angkatan,ipk_terakhir)
             VALUES($1,$2,$3,$4,$5,$6,$7)`,
			nim, name, prodi, adminID, userID, angkatan, ipk,
		)
		if err != nil {
			_ = tx.Rollback(ctx)
			log.Fatalf("Gagal membuat mahasiswa %s: %v", nim, err)
		}

		if err = tx.Commit(ctx); err != nil {
			log.Fatal("Gagal commit data mahasiswa: ", err)
		}
		created++
	}

	fmt.Printf("Seed selesai. Mahasiswa baru: %d, mahasiswa dilewati: %d\n", created, skipped)
	fmt.Println("Selesai.")
}
