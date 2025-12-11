package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/upload", uploadAndAnalyzeHandler)
	http.HandleFunc("/files", proxyToService("http://file-storing-service:8082/files"))
	http.HandleFunc("/files/", proxyToService("http://file-storing-service:8082/files/"))
	http.HandleFunc("/analyze", proxyToService("http://file-analysis-service:8081/analyze"))
	http.HandleFunc("/reports", proxyToService("http://file-analysis-service:8081/reports"))
	http.HandleFunc("/reports/", proxyToService("http://file-analysis-service:8081/reports/"))
	http.HandleFunc("/wordCloud/", proxyToService("http://file-analysis-service:8081/wordCloud/"))

	fmt.Println("API Gateway запущен на http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "OK", "message": "API Gateway is running"}`))
}

func proxyToService(targetURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		fullURL := targetURL + r.RequestURI[len(getBasePath(targetURL)):]
		req, err := http.NewRequest(r.Method, fullURL, r.Body)
		if err != nil {
			http.Error(w, "Ошибка создания прокси-запроса", http.StatusInternalServerError)
			return
		}
		for name, values := range r.Header {
			for _, value := range values {
				req.Header.Add(name, value)
			}
		}
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "Сервис недоступен", http.StatusBadGateway)
		}
		defer resp.Body.Close()
		for name, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

func getBasePath(url string) string {
	parts := strings.SplitN(url, "://", 2)
	if len(parts) < 2 {
		return ""
	}
	remaining := parts[1]
	slashIdx := strings.Index(remaining, "/")
	if slashIdx == -1 {
		return ""
	}
	return remaining[slashIdx:]
}

func uploadAndAnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	resp, err := http.Post(
		"http://file-storing-service:8082/upload",
		r.Header.Get("Content-Type"),
		r.Body,
	)
	if err != nil {
		http.Error(w, "Ошибка связи с File Storing Service", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		w.Write(bodyBytes)
		return
	}
	var uploadResp map[string]interface{}
	json.Unmarshal(bodyBytes, &uploadResp)

	go func() {
		var fileID int
		if idFloat, ok := uploadResp["file_id"].(float64); ok {
			fileID = int(idFloat)
		} else {
			if idInt, ok := uploadResp["file_id"].(int); ok {
				fileID = idInt
			} else {
				fmt.Println("Ошибка: не удалось получить file_id из ответа, он равен 0")
				return
			}
		}
		filePath, _ := uploadResp["file_path"].(string)
		studentID, _ := uploadResp["student_id"].(string)
		assignmentID, _ := uploadResp["assignment_id"].(string)

		analyzeReq := map[string]interface{}{
			"file_id":       int(fileID),
			"file_path":     filePath,
			"student_id":    studentID,
			"assignment_id": assignmentID,
		}

		jsonData, _ := json.Marshal(analyzeReq)
		_, err := http.Post("http://file-analysis-service:8081/analyze", "application/json", bytes.NewBuffer(jsonData))

		if err != nil {
			fmt.Printf("Ошибка авто-запуска анализа для файла %d: %v\n", int(fileID), err)
		} else {
			fmt.Printf("Авто-анализ запущен для файла %d\n", int(fileID))
		}
	}()
	uploadResp["analysis_status"] = "started"
	finalResponse, _ := json.Marshal(uploadResp)
	w.Write(finalResponse)
}
