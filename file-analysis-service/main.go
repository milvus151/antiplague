package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/glebarez/go-sqlite"
)

type AnalysisRequest struct {
	FileID       int    `json:"file_id"`
	FilePath     string `json:"file_path"`
	StudentID    string `json:"student_id"`
	AssignmentID string `json:"assignment_id"`
}

type PlagiarismReport struct {
	ID              int     `json:"id"`
	FileID          int     `json:"file_id"`
	PlagiarismScore float64 `json:"plagiarism_score"`
	IsPlagiarism    bool    `json:"is_plagiarism"`
	MatchedFileID   int     `json:"matched_file_id"`
	AnalysisState   string  `json:"analysis_state"`
	SameDetails     string  `json:"same_details"`
}

var db *sql.DB

func init() {
	var err error
	db, err = sql.Open("sqlite", "/app/files.db")
	if err != nil {
		panic("Ошибка подключения к БД: " + err.Error())
	}
	fmt.Println("Успешно подключились к БД!")
	createReportsTable()
}

func createReportsTable() {
	query := `
	CREATE TABLE IF NOT EXISTS reports (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_id INTEGER NOT NULL,
		plagiarism_score INTEGER NOT NULL,
		is_plagiarism BOOLEAN NOT NULL,
		matched_file_id INTEGER NOT NULL,
		analysis_state STRING NOT NULL,
		same_details STRING NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)
	`
	_, err := db.Exec(query)
	if err != nil {
		panic("Ошибка создания таблицы отчётов: " + err.Error())
	}
	fmt.Println("Таблица отчётов по плагиату готова")
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/analyze", analyzeHandler)
	http.HandleFunc("/reports", getAllReportsHandler)
	http.HandleFunc("/reports/", getReportHandler)
	http.HandleFunc("/wordCloud/", getWordCloudHandler)

	fmt.Println("File Analysis Service запущен на http://localhost:8081")
	http.ListenAndServe(":8081", nil)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "ok",
		"service": "File Analysis Service",
	}
	json.NewEncoder(w).Encode(response)
}

func analyzeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is supported.", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	var req AnalysisRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, `{"error": "Ошибка при парсинге JSON"}`, http.StatusBadRequest)
		return
	}
	fmt.Printf("Анализ файла: %s (File ID: %d)\n", req.FilePath, req.FileID)
	ext := filepath.Ext(req.FilePath)
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
	if !supportedExts[ext] {
		fmt.Printf("Пропуск файла %s: неподдерживаемый формат %s\n", req.FilePath, ext)
		report := PlagiarismReport{
			FileID:          req.FileID,
			AnalysisState:   "skipped",
			PlagiarismScore: 0,
			IsPlagiarism:    false,
			SameDetails:     fmt.Sprintf("Формат %s не поддерживается. Разрешены: txt, go, py, java, cpp, c, h, js, ts, md", ext),
		}
		SaveReport(req.FileID, 0, false, 0, "skipped because of incorrect extension")
		json.NewEncoder(w).Encode(report)
		return
	}
	newFileText, err := os.ReadFile(req.FilePath)
	if err != nil {
		fmt.Printf("Ошибка чтения файла: %v/n", err)
		report := PlagiarismReport{
			FileID:          req.FileID,
			AnalysisState:   "error",
			PlagiarismScore: 0,
			IsPlagiarism:    false,
		}
		json.NewEncoder(w).Encode(report)
		return
	}
	newFileContent := string(newFileText)
	fmt.Printf("Файл прочитан, размер файла: %d символов\n", len(newFileContent))

	plagiarismScore, matchedFileID := comparePlagiarism(newFileContent, req.StudentID, req.FileID)
	isPlagiarism := plagiarismScore > 0.5
	fmt.Printf("Результат плагиата: %.2f%% \n ", plagiarismScore*100)

	report := SaveReport(req.FileID, plagiarismScore, isPlagiarism, matchedFileID, "completed")

	fmt.Printf("Анализ завершен. Результат отправляем...\n")
	json.NewEncoder(w).Encode(report)
}

func SaveReport(fileID int, score float64, isPlagiarism bool, matchedFileID int, status string) PlagiarismReport {
	isPlagiarismInt := 0
	if isPlagiarism {
		isPlagiarismInt = 1
	}
	query := `
	INSERT INTO reports (
	    file_id,
    	plagiarism_score,
	    is_plagiarism,
	    matched_file_id,
        analysis_state,
        same_details
	) VALUES (?, ?, ?, ?, ?, ?)
	`
	details := fmt.Sprintf("Совпадение %.2f%% с File ID %d", score*100, matchedFileID)
	result, err := db.Exec(query, fileID, score, isPlagiarismInt, matchedFileID, status, details)
	if err != nil {
		fmt.Println("Ошибка при создании отчёта")
		return PlagiarismReport{
			FileID:          fileID,
			AnalysisState:   "error",
			PlagiarismScore: 0,
			IsPlagiarism:    false,
			SameDetails:     "Ошибка при сохранении отчёта",
		}
	}
	reportID, _ := result.LastInsertId()
	report := PlagiarismReport{
		ID:              int(reportID),
		FileID:          fileID,
		PlagiarismScore: score,
		IsPlagiarism:    isPlagiarism,
		MatchedFileID:   matchedFileID,
		AnalysisState:   "completed",
		SameDetails:     details,
	}
	return report
}

func comparePlagiarism(newFileContent string, curStudentID string, curFileID int) (float64, int) {
	query := `
	SELECT id, file_path, student_id 
	FROM files 
	WHERE id != ? AND student_id != ?
	ORDER BY id ASC
	`
	rows, err := db.Query(query, curFileID, curStudentID)
	if err != nil {
		fmt.Println("Ошибка при запросе к БД", err)
		return 0, 0
	}
	defer rows.Close()

	maxSimilarity := 0.0
	matchedFileID := 0
	for rows.Next() {
		var fileID int
		var filePath, studentID string

		err = rows.Scan(&fileID, &filePath, &studentID)
		if err != nil {
			continue
		}
		oldFileText, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Ошибка чтения файла %s: %v\n", filePath, err)
			continue
		}
		oldFileContent := string(oldFileText)
		similarity := countSim(newFileContent, oldFileContent)

		fmt.Printf("Сравнение с File ID %d: %.2f%% совпадения\n", fileID, similarity*100)
		if similarity > maxSimilarity {
			maxSimilarity = similarity
			matchedFileID = fileID
		}
	}
	fmt.Printf("Максимальное совпадение: %.2f%% (с File ID: %d)\n", maxSimilarity*100, matchedFileID)
	return maxSimilarity, matchedFileID
}

func countSim(newFileContent string, oldFileContent string) float64 {
	words1 := strings.Fields(strings.ToLower(newFileContent))
	words2 := strings.Fields(strings.ToLower(oldFileContent))

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	matchCount := 0
	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 {
				matchCount++
				break
			}
		}
	}
	maxLen := len(words1)
	if len(words2) > maxLen {
		maxLen = len(words2)
	}
	return float64(matchCount) / float64(maxLen)
}

func getReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is supported.", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	reportID := r.URL.Path[len("/reports/"):]

	query := `
	SELECT id, file_id, plagiarism_score, is_plagiarism, matched_file_id, analysis_state, same_details
	FROM reports
	WHERE id = ?
	`
	row := db.QueryRow(query, reportID)

	var report PlagiarismReport
	var isPlagiarismInt int

	err := row.Scan(
		&report.ID,
		&report.FileID,
		&report.PlagiarismScore,
		&isPlagiarismInt,
		&report.MatchedFileID,
		&report.AnalysisState,
		&report.SameDetails,
	)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Отчёт не найден"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Ошибка при запросе к БД"}`, http.StatusInternalServerError)
		return
	}
	report.IsPlagiarism = isPlagiarismInt == 1
	json.NewEncoder(w).Encode(report)
}

func getAllReportsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is supported.", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	query := `
	SELECT id, file_id, plagiarism_score, is_plagiarism, matched_file_id, analysis_state, same_details
	FROM reports
	`
	rows, err := db.Query(query)
	if err != nil {
		http.Error(w, `{"error": "Ошибка при запросе к БД"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	reports := []PlagiarismReport{}
	for rows.Next() {
		var report PlagiarismReport
		var isPlagiarismInt int
		err := rows.Scan(
			&report.ID,
			&report.FileID,
			&report.PlagiarismScore,
			&isPlagiarismInt,
			&report.MatchedFileID,
			&report.AnalysisState,
			&report.SameDetails,
		)
		if err != nil {
			continue
		}
		report.IsPlagiarism = isPlagiarismInt == 1
		reports = append(reports, report)
	}
	json.NewEncoder(w).Encode(reports)
}

func getWordCloudHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is supported.", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fileID := r.URL.Path[len("/wordcloud/"):]
	var filePath string
	query := `
	SELECT file_path FROM files WHERE id = ?
	`
	row := db.QueryRow(query, fileID)
	err := row.Scan(&filePath)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error": "Файл не найден"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error": "Ошибка при запросе к БД"}`, http.StatusInternalServerError)
		return
	}
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, `{"error": "Ошибка чтения файла"}`, http.StatusInternalServerError)
		return
	}
	quickchartURL := "https://quickchart.io/wordcloud"
	payload := map[string]interface{}{
		"format":          "png",
		"width":           1000,
		"height":          1000,
		"backgroundColor": "#2b2b2b",
		"fontScale":       20,
		"scale":           "sqrt",
		"removeStopwords": true,
		"minWordLength":   3,
		"text":            string(fileContent),
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, `{"error": "Ошибка формирования запроса"}`, http.StatusInternalServerError)
		return
	}
	resp, err := http.Post(quickchartURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		http.Error(w, `{"error": "Ошибка создания облака слов"}`, http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		http.Error(w, `{"error": "Сервис облака слов недоступен"}`, http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "image/json")
	io.Copy(w, resp.Body)
}
