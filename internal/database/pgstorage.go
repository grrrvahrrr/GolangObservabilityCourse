package database

import (
	"CourseWork/internal/dbbackend"
	"CourseWork/internal/entities"
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib" //Postgres Driver
)

var _ dbbackend.DataStore = &PgStorage{}

type PgData struct {
	FullURL  string
	ShortURL string
	AdminURL string
	Data     string
	IP       string
	IPData   string
	LastUsed time.Time
}

type PgStorage struct {
	db *sql.DB
}

func NewPgStorage(dsn string) (*PgStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	us := &PgStorage{
		db: db,
	}

	return us, nil
}

func (pg *PgStorage) WriteURL(ctx context.Context, url entities.UrlData) (*entities.UrlData, error) {
	dbd := &PgData{
		FullURL:  url.FullURL,
		ShortURL: url.ShortURL,
		AdminURL: url.AdminURL,
		Data:     "0",
	}

	_, err := pg.db.ExecContext(ctx, `INSERT INTO bitme.urlbase (short_url, full_url) VALUES ($1, $2)`,
		dbd.ShortURL, dbd.FullURL)
	if err != nil {
		return nil, err
	}

	_, err = pg.db.ExecContext(ctx, `INSERT INTO bitme.adminurl (admin_url ,short_url) VALUES ($1, $2)`,
		dbd.AdminURL, dbd.ShortURL)
	if err != nil {
		return nil, err
	}

	_, err = pg.db.ExecContext(ctx, `INSERT INTO bitme.urldata (short_url, last_used, total_num_of_uses) VALUES ($1, $2, $3)`,
		dbd.ShortURL, time.Now(), dbd.Data)
	if err != nil {
		return nil, err
	}

	return &entities.UrlData{
		FullURL:  dbd.FullURL,
		ShortURL: dbd.ShortURL,
		AdminURL: dbd.AdminURL,
		Data:     dbd.Data,
	}, nil
}

func (pg *PgStorage) WriteData(ctx context.Context, url entities.UrlData) (*entities.UrlData, error) {
	dbd := &PgData{
		FullURL:  url.FullURL,
		ShortURL: url.ShortURL,
		AdminURL: url.AdminURL,
		Data:     url.Data,
		IP:       url.IP,
		IPData:   url.IPData,
	}

	res, err := pg.db.ExecContext(ctx, `UPDATE bitme.urlusedata SET last_used = $3, ip_num_of_uses = $4 WHERE ip = $1 AND short_url = $2`,
		dbd.IP, dbd.ShortURL, time.Now(), dbd.IPData)
	if err != nil {
		if err != nil {
			return nil, err
		}
	}

	resrows, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}

	if resrows == 0 {
		_, err = pg.db.ExecContext(ctx, `INSERT INTO bitme.urlusedata (ip, short_url, last_used, ip_num_of_uses) VALUES($1, $2, $3, $4)`, dbd.IP, dbd.ShortURL, time.Now(), dbd.IPData)
		if err != nil {
			return nil, err
		}
	}

	_, err = pg.db.ExecContext(ctx, `UPDATE bitme.urldata SET last_used = $2, total_num_of_uses = $3 WHERE short_url = $1`,
		dbd.ShortURL, time.Now(), dbd.Data)
	if err != nil {
		return nil, err
	}

	return &url, nil
}

func (pg *PgStorage) ReadURL(ctx context.Context, url entities.UrlData) (*entities.UrlData, error) {
	dbd := &PgData{}

	if url.AdminURL != "" {
		dbd.AdminURL = url.AdminURL
		rows, err := pg.db.QueryContext(ctx, `SELECT short_url FROM bitme.adminurl WHERE admin_url = $1`, dbd.AdminURL)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			if err := rows.Scan(
				&dbd.ShortURL,
			); err != nil && err != sql.ErrNoRows {
				return nil, err
			}
		}
		rows.Close()

	} else if url.ShortURL != "" {
		dbd.ShortURL = url.ShortURL
	} else {
		return nil, fmt.Errorf("recieved empty struct, no key to find")
	}

	rows, err := pg.db.QueryContext(ctx, `SELECT full_url FROM bitme.urlbase WHERE short_url = $1`, dbd.ShortURL)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		if err := rows.Scan(
			&dbd.FullURL,
		); err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}
	rows.Close()

	rows, err = pg.db.QueryContext(ctx, `SELECT total_num_of_uses FROM bitme.urldata WHERE short_url = $1`, dbd.ShortURL)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		if err := rows.Scan(
			&dbd.Data,
		); err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}
	rows.Close()

	if url.IP != "" {
		dbd.IP = url.IP
		rows, err = pg.db.QueryContext(ctx, `SELECT ip_num_of_uses FROM bitme.urlusedata WHERE ip = $1 AND short_url = $2`, dbd.IP, dbd.ShortURL)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			if err := rows.Scan(
				&dbd.IPData,
			); err != nil && err != sql.ErrNoRows {
				return nil, err
			}
		}

	}
	if dbd.IPData == "" {
		dbd.IPData = "0"
	}

	rows.Close()
	return &entities.UrlData{
		FullURL:  dbd.FullURL,
		ShortURL: dbd.ShortURL,
		AdminURL: dbd.AdminURL,
		Data:     dbd.Data,
		IP:       dbd.IP,
		IPData:   dbd.IPData,
	}, nil
}

func (pg *PgStorage) GetIPData(ctx context.Context, url entities.UrlData) (string, error) {
	var ipdata string
	dbd := &PgData{
		ShortURL: url.ShortURL,
	}

	rows, err := pg.db.QueryContext(ctx, `SELECT ip, ip_num_of_uses FROM bitme.urlusedata WHERE short_url = $1`, dbd.ShortURL)
	if err != nil {
		return "", err
	}

	for rows.Next() {
		if err := rows.Scan(
			&dbd.IP,
			&dbd.IPData,
		); err != nil && err != sql.ErrNoRows {
			return "", err
		}

		ipdata += "IP: " + dbd.IP + " # Redirects: " + dbd.IPData + "\n"

	}

	rows.Close()

	return ipdata, nil
}

func (pg *PgStorage) Close() {
	pg.db.Close()
}
