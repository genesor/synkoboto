package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	"github.com/bwmarrin/discordgo"
	synkoboto "github.com/genesor/synkoboto/pkg"
	"github.com/genesor/synkoboto/pkg/synctube"
	"github.com/heetch/confita"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func main() {
	ctx := context.Background() // TODO maybe have a timeout ctx

	if err := godotenv.Load(".env"); err != nil {
		if !os.IsNotExist(err) {
			logger.WithError(err).Fatal("unable to load .env file")
		}
		logger.WithError(err).Info("no .env file found")
	}

	cfg := &synkoboto.Configuration{
		RoomName: "Synkoboto",
	}
	if err := confita.NewLoader().Load(ctx, cfg); err != nil {
		logger.WithError(err).Fatal("error getting bot configuration from env")
	}

	logger.Info("configuration loaded")

	d, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		logger.WithError(err).Fatal("error creating new discord client")
	}

	d.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		logger.Infof("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})

	if err := d.Open(); err != nil {
		logger.WithError(err).Fatal("Cannot open the session")
	}

	g, err := d.Guild(cfg.ServerID)
	if err != nil {
		logger.WithError(err).Fatal("cannot retrieve server information")
	}

	logger.Infof("bot running on server %s #%s", g.Name, g.ID)

	logger.Info("Adding commands")

	cmd, err := d.ApplicationCommandCreate(cfg.AppID, cfg.ServerID, &discordgo.ApplicationCommand{
		Name:        "synctube",
		Description: "Create a new synctube room",
	})
	if err != nil {
		logger.WithError(err).Fatal("error registering command %s", "synctube")
	}
	logger.Infof("command %s added with ID #%s", cmd.Name, cmd.ID)

	creator := synctube.NewCreator(cfg, http.DefaultClient, logger)

	d.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		uName := ""
		if i.User != nil {
			uName = i.User.Username
		} else if i.Member != nil {
			uName = i.Member.User.Username
		}
		cmdStr := i.ApplicationCommandData().Name

		logger.Infof("user %s triggered command %s", uName, cmdStr)

		if cmdStr != "synctube" {
			logger.Errorf("unknown command %s", cmdStr)
			return
		}

		r, err := creator.CreateRoom()
		if err != nil {
			logger.WithError(err).Error("error creating room")

			publishInteractionReponse(s, i, "error creating synctube room")
			return
		}
		logger.Infof("room %s created", r.ID)

		if err := creator.SetPermissions(r); err != nil {
			logger.WithError(err).Error("error setting room permissions")

			publishInteractionReponse(s, i, "error setting up room permissions\nRoom avaiable: "+r.URL)
			return
		}

		logger.Infof("room %s finalized", r.ID)

		publishInteractionReponse(s, i, r.URL)
		return
	})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, os.Kill)
	logger.Warn("Press Ctrl+C to exit")
	<-stop

	if err := d.ApplicationCommandDelete(cfg.AppID, cfg.ServerID, cmd.ID); err != nil {
		logger.WithError(err).Panicf("Cannot delete %v command #%s", cmd.Name, cmd.ID)
	}
	logger.Infof("Command %v #%s deleted", cmd.Name, cmd.ID)
}

func publishInteractionReponse(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
		},
	})
	if err != nil {
		logger.WithError(err).Error("error publishing interaction response")
	}
}
