package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// 全局代理配置
var (
	ProxyURL string = "http://127.0.0.1:10810" // 在这里设置您的代理地址，例如: "http://127.0.0.1:7890"
)

type GmailService struct {
	service   *gmail.Service
	userEmail string
}

// SetProxy 设置代理地址
func SetProxy(proxyURL string) {
	ProxyURL = proxyURL
	log.Printf("已设置代理: %s", proxyURL)
}

// NewGmailService 创建Gmail服务实例
func NewGmailService(credentialsFile, tokenFile, userEmail string) (*GmailService, error) {
	// 检查凭据文件是否存在
	if _, err := os.Stat(credentialsFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("Gmail凭据文件不存在: %s\n请按照以下步骤获取凭据：\n1. 访问 https://console.cloud.google.com/\n2. 创建OAuth 2.0客户端ID（桌面应用）\n3. 下载JSON文件并重命名为 credentials.json", credentialsFile)
	}

	b, err := ioutil.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("无法读取凭证文件: %v", err)
	}

	// 解析OAuth 2.0配置
	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope, gmail.GmailSendScope, gmail.GmailModifyScope)
	if err != nil {
		return nil, fmt.Errorf("无法解析OAuth 2.0配置: %v\n请确保下载的是OAuth 2.0客户端ID，而不是API密钥", err)
	}

	// 获取OAuth2 token
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		// 创建支持代理的HTTP客户端用于token获取
		proxyClient := createHTTPClientWithProxy()
		tok = getTokenFromWeb(config, proxyClient)
		saveToken(tokenFile, tok)
	} else {
		// 检查token是否有效，如果无效则重新获取
		if !isTokenValid(tok) {
			log.Printf("Token已过期，重新获取...")
			proxyClient := createHTTPClientWithProxy()
			tok = getTokenFromWeb(config, proxyClient)
			saveToken(tokenFile, tok)
		}
	}
	
	// 创建支持代理的HTTP客户端
	proxyClient := createHTTPClientWithProxy()
	
	// 创建OAuth2客户端，并确保代理配置正确应用
	var oauthClient *http.Client
	if proxyClient.Transport != nil {
		// 如果设置了代理，创建一个新的Transport，同时保持OAuth2认证
		oauthClient = &http.Client{
			Transport: &oauth2.Transport{
				Base:   proxyClient.Transport,
				Source: config.TokenSource(context.Background(), tok),
			},
		}
	} else {
		// 如果没有代理，使用标准的OAuth2客户端
		oauthClient = config.Client(context.Background(), tok)
	}
	
	srv, err := gmail.NewService(context.Background(), option.WithHTTPClient(oauthClient))
	if err != nil {
		return nil, fmt.Errorf("无法创建Gmail服务: %v", err)
	}

	return &GmailService{
		service:   srv,
		userEmail: userEmail,
	}, nil
}



// createHTTPClientWithProxy 创建支持代理的HTTP客户端
func createHTTPClientWithProxy() *http.Client {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// 优先使用代码中设置的代理
	if ProxyURL != "" {
		proxyURL, err := url.Parse(ProxyURL)
		if err != nil {
			log.Printf("警告: 无法解析代理URL %s: %v", ProxyURL, err)
		} else {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
			log.Printf("已配置代理: %s", ProxyURL)
		}
		return client
	}
	
	// 如果没有设置代理，检查环境变量
	httpProxy := os.Getenv("HTTP_PROXY")
	httpsProxy := os.Getenv("HTTPS_PROXY")
	
	// 如果没有设置HTTPS代理，尝试使用HTTP代理
	if httpsProxy == "" {
		httpsProxy = httpProxy
	}
	
	// 如果没有设置HTTP代理，尝试使用小写的环境变量
	if httpProxy == "" {
		httpProxy = os.Getenv("http_proxy")
	}
	if httpsProxy == "" {
		httpsProxy = os.Getenv("https_proxy")
	}
	
	// 如果设置了代理，配置代理
	if httpsProxy != "" {
		proxyURL, err := url.Parse(httpsProxy)
		if err != nil {
			log.Printf("警告: 无法解析代理URL %s: %v", httpsProxy, err)
		} else {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
			log.Printf("已配置代理: %s", httpsProxy)
		}
	} else {
		log.Println("未配置代理")
	}
	
	return client
}

// getTokenFromWeb 从Web获取token
func getTokenFromWeb(config *oauth2.Config, client *http.Client) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("在浏览器中打开以下链接进行授权: \n%v\n", authURL)

	var authCode string
	fmt.Print("粘贴授权码: ")
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("无法读取授权码: %v", err)
	}

	// 使用配置了代理的客户端进行token交换
	ctx := context.WithValue(context.TODO(), oauth2.HTTPClient, client)
	tok, err := config.Exchange(ctx, authCode)
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

// isTokenValid 检查token是否有效
func isTokenValid(token *oauth2.Token) bool {
	if token == nil {
		return false
	}
	
	// 检查token是否过期
	if token.Expiry.Before(time.Now()) {
		return false
	}
	
	// 检查是否有访问令牌
	if token.AccessToken == "" {
		return false
	}
	
	return true
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

// GetUnreadEmails 获取未读邮件（分批处理）
func (gs *GmailService) GetUnreadEmails() ([]*EmailMessage, error) {
	// 使用分批处理，确保处理完所有邮件
	// 每批50封，最多10批（总共500封）
	return gs.GetUnreadEmailsBatch(50, 10)
}

// GetUnreadEmailsWithLimit 获取指定数量的未读邮件
func (gs *GmailService) GetUnreadEmailsWithLimit(maxResults int64) ([]*EmailMessage, error) {
	if maxResults <= 0 {
		maxResults = 50 // 默认限制
	}
	if maxResults > 500 {
		maxResults = 500 // 最大限制
	}

	query := "is:unread -in:trash -in:spam"
	
	req := gs.service.Users.Messages.List("me").Q(query).MaxResults(maxResults)
	
	r, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf("无法获取邮件列表: %v", err)
	}

	log.Printf("获取到 %d 封未读邮件（最大限制: %d）", len(r.Messages), maxResults)

	var emails []*EmailMessage
	
	// 使用goroutine批量处理邮件详情获取
	emailChan := make(chan *EmailMessage, len(r.Messages))
	errorChan := make(chan error, len(r.Messages))
	
	// 限制并发数量，避免API调用过于频繁
	semaphore := make(chan struct{}, 10) // 最多10个并发
	
	for _, m := range r.Messages {
		go func(messageID string) {
			semaphore <- struct{}{} // 获取信号量
			defer func() { <-semaphore }() // 释放信号量
			
			msg, err := gs.service.Users.Messages.Get("me", messageID).Do()
			if err != nil {
				log.Printf("无法获取邮件详情 %s: %v", messageID, err)
				errorChan <- err
				return
			}

			email := parseEmailMessage(msg)
			if email != nil {
				emailChan <- email
			} else {
				errorChan <- fmt.Errorf("解析邮件失败: %s", messageID)
			}
		}(m.Id)
	}

	// 收集结果
	for i := 0; i < len(r.Messages); i++ {
		select {
		case email := <-emailChan:
			emails = append(emails, email)
		case err := <-errorChan:
			log.Printf("处理邮件时出错: %v", err)
		}
	}

	log.Printf("成功处理 %d 封邮件", len(emails))
	return emails, nil
}

// GetUnreadEmailsBatch 分批获取所有未读邮件
func (gs *GmailService) GetUnreadEmailsBatch(batchSize int64, maxBatches int) ([]*EmailMessage, error) {
	if batchSize <= 0 {
			batchSize = 50
	}
	if maxBatches <= 0 {
			maxBatches = 10
	}
	var allEmails []*EmailMessage
	pageToken := ""
	batchCount := 0
	seenIDs := make(map[string]bool) // 用于去重

	for batchCount < maxBatches {
			query := "is:unread -in:trash -in:spam"
			req := gs.service.Users.Messages.List("me").Q(query).MaxResults(batchSize).IncludeSpamTrash(false)
			
			if pageToken != "" {
					req = req.PageToken(pageToken)
			}

			r, err := req.Do()
			if err != nil {
					return allEmails, fmt.Errorf("failed to list batch %d: %v", batchCount+1, err)
			}

			log.Printf("Processing batch %d: %d messages", batchCount+1, len(r.Messages))

			// 批量获取邮件详情（可改用 goroutine 并发，控制并发数）
			for _, m := range r.Messages {
					if seenIDs[m.Id] {
							continue // 跳过已处理的邮件
					}
					seenIDs[m.Id] = true

					msg, err := gs.service.Users.Messages.Get("me", m.Id).Format("full").Do()
					if err != nil {
							log.Printf("Failed to get message %s (will retry): %v", m.Id, err)
							// 可加入重试逻辑
							continue
					}

					if email := parseEmailMessage(msg); email != nil {
							allEmails = append(allEmails, email)
					}
			}

			if r.NextPageToken == "" {
					break
			}

			pageToken = r.NextPageToken
			batchCount++
			time.Sleep(50 * time.Millisecond) // 更短的间隔
	}

	if batchCount >= maxBatches {
			log.Printf("Stopped after reaching max batches (%d), total emails: %d", maxBatches, len(allEmails))
	}
	return allEmails, nil
}

// SendEmail 发送邮件
func (gs *GmailService) SendEmail(to, subject, body string) error {
	var message gmail.Message

	// 对邮件标题进行UTF-8编码处理
	encodedSubject := encodeSubject(subject)
	
	emailTo := "To: " + to + "\r\n"
	emailSubject := "Subject: " + encodedSubject + "\r\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	msg := []byte(emailTo + emailSubject + mime + "\r\n" + body)

	message.Raw = base64.URLEncoding.EncodeToString(msg)

	_, err := gs.service.Users.Messages.Send("me", &message).Do()
	if err != nil {
		return fmt.Errorf("无法发送邮件: %v", err)
	}

	return nil
}

// encodeSubject 编码邮件标题
func encodeSubject(subject string) string {
	// 检查是否包含非ASCII字符
	hasNonASCII := false
	for _, r := range subject {
		if r > 127 {
			hasNonASCII = true
			break
		}
	}
	
	if !hasNonASCII {
		return subject
	}
	
	// 使用UTF-8编码
	return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="
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