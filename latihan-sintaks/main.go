package main

import "fmt"

func main() {
	nama := "Zidane"
	umur := 20
	ipk := 3.02
	aktif := true
	mataKuliah := []string{"Desain UI/UX", "Microservice", "Artificial Intellegent"}

	mahasiswa := make(map[string]int)
	mahasiswa["Zidane"] = 90
	mahasiswa["Budi"] = 85
	mahasiswa["Andi"] = 88

	nilai, ada := mahasiswa["Zidane"]
if ada {
	fmt.Println("Nilai Zidane:", nilai)
} else {
	fmt.Println("Zidane tidak ditemukan")
}

delete(mahasiswa, "Budi")
fmt.Println(mahasiswa)

	fmt.Println(nama)
	fmt.Println(umur)
	fmt.Println(ipk)
	fmt.Println(aktif)
	fmt.Println(mataKuliah)
}