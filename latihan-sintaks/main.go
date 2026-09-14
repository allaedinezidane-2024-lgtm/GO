package main

import "fmt"

type Student struct {
	ID       int
	Name     string
	Grade    float64
	IsActive bool
}

func (s Student) GetInfo() string {
	return fmt.Sprintf("ID: %d, Nama: %s, Nilai: %.2f, Aktif: %v",
		s.ID, s.Name, s.Grade, s.IsActive)
}

func (s *Student) UpdateGrade(grade float64) {
	s.Grade = grade
}

func (s *Student) Activate() {
	s.IsActive = true
}

func (s *Student) Deactivate() {
	s.IsActive = false
}

func swap(a, b *int) {
	*a, *b = *b, *a
}

func updateSlice(s *[]string, newItem string) {
	*s = append(*s, newItem)
}

func ubahNilai(x int) {
	x = 100
}

func ubahLewatPointer(x *int) {
	*x = 100
}
func main() {
	nama := "Zidane"
	umur := 20
	ipk := 3.02
	aktif := true
	mataKuliah := []string{"Desain UI/UX", "Microservice", "Artificial Intellegent"}
	updateSlice(&mataKuliah, "Pemrograman Go")

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

	for key, nilai := range mahasiswa {
		fmt.Println(key, ":", nilai)
	}
	fmt.Println(nama)
	fmt.Println(umur)
	fmt.Println(ipk)
	fmt.Println(aktif)
	fmt.Println(mataKuliah)

	a := 10
	b := 20

	fmt.Println("Sebelum ditukar:", a, b)

	swap(&a, &b)

	fmt.Println("Setelah ditukar:", a, b)

	student := Student{
		ID:       1,
		Name:     "Zidane",
		Grade:    85.5,
		IsActive: false,
	}

	fmt.Println(student.GetInfo())

	student.UpdateGrade(90.5)
	student.Activate()

	fmt.Println(student.GetInfo())

	student.Deactivate()

	fmt.Println(student.GetInfo())

	angka := 50

	fmt.Println("Sebelum:", angka)

	ubahNilai(angka)
	fmt.Println("Setelah pass by value:", angka)

	ubahLewatPointer(&angka)
	fmt.Println("Setelah pass by pointer:", angka)
}
