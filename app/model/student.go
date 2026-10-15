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
    NIM   string `json:"nim"`
    Nama  string `json:"nama"`
    Prodi string `json:"prodi"`
}

type UpdateStudentRequest struct {
    NIM   *string `json:"nim,omitempty"`
    Nama  *string `json:"nama,omitempty"`
    Prodi *string `json:"prodi,omitempty"`
}
