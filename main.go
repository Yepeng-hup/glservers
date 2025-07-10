package main

import (
	"bufio"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

var (
	mu            sync.Mutex
	gameLogAll            = make([]gameLog, 0)
	searchGameLog         = make([]gameLog, 0)
	errLog                = make([]gameLog, 0)
	cfg           *config = nil
	fs                    = http.FileServer(http.Dir("./static"))
)

type (
	gameLog struct {
		Time string
		Eip  string
		Pro  string
		Log  string
	}

	config struct {
		ServiceIpPort    string   `json:"server_ip_port"`
		LogLimitationNum int      `json:"log_limitation_num"`
		CustomKeywords   []string `json:"custom_keywords"`
	}

	gameLogJson struct {
		DateTime string `json:"dateTime"`
		Eip      string `json:"eip"`
		Title    string `json:"title"`
		Content  string `json:"content"`
	}
)

const (
	tmplFilePath   = "./temp/log.tmpl"
	configJsonFile = "./config/glserver.json"
	apiSecretKey   = "log-server-api-key-abc123-qwertyuiop0912873465"
)

func readConfigFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal("open json file: ", err.Error())
	}
	defer file.Close()
	f := bufio.NewReader(file)
	configObj := json.NewDecoder(f)
	if err = configObj.Decode(&cfg); err != nil {
		return err
	}
	return nil
}

func postDelLog(w http.ResponseWriter, r *http.Request) {
	gameLogAll = nil
	log.Println("WARN ---> The client has performed a complete delete operation.Metadata cleared.")
	w.Header().Set("Cache-Control", "must-revalidate, no-store")
	w.Header().Set("Content-Type", " text/html;charset=UTF-8")
	//url config
	w.Header().Set("Location", "/getlog")
	w.WriteHeader(http.StatusFound)
}

func postReceiveLog(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var p gameLogJson
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			log.Println("ERROR ---> ", "invalid JSON,"+err.Error())
			message := `{"status": 500, "error": "invalid JSON."}`
			w.Write([]byte(message))
			return
		}
		if len(gameLogAll) >= cfg.LogLimitationNum {
			message := `{"status": 500, "error": "log overflow max limit num."}`
			w.Write([]byte(message))
			return
		}

		go func() {
			log.Println("INFO: postReceiveLog api connection successful.")
			mu.Lock()
			defer mu.Unlock()

			g := gameLog{
				Time: p.DateTime,
				Eip:  p.Eip,
				Pro:  p.Title,
				Log:  p.Content,
			}
			gameLogAll = append(gameLogAll, g)
		}()

		message := `{"status": 200}`
		w.Write([]byte(message))
		return
	}
	message := `{"msg": "error illegal request."}`
	w.Write([]byte(message))

}

func getLog(w http.ResponseWriter, r *http.Request) {

	z, err := template.ParseFiles(tmplFilePath)
	if err != nil {
		log.Printf("ERROR ---> tmpl: %v \n", err.Error())
		return
	}

	err = z.Execute(w, map[string]interface{}{
		"logNum":  len(gameLogAll),
		"gameLog": gameLogAll,
	})
	if err != nil {
		log.Printf("ERROR ---> z.Execute rendering error: %v \n", err.Error())
		return
	}
}

func getIndex(w http.ResponseWriter, r *http.Request) {

	z, err := template.ParseFiles(tmplFilePath)
	if err != nil {
		log.Printf("ERROR ---> tmpl: %v \n", err.Error())
		return
	}

	err = z.Execute(w, map[string]interface{}{
		"logNum": 0,
	})
	if err != nil {
		log.Printf("ERROR ---> z.Execute rendering error: %v \n", err.Error())
		return
	}
}

func postSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if len(searchGameLog) != 0 {
			searchGameLog = nil
		}
		err := r.ParseForm()
		if err != nil {
			log.Println("ERROR ---> ", err.Error())
		}

		searchName := r.PostForm.Get("queryStr")
		searchMode := r.PostForm.Get("selectxz")
		if searchMode == "vague" {
			// Vague
			for _, v := range gameLogAll {
				if strings.Contains(v.Pro, searchName) {
					searchGameLog = append(searchGameLog, v)
				}
				continue
			}
		} else {
			// Precise
			for _, l := range gameLogAll {
				regexStr := `\b` + searchName + `\b`
				matched, _ := regexp.MatchString(regexStr, l.Pro)
				if matched {
					searchGameLog = append(searchGameLog, l)
				}
			}
		}

		z, err := template.ParseFiles(tmplFilePath)
		if err != nil {
			log.Printf("ERROR ---> tmpl: %v \n", err.Error())
			return
		}

		err = z.Execute(w, map[string]interface{}{
			"logNum":  len(searchGameLog),
			"gameLog": searchGameLog,
		})
		if err != nil {
			log.Printf("ERROR ---> z.Execute rendering error: %v \n", err.Error())
			return
		}
	}
}

func (g gameLog) String() string {
	return g.Log
}

func getErrLog(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		if len(errLog) != 0 {
			errLog = nil
		}

		for _, l := range gameLogAll {
			for _, v := range cfg.CustomKeywords {
				regexStr := `\b` + v + `\b`
				matched, _ := regexp.MatchString(regexStr, l.String())
				if matched {
					errLog = append(errLog, l)
				}
			}
		}

		z, err := template.ParseFiles(tmplFilePath)
		if err != nil {
			log.Printf("ERROR ---> tmpl: %v \n", err.Error())
			return
		}

		err = z.Execute(w, map[string]interface{}{
			"logNum":  len(errLog),
			"gameLog": errLog,
		})
		if err != nil {
			log.Printf("ERROR ---> z.Execute rendering error: %v \n", err.Error())
			return
		}
	}
}

type ServerStatus struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func serverCheck(w http.ResponseWriter, r *http.Request) {
	//apiKey := r.Header.Get("X-API-KEY")
	//if apiKey != apiSecretKey {
	//	w.WriteHeader(http.StatusUnauthorized)
	//	json.NewEncoder(w).Encode(ServerStatus{
	//		Status:  "Unauthorized",
	//		Message: "Invalid API Key",
	//	})
	//	return
	//}

	w.Header().Set("Content-Type", "application/json")
	status := ServerStatus{
		Status:  "OK",
		Message: "Server is running",
	}

	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	if err := readConfigFile(configJsonFile); err != nil {
		log.Fatalln("ERROR ---> init jsonfile config fail,", err.Error())
	}
	http.HandleFunc("/", getIndex)
	http.HandleFunc("/del", postDelLog)
	http.HandleFunc("/log", postReceiveLog)
	http.HandleFunc("/getlog", getLog)
	http.HandleFunc("/ss", postSearch)
	http.HandleFunc("/log/err", getErrLog)
	http.HandleFunc("/check", serverCheck)
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	log.Printf("INFO ---> http server listen ip and port: %s\n", cfg.ServiceIpPort)
	if err := http.ListenAndServe(cfg.ServiceIpPort, nil); err != nil {
		log.Fatalln("ERROR ---> start glserver fail,", err.Error())
	}
}
