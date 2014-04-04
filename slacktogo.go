//  project main.go
package main

import (
	"fmt"
	"log"
	"os"
	"net/http"
	"net/url"
	"time"
	"strings"
	"encoding/json"
	"bitbucket.org/kardianos/service"
	"bitbucket.org/kardianos/osext"
	"github.com/BurntSushi/toml"
	"path"
)

type TeamID string
type TeamName string
type Channel string

type Team struct {
	id TeamID
	url string
}

type Destination struct {
	url string
	channel Channel
}

type MappingID struct {
	id TeamID
	channel Channel
}

// Special Structures for the config parsing

type configTeam struct {
	ID TeamID
	URL string
}

type configMapping struct {
	FromTeam TeamName
	FromChannel Channel
	ToTeam TeamName
	ToChannel Channel
}

type tomlConfig struct {
	BindAddr string
	Port uint16
	SSLCrt string
	SSLKey string
	Teams map[TeamName]configTeam `toml:"team"`
	Mappings []configMapping `toml:"mapping"`
}

var teams=make(map[TeamName]Team)
var team_namen=make(map[TeamID]TeamName)
var mapping=make(map[MappingID][]Destination)

var echos=0

func AddMapping(from_team TeamName, from_channel Channel, to_team TeamName, to_channel Channel) {
	id:=MappingID{
		id:teams[from_team].id,
		channel:from_channel,
	}
	mapping[id]=append(mapping[id],Destination{
			url: teams[to_team].url,
			channel: to_channel })
}

func AddTeam(name TeamName, id TeamID, url string) {
	teams[name]=Team{
		id: id,
		url: url }
	team_namen[id]=name
}

func NoCache(w http.ResponseWriter) {
	h:=w.Header()
	h.Add("Expires","Mon, 1 Jan 2000 00:00:00 UTC")
	h.Add("Last-Modified",time.Now().UTC().Format(time.RFC1123))
	h.Add("Cache-Control", "no-store, no-cache, must-revalidate")
	//h.Add("Cache-Control", "post-check=0, pre-check=0");
	h.Add("Pragma","no-cache")
}

func BotAnswers(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "application/json")
	tmp := map[string]interface{}{"text": text }
	res, _ := json.Marshal(tmp)
	fmt.Fprintf(w, string(res));
	return
}

func OnRequest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	NoCache(w)	// well we never want it cached!

	team_id:=TeamID(r.PostFormValue("team_id"))

	if(team_id=="") {
		w.WriteHeader(http.StatusPreconditionFailed)
		fmt.Fprintf(w, "Illegal access to Slackgate... (c) METATEXX GmbH 2014")
		return
	}

	from_channel:=Channel("#"+r.PostFormValue("channel_name"))
	mapping_id:=MappingID{
		id: team_id,
		channel: from_channel,
	}

	user_name:=r.PostFormValue("user_name");
	if(user_name=="slackbot") {
		echos++
		w.WriteHeader(http.StatusNoContent);
		return
	}

	dsts, ok:=mapping[mapping_id]
	if(! ok) {
		BotAnswers(w, fmt.Sprintf("Unknown Team: %q",mapping_id))
		return
	}

	msg_body:=r.PostFormValue("text");

	fmt.Printf("%s %s / %s: %q\n",team_id,from_channel,user_name,msg_body)

	if(msg_body=="+info") {
		BotAnswers(w, "This is SlackToGo Server V0.2.0 (c) METATEXX GmbH 2014 - Written by Hans Raaf")
		return
	}

	if(msg_body=="+date") {
		BotAnswers(w, time.Now().UTC().Format(time.RFC1123))
		return
	}

	if(msg_body=="+echos") {
		BotAnswers(w, fmt.Sprintf("Echos: %d",echos))
		return
	}

	if(msg_body=="+routing") {
		out:=make([]string,0)
		for mid, maps := range mapping {
			if(len(out)>0) {
				out=append(out,"");
			}
			out=append(out,fmt.Sprintf("Routing messages from %q (%s) %s to:", team_namen[mid.id], mid.id, mid.channel));
			for idx, dst := range maps {
				out=append(out,fmt.Sprintf("> %d:  %q @ %q",idx,dst.url, dst.channel));
			}
		}
		BotAnswers(w, strings.Join(out,"\n"))
		return
	}

	for _, dst := range dsts {
		tmp := map[string]interface{}{
			"channel": dst.channel,
			"username": fmt.Sprintf("*%s",user_name),
			"text": msg_body,
			"icon_emoji": ":twisted_rightwards_arrows:" }
		res, _ := json.Marshal(tmp)

		resp, err := http.PostForm(dst.url,
			url.Values{"payload": { string(res)}})
		if(err!=nil) {
			w.Header().Set("Content-Type", "application/json")
			tmp := map[string]interface{}{"text": fmt.Sprintf("Error: %q!",err.Error()) }
			res, _ := json.Marshal(tmp)
			fmt.Fprintf(w, string(res));
			return
		}

		if(resp.StatusCode!=200) {
			fmt.Println(resp)
		}

		resp.Body.Close();
	}

	return
}

func checkConfig() {
	fmt.Printf("BindAddr: %q\n",config.BindAddr)
	fmt.Printf("Port: %d\n",config.Port)
	fmt.Printf("SSL Certificate: %q\n",config.SSLCrt)
	fmt.Printf("SSL Key: %q\n",config.SSLKey)

	fmt.Println("\nTeams:")
	for name, team := range config.Teams {
		fmt.Printf("%s (%s) %s\n",
			name,
			team.ID,
			team.URL,
		)
	}

	fmt.Println("\nMappings:")
	for idx, mapping := range config.Mappings {
		fmt.Printf("%d: %s %s -> %s %s\n",
			idx,
			mapping.FromTeam,
			mapping.FromChannel,
			mapping.ToTeam,
			mapping.ToChannel,
		)
	}
}

var logit service.Logger

var config tomlConfig

func main() {

	// Config Defaults
	config.BindAddr="[::]"
	config.Port=8080
	config.SSLCrt=""
	config.SSLKey=""

	// The "go run" command creates the executable in the tmp dir of the system
	// the "service" wrapper runs a strange current dir

	curdir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	//dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	prgdir, err := osext.ExecutableFolder()
	if err != nil {
		log.Fatal(err)
	}

	// trying to find the config file
	if _, err := toml.DecodeFile(path.Join(prgdir,"config.toml"), &config); err != nil {
		if _, err := toml.DecodeFile(path.Join(curdir,"config.toml"), &config); err != nil {
			fmt.Println(err)
			return
		}
	}

	// Service Setup
	var name = "SlackToGo"
	var displayName = "Slack Team Gateway"
	var desc = "This Gateway connects two channels on different teams ins slack with each other."

	s, err := service.NewService(name, displayName, desc)
	logit = s

	if err != nil {
		fmt.Printf("%s unable to start: %s", displayName, err)
		return
	}

	if len(os.Args) > 1 {
		var err error
		verb := os.Args[1]
		switch verb {
		case "install":
			err = s.Install()
			if err != nil {
				fmt.Printf("Failed to install: %s\n", err)
				return
			}
			fmt.Printf("Service %q installed.\n", displayName)
		case "remove":
			err = s.Remove()
			if err != nil {
				fmt.Printf("Failed to remove: %s\n", err)
				return
			}
			fmt.Printf("Service %q removed.\n", displayName)
		case "start":
			err = s.Start()
			if err != nil {
				fmt.Printf("Failed to start: %s\n", err)
				return
			}
			fmt.Printf("Service %q started.\n", displayName)
		case "stop":
			err = s.Stop()
			if err != nil {
				fmt.Printf("Failed to stop: %s\n", err)
				return
			}
			fmt.Printf("Service %q stopped.\n", displayName)
		case "restart":
			err = s.Stop()
			if err != nil {
				fmt.Printf("Failed to stop: %s\n", err)
				return
			}
			fmt.Printf("Service %q stopped.\n", displayName)
			err = s.Start()
			if err != nil {
				fmt.Printf("Failed to start: %s\n", err)
				return
			}
			fmt.Printf("Service %q started.\n", displayName)
		case "check":
			checkConfig()
		case "run":
			doWork()
		}
		return
	}
	err = s.Run(func() error {
			// start
			go doWork()
			return nil
		}, func() error {
			// stop
			stopWork()
			return nil
		})
	if err != nil {
		logit.Error(err.Error())
	}
}

func msgCreator() {
	logit.Info("Notifying all target teams and channels!")

	for _, all := range mapping {
		for _, dst := range all {
			tmp := map[string]interface{}{
				"channel": dst.channel,
				"username": "*SlackToGo",
				"text": "Up and ready!",
				"icon_emoji": ":twisted_rightwards_arrows:" }
			res, _ := json.Marshal(tmp)

			resp, err := http.PostForm(dst.url,
				url.Values{"payload": { string(res)}})
			if(err!=nil) {
				fmt.Printf("Error: %q!\n",err.Error())
				return
			}

			if (resp.StatusCode != 200) {
				fmt.Println(resp)
			}
			resp.Body.Close();
		}
	}
}

func doWork() {
	logit.Info("SlackToGo Server started!")

	// SetupRouting
	for name, team := range config.Teams {
		AddTeam(name,team.ID,team.URL)
	}

	for _, mp := range config.Mappings {
		AddMapping(mp.FromTeam,mp.FromChannel,mp.ToTeam,mp.ToChannel)
	}

	msgCreator()

	http.HandleFunc("/", OnRequest)

	if(config.SSLCrt=="" || config.SSLKey=="") {
		log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d",config.BindAddr,config.Port),
			nil))
	} else {
		log.Fatal(http.ListenAndServeTLS(fmt.Sprintf("%s:%d",config.BindAddr,config.Port),
			config.SSLCrt,
			config.SSLKey,
			nil))
	}

	select {}
}

func stopWork() {
	logit.Info("SlackToGo Server stopping!")
}
