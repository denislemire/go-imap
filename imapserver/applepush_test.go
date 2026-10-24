package imapserver

import (
	"bufio"
	"strings"
	"testing"

	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func TestReadApplePush(t *testing.T) {
	valid := ` aps-version 2 aps-account-id "account-id" aps-device-token aabbcc aps-subtopic com.apple.mobilemail mailboxes (INBOX "Sent Items")` + "\r\n"
	for _, tc := range []struct {
		name, wire string
		valid      bool
	}{
		{"valid", valid, true},
		{"reordered", ` mailboxes (INBOX) aps-subtopic com.apple.mobilemail aps-device-token aabbcc aps-account-id account aps-version 2` + "\r\n", true},
		{"unsupported version", strings.Replace(valid, "aps-version 2", "aps-version 3", 1), false},
		{"duplicate", strings.Replace(valid, "aps-account-id", "aps-version", 1), false},
		{"wrong subtopic", strings.Replace(valid, "com.apple.mobilemail", "another.app", 1), false},
		{"unknown", strings.Replace(valid, "aps-account-id", "unknown", 1), false},
		{"too many mailboxes", strings.Replace(valid, `(INBOX "Sent Items")`, "("+strings.Repeat("INBOX ", 33)+")", 1), false},
		{"extra arguments", strings.TrimSuffix(valid, "\r\n") + " extra value\r\n", false},
		{"oversized", strings.Replace(valid, "aabbcc", strings.Repeat("a", 513), 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader(tc.wire)), imapwire.ConnSideServer)
			_, err := readApplePush(dec)
			if (err == nil) != tc.valid {
				t.Fatalf("accepted=%v, err=%v", err == nil, err)
			}
		})
	}
}
