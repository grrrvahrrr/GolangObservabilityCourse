package pgxstorage

import (
	"CourseWork/internal/dbbackend"
	"CourseWork/internal/entities"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/jackc/pgx/v4/stdlib"
)

var _ dbbackend.DataStore = &PgxStorage{}

type PgxData struct {
	FullURL  string
	ShortURL string
	AdminURL string
	Data     int
	IP       string
	IPData   int
	LastUsed time.Time
}

type PgxStorage struct {
	db *pgxpool.Pool
}

func NewPgxConfig(dsn string, maxConns int32, minConns int32, lifeTime int32, idleTimeSec int32) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	cfg.MaxConns = maxConns
	cfg.MinConns = minConns
	cfg.MaxConnLifetime = time.Duration(lifeTime) * time.Second
	cfg.MaxConnIdleTime = time.Duration(idleTimeSec) * time.Second

	return cfg, nil
}

func NewPgxStorage(ctx context.Context, cfg *pgxpool.Config) (*PgxStorage, error) {
	dbpool, err := pgxpool.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	us := &PgxStorage{
		db: dbpool,
	}

	return us, nil
}

func (pgxs *PgxStorage) WriteURL(ctx context.Context, url entities.UrlData) (*entities.UrlData, error) {
	dbd := &PgxData{
		FullURL:  url.FullURL,
		ShortURL: url.ShortURL,
		AdminURL: url.AdminURL,
		Data:     0,
	}

	_, err := pgxs.db.Query(ctx, `INSERT INTO bitme.urlbase (short_url, full_url) VALUES ($1, $2)`,
		dbd.ShortURL, dbd.FullURL)
	if err != nil {
		return nil, err
	}

	_, err = pgxs.db.Query(ctx, `INSERT INTO bitme.adminurl (admin_url ,short_url) VALUES ($1, $2)`,
		dbd.AdminURL, dbd.ShortURL)
	if err != nil {
		return nil, err
	}

	_, err = pgxs.db.Query(ctx, `INSERT INTO bitme.urldata (short_url, last_used, total_num_of_uses) VALUES ($1, $2, $3)`,
		dbd.ShortURL, time.Now(), dbd.Data)
	if err != nil {
		return nil, err
	}

	data := strconv.Itoa(dbd.Data)

	return &entities.UrlData{
		FullURL:  dbd.FullURL,
		ShortURL: dbd.ShortURL,
		AdminURL: dbd.AdminURL,
		Data:     data,
	}, nil
}

func (pgxs *PgxStorage) WriteData(ctx context.Context, url entities.UrlData) (*entities.UrlData, error) {
	data, err := strconv.Atoi(url.Data)
	if err != nil {
		return nil, err
	}

	ipData, err := strconv.Atoi(url.IPData)
	if err != nil {
		return nil, err
	}

	dbd := &PgxData{
		FullURL:  url.FullURL,
		ShortURL: url.ShortURL,
		AdminURL: url.AdminURL,
		Data:     data,
		IP:       url.IP,
		IPData:   ipData,
	}

	rows, err := pgxs.db.Query(ctx, `UPDATE bitme.urlusedata SET last_used = $3, ip_num_of_uses = $4 WHERE ip = $1 AND short_url = $2`,
		dbd.IP, dbd.ShortURL, time.Now(), dbd.IPData)
	if err != nil {
		return nil, err
	}

	rows.Close()

	if rows.CommandTag().RowsAffected() == 0 {
		_, err = pgxs.db.Query(ctx, `INSERT INTO bitme.urlusedata (ip, short_url, last_used, ip_num_of_uses) VALUES($1, $2, $3, $4)`, dbd.IP, dbd.ShortURL, time.Now(), dbd.IPData)
		if err != nil {
			return nil, err
		}
	}

	_, err = pgxs.db.Query(ctx, `UPDATE bitme.urldata SET last_used = $2, total_num_of_uses = $3 WHERE short_url = $1`,
		dbd.ShortURL, time.Now(), dbd.Data)
	if err != nil {
		return nil, err
	}

	return &url, nil
}

func (pgxs *PgxStorage) ReadURL(ctx context.Context, url entities.UrlData) (*entities.UrlData, error) {
	dbd := &PgxData{}

	if url.AdminURL != "" {
		dbd.AdminURL = url.AdminURL
		err := pgxs.db.QueryRow(ctx, `SELECT short_url FROM bitme.adminurl WHERE admin_url = $1`, dbd.AdminURL).Scan(&dbd.ShortURL)
		if err != nil && err != pgx.ErrNoRows {
			return nil, err
		}
	} else if url.ShortURL != "" {
		dbd.ShortURL = url.ShortURL
	} else {
		return nil, fmt.Errorf("recieved empty struct, no key to find")
	}

	err := pgxs.db.QueryRow(ctx, `SELECT full_url FROM bitme.urlbase WHERE short_url = $1`, dbd.ShortURL).Scan(&dbd.FullURL)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	err = pgxs.db.QueryRow(ctx, `SELECT total_num_of_uses FROM bitme.urldata WHERE short_url = $1`, dbd.ShortURL).Scan(&dbd.Data)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	if url.IP != "" {
		dbd.IP = url.IP
		err = pgxs.db.QueryRow(ctx, `SELECT ip_num_of_uses FROM bitme.urlusedata WHERE ip = $1 AND short_url = $2`, dbd.IP, dbd.ShortURL).Scan(&dbd.IPData)
		if err != nil && err != pgx.ErrNoRows {
			return nil, err
		}
	}

	var ipData string
	if dbd.IPData == 0 {
		ipData = "0"
	} else {
		ipData = strconv.Itoa(dbd.IPData)
	}

	data := strconv.Itoa(dbd.Data)

	return &entities.UrlData{
		FullURL:  dbd.FullURL,
		ShortURL: dbd.ShortURL,
		AdminURL: dbd.AdminURL,
		Data:     data,
		IP:       dbd.IP,
		IPData:   ipData,
	}, nil
}

func (pgxs *PgxStorage) GetIPData(ctx context.Context, url entities.UrlData) (string, error) {
	var ipdata string

	dbd := &PgxData{
		ShortURL: url.ShortURL,
	}

	rows, err := pgxs.db.Query(ctx, `SELECT ip, ip_num_of_uses FROM bitme.urlusedata WHERE short_url = $1`, dbd.ShortURL)
	if err != nil && err != pgx.ErrNoRows {
		return "", err
	}

	for rows.Next() {
		if err := rows.Scan(
			&dbd.IP,
			&dbd.IPData,
		); err != nil && err != pgx.ErrNoRows {
			return "", err
		}

		ipData := strconv.Itoa(dbd.IPData)

		ipdata += "IP: " + dbd.IP + " # Redirects: " + ipData + "\n"

	}

	rows.Close()

	return ipdata, nil
}

func (pgx *PgxStorage) Close() {
	pgx.db.Close()
}
