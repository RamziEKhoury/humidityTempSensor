package db

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"os"

	"github.com/go-sql-driver/mysql"
)

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

func init() {
	rootCertPool := x509.NewCertPool()

	pem, err := os.ReadFile("certs/do-mysql-ca.pem")
	if err != nil {
		panic(fmt.Errorf("failed to load CA cert: %w", err))
	}

	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		panic("failed to add CA cert to pool")
	}

	err = mysql.RegisterTLSConfig("do", &tls.Config{
		RootCAs: rootCertPool,
	})
	if err != nil {
		panic(fmt.Errorf("failed to register TLS config: %w", err))
	}
}

func OpenDB(cfg *DBConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true&tls=do",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
