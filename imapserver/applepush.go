package imapserver

import (
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

// ApplePushRegistration contains the version-2 XAPPLEPUSHSERVICE registration.
// DeviceToken and AccountID are sensitive identifiers and must not be logged.
type ApplePushRegistration struct {
	AccountID   string
	DeviceToken string
	Subtopic    string
	Mailboxes   []string
}

// SessionApplePush optionally implements Apple's Mail push registration extension.
// Returning an empty topic disables both advertising and registration.
type SessionApplePush interface {
	ApplePushTopic() string
	RegisterApplePush(*ApplePushRegistration) error
}

func (c *Conn) handleApplePush(dec *imapwire.Decoder) error {
	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}
	session, ok := c.session.(SessionApplePush)
	if !ok || session.ApplePushTopic() == "" {
		return newClientBugError("XAPPLEPUSHSERVICE is not enabled")
	}
	registration, err := readApplePush(dec)
	if err != nil {
		return err
	}
	if err = session.RegisterApplePush(registration); err != nil {
		return err
	}
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("XAPPLEPUSHSERVICE").SP().Atom("aps-version").SP().Number(2).SP().Atom("aps-topic").SP().String(session.ApplePushTopic())
	return enc.CRLF()
}

func readApplePush(dec *imapwire.Decoder) (*ApplePushRegistration, error) {
	registration := &ApplePushRegistration{}
	seen := map[string]bool{}
	version := ""
	for i := 0; i < 5; i++ {
		var key, value string
		if !dec.ExpectSP() || !dec.ExpectAString(&key) || !dec.ExpectSP() {
			return nil, dec.Err()
		}
		key = strings.ToLower(key)
		if seen[key] {
			return nil, newClientBugError("Duplicate Apple push parameter")
		}
		seen[key] = true
		if key == "mailboxes" {
			err := dec.ExpectList(func() error {
				if len(registration.Mailboxes) >= 32 {
					return newClientBugError("Too many Apple push mailboxes")
				}
				var name string
				if !dec.ExpectMailbox(&name) {
					return dec.Err()
				}
				if name == "" || len(name) > 200 {
					return newClientBugError("Invalid Apple push mailbox")
				}
				registration.Mailboxes = append(registration.Mailboxes, name)
				return nil
			})
			if err != nil {
				return nil, err
			}
			continue
		}
		if !dec.ExpectAString(&value) {
			return nil, dec.Err()
		}
		if value == "" || len(value) > 512 {
			return nil, newClientBugError("Invalid Apple push parameter length")
		}
		switch key {
		case "aps-version":
			version = value
		case "aps-account-id":
			registration.AccountID = value
		case "aps-device-token":
			registration.DeviceToken = value
		case "aps-subtopic":
			registration.Subtopic = value
		default:
			return nil, newClientBugError("Unknown Apple push parameter")
		}
	}
	if !dec.ExpectCRLF() {
		return nil, dec.Err()
	}
	if version != "2" || registration.AccountID == "" || registration.DeviceToken == "" || registration.Subtopic != "com.apple.mobilemail" || !seen["mailboxes"] {
		return nil, &imap.Error{Type: imap.StatusResponseTypeBad, Text: "Invalid Apple push registration"}
	}
	return registration, nil
}
