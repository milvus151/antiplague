package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

type FileInfo struct {
	ID           string      `json:"id"`
	StudentID    string      `json:"student_id"`
	AssignmentID string      `json:"assignment_id"`
	FilePath     string      `json:"file_path"`
	UploadedAt   interface{} `json:"uploaded_at"`
	Status       string      `json:"status"`
}

var db *sql.DB

func init() {
	var err error
	db, err = sql.Open("sqlite", "/app/files.db")
	if err != nil {
		panic("Ошибка подключения к БД: " + err.Error())
	}
	err = db.Ping()
	if err != nil {
		panic("БД не отвечает: " + err.Error())
	}
	fmt.Println("File Storing Service подключен к БД")
	createTable()
}
func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		student_id TEXT NOT NULL,
		assignment_id TEXT NOT NULL,
		file_path TEXT NOT NULL,
		uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		status TEXT DEFAULT 'pending'    
	)
	`
	_, err := db.Exec(query)
	if err != nil {
		panic("Ошибка создания таблицы: " + err.Error())
	}
	fmt.Println("Таблица для файлов создана")
}

func main() {
	os.MkdirAll("/app/uploads", 0755)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/files", listFilesHandler)
	http.HandleFunc("/files/", getFileHandler)

	fmt.Println("File Storing Service запущен на http://localhost:8082")
	http.ListenAndServe(":8082", nil)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "OK",
		"message": "File Storing работает",
	}
	json.NewEncoder(w).Encode(response)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	err := r.ParseMultipartForm(15 << 20)
	if err != nil {
		http.Error(w, `{"error": "Ошибка при парсинге формы"}`, http.StatusBadRequest)
		return
	}
	studentID := r.FormValue("student_id")
	assignmentID := r.FormValue("assignment_id")
	if studentID == "" || assignmentID == "" {
		http.Error(w, `{"error": "student_id и assignment_id обязательны"}`, http.StatusBadRequest)
		return
	}
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error": "Файл не найден в запросе"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := filepath.Ext(handler.Filename)
	supportedExts := map[string]bool{
		".txt":  true,
		".go":   true,
		".py":   true,
		".js":   true,
		".java": true,
		".cpp":  true,
		".c":    true,
		".h":    true,
		".ts":   true,
		".md":   true,
	}
	if !supportedExts[strings.ToLower(ext)] {
		http.Error(w, fmt.Sprintf(`{"error": "Формат %s не поддерживается. Разрешены: txt, go, py, java, cpp, c, h, js, ts, md"}`, ext), http.StatusUnsupportedMediaType)
		return
	}
	timestamp := time.Now().Unix()
	// Создаём имя формата: work_student001_task1_1732874940.txt
	safeFilename := fmt.Sprintf("work_%s_%s_%d%s", studentID, assignmentID, timestamp, ext)

	uploadsDir := "/app/uploads"
	if _, err := os.Stat(uploadsDir); os.IsNotExist(err) {
		uploadsDir = "./uploads"
	}

	abspath, err := filepath.Abs(filepath.Join(uploadsDir, safeFilename))
	if err != nil {
		fmt.Println("Ошибка при получении абсолютного пути:", err)
		http.Error(w, `{"error": "Ошибка при получении пути"}`, http.StatusInternalServerError)
		return
	}
	fmt.Println("Пытаемся создать файл:", abspath)

	dst, err := os.Create(abspath)
	if err != nil {
		fmt.Println("Ошибка при создании файла:", err)
		http.Error(w, `{"error": "Ошибка при создании файла"}`, http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, `{"error": "Ошибка при копировании файла"}`, http.StatusInternalServerError)
		return
	}
	query := `
	INSERT INTO files (student_id, assignment_id, file_path, status)
	VALUES (?, ?, ?, 'pending')
	`
	result, err := db.Exec(query, studentID, assignmentID, abspath)
	if err != nil {
		http.Error(w, `{"error": "Ошибка при сохранении данных в БД"}`, http.StatusBadRequest)
		return
	}

	fileID, err := result.LastInsertId()
	if err != nil {
		http.Error(w, `{"error": "Ошибка при получении ID"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":        "success",
		"message":       "Файл получен, работа зарегистрирована",
		"file_id":       fileID,
		"student_id":    studentID,
		"assignment_id": assignmentID,
		"filename":      handler.Filename,
		"file_path":     abspath,
	}
	json.NewEncoder(w).Encode(response)
}

func getFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	file_id := r.URL.Path[len("/files/"):]
	query := `
	SELECT id, student_id, assignment_id, file_path, uploaded_at, status FROM files WHERE id = ?
	`
	row := db.QueryRow(query, file_id)
	var file FileInfo
	err := row.Scan(
		&file.ID,
		&file.StudentID,
		&file.AssignmentID,
		&file.FilePath,
		&file.UploadedAt,
		&file.Status,
	)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Файл не найден"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Ошибка при запросе к БД"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(file)
}

func listFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	query := `SELECT id, student_id, assignment_id, file_path, uploaded_at, status FROM files`
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, `{"error": "Ошибка при запросе к БД"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	files := []FileInfo{}
	for rows.Next() {
		var file FileInfo
		err := rows.Scan(
			&file.ID,
			&file.StudentID,
			&file.AssignmentID,
			&file.FilePath,
			&file.UploadedAt,
			&file.Status,
		)
		if err != nil {
			continue
		}
		files = append(files, file)
	}
	json.NewEncoder(w).Encode(files)
}
