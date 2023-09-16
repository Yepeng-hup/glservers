package main

import (
	"bufio"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	timeSlice = make([] string, 0)
	proSlice = make([] string, 0)
	logSlice = make([] string, 0)
	gameLogAll = make([] gameLog, 0)
	searchGameLog = make([] gameLog, 0)
	errLog = make([]gameLog, 0)
	cfg *config = nil
)

type (
	gameLog struct {
		Time string
		Pro string
		Log string
	}
	config struct {
		ServiceIpPort string `json:"server_ip_port"`
	}
)

const (
	tmplFilePath = "./temp/log.tmpl"
	configJsonFile = "./config/glserver.json"
	logLimitationNum = 1200
)

func readConfigFile(path string) (*config, error) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal("open json file: ", err.Error())
	}
	defer file.Close()
	f := bufio.NewReader(file)
	configObj := json.NewDecoder(f)
	if err = configObj.Decode(&cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}


func postDelLog(w http.ResponseWriter, r *http.Request) {
	timeSlice = nil
	proSlice = nil
	logSlice = nil
	gameLogAll = nil
	log.Println("WARN ---> The client has performed a complete delete operation.Metadata cleared.")
	w.Header().Set("Cache-Control", "must-revalidate, no-store")
	w.Header().Set("Content-Type", " text/html;charset=UTF-8")
	//url config
	w.Header().Set("Location", "/getlog")
	w.WriteHeader(http.StatusFound)
}


func postReceiveLog(w http.ResponseWriter, r *http.Request){
	if r.Method == "POST" {
		err := r.ParseForm()
		if err != nil {
			log.Println("ERROR ---> ",err.Error())
			return
		}
		if len(gameLogAll) >= logLimitationNum {
			message := `{"status": 500, "error": "log overflow max 1200."}`
			w.Write([]byte(message))
			return
		}
		timeSlice = append(timeSlice, r.PostForm.Get("dateTime"))
		proSlice = append(proSlice, r.PostForm.Get("pro"))
		logSlice = append(logSlice,  r.PostForm.Get("gamelog"))
		index := len(proSlice)-1
		g := gameLog{
			Time: timeSlice[index],
			Pro: proSlice[index],
			Log: logSlice[index],
		}
		gameLogAll = append(gameLogAll, g)
		message := `{"status": 200}`
		w.Write([]byte(message))
		return
	}
	message := `{"msg": "error Illegal request."}`
	w.Write([]byte(message))
	return
}


func getLog(w http.ResponseWriter, r *http.Request) {

	z,err := template.ParseFiles(tmplFilePath)
	if err != nil {
		log.Printf("ERROR ---> tmpl: %v \n",err.Error() )
		return
	}

	err = z.Execute(w, map[string]interface{}{
		"logNum": len(gameLogAll),
		"gameLog": gameLogAll,
	})
	if err != nil {
		log.Printf("ERROR ---> z.Execute use, lr: %v \n", err.Error())
		return
	}
	return
}


func getIndex(w http.ResponseWriter, r *http.Request) {

	z,err := template.ParseFiles(tmplFilePath)
	if err != nil {
		log.Printf("ERROR ---> tmpl: %v \n",err.Error() )
		return
	}

	err = z.Execute(w, map[string]interface{}{
		"logNum": 0,
	})
	if err != nil {
		log.Printf("ERROR ---> z.Execute use, lr: %v \n", err.Error())
		return
	}
	return
}


func postSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if len(searchGameLog)!=0 {
			searchGameLog = nil
		}
		err := r.ParseForm()
		if err != nil {
			log.Println("ERROR ---> ",err.Error())
		}
		searchName := r.PostForm.Get("queryStr")
		for _,v := range gameLogAll {
			if strings.Contains(v.Pro, searchName) {
				searchGameLog = append(searchGameLog, v)
			}
			continue
		}
		z,err := template.ParseFiles(tmplFilePath)
		if err != nil {
			log.Printf("ERROR ---> tmpl: %v \n",err.Error() )
			return
		}

		err = z.Execute(w, map[string]interface{}{
			"logNum": len(searchGameLog),
			"gameLog": searchGameLog,
		})
		if err != nil {
			log.Printf("ERROR ---> z.Execute use, lr: %v \n", err.Error())
			return
		}
	}
}

func getErrLog(w http.ResponseWriter, r *http.Request) {

	if r.Method == "GET"{
		if len(errLog) != 0 {
			errLog = nil
		}

		for _,v := range gameLogAll {
			if strings.Contains(v.Log, "error") || strings.Contains(v.Log, "fail") {
				errLog = append(errLog, v)
			}
		}

		z,err := template.ParseFiles(tmplFilePath)
		if err != nil {
			log.Printf("ERROR ---> tmpl: %v \n",err.Error() )
			return
		}

		err = z.Execute(w, map[string]interface{}{
			"logNum": len(errLog),
			"gameLog": errLog,
		})
		if err != nil {
			log.Printf("ERROR ---> z.Execute use, lr: %v \n", err.Error())
			return
		}
	}
}


func main() {
	j, err := readConfigFile(configJsonFile)
	if err != nil {
		log.Fatalln(err.Error())
	}
	http.HandleFunc("/", getIndex)
	http.HandleFunc("/del", postDelLog)
	http.HandleFunc("/log", postReceiveLog)
	http.HandleFunc("/getlog", getLog)
	http.HandleFunc("/ss", postSearch)
	http.HandleFunc("/log/err", getErrLog)
	log.Printf("INFO ---> http server listen ip and port: %s\n", j.ServiceIpPort)
	e := http.ListenAndServe(j.ServiceIpPort, nil)
	if e != nil {
		log.Fatal("ERROR ---> start glserver fail,", e.Error())
	}
}
