package imapserver

import (
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) handleEnable(dec *imapwire.Decoder) error {
	var requested []imap.Cap
	for dec.SP() {
		cap, err := internal.ExpectCap(dec)
		if err != nil {
			return err
		}
		requested = append(requested, cap)
	}
	if !dec.ExpectCRLF() {
		return dec.Err()
	}

	if err := c.checkState(imap.ConnStateAuthenticated); err != nil {
		return err
	}

	var enabled []imap.Cap
	for _, req := range requested {
		switch req {
		case imap.CapCondStore, imap.CapQResync:
			if c.server.options.caps().Has(req) {
				enabled = append(enabled, req)
			}
		case imap.CapIMAP4rev2, imap.CapUTF8Accept:
			enabled = append(enabled, req)
		}
	}

	for _, cap := range enabled {
		if cap == imap.CapCondStore || cap == imap.CapQResync {
			if err := c.useCondStore(); err != nil {
				return err
			}
			break
		}
	}
	c.mutex.Lock()
	for _, e := range enabled {
		c.enabled[e] = struct{}{}
	}
	c.mutex.Unlock()

	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("ENABLED")
	for _, c := range enabled {
		enc.SP().Atom(string(c))
	}
	return enc.CRLF()
}
