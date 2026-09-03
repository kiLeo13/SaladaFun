package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	appaccounttree "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/accounttree"
	appbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/birthday"
	apppreferences "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/preferences"
	appquote "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/quote"
	appvoiceactivity "github.com/kiLeo13/SaladaFun/discord/padinho/internal/application/voiceactivity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/commands"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/config"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/database"
	padinhodiscord "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord"
	discordbirthday "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/birthday"
	discordourochest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/ourochest"
	discordouroquest "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/ouroquest"
	discordvoiceactivity "github.com/kiLeo13/SaladaFun/discord/padinho/internal/discord/voiceactivity"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/job"
	birthdayjob "github.com/kiLeo13/SaladaFun/discord/padinho/internal/job/birthday"
	"github.com/kiLeo13/SaladaFun/discord/padinho/internal/persistence/mysql"
)

const birthdayCheckSchedule = "0 * * * *"

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
	preferencesRepo := mysql.NewUserPreferencesRepository(db)
	quoteRepo := mysql.NewQuoteRepository(db)
	accountTreeRepo := mysql.NewDiscordAccountLinkRepository(db)
	voiceActivityRepo := mysql.NewVoiceActivityLogRepository(db)

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
	voiceActivityChannelID, err := configRepo.Get(config.VoiceActivityLogChannel)
	if err != nil {
		fail(logger, err)
	}
	if voiceActivityChannelID == "" {
		fail(logger, errors.New("channels.logs.voice is empty"))
	}
	betaSpiritURL, err := configRepo.Get(config.BetaSpiritYouTubeURL)
	if err != nil {
		fail(logger, err)
	}
	if betaSpiritURL == "" {
		fail(logger, errors.New("urls.youtube.betaSpirit is empty"))
	}
	mudaeSettings, err := configRepo.MudaeOQ()
	if err != nil {
		fail(logger, err)
	}

	// Services
	birthdayService := appbirthday.NewService(birthdayRepo, configRepo)
	preferencesService := apppreferences.NewService(preferencesRepo)
	quoteService := appquote.NewService(quoteRepo)
	accountTreeService := appaccounttree.NewService(accountTreeRepo)
	voiceActivityService := appvoiceactivity.NewService(voiceActivityRepo)

	// Discord
	routes := padinhodiscord.NewRoutes()
	gateway, err := padinhodiscord.New(token, routes, logger)
	if err != nil {
		fail(logger, err)
	}
	ouroChestListener, err := discordourochest.New(
		mudaeSettings.BotID,
		discordourochest.EmojiIDs{
			Blue: mudaeSettings.BlueEmojiID, Teal: mudaeSettings.TealEmojiID,
			Green: mudaeSettings.GreenEmojiID, Yellow: mudaeSettings.YellowEmojiID,
			Orange: mudaeSettings.OrangeEmojiID, Red: mudaeSettings.RedEmojiID,
		},
		gateway,
		preferencesService,
		gateway,
		logger,
	)
	if err != nil {
		fail(logger, err)
	}
	if err := gateway.AddSubscriber(ouroChestListener); err != nil {
		fail(logger, err)
	}
	ouroQuestListener, err := discordouroquest.New(
		mudaeSettings.BotID,
		discordouroquest.EmojiIDs{
			Blue: mudaeSettings.BlueEmojiID, Teal: mudaeSettings.TealEmojiID,
			Green: mudaeSettings.GreenEmojiID, Yellow: mudaeSettings.YellowEmojiID,
			Orange: mudaeSettings.OrangeEmojiID, Purple: mudaeSettings.PurpleEmojiID,
			Red: mudaeSettings.RedEmojiID,
		},
		gateway, preferencesService, gateway, logger,
	)
	if err != nil {
		fail(logger, err)
	}
	if err := gateway.AddSubscriber(ouroQuestListener); err != nil {
		fail(logger, err)
	}
	voiceActivityListener, err := discordvoiceactivity.New(voiceActivityChannelID, gateway, voiceActivityService, logger)
	if err != nil {
		fail(logger, err)
	}
	if err := gateway.AddSubscriber(voiceActivityListener); err != nil {
		fail(logger, err)
	}
	commands.Register(
		routes, birthdayService, quoteService, accountTreeService, betaSpiritURL,
		gateway, ouroChestListener, ouroQuestListener,
	)
	if err := routes.Freeze(); err != nil {
		fail(logger, err)
	}

	// Scheduled jobs
	birthdaySender := discordbirthday.NewSender(gateway, channelID)
	birthdayJob := birthdayjob.New(birthdayService, birthdaySender)
	scheduler := job.NewScheduler(logger)
	if err := scheduler.Schedule(birthdayCheckSchedule, birthdayJob.Run); err != nil {
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
