package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/KristinaEtc/config"
	_ "github.com/KristinaEtc/slflog"
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

// ConfFile is a file with all program options
type ConfFile struct {
	Name        string
	Address     string
	User        string
	Password    string
	Path        string
	ProjectName string
}

var globalOpt = ConfFile{
	Name:        "of-the-day program",
	Address:     "localhost",
	User:        "guest",
	Password:    "guest",
	Path:        "/rest/api/2/user/assignable/search?project=",
	ProjectName: "MyProject",
}

type responceBody []colleague

type colleague struct {
	DisplayName string `json:"displayName"`
}

func initScopeTable(colleagues []colleague) (scopeTable map[string]int) {
	scopeTable = make(map[string]int)
	log.Infof("%+v", colleagues)
	for _, c := range colleagues {
		scopeTable[c.DisplayName] = 0
	}
	return
}

func getColleagues() (responceBody, error) {
	uri := globalOpt.Address + globalOpt.Path + globalOpt.ProjectName
	log.Debugf("URI= %s", uri)

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, fmt.Errorf("creating a newrequest [%s]: %s", uri, err.Error())
	}

	req.SetBasicAuth(globalOpt.User, globalOpt.Password)
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
	config.ReadGlobalConfig(&globalOpt, "template options")
	log.Infof("%s", globalOpt.Name)
	log.Error("----------------------------------------------")

	log.Infof("BuildDate=%s\n", BuildDate)
	log.Infof("GitCommit=%s\n", GitCommit)
	log.Infof("GitBranch=%s\n", GitBranch)
	log.Infof("GitState=%s\n", GitState)
	log.Infof("GitSummary=%s\n", GitSummary)
	log.Infof("VERSION=%s\n", Version)

	colleagues, err := getColleagues()
	if err != nil {
		log.Errorf("Getting colleagues from project %s: %s", globalOpt.ProjectName, err.Error())
		return
	}
	log.Infof("%+v", colleagues)

	scopeTable := initScopeTable(colleagues)
	log.Infof("Scope table: %+v", scopeTable)
}
