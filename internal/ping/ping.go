package ping

import (
	"crypto/rand"
	"fmt"
)

type Ping struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	IP          string `json:"ip"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	TotalResults int   `json:"total_results"`
}

const (
	StatusOn  = "on"
	StatusOff = "off"
)

func ValidStatus(status string) bool {
	return status == StatusOn || status == StatusOff
}

func NewUUID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16]), nil
}
