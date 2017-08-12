package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	_ "github.com/KristinaEtc/slflog"

	"github.com/KristinaEtc/config"
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

// ConfFile is a file with all program options
type ConfFile struct {
	Name     string
	Choosing string
	Jira     JiraConf
}

var globalConf = ConfFile{
	Name:     "of-the-day program",
	Choosing: "The fool",
	Jira: JiraConf{
		Address:     "localhost",
		User:        "guest",
		Password:    "guest",
		Path:        "/rest/api/2/user/assignable/search?project=",
		ProjectName: "MyProject",
	},
}

type responceBody []colleague

type colleague struct {
	DisplayName string `json:"displayName"`
}

func choose(scopeTable map[string]int) string {
	// TODO: choose graceful with metrics considering
	for k := range scopeTable {
		return k
	}
	return ""
}

func initScopeTable(colleagues []colleague) (scopeTable map[string]int) {
	scopeTable = make(map[string]int)
	log.Infof("%+v", colleagues)
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
	//}err == io.EOF {

	return pData, nil
}

func main() {
	config.ReadGlobalConfig(&globalConf, "of-the-day")
	log.Infof("%s", globalConf.Name)
	log.Error("----------------------------------------------")

	log.Infof("BuildDate=%s\n", BuildDate)
	log.Infof("GitCommit=%s\n", GitCommit)
	log.Infof("GitBranch=%s\n", GitBranch)
	log.Infof("GitState=%s\n", GitState)
	log.Infof("GitSummary=%s\n", GitSummary)
	log.Infof("VERSION=%s\n", Version)

	colleagues, err := getColleagues(globalConf.Jira)
	if err != nil {
		log.Errorf("Getting colleagues from project %s: %s", globalConf.Jira.ProjectName, err.Error())
		return
	}

	scopeTable := initScopeTable(colleagues)
	log.Debugf("Scope table: %+v", scopeTable)

	winner := choose(scopeTable)
	log.Infof("%s of the day is... %s. GRATS!", globalConf.Choosing, winner)

}
