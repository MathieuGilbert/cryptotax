package main

import (
	"encoding/csv"
	"log"
	"net/http"
	"runtime"

	_ "github.com/jinzhu/gorm/dialects/postgres" // db driver
	"github.com/julienschmidt/httprouter"
	"github.com/mathieugilbert/cryptotax/cmd/config"
	"github.com/mathieugilbert/cryptotax/cmd/parsers"
	"github.com/mathieugilbert/cryptotax/database"
	"github.com/mathieugilbert/cryptotax/models"
)

// Env - holds the handlers
type Env struct {
	db models.Datastore
}

// Parser is an interface for exchange-specific parsing logic
type Parser interface {
	Parse(*csv.Reader) ([]parsers.Trade, error)
}

// Presenter defines template data
type Presenter struct {
	LoggedIn  bool
	CSRFToken string
	Data      interface{}
	Form      interface{}
}

// FormField for persistent form values and messages in responses
type FormField struct {
	Value   string
	Message string
	Success bool
}

// Form to hold persistent posted-back values and a message
type Form struct {
	Fields  map[string]*FormField
	Message string
	Success bool
}

var (
	// SupportedCurrencies is a list of supported base currencies
	SupportedCurrencies = []string{
		"CAD",
		"USD",
	}
	// SupportedExchanges is a list of supported source exchanges
	SupportedExchanges = []string{
		"Coinbase",
		"Kucoin",
		"Cryptotax",
	}
	// TemplateFiles is a list of common template files needed for rendering
	TemplateFiles = []string{
		"web/templates/index.html.tmpl",
		"web/templates/hero.html.tmpl",
		"web/templates/navigation.html.tmpl",
		"web/templates/footer.html.tmpl",
	}
	// Config holds the app configuration
	Config *config.Configuration
)

func init() {
	// Use all CPU cores
	runtime.GOMAXPROCS(runtime.NumCPU())
	// load config values
	loadConfig()
}

// main: connect to database, set up router handlers and start web server.
func main() {
	// initialize database
	db, err := initDB()
	if err != nil {
		log.Fatal(err)
	}
	// run the latest migrations
	database.Migrate(Config.DBString())

	// wrap DB
	env := &Env{db}

	// add router endpoints and handlers
	router := httprouter.New()

	router.GET("/", env.wrapHandler(env.requireSession(env.getRoot)))

	router.GET("/register", env.wrapHandler(env.requireSession(env.notLoggedIn(env.getRegister))))
	router.POST("/register", env.wrapHandler(env.requireSession(env.notLoggedIn(env.postRegister))))
	router.GET("/verify", env.wrapHandler(env.getVerify))

	router.GET("/login", env.wrapHandler(env.requireSession(env.notLoggedIn(env.getLogin))))
	router.POST("/login", env.wrapHandler(env.requireSession(env.notLoggedIn(env.postLogin))))
	router.GET("/logout", env.wrapHandler(env.getLogout))

	router.GET("/files", env.wrapHandler(env.loggedInOnly(env.getFiles)))
	router.POST("/upload", env.wrapHandler(env.loggedInOnly(env.postUploadAsync)))
	router.DELETE("/file", env.wrapHandler(env.loggedInOnly(env.deleteFileAsync)))
	router.GET("/filetrades", env.wrapHandler(env.loggedInOnly(env.getFileTradesAsync)))

	router.GET("/trades", env.wrapHandler(env.loggedInOnly(env.getTrades)))
	router.POST("/trade", env.wrapHandler(env.loggedInOnly(env.postTradeAsync)))
	router.DELETE("/trade", env.wrapHandler(env.loggedInOnly(env.deleteTradeAsync)))

	router.GET("/reports", env.wrapHandler(env.loggedInOnly(env.getReports)))
	router.GET("/rateRequest", env.wrapHandler(env.loggedInOnly(env.getRateRequestAsync)))
	router.POST("/report", env.wrapHandler(env.loggedInOnly(env.postReportAsync)))

	// serve static files
	router.ServeFiles("/web/js/*filepath", http.Dir("web/js"))
	router.ServeFiles("/web/components/*filepath", http.Dir("web/components"))
	router.ServeFiles("/web/css/*filepath", http.Dir("web/css"))
	router.ServeFiles("/web/images/*filepath", http.Dir("web/images"))

	// start the web server
	log.Println("Web server ready.")
	log.Fatal(http.ListenAndServe(":5000", router))
}

// loadConfig reads the config file into the global Config variable
func loadConfig() {
	c, err := config.Read("private/config.json")
	if err != nil {
		log.Fatal(err)
	}
	Config = c
}

// initDB initializes the DB connection and returns a new instance
func initDB() (*models.DB, error) {
	db, err := database.NewDB(Config.DBString())
	if err != nil {
		log.Fatal(err)
	}
	db.LogMode(true)

	return db, nil
}

/*
func (env *Env) downloadTrades(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// requires active session
	s, err := env.session(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// get manual trades
	ts, err := env.db.GetManualTrades(s.ReportID)
	if err != nil {
		http.Error(w, "Unable to fetch trades", http.StatusInternalServerError)
		return
	}

	c := &parsers.Custom{}
	csv, err := c.Generate(ts)
	if err != nil {
		http.Error(w, "Error generating CSV", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=cryptotax-trades.csv")
	w.Header().Set("Content-Type", "text/csv")
	w.Write(csv)
}
*/

func parse(p Parser, r *csv.Reader) ([]parsers.Trade, error) {
	return p.Parse(r)
}
