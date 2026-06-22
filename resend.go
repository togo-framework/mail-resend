// Package resend is a Resend (resend.com) driver for togo mail. It shows the
// driver-plugin pattern: a plugin that depends on another plugin (mail) and
// registers itself with it. Blank-import + set MAIL_DRIVER=resend, RESEND_API_KEY.
package resend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/togo-framework/mail"
	"github.com/togo-framework/togo"
)

const endpoint = "https://api.resend.com/emails"

func init() {
	mail.RegisterDriver("resend", func(k *togo.Kernel) (mail.Mailer, error) {
		key := os.Getenv("RESEND_API_KEY")
		if key == "" {
			return nil, errors.New("mail-resend: RESEND_API_KEY not set")
		}
		return &mailer{key: key, client: &http.Client{Timeout: 15 * time.Second}}, nil
	})
}

type mailer struct {
	key    string
	client *http.Client
}

func (m *mailer) Send(ctx context.Context, msg mail.Message) error {
	body := map[string]any{
		"from":    msg.From,
		"to":      msg.To,
		"subject": msg.Subject,
	}
	if msg.HTML != "" {
		body["html"] = msg.HTML
	}
	if msg.Text != "" {
		body["text"] = msg.Text
	}
	if len(msg.Cc) > 0 {
		body["cc"] = msg.Cc
	}
	if len(msg.Bcc) > 0 {
		body["bcc"] = msg.Bcc
	}
	buf, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+m.key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mail-resend: status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
