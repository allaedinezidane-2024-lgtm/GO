package model

import "time"

type Student struct {
	ID        int       `json:"id"`
	NIM       string    `json:"nim"`
	Nama      string    `json:"nama"`
	Prodi     string    `json:"prodi"`
	OwnerID   int       `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateStudentRequest struct {
	NIM   string `json:"nim" validate:"required,max=20"`
	Nama  string `json:"nama" validate:"required,max=100"`
	Prodi string `json:"prodi" validate:"required,max=100"`
}

type UpdateStudentRequest struct {
	NIM   *string `json:"nim,omitempty" validate:"omitnil,required,max=20"`
	Nama  *string `json:"nama,omitempty" validate:"omitnil,required,max=100"`
	Prodi *string `json:"prodi,omitempty" validate:"omitnil,required,max=100"`
}
