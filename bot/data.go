package bot

import (
	log "bitbucket.org/aisee/minilog"
	"encoding/json"
	"github.com/aiseeq/helpers/pkg/file"
	"github.com/aiseeq/s2l/protocol/client"
	"io/ioutil"
	"os"
)

type GameData struct {
	Version string
	Result  string
	Cheeze  bool
}

func SaveGameData(result string, cheeze bool) {
	if !file.Exists("data") {
		log.Warning("No data dir")
		if err := os.MkdirAll("data", 0755); err != nil {
			log.Error(err)
			return
		}
	}
	data, err := json.MarshalIndent(GameData{
		Version: "1.0",
		Result:  result,
		Cheeze:  cheeze,
	}, "", "\t")
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

func LoadGameData(cheeze bool) bool { // Returns if we want to cheeze. Default result in param
	if !file.Exists("data") || !file.Exists("data/"+client.LadderOpponentID+".json") {
		return cheeze
	}
	data, err := ioutil.ReadFile("data/"+client.LadderOpponentID+".json")
	if err != nil {
		log.Error(err)
		return cheeze
	}

	var gd GameData
	err = json.Unmarshal(data, &gd)
	if err != nil {
		log.Error(err)
		return cheeze
	}
	if gd.Result != "Victory" {
		return !gd.Cheeze // Switch tactics
	}
	return gd.Cheeze
}
