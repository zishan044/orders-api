package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/zishan044/orders-api/application"
)

func main() {
	app := application.New()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := app.Start(ctx); err != nil {
		fmt.Errorf("failed to start app: %w", err)
	}
}
