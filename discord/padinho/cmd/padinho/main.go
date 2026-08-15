package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	appbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/birthday"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/commands"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/config"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
	padinhodiscord "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	discordbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/birthday"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/job"
	birthdayjob "github.com/kiLeo13/SaladaFun/discord/padinho/internal/job/birthday"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/persistence/mysql"
)

const birthdayCheckInterval = time.Minute

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	db, err := database.Open()
	if err != nil {
		fail(logger, err)
	}
	defer database.Close(db)

	// Repositories
	configRepo := config.New(db)
	birthdayRepo := mysql.NewBirthdayRepository(db)

	token, err := configRepo.Get(config.AppToken)
	if err != nil {
		fail(logger, err)
	}
	if token == "" {
		fail(logger, errors.New("app.token is empty"))
	}
	channelID, err := configRepo.Get(config.BirthdayChannelID)
	if err != nil {
		fail(logger, err)
	}
	if channelID == "" {
		fail(logger, errors.New("birthday.channel_id is empty"))
	}

	// Services
	birthdayService := appbirthday.NewService(birthdayRepo)

	// Discord
	routes := padinhodiscord.NewRoutes()
	commands.Register(routes, birthdayService)
	if err := routes.Freeze(); err != nil {
		fail(logger, err)
	}

	gateway, err := padinhodiscord.New(token, routes, logger)
	if err != nil {
		fail(logger, err)
	}

	// Scheduled jobs
	birthdaySender := discordbirthday.NewSender(gateway, channelID)
	birthdayJob := birthdayjob.New(birthdayService, birthdaySender)
	scheduler := job.NewScheduler(logger)
	if err := scheduler.Every(birthdayCheckInterval, birthdayJob.Run); err != nil {
		fail(logger, err)
	}

	// Runtime
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	scheduler.Start(ctx)
	defer scheduler.Wait()
	if err := gateway.Run(ctx); err != nil {
		fail(logger, err)
	}
}

func fail(logger *slog.Logger, err error) {
	logger.Error("Padinho stopped", "error", err)
	os.Exit(1)
}
