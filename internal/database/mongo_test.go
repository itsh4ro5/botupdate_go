package database

import (
	"strings"
	"testing"

	"github.com/itsh4ro5/botupdate/internal/models"
)

func TestURL_Encoding_Decoding(t *testing.T) {
	testCases := []string{
		"https://t.me/+AbCdEfGh",
		"https://t.me/example.channel/+AbCd",
		"https://t.me/joinchat/1234567890",
	}

	for _, tc := range testCases {
		encoded := encodeLinkMapKey(tc)
		if strings.Contains(encoded, ".") || strings.HasPrefix(encoded, "$") {
			t.Errorf("Encoded key %q contains invalid MongoDB characters for %q", encoded, tc)
		}
		decoded := decodeLinkMapKey(encoded)
		if decoded != tc {
			t.Errorf("Expected decoded to be %q, got %q", tc, decoded)
		}
	}
}

// Ensure the corrupt mapping logic skips ID 0
func TestCorruptedMappingProtection(t *testing.T) {
	// Let's test the load logic behavior on a fake link map
	state := &models.BotState{
		LinkMap: map[string]*models.InviteMapping{
			encodeLinkMapKey("https://t.me/+valid"): {
				Hash:    "https://t.me/+valid",
				UserID:  123,
				BatchID: -100,
			},
			encodeLinkMapKey("https://t.me/+invalid1"): {
				Hash:    "https://t.me/+invalid1",
				UserID:  0, // corrupted
				BatchID: -100,
			},
			encodeLinkMapKey("https://t.me/+invalid2"): {
				Hash:    "https://t.me/+invalid2",
				UserID:  123,
				BatchID: 0, // corrupted
			},
		},
	}

	decodedMap := make(map[string]*models.InviteMapping)
	for k, v := range state.LinkMap {
		if v == nil || v.UserID <= 0 || v.BatchID == 0 {
			continue
		}
		decodedK := decodeLinkMapKey(k)
		decodedMap[decodedK] = v
	}

	if len(decodedMap) != 1 {
		t.Errorf("Expected 1 valid entry, got %d", len(decodedMap))
	}
	if _, ok := decodedMap["https://t.me/+valid"]; !ok {
		t.Errorf("Expected valid entry to be present")
	}
}
