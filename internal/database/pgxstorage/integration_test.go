//go:build integration_tests

package pgxstorage

import (
	"CourseWork/internal/entities"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"testing"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var dsn string

type setupResult struct {
	Pool              *dockertest.Pool
	PostgresContainer *dockertest.Resource
}

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	setupResult, err := setup()
	if err != nil {
		log.Panicln("setup err: ", err)
		return -1
	}
	defer teardown(setupResult)
	return m.Run()
}

func setup() (r *setupResult, err error) {
	testFileDir, err := getTestFileDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get the script dir: %w", err)
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("failed to create a new docketest pool: %w", err)
	}
	pool.MaxWait = time.Second * 5

	postgresContainer, err := runPostgresContainer(pool, testFileDir)
	if err != nil {
		return nil, fmt.Errorf("failed to run the Postgres container: %w", err)
	}
	defer func() {
		if err != nil {
			if err := pool.Purge(postgresContainer); err != nil {
				log.Println("failed to purge the postgres container: %w", err)
			}
		}
	}()

	return &setupResult{
		Pool:              pool,
		PostgresContainer: postgresContainer,
	}, nil
}

func getTestFileDir() (string, error) {
	_, fileName, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("failed to get the caller info")
	}
	fileDir := filepath.Dir(fileName)
	dir, err := filepath.Abs(fileDir)
	if err != nil {
		return "", fmt.Errorf("failed to get the absolute path to the directory %s: %w", dir, err)
	}
	log.Println(fileDir)
	return fileDir, nil
}

func runPostgresContainer(pool *dockertest.Pool, testFileDir string) (*dockertest.Resource, error) {
	postgresContainer, err := pool.RunWithOptions(
		&dockertest.RunOptions{
			Repository: "postgres",
			Tag:        "14.0",
			Env: []string{
				"POSTGRES_PASSWORD=123",
			},
		},
		func(config *docker.HostConfig) {
			config.AutoRemove = false
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
			config.Mounts = []docker.HostMount{
				{
					Target: "/docker-entrypoint-initdb.d",
					Source: filepath.Join(testFileDir, "testinit"),
					Type:   "bind",
				},
			}
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start the postgres docker container: %w", err)
	}

	postgresContainer.Expire(120)
	port := postgresContainer.GetPort("5432/tcp")
	dsn = fmt.Sprintf("postgres://bitusertest:123@localhost:%s/bitmedbtest?sslmode=disable", port)

	// Wait for the DB to start
	if err := pool.Retry(func() error {
		db, err := getDBConnector()
		if err != nil {
			return fmt.Errorf("failed to get a DB connector: %w", err)
		}
		return db.Ping(context.Background())
	}); err != nil {
		pool.Purge(postgresContainer)
		return nil, fmt.Errorf("failed to ping the created DB: %w", err)
	}
	return postgresContainer, nil
}

func getDBConnector() (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create the PGX pool config from connection string: %w", err)
	}
	cfg.ConnConfig.ConnectTimeout = time.Second * 1
	db, err := pgxpool.ConnectConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the postgres DB using a PGX connection pool: %w", err)
	}
	return db, nil
}

func teardown(r *setupResult) {
	if err := r.Pool.Purge(r.PostgresContainer); err != nil {
		log.Printf("failed to purge the Postgres container: %v", err)
	}
}

func TestWriteURL(t *testing.T) {
	tdu := entities.UrlData{
		FullURL:  "testFullUrl",
		ShortURL: "testSU",
		AdminURL: "testAU",
	}

	conn, err := getDBConnector()
	if err != nil {
		t.Fatalf("failed to get a connector to the DB: %v", err)
	}

	ctx := context.Background()

	pgxs := PgxStorage{
		db: conn,
	}

	_, err = pgxs.WriteURL(ctx, tdu)
	if err != nil {
		t.Fatalf("failed to WriteURL to the DB: %v", err)
	}
}

func TestWriteData(t *testing.T) {
	tdu := entities.UrlData{
		FullURL:  "testFullUrl",
		ShortURL: "testSU",
		AdminURL: "testAU",
		Data:     "1",
		IP:       "0.0.0.0",
		IPData:   "1",
	}

	conn, err := getDBConnector()
	if err != nil {
		t.Fatalf("failed to get a connector to the DB: %v", err)
	}

	ctx := context.Background()

	pgxs := PgxStorage{
		db: conn,
	}

	_, err = pgxs.WriteData(ctx, tdu)
	if err != nil {
		t.Fatalf("failed to WriteData to the DB: %v", err)
	}
}

func TestReadURL(t *testing.T) {
	tdu := entities.UrlData{
		AdminURL: "testAU",
		IP:       "0.0.0.0",
	}

	conn, err := getDBConnector()
	if err != nil {
		t.Fatalf("failed to get a connector to the DB: %v", err)
	}

	ctx := context.Background()

	pgxs := PgxStorage{
		db: conn,
	}

	newtdu, err := pgxs.ReadURL(ctx, tdu)
	if err != nil {
		t.Fatalf("failed to WriteData to the DB: %v", err)
	}

	fmt.Println("result: ", newtdu)

	if newtdu.FullURL != "testFullUrl" {
		t.Errorf("Failed to Read Full Url, got %s", newtdu.FullURL)
	}

	if newtdu.ShortURL != "testSU" {
		t.Errorf("Failed to Read Short Url, got %s", newtdu.ShortURL)
	}

	if newtdu.AdminURL != "testAU" {
		t.Errorf("Failed to Read Admin Url, got %s", newtdu.AdminURL)
	}

	if newtdu.Data != "1" {
		t.Errorf("Failed to Read Data, got %s", newtdu.Data)
	}

	if newtdu.IP != "0.0.0.0" {
		t.Errorf("Failed to Read IP, got %s", newtdu.IP)
	}

	if newtdu.IPData != "1" {
		t.Errorf("Failed to Read IPData, got %s", newtdu.IPData)
	}

}

func TestGetIPData(t *testing.T) {
	tdu := entities.UrlData{
		ShortURL: "testSU",
	}

	conn, err := getDBConnector()
	if err != nil {
		t.Fatalf("failed to get a connector to the DB: %v", err)
	}

	ctx := context.Background()

	pgxs := PgxStorage{
		db: conn,
	}

	ipdata, err := pgxs.GetIPData(ctx, tdu)
	if err != nil {
		t.Fatalf("failed to WriteData to the DB: %v", err)
	}

	if ipdata != "IP: 0.0.0.0 # Redirects: 1\n" {
		t.Errorf("Failed to Read IPData, got %s", ipdata)
	}
}
