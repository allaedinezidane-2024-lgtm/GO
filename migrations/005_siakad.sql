BEGIN;

-- Tambahkan role mahasiswa tanpa menghapus role lama.
INSERT INTO roles (name)
VALUES ('mahasiswa')
ON CONFLICT (name) DO NOTHING;

-- Lengkapi data mahasiswa sesuai soal UTS.
ALTER TABLE students
    ADD COLUMN IF NOT EXISTS user_id INTEGER
        REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS angkatan INTEGER NOT NULL DEFAULT 2024,
    ADD COLUMN IF NOT EXISTS ipk_terakhir NUMERIC(3,2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS students_user_id_key
    ON students(user_id)
    WHERE user_id IS NOT NULL;

-- Tabel mata kuliah.
CREATE TABLE IF NOT EXISTS courses (
    id SERIAL PRIMARY KEY,
    kode_mk VARCHAR(20) NOT NULL UNIQUE,
    nama_mk VARCHAR(150) NOT NULL,
    sks INTEGER NOT NULL CHECK (sks BETWEEN 1 AND 6),
    semester INTEGER NOT NULL CHECK (semester BETWEEN 1 AND 14),
    kuota INTEGER NOT NULL CHECK (kuota >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tabel KRS.
CREATE TABLE IF NOT EXISTS enrollments (
    id SERIAL PRIMARY KEY,
    student_id INTEGER NOT NULL
        REFERENCES students(id) ON DELETE RESTRICT,
    course_id INTEGER NOT NULL
        REFERENCES courses(id) ON DELETE RESTRICT,
    tahun_akademik VARCHAR(30) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT enrollments_student_course_year_key
        UNIQUE (student_id, course_id, tahun_akademik)
);

CREATE INDEX IF NOT EXISTS idx_students_deleted_at
    ON students(deleted_at);

CREATE INDEX IF NOT EXISTS idx_enrollments_student_year
    ON enrollments(student_id, tahun_akademik);

CREATE INDEX IF NOT EXISTS idx_enrollments_course
    ON enrollments(course_id);

-- Permission tambahan untuk fitur SIAKAD.
INSERT INTO permissions (name)
VALUES
    ('course:list'),
    ('enrollment:create'),
    ('enrollment:delete')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name)
VALUES
    ('admin', 'course:list'),
    ('admin', 'enrollment:create'),
    ('admin', 'enrollment:delete'),
    ('mahasiswa', 'course:list'),
    ('mahasiswa', 'enrollment:create'),
    ('mahasiswa', 'enrollment:delete')
ON CONFLICT DO NOTHING;

COMMIT;
