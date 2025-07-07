package handler

import (
	"fmt"
	"io"
	"net/http"
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

func Ytmp4(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, fmt.Sprintf("Failed to send video: %v", err), http.StatusInternalServerError)
	}
}

// For Vercel: main entry point for /api/ytmp4
func Handler(w http.ResponseWriter, r *http.Request) {
	Ytmp4(w, r)
}
