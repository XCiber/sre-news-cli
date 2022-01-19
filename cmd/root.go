/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/XCiber/sre-news-cli/internal/news"
	"github.com/XCiber/sre-news-cli/internal/slack"
	"github.com/XCiber/sre-news-cli/internal/utils"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var cfgFile string
var newsAPIURL string
var nasdaqHolidays news.NasdaqHolidays
var slackWebHook string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sre-news-cli",
	Short: "Post daily news to slack",
	Long:  `Command line tool which can grab news from news service and post them in slack with proper formatting.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		newsAPIURL = viper.GetString("news-api-url")
		slackWebHook = viper.GetString("slack-webhook")
		err := viper.UnmarshalKey("nasdaq-holidays", &nasdaqHolidays)
		if err != nil {
			log.Error(err)
		}
	},
	// Uncomment the following line if your bare application
	//has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {

		httpClient := &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     10 * time.Second,
				TLSHandshakeTimeout: 5 * time.Second,
				DialContext: (&net.Dialer{
					Timeout: 10 * time.Second,
				}).DialContext,
			},
		}
		newsClient := news.New(
			news.SetAPIBaseURL(newsAPIURL), //https://news.b2b.prod.env:8443
			news.SetHTTPClient(httpClient))

		from := utils.StartTimeOfDay(time.Now()).Unix()
		to := utils.EndTimeOfDay(time.Now()).Unix()

		res, err := newsClient.GetNews(from, to)
		if err != nil {
			log.Error(err)
		}

		res.Data.AddNasdaqOpening(nasdaqHolidays)

		dataForSlack := res.Data.MergeForSlack("USD|EUR|XAU|XAG|BTC")
		slackMessage, err := dataForSlack.RenderForSlack()
		if err != nil {
			log.Error(err)
		}
		slackClient := slack.New(slackWebHook)
		if err := slackClient.Send(slackMessage); err != nil {
			log.Error(err)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
	rootCmd.PersistentFlags().StringVar(&newsAPIURL, "news-api-url", "", "news api url")
	viper.BindPFlag("news-api-url", rootCmd.Flags().Lookup("news-api-url"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("SRE")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Info("Using config file: ", viper.ConfigFileUsed())
	}
}
