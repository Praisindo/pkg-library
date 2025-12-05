package mail

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Praisindo/pkg-library/app/pkg/filevalidation"
	"github.com/wneessen/go-mail"
)

type Config struct {
	Configsmtpservice  string `validate:"required,eq=plain|eq=gmail|eq=outlook" name:"CONFIG_SMTP_SERVICE"`
	Configsmtphost     string `validate:"required" name:"CONFIG_SMTP_HOST"`
	Configsmtpport     int    `validate:"required" name:"CONFIG_SMTP_PORT"`
	Configsendername   string `validate:"required" name:"CONFIG_SENDER_NAME"`
	Configauthemail    string `validate:"required" name:"CONFIG_AUTH_EMAIL"`
	Configauthpassword string `validate:"required" name:"CONFIG_AUTH_PASSWORD"`
	Configsmtpssl      bool   `validate:"boolean" name:"CONFIG_SMTP_SSL"`
}

func NewMailClient(config *Config) *mail.Client {
	mailService := config.Configsmtpservice
	var client *mail.Client
	var err error
	if mailService == "gmail" {
		client, err = mail.NewClient(config.Configsmtphost, mail.WithPort(config.Configsmtpport), mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(config.Configauthemail), mail.WithPassword(config.Configauthpassword))
		if err != nil {
			slog.Error("failed to create mail client: %s", err)
			return nil
		}
	} else if mailService == "outlook" {
		client, err = mail.NewClient(config.Configsmtphost, mail.WithPort(config.Configsmtpport), mail.WithSMTPAuth(mail.SMTPAuthLogin),
			mail.WithUsername(config.Configauthemail), mail.WithPassword(config.Configauthpassword))
		if err != nil {
			slog.Error("failed to create mail client: %s", err)
			return nil
		}
	} else {
		slog.Error("mail service not supported")
		return nil
	}
	if config.Configsmtpssl {
		client.SetSSL(true)
	}
	return client
}

func NewMail(client *mail.Client, email, subject, body, attachmentURL, sender string, cc []string) error {

	m := mail.NewMsg()
	if err := m.From(sender); err != nil {
		slog.Error("failed to set From address", slog.Any("error", err))
	}
	if strings.Contains(email, ",") {
		emailArray := strings.Split(email, ",")
		if err := m.To(emailArray...); err != nil {
			slog.Error("failed to set To address", slog.Any("error", err))
		}
	} else {
		if err := m.To(email); err != nil {
			slog.Error("failed to set To address", slog.Any("error", err))
		}
	}
	if err := m.Cc(cc...); err != nil {
		slog.Error("failed to set To address", slog.Any("error", err))
	}
	m.Subject(subject)
	m.SetBodyString(mail.TypeTextHTML, body)
	var pathFile string
	if attachmentURL != "" {
		filecheck := isUrlOrFile(attachmentURL)
		if filecheck == "url" {

			client := http.Client{
				CheckRedirect: func(r *http.Request, via []*http.Request) error {
					r.URL.Opaque = r.URL.Path
					return nil
				},
			}

			// Put content on file
			resp, err := client.Get(attachmentURL)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			fileURL := resp.Request.URL.RequestURI()
			segments := strings.Split(fileURL, "/")
			fileName := segments[len(segments)-1]
			pathFile = "./app/api/asset/" + fileName
			// Create blank file
			file, err := os.Create(pathFile)
			if err != nil {
				return err
			}
			_, err = io.Copy(file, resp.Body)
			if err != nil {
				return err
			}

			file, err = os.Open(pathFile)
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = filevalidation.ValidateFile(pathFile)

			if err != nil {
				return err
			}
			m.AttachReader(fileName, file)
		} else if filecheck == "file" {
			file, err := os.Open(attachmentURL)
			if err != nil {
				slog.Any("error", err)
			}
			defer file.Close()
			extension, err := filevalidation.ValidateImageReader(file)
			if err != nil {
				return err
			}
			m.AttachReader("attachment"+extension, file)
		}
	}
	var sendError error
	for retry := 0; retry < 3; retry++ {
		sendError = client.DialAndSend(m)
		if sendError == nil {
			// Email sent successfully, break the loop
			break
		}
		// Sleep for a moment before retrying (you can adjust the sleep duration)
		time.Sleep(2 * time.Second)
	}

	if sendError != nil {
		// Return an error if all retries fail
		return sendError
	}
	slog.Error("Mail Sent to: " + email)
	defer os.Remove(pathFile)
	return nil
}

func isUrlOrFile(input string) string {
	// Check if input is a URL
	_, err := url.ParseRequestURI(input)
	if err == nil && strings.Contains(input, "://") {
		return "url"
	}

	// Check if input is a local file
	_, err = os.Stat(input)
	if err == nil {
		return "file"
	}

	return ""
}
