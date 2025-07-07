package handler

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/kkdai/youtube/v2"
)

func sanitizeFilename(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' {
			return '-'
		}
		return r
	}, name)
}

func Ytm3(w http.ResponseWriter, r *http.Request) {
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

	// ffmpeg must be available in Vercel's serverless environment (may not be by default!)
	cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "mp3", "-ab", "192000", "-vn", "pipe:1")
	cmd.Stdin = stream
	cmd.Stdout = w

	err = cmd.Run()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to convert audio to mp3: %v", err), http.StatusInternalServerError)
		return
	}
}

// For Vercel: main entry point for /api/ytm3
func Handler(w http.ResponseWriter, r *http.Request) {
	Ytm3(w, r)
}
