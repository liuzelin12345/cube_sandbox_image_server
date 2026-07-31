// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"

	"github.com/TencentCloudAgentRuntime/ags-cookbook/examples/custom-image-go-sdk/cube_sandbox_image_server/internal/apperror"
)

type ApiKeyAuthMiddleware struct {
	keyHashes [][sha256.Size]byte
}

func NewApiKeyAuthMiddleware(apiKeys []string) *ApiKeyAuthMiddleware {
	keyHashes := make([][sha256.Size]byte, 0, len(apiKeys))
	for _, apiKey := range apiKeys {
		keyHashes = append(keyHashes, sha256.Sum256([]byte(apiKey)))
	}
	return &ApiKeyAuthMiddleware{keyHashes: keyHashes}
}

func (m *ApiKeyAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providedHash := sha256.Sum256([]byte(r.Header.Get("X-API-Key")))
		authorized := 0
		for _, expectedHash := range m.keyHashes {
			authorized |= subtle.ConstantTimeCompare(providedHash[:], expectedHash[:])
		}
		if authorized != 1 {
			apperror.Write(w, apperror.New(
				http.StatusUnauthorized,
				"UNAUTHORIZED",
				"缺少或无效的 API Key",
				nil,
			))
			return
		}
		next(w, r)
	}
}
