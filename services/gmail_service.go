package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type GmailService struct {
	service   *gmail.Service
	userEmail string
}

// NewGmailService 创建Gmail服务实例
func NewGmailService(credentialsFile, tokenFile, userEmail string) (*GmailService, error) {
	b, err := ioutil.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("无法读取凭证文件: %v", err)
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope, gmail.GmailSendScope)
	if err != nil {
		return nil, fmt.Errorf("无法解析凭证文件: %v", err)
	}

	client := getClient(config, tokenFile)

	srv, err := gmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("无法创建Gmail服务: %v", err)
	}

	return &GmailService{
		service:   srv,
		userEmail: userEmail,
	}, nil
}

// getClient 获取OAuth2客户端
func getClient(config *oauth2.Config, tokFile string) *http.Client {
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// getTokenFromWeb 从Web获取token
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("在浏览器中打开以下链接进行授权: \n%v\n", authURL)

	var authCode string
	fmt.Print("粘贴授权码: ")
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("无法读取授权码: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("无法获取token: %v", err)
	}
	return tok
}

// tokenFromFile 从文件读取token
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// saveToken 保存token到文件
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("保存凭证文件到: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("无法缓存oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// GetUnreadEmails 获取未读邮件
func (gs *GmailService) GetUnreadEmails() ([]*EmailMessage, error) {
	query := "is:unread"
	
	req := gs.service.Users.Messages.List("me").Q(query)
	
	r, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf("无法获取邮件列表: %v", err)
	}

	var emails []*EmailMessage
	
	for _, m := range r.Messages {
		msg, err := gs.service.Users.Messages.Get("me", m.Id).Do()
		if err != nil {
			log.Printf("无法获取邮件详情 %s: %v", m.Id, err)
			continue
		}

		email := parseEmailMessage(msg)
		if email != nil {
			emails = append(emails, email)
		}
	}

	return emails, nil
}

// SendEmail 发送邮件
func (gs *GmailService) SendEmail(to, subject, body string) error {
	var message gmail.Message

	emailTo := "To: " + to + "\r\n"
	emailSubject := "Subject: " + subject + "\r\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	msg := []byte(emailTo + emailSubject + mime + "\r\n" + body)

	message.Raw = base64.URLEncoding.EncodeToString(msg)

	_, err := gs.service.Users.Messages.Send("me", &message).Do()
	if err != nil {
		return fmt.Errorf("无法发送邮件: %v", err)
	}

	return nil
}

// MarkAsRead 标记邮件为已读
func (gs *GmailService) MarkAsRead(messageID string) error {
	req := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}

	_, err := gs.service.Users.Messages.Modify("me", messageID, req).Do()
	if err != nil {
		return fmt.Errorf("无法标记邮件为已读: %v", err)
	}

	return nil
}

// EmailMessage 邮件消息结构
type EmailMessage struct {
	ID          string
	Subject     string
	From        string
	To          string
	Body        string
	ReceivedAt  time.Time
}

// parseEmailMessage 解析邮件消息
func parseEmailMessage(msg *gmail.Message) *EmailMessage {
	email := &EmailMessage{
		ID: msg.Id,
	}

	// 解析头部信息
	for _, header := range msg.Payload.Headers {
		switch header.Name {
		case "Subject":
			email.Subject = header.Value
		case "From":
			email.From = header.Value
		case "To":
			email.To = header.Value
		case "Date":
			if t, err := time.Parse(time.RFC1123Z, header.Value); err == nil {
				email.ReceivedAt = t
			}
		}
	}

	// 解析邮件正文
	email.Body = extractBody(msg.Payload)

	return email
}

// extractBody 提取邮件正文
func extractBody(payload *gmail.MessagePart) string {
	var body string

	if payload.Body != nil && payload.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			body = string(data)
		}
	}

	// 如果是多部分邮件，递归提取
	for _, part := range payload.Parts {
		if strings.Contains(part.MimeType, "text/") {
			if partBody := extractBody(part); partBody != "" {
				body += partBody
			}
		}
	}

	return body
}