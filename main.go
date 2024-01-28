package main

import (
	"context"
	"fmt"

	"github.com/zishan044/orders-api/application"
)

func main() {
	app := application.New()
	if err := app.Start(context.TODO()); err != nil {
		fmt.Errorf("failed to start app: %w", err)
	}
}
