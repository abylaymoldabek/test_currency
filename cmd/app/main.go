package main

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/abylaymoldabek/test_currency/config"
	"github.com/abylaymoldabek/test_currency/internal/entity"
	_ "github.com/denisenkom/go-mssqldb"
	"github.com/gorilla/mux"
)

const (
	server   = "localhost"
	port     = 1433
	user     = "kursUSER"
	password = "kursPswd"
	database = "TEST"
)

func initDB() {
	connectionString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=%s;",
		server, user, password, port, database)

	var err error
	db, err = sql.Open("sqlserver", connectionString)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to SQL Server")
}

var db *sql.DB
var wg sync.WaitGroup

type ErrorResponse struct {
	Message string `json:"message"`
}

func createTable() {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS R_CURRENCY (
			ID SERIAL PRIMARY KEY,
			TITLE VARCHAR(60) NOT NULL,
			CODE VARCHAR(3) NOT NULL,
			VALUE NUMERIC(18,2) NOT NULL,
			A_DATE VARCHAR(60) NOT NULL
		)
	`)
	if err != nil {
		log.Fatal(err)
	}
}

func insertCurrencyAsync(item entity.Currency, date string, ch chan error) {
	defer wg.Done()

	_, err := db.Exec("INSERT INTO R_CURRENCY (TITLE, CODE, VALUE, A_DATE) VALUES ($1, $2, $3, $4)",
		item.Name, item.Code, item.Rate, date)
	if err != nil {
		log.Println("Error saving data to database:", err)
		ch <- err
		return
	}
	ch <- nil
}

func fetchAndSaveDataAsync(fdate string) error {
	url := fmt.Sprintf("https://nationalbank.kz/rss/get_rates.cfm?fdate=%s", fdate)

	resp, err := http.Get(url)
	if err != nil {
		log.Println("Error fetching data:", err)
		return err
	}
	defer resp.Body.Close()

	var result entity.NBKData
	if err := xml.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Println("Error decoding XML:", err)
		return err
	}

	errCh := make(chan error, len(result.Currencies))
	wg.Add(len(result.Currencies))
	for _, item := range result.Currencies {

		go insertCurrencyAsync(item, result.Date, errCh)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

func handleGetRates(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	fdate, ok := params["fdate"]
	if !ok {
		http.Error(w, "Missing 'fdate' parameter", http.StatusBadRequest)
		return
	}

	err := fetchAndSaveDataAsync(fdate)
	if err != nil {
		sendErrorResponse(w, "Error where saving currency", http.StatusBadRequest)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Request accepted. Data will be processed asynchronously.\n"))
}

func getCurrencies(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	date, ok := params["date"]
	if !ok {
		sendErrorResponse(w, "Missing 'date' parameter", http.StatusBadRequest)
		return
	}

	code, codeExists := params["code"]
	var rows *sql.Rows
	var err error
	if codeExists {
		rows, err = db.Query("SELECT title, code, value, a_date FROM R_CURRENCY WHERE a_date = $1 AND code = $2", date, code)
	} else {
		rows, err = db.Query("SELECT title, code, value, a_date FROM R_CURRENCY WHERE a_date = $1", date)
	}

	if err != nil {
		log.Println("Error querying data from database:", err)
		sendErrorResponse(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var currencies []entity.Currency
	for rows.Next() {
		var currency entity.Currency
		if err := rows.Scan(&currency.Name, &currency.Code, &currency.Rate, &currency.Date); err != nil {
			log.Println("Error scanning rows:", err)
			sendErrorResponse(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		currencies = append(currencies, currency)
	}

	if len(currencies) == 0 {
		sendErrorResponse(w, "No data found for the specified parameters", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currencies)
}

func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errorResponse := ErrorResponse{Message: message}
	json.NewEncoder(w).Encode(errorResponse)
}

func main() {

	config, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatal("Error loading configuration:", err)
	}
	fmt.Println(strconv.Itoa(config.ServerConfig.Port))
	initDB()
	defer db.Close()

	go createTable()

	router := mux.NewRouter()
	router.HandleFunc("/get_rates/{fdate}", handleGetRates).Methods("GET")
	router.HandleFunc("/currency/{date}/{code}", getCurrencies).Methods("GET")
	router.HandleFunc("/currency/{date}", getCurrencies).Methods("GET").MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
		rm.MatchErr = nil
		return true
	})

	log.Fatal(http.ListenAndServe(strconv.Itoa(config.ServerConfig.Port), router))
}
