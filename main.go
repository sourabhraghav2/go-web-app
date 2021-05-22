package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/schema"
	env "github.com/joho/godotenv"
)

const envFile = ".env"
const dataFile = "data/forms.json"

var loadEnv = env.Load

type formInput struct {
	FirstName   string `schema:"first_name"`
	LastName    string `schema:"last_name"`
	Email       string `schema:"email"`
	PhoneNumber string `schema:"phone_number"`
}
type State struct {
	DetailList  []formInput
	PlaceHolder formInput
	PageType    string
	FirstName   string
	LastName    string
	Email       string
	PhoneNumber string
}

func (f formInput) validate() error {
	if f.FirstName == "" || f.LastName == "" || f.Email == "" || f.PhoneNumber == "" {
		return errors.New("invalid input")
	}
	return nil
}

func (f formInput) save() error {
	file, err := ioutil.ReadFile(dataFile)
	if err != nil {
		return err
	}
	var olrForm []formInput
	err = json.Unmarshal(file, &olrForm)
	if err != nil {
		return err
	}
	forms := []formInput{f}
	toSave, err := json.Marshal(append(forms, olrForm...))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(dataFile, toSave, os.ModeAppend)
	return err
}

func handleFunc(resp http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case http.MethodPost:
		log.Println("endPoint : POST")
		postDataToDB(resp, req)

	case http.MethodGet:
		log.Println("endPoint : GET")
		fetchDetailsList(resp, req)

	default:
		log.Println("error no 404")
		resp.WriteHeader(http.StatusNotFound)
		fmt.Fprint(resp, "not found")
	}
}
func handleError(err error, resp http.ResponseWriter, req *http.Request) {
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(resp, err.Error())
		return
	}
}
func postDataToDB(resp http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	handleError(err, resp, req)
	err = req.ParseForm()
	handleError(err, resp, req)
	f := new(formInput)
	decoder := schema.NewDecoder()
	err = decoder.Decode(f, req.Form)
	handleError(err, resp, req)
	err = f.validate()
	handleError(err, resp, req)
	err = f.save()
	handleError(err, resp, req)
	tpl.ExecuteTemplate(resp, "index", State{
		PageType: "VIEW",
	})

}
func fetchDetailsList(w http.ResponseWriter, r *http.Request) {
	data, err := ioutil.ReadFile(dataFile)
	log.Println("Fetching data")
	if err != nil {
		fmt.Print(err)
	}
	var formList []formInput

	json.Unmarshal(data, &formList)
	tpl.ExecuteTemplate(w, "index", State{
		DetailList: formList,
		PageType:   "VIEW",
	})

}

var tpl = template.Must(template.ParseFiles("public/index.html", "public/component/list.html", "public/component/form.html"))

func indexHandler(w http.ResponseWriter, r *http.Request) {

	tpl.ExecuteTemplate(w, "index", State{
		FirstName:   "Sourabh",
		LastName:    "Raghav",
		Email:       "sourabhraghav2@gmail.com",
		PhoneNumber: "+66657126626",
		PageType:    "EDIT",
	})
}
func run() (s *http.Server) {
	err := loadEnv(envFile)
	if err != nil {
		log.Fatal(err)
	}
	port, exist := os.LookupEnv("PORT")
	if !exist {
		log.Fatal("no port specified")
	}
	port = fmt.Sprintf(":%s", port)

	log.Println("Initializing endPoints")
	mux := http.NewServeMux()
	mux.HandleFunc("/rest", handleFunc)
	mux.HandleFunc("/", indexHandler)

	s = &http.Server{
		Addr:           port,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	return
}

func main() {
	log.Println("App  init")
	s := run()
	quit := make(chan os.Signal)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown")
	}
	log.Println("Server exiting")
}
