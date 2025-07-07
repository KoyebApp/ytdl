package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/kkdai/youtube/v2"
)

func main() {
	http.HandleFunc("/ytmp4", ytmp4Handler)
	http.HandleFunc("/ytm3", ytm3Handler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("Server starting on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
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
