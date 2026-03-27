package main

import (
    "fmt"
    "log"
    "os"

    _ "github.com/go-sql-driver/mysql"
    "github.com/jmoiron/sqlx"

    "event-engine-starter/config"
)

func main() {
    log.SetFlags(log.LstdFlags | log.Lmsgprefix)
    log.SetPrefix("[cleanup] ")

    cfgPath := "config.yaml"
    if os.Getenv("RAILWAY_ENVIRONMENT") != "" {
        cfgPath = "config.railway.yaml"
    } else if p := os.Getenv("CONFIG_PATH"); p != "" {
        cfgPath = p
    }

    cfg, err := config.Load(cfgPath)
    if err != nil {
        log.Fatalf("load config: %v", err)
    }

    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=UTC",
        cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name)

    db, err := sqlx.Connect("mysql", dsn)
    if err != nil {
        log.Fatalf("connect to MySQL: %v", err)
    }
    defer db.Close()

    log.Printf("connected to MySQL at %s:%d/%s", cfg.DB.Host, cfg.DB.Port, cfg.DB.Name)

    stmts := []string{
        "DELETE FROM event_error_log",
        "DELETE FROM event_payload",
        "DELETE FROM event",
    }

    for _, stmt := range stmts {
        if _, err := db.Exec(stmt); err != nil {
            log.Fatalf("exec %q: %v", stmt, err)
        }
    }

    log.Println("demo data cleared (event, event_payload, event_error_log)")
}
