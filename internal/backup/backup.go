package backup

import (
	"context"
	"fmt"
)
import "github.com/tderick/backup-companion-go/internal/config"

func Execute(ctx context.Context, cfg *config.Config) {
	fmt.Printf("Loaded config: %+v\n", cfg)
	fmt.Printf("Context %v\n", ctx)
	fmt.Println("Hello World")
}
