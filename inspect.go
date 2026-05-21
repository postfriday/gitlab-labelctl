package main

import (
    "context"
    "fmt"
    "os"
    "github.com/postfriday/gitlab-labelctl/internal/config"
)

func main() {
    cfg, err := config.Load(context.Background(), "configs/root.yml")
    if err != nil {
        fmt.Println("ERROR:", err)
        os.Exit(1)
    }
    fmt.Printf("loaded version=%d include=%v\n", cfg.Version, cfg.Include)
}
