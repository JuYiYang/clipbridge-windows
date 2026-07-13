package protocol

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"
)

const PlainTextType = "public.utf8-plain-text"

type ClipboardContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type ClipboardItem struct {
	ID             string             `json:"id"`
	Title          string             `json:"title"`
	Application    *string            `json:"application,omitempty"`
	FirstCopiedAt  time.Time          `json:"firstCopiedAt"`
	LastCopiedAt   time.Time          `json:"lastCopiedAt"`
	NumberOfCopies int                `json:"numberOfCopies"`
	Pin            *string            `json:"pin,omitempty"`
	Contents       []ClipboardContent `json:"contents"`
	SourceDeviceID string             `json:"sourceDeviceID"`
}

type PushRequest struct {
	DeviceID string          `json:"deviceID"`
	Items    []ClipboardItem `json:"items"`
}

type PushResponse struct {
	Accepted  int     `json:"accepted"`
	Stored    int     `json:"stored"`
	NextSince float64 `json:"nextSince"`
}

type PullResponse struct {
	Items     []ClipboardItem `json:"items"`
	NextSince *float64        `json:"nextSince,omitempty"`
}

func NewTextItem(deviceID string, text string, copiedAt time.Time) ClipboardItem {
	data := []byte(text)
	return ClipboardItem{
		ID:             ContentID(PlainTextType, data),
		Title:          titleFromText(text),
		FirstCopiedAt:  copiedAt,
		LastCopiedAt:   copiedAt,
		NumberOfCopies: 1,
		Contents: []ClipboardContent{{
			Type:  PlainTextType,
			Value: base64.StdEncoding.EncodeToString(data),
		}},
		SourceDeviceID: deviceID,
	}
}

func ContentID(contentType string, data []byte) string {
	hasher := sha256.New()
	hasher.Write([]byte(contentType))
	hasher.Write([]byte{0})
	hasher.Write(data)
	hasher.Write([]byte{31})
	return hex.EncodeToString(hasher.Sum(nil))
}

func titleFromText(text string) string {
	title := strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	title = strings.ReplaceAll(title, "\n", " ")
	if title == "" {
		return "(Empty text)"
	}
	const maxTitleLength = 120
	if len([]rune(title)) <= maxTitleLength {
		return title
	}
	runes := []rune(title)
	return string(runes[:maxTitleLength]) + "..."
}
