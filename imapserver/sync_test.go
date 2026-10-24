package imapserver

import (
	"bufio"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
	"strings"
	"testing"
)

func TestSelectSyncParser(t *testing.T) {
	for _, tc := range []struct {
		args    string
		enabled bool
		valid   bool
	}{
		{" (CONDSTORE)", false, true}, {" (QRESYNC (1 2))", true, true}, {" (QRESYNC (1 2 1:3))", true, true}, {" (QRESYNC (1 2 (1:3 1:3)))", true, true}, {" (QRESYNC (1 2 1:3 (1:3 1:3)))", true, true},
		{" (QRESYNC (1 2))", false, false}, {" (QRESYNC (0 2))", true, false}, {" (QRESYNC (1 0))", true, false}, {" (QRESYNC (1 2 1:*))", true, false}, {" (QRESYNC (1 9223372036854775808))", true, false}, {" (BOGUS)", true, false},
	} {
		t.Run(tc.args, func(t *testing.T) {
			c := &Conn{state: imap.ConnStateAuthenticated, enabled: imap.CapSet{}, server: New(&Options{Caps: imap.CapSet{imap.CapIMAP4rev1: {}, imap.CapCondStore: {}, imap.CapQResync: {}}})}
			if tc.enabled {
				c.enabled[imap.CapQResync] = struct{}{}
			}
			dec := imapwire.NewDecoder(bufio.NewReader(strings.NewReader(tc.args+"\r\n")), imapwire.ConnSideServer)
			var o imap.SelectOptions
			err := c.readSelectSync(dec, &o)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v err=%v", tc.valid, err)
			}
			if err == nil && !dec.ExpectCRLF() {
				t.Fatal(dec.Err())
			}
		})
	}
}
