package main

import (
	"CourseWork/internal/apichi"
	"CourseWork/internal/apichi/openapichi"
	"CourseWork/internal/config"
	"CourseWork/internal/database/pgxstorage"
	"CourseWork/internal/dbbackend"
	"CourseWork/internal/logging"
	"CourseWork/internal/server"
	"context"
	_ "embed"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"time"

	log "github.com/sirupsen/logrus"
)

//go:embed config/config.env
var cfg string

func main() {
	//Generate random seed
	rand.Seed(time.Now().UnixNano())

	//Creating Context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

	//Logging
	// f, err := logging.LogErrors("error.log")
	// if err != nil {
	// 	log.Fatal("Error opening file: ", err)
	// }
	// defer f.Close()

	logger := logging.NewLogger()

	//Tracing
	var ju openapichi.JaegerUtils
	tracer, closer, err := ju.InitJaeger("localhost:6831", "bitme", logger)
	if err != nil {
		log.Fatal("Error loading tracer: ", err)
	}
	defer closer.Close()

	//Load Config
	cfg, err := config.LoadConfig(cfg, logger)
	if err != nil {
		log.Fatal("Error loading config: ", err)
	}

	//Creating Storage
	//const dsn = "postgres://bituser:bit@localhost:5433/bitmedb?sslmode=disable"
	// udf, err := database.NewPgStorage(os.Getenv("PG_DSN"))
	// if err != nil {
	// 	log.Fatal("Error creating database files: ", err)
	// }

	pgxcfg, err := pgxstorage.NewPgxConfig(os.Getenv("PG_DSN"), 50, 10, 1, 2, logger)
	if err != nil {
		log.Fatal("Error creating database config: ", err)
	}

	udf, err := pgxstorage.NewPgxStorageChached(ctx, pgxcfg)
	if err != nil {
		log.Fatal("Error creating database files: ", err)
	}

	dbbe := dbbackend.NewDataStorage(udf)

	//Init metrics
	m := &openapichi.BitmeMetrics{}
	err = m.Init()
	if err != nil {
		log.Fatal("Error initializing metrics: ", err)
	}

	//Creating router and server
	hs := apichi.NewHandlers(dbbe)
	rt := openapichi.NewOpenApiRouter(hs, m, logger, tracer)
	srv := server.NewServer(":8000", rt, cfg, logger, tracer)

	//Starting
	srv.Start(dbbe)

	fmt.Println("Hello, Bitme!")

	//Shutting down
	<-ctx.Done()

	srv.Stop()
	cancel()
	udf.Close()
	ju.Close()

	fmt.Print("Server shutdown.")
}
