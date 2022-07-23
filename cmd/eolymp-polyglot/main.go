package main

import (
	"flag"
	"github.com/eolymp/go-packages/httpx"
	"github.com/eolymp/go-packages/oauth"
	"github.com/eolymp/go-sdk/eolymp/atlas"
	"github.com/eolymp/go-sdk/eolymp/keeper"
	"github.com/eolymp/go-sdk/eolymp/typewriter"
	c "github.com/eolymp/polyglot/cmd/config"
	"github.com/spf13/viper"
	"log"
	"net/http"
	"os"
	"time"
)

var client httpx.Client
var atl *atlas.AtlasService
var tw *typewriter.TypewriterService
var kpr *keeper.KeeperService
var conf c.Configuration

func main() {

	viper.SetConfigName("config")
	viper.AddConfigPath("./cmd/config")
	viper.AutomaticEnv()
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Error reading config file, %s", err)
	}

	err := viper.Unmarshal(&conf)
	if err != nil {
		log.Printf("Unable to decode into struct, %v", err)
	}

	client = httpx.NewClient(
		&http.Client{Timeout: 300 * time.Second},
		httpx.WithCredentials(oauth.PasswordCredentials(
			oauth.NewClient(conf.Eolymp.ApiUrl),
			conf.Eolymp.Username,
			conf.Eolymp.Password,
		)),
		httpx.WithHeaders(map[string][]string{
			"Space-ID": {conf.SpaceId},
		}),
	)

	atl = atlas.NewAtlas(client)

	tw = typewriter.NewTypewriter(client)
	kpr = keeper.NewKeeper(client)

	pid := flag.String("id", "", "Problem ID")
	flag.Parse()

	command := flag.Arg(0)

	switch command {
	case "ic":
		if contestId := flag.Arg(1); contestId == "" {
			log.Println("Path argument is empty")
			flag.Usage()
			os.Exit(-1)
		} else {
			ImportContest(contestId)
		}
	case "uc":
		if contestId := flag.Arg(1); contestId == "" {
			log.Println("Path argument is empty")
			flag.Usage()
			os.Exit(-1)
		} else {
			UpdateContest(contestId)
		}
	case "ip":
		for i, path := 1, flag.Arg(1); path != ""; i, path = i+1, flag.Arg(i+1) {
			if err := ImportProblem(path, pid); err != nil {
				log.Fatal(err)
			}
		}
	case "dp":
		for i, link := 1, flag.Arg(1); link != ""; i, link = i+1, flag.Arg(i+1) {
			if err := DownloadAndImportProblem(link, pid); err != nil {
				log.Fatal(err)
			}
		}
	case "up":
		for i, link := 1, flag.Arg(1); link != ""; i, link = i+1, flag.Arg(i+1) {
			if err := UpdateProblem(link); err != nil {
				log.Fatal(err)
			}
		}
	default:
		log.Fatal("no command found")
	}

}
