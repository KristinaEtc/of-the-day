package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	_ "github.com/KristinaEtc/slflog"

	"github.com/KristinaEtc/config"
	"github.com/Syfaro/telegram-bot-api"
	"github.com/ventu-io/slf"
)

var log = slf.WithContext("of-the-day")

// These fields are populated by govvv
var (
	BuildDate  string
	GitCommit  string
	GitBranch  string
	GitState   string
	GitSummary string
	Version    string
)

// JiraConf stores all configurations about jira request
type JiraConf struct {
	Address     string
	User        string
	Password    string
	Path        string
	ProjectName string
}

// TelegramConf is a file with telegram stuff.
type TelegramConf struct {
	Token   string
	Timeout int
	Debug   bool
}

// ConfFile is a file with all program options
type ConfFile struct {
	Name       string
	Nomination string
	Jira       JiraConf
	Telegram   TelegramConf
}

var globalConf = ConfFile{
	Name:       "of-the-day program",
	Nomination: "The fool",
	Jira: JiraConf{
		Address:     "localhost",
		User:        "guest",
		Password:    "guest",
		Path:        "/rest/api/2/user/assignable/search?project=",
		ProjectName: "MyProject",
	},
	Telegram: TelegramConf{
		Token:   "token",
		Timeout: 60,
		Debug:   true,
	},
}

type responceBody []colleague

type colleague struct {
	DisplayName string `json:"displayName"`
}

var winnerUser winner

type winner struct {
	nomitation string
	winner     string
	updateDay  int
	m          *sync.Mutex
}

func getRandomColleague(scopeTable map[string]int) string {
	for k := range scopeTable {
		return k
	}
	return ""
}

func updateWinner(scopeTable map[string]int) {
	// TODO: choose graceful with metrics considering
	winnerUser.m.Lock()
	defer winnerUser.m.Unlock()

	_, _, dayNow := time.Now().Date()
	if winnerUser.updateDay == dayNow {
		return
	}
	winnerUser.winner = getRandomColleague(scopeTable)
}

func initScopeTable(colleagues []colleague) (scopeTable map[string]int) {
	scopeTable = make(map[string]int)
	for _, c := range colleagues {
		scopeTable[c.DisplayName] = 0
	}
	return
}

func getColleagues(conf JiraConf) (responceBody, error) {
	uri := conf.Address + conf.Path + conf.ProjectName
	log.Debugf("URI= %s", uri)

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("creating a newrequest [%s]: %s", uri, err.Error())
	}

	req.SetBasicAuth(conf.User, conf.Password)
	req.Header.Add("Content-Type", "application/json")

	c := http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request [%s]: %s", uri, err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("executing request [%s]: responce with status %d", uri, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading request body: %s", err.Error())
	}

	decoder := json.NewDecoder(strings.NewReader(string(body)))
	var pData responceBody
	if err := decoder.Decode(&pData); err != nil {
		return nil, fmt.Errorf("decoding responce body: %s", err.Error())
	}
	return pData, nil
}

func run(conf TelegramConf, scopeTable map[string]int) (err error) {
	bot, err := tgbotapi.NewBotAPI(conf.Token)
	if err != nil {
		return fmt.Errorf("connecting to a bot: %s", err.Error())
	}

	if conf.Debug {
		bot.Debug = true
	}
	log.Infof("Authorized on account %s", bot.Self.UserName)
	log.Debugf("Token=[%s]", conf.Token)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = conf.Timeout

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		return fmt.Errorf("getting update chan: %s", err.Error())
	}

	var responseMsg string
	var msg tgbotapi.MessageConfig

	for update := range updates {
		if update.Message == nil {
			continue
		}
		log.Debugf("[%s] %s", update.Message.From.UserName, update.Message.Text)
		switch update.Message.Text {
		case "lox":
			log.Debug("lox detected")
			updateWinner(scopeTable)
			responseMsg = fmt.Sprintf("%s of the day is... %s. GRATS!", globalConf.Nomination, winnerUser.winner)
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, responseMsg)
			msg.ReplyToMessageID = update.Message.MessageID
		default:
			responseMsg = "Slabo stelish, pirozhok. Poprobuj \"lox\" command"
			msg = tgbotapi.NewMessage(update.Message.Chat.ID, responseMsg)
			msg.ReplyToMessageID = update.Message.MessageID
		}
		bot.Send(msg)
	}
	return nil
}

func main() {
	config.ReadGlobalConfig(&globalConf, "of-the-day")
	log.Infof("%s", globalConf.Name)
	log.Error("----------------------------------------------")

	log.Infof("BuildDate=%s", BuildDate)
	log.Infof("GitCommit=%s", GitCommit)
	log.Infof("GitBranch=%s", GitBranch)
	log.Infof("GitState=%s", GitState)
	log.Infof("GitSummary=%s", GitSummary)
	log.Infof("VERSION=%s\n", Version)

	colleagues, err := getColleagues(globalConf.Jira)
	if err != nil {
		log.Errorf("Getting colleagues from project %s: %s", globalConf.Jira.ProjectName, err.Error())
		return
	}

	scopeTable := initScopeTable(colleagues)
	log.Debugf("Scope table: %+v", scopeTable)

	_, _, currDay := time.Now().Date()
	winnerUser = winner{
		nomitation: globalConf.Nomination,
		updateDay:  currDay,
		m:          &sync.Mutex{},
		winner:     getRandomColleague(scopeTable),
	}

	err = run(globalConf.Telegram, scopeTable)
	if err != nil {
		log.Errorf("running: %s", err.Error())
	}
}
