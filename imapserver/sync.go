package imapserver

import (
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func (c *Conn) useCondStore() error {
	if !c.server.options.caps().Has(imap.CapCondStore) {
		return newClientBugError("CONDSTORE not supported")
	}
	wasEnabled := c.enabled.Has(imap.CapCondStore)
	c.mutex.Lock()
	c.enabled[imap.CapCondStore] = struct{}{}
	c.mutex.Unlock()
	if !wasEnabled && c.state == imap.ConnStateSelected {
		if s, ok := c.session.(interface{ HighestModSeq() uint64 }); ok {
			return c.writeHighestModSeq(s.HighestModSeq())
		}
	}
	return nil
}
func (c *Conn) readSelectSync(dec *imapwire.Decoder, o *imap.SelectOptions) error {
	if !dec.SP() {
		return nil
	}
	return dec.ExpectList(func() error {
		var name string
		if !dec.ExpectAtom(&name) {
			return dec.Err()
		}
		switch strings.ToUpper(name) {
		case "CONDSTORE":
			o.CondStore = true
			return c.useCondStore()
		case "QRESYNC":
			if !c.enabled.Has(imap.CapQResync) {
				return newClientBugError("ENABLE QRESYNC required")
			}
			if o.QResync != nil {
				return newClientBugError("Duplicate QRESYNC")
			}
			q := &imap.SelectQResyncOptions{}
			o.QResync = q
			o.CondStore = true
			if !dec.ExpectSP() || !dec.ExpectSpecial('(') || !dec.ExpectNumber(&q.UIDValidity) || q.UIDValidity == 0 || !dec.ExpectSP() || !dec.ExpectModSeq(&q.ModSeq) || q.ModSeq == 0 {
				return newClientBugError("Invalid QRESYNC state")
			}
			if dec.SP() {
				if !dec.Special('(') {
					if !dec.ExpectUIDSet(&q.KnownUIDs) || q.KnownUIDs.Dynamic() {
						return newClientBugError("Invalid known UIDs")
					}
					if !dec.SP() {
						if !dec.ExpectSpecial(')') {
							return dec.Err()
						}
						return c.useCondStore()
					}
					if !dec.ExpectSpecial('(') {
						return dec.Err()
					}
				}
				var seq imap.NumSet
				var uid imap.UIDSet
				if !dec.ExpectNumSet(imapwire.NumKindSeq, &seq) || !dec.ExpectSP() || !dec.ExpectUIDSet(&uid) || !dec.ExpectSpecial(')') {
					return dec.Err()
				}
				if seq.Dynamic() || uid.Dynamic() {
					return newClientBugError("Invalid sequence match data")
				}
			}
			if !dec.ExpectSpecial(')') {
				return dec.Err()
			}
			return c.useCondStore()
		default:
			return newClientBugError("Unknown SELECT parameter")
		}
	})
}
func (c *Conn) writeHighestModSeq(n uint64) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	return enc.Atom("*").SP().Atom("OK").SP().Special('[').Atom("HIGHESTMODSEQ").SP().ModSeq(n).Special(']').SP().Text("Mailbox change number").CRLF()
}
func (c *Conn) writeVanished(uids imap.UIDSet, earlier bool) error {
	enc := newResponseEncoder(c)
	defer enc.end()
	enc.Atom("*").SP().Atom("VANISHED").SP()
	if earlier {
		enc.Special('(').Atom("EARLIER").Special(')').SP()
	}
	return enc.NumSet(uids).CRLF()
}

// WriteVanished reports previously expunged UIDs in a QRESYNC FETCH.
func (w *FetchWriter) WriteVanished(uids imap.UIDSet) error { return w.conn.writeVanished(uids, true) }

// WriteModSeq includes a persistent message change number.
func (w *FetchResponseWriter) WriteModSeq(n uint64) {
	w.writeItemSep()
	w.enc.Atom("MODSEQ").SP().Special('(').ModSeq(n).Special(')')
}

func (w *FetchWriter) CondStore() bool { return w.conn.enabled.Has(imap.CapCondStore) }
