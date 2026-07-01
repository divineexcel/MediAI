package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func GenerateReference(prefix string) string {
	return fmt.Sprintf("%s-%s-%s",
		strings.ToUpper(prefix),
		uuid.NewString()[:8],
		time.Now().Format("20060102"),
	)
}

func NewUUID() string {
	return uuid.NewString()
}

func NormEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
