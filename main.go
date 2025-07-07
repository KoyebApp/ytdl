package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kkdai/youtube/v2"
)

func main() {
	http.HandleFunc("/ytmp4", ytmp4Handler)
	http.HandleFunc("/ytm3", ytm3Handler)

	// Start periodic fetcher goroutine
	go startPeriodicFetch()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server starting on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Periodically fetches the API URL from FETCH_API_URL env var every 5 minutes
func startPeriodicFetch() {
	apiURL := os.Getenv("FETCH_API_URL")
	if apiURL == "" {
		log.Println("FETCH_API_URL not set; periodic fetcher will not run.")
		return
	}
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Fetch once immediately at startup
	fetchAndLog(apiURL)
	for range ticker.C {
		fetchAndLog(apiURL)
	}
}

func fetchAndLog(url string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching %s: %v", url, err)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	log.Printf("Fetched %s: status %s, body: %s", url, resp.Status, string(body))
}

func ytmp4Handler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
		return
	}
	client := youtube.Client{}
	video, err := client.GetVideo(url)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get video info: %v", err), http.StatusInternalServerError)
		return
	}
	formats := video.Formats.WithAudioChannels()
	if len(formats) == 0 {
		http.Error(w, "No video+audio format available", http.StatusInternalServerError)
		return
	}
	stream, err := client.GetStream(video, &formats[0])
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get video stream: %v", err), http.StatusInternalServerError)
		return
	}
	filename := sanitizeFilename(video.Title) + ".mp4"
	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	_, err = io.Copy(w, stream)
	if err != nil {
		log.Printf("Failed to send video: %v", err)
	}
}

func ytm3Handler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "Missing 'url' parameter", http.StatusBadRequest)
		return
	}
	client := youtube.Client{}
	video, err := client.GetVideo(url)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get video info: %v", err), http.StatusInternalServerError)
		return
	}
	audioFormats := video.Formats.Type("audio")
	if len(audioFormats) == 0 {
		http.Error(w, "No audio format available", http.StatusInternalServerError)
		return
	}
	stream, err := client.GetStream(video, &audioFormats[0])
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get audio stream: %v", err), http.StatusInternalServerError)
		return
	}
	filename := sanitizeFilename(video.Title) + ".mp3"
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "mp3", "-ab", "192000", "-vn", "pipe:1")
	cmd.Stdin = stream
	cmd.Stdout = w
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to convert audio to mp3: %v", err), http.StatusInternalServerError)
		return
	}
}

func sanitizeFilename(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' {
			return '-'
		}
		return r
	}, name)
}
