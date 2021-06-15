package bot

import (
	log "bitbucket.org/aisee/minilog"
	"encoding/json"
	"github.com/aiseeq/helpers/pkg/file"
	"github.com/aiseeq/s2l/protocol/client"
	"io/ioutil"
	"os"
	"path/filepath"
)

type StrategyResults map[Strategy]struct {
	Victories int
	Defeats   int
}

type GameData struct {
	Version      string
	History      StrategyResults
	LastResult   string
	LastStrategy Strategy
}

func SaveGameData(gd *GameData, strategy Strategy, result string) {
	if !file.Exists("data") {
		log.Warning("No data dir")
		if err := os.MkdirAll("data", 0755); err != nil {
			log.Error(err)
			return
		}
	}

	sh := gd.History[strategy]
	if result == "Victory" {
		sh.Victories++
	} else {
		sh.Defeats++
	}
	gd.History[strategy] = sh
	gd.LastStrategy = strategy
	gd.LastResult = result
	gd.Version = "2.0"

	data, err := json.MarshalIndent(gd, "", "\t")
	if err != nil {
		log.Error(err)
		return
	}
	err = ioutil.WriteFile("data/"+client.LadderOpponentID+".json", data, 0644)
	if err != nil {
		log.Error(err)
		return
	}
}

func LoadGameData() *GameData {
	var gd GameData
	gd.History = StrategyResults{}
	if !file.Exists("data") || !file.Exists("data/"+client.LadderOpponentID+".json") {
		return &gd
	}
	data, err := ioutil.ReadFile("data/" + client.LadderOpponentID + ".json")
	if err != nil {
		log.Error(err)
		return &gd
	}

	err = json.Unmarshal(data, &gd)
	if err != nil {
		log.Error(err)
		return &gd
	}
	return &gd
}

func DebugGameData() {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	log.Infof("I'm in dir %s running as uid %d", dir, os.Getuid())
	if err := filepath.Walk(".", func(path string, f os.FileInfo, err error) error {
		log.Infof("%s - dir: %v, mode: %v, size: %v, err: %v", path, f.IsDir(), f.Mode(), f.Size(), err)
		return err
	}); err != nil {
		log.Error(err)
	}
}
