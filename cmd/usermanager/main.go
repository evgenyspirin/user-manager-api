package main

import (
	"context"
	"log"
	"os"

	"user-manager-api/internal"
)

// I focused on implementing more important features and left this list for later.
// todo: Implement:
// todo: Linters(GolangCiLint)
// todo: Central error handling pattern "SPE":
// https://medium.com/@yevheniikulhaviuk/golang-architectural-pattern-for-errors-531c0e54d67b

func main() {
	ctx := context.Background()

	app, err := internal.NewApp(ctx)
	if err != nil {
		log.Fatalf("init app failed: %v", err)
	}
	defer app.Close()

	app.InitControllers()

	if err = app.Run(ctx); err != nil {
		app.Logger().Sugar().Errorf("usermanagerapi stopped with error: %v", err)
		os.Exit(1)
	}
}
