package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"go-stripe/internal/driver"
	"go-stripe/internal/models"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
)

const version = "1.0.0"
const cssVersion = "1"

var session *scs.SessionManager

type config struct {
	port int
	env  string
	api  string
	db   struct {
		dsn string
	}
	stripe struct {
		secret string
		key    string
	}
	secretKey string
	frontend  string
}

type application struct {
	config        config
	infoLog       *log.Logger
	errorLog      *log.Logger
	templateCache map[string]*template.Template
	version       string
	DB            models.DBModel
	Session       *scs.SessionManager
}

func (app *application) serve() error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.port),
		Handler:           app.routes(),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	app.infoLog.Printf(
		"Starting HTTP server in %s mode on port %d",
		app.config.env,
		app.config.port,
	)

	return srv.ListenAndServe()
}

var STRIPE_SECRET string = "sk_test_51MJC5vCxqx34Mypk5vmaPZ0dagAL3pwJVkGwuNU2TobqaUZYSoVl2Issi3ltgniL5VqAL2HiMxaqR2ynBFsm1DRt00WjhTTrBq"
var STRIPE_KEY string = "pk_test_51MJC5vCxqx34Mypk8kKG4OrBbDTmBFIJt9EfpsOfcIH092mCyV5J3V0dgxwF22p3qViJwkly12V8W8PHCzksC8Nb00m6laJtP6"

func main() {
	gob.Register(TransactionData{})

	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "Server port to listen on")
	flag.StringVar(&cfg.env, "env", "development", "App enviorment {development|production}")
	flag.StringVar(&cfg.db.dsn, "dsn", "root:secret@tcp(localhost:3307)/widgets?parseTime=true&tls=false", "DSN")
	flag.StringVar(&cfg.api, "api", "http://localhost:4001", "URL to api")

	flag.StringVar(&cfg.secretKey, "secret", "bRWmrwNUTqNUuzckjxsFlHZjxHkjrzKP", "secret key")
	flag.StringVar(&cfg.frontend, "frontend", "http://localhost:4000", "url to frontend")
	flag.Parse()

	// cfg.stripe.key = os.Getenv("STRIPE_KEY")
	// cfg.stripe.secret = os.Getenv("STRIPE_SECRET")
	cfg.stripe.key = STRIPE_KEY
	cfg.stripe.secret = STRIPE_SECRET

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	conn, err := driver.OpenDB(cfg.db.dsn)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer conn.Close()

	// set up session
	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Store = mysqlstore.New(conn)

	tc := make(map[string]*template.Template)

	app := &application{
		config:        cfg,
		infoLog:       infoLog,
		errorLog:      errorLog,
		templateCache: tc,
		version:       version,
		DB:            models.DBModel{DB: conn},
		Session:       session,
	}

	err = app.serve()
	if err != nil {
		app.errorLog.Println(err)
		log.Fatal(err)
	}
}
