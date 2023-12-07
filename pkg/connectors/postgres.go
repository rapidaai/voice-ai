package connectors

import (
	"context"
	"fmt"
	"time"

	commons "github.com/lexatic/web-backend/pkg/commons"
	configs "github.com/lexatic/web-backend/pkg/configs"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresConnector interface {
	Connector
	Query(ctx context.Context, qry string, dest interface{}) error
	DB(ctx context.Context) *gorm.DB
}

type postgresConnector struct {
	cfg    *configs.PostgresConfig
	db     *gorm.DB
	logger commons.Logger
}

func NewPostgresConnector(config *configs.PostgresConfig, logger commons.Logger) PostgresConnector {
	return &postgresConnector{cfg: config, logger: logger}
}

func (psql *postgresConnector) DB(ctx context.Context) *gorm.DB {
	return psql.db.WithContext(ctx)
}

// generating connection string from configuration
func (psql *postgresConnector) connectionString() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s", psql.cfg.Host, psql.cfg.Auth.User, psql.cfg.Auth.Password, psql.cfg.DBName, psql.cfg.Port, psql.cfg.SslMode)
}

func (psql *postgresConnector) Connect(ctx context.Context) error {
	psql.logger.Debugf("Creating postgres client for postgres  %s", psql.connectionString())
	db, err := gorm.Open(postgres.Open(psql.connectionString()), &gorm.Config{})
	if err != nil {
		psql.logger.Errorf("Failed to open postgres connection %s.", err)
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		psql.logger.Errorf("Failed to create postgres client connection pool %s.", err)
		return err
	}
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(psql.cfg.MaxIdealConnection)
	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(psql.cfg.MaxOpenConnection)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)

	// set db to instance
	psql.db = db

	return nil

}

func (psql *postgresConnector) Name() string {
	return fmt.Sprintf("PSQL psql://%s:%d", psql.cfg.Host, psql.cfg.Port)
}
func (psql *postgresConnector) IsConnected(ctx context.Context) bool {

	psql.logger.Debugf("Calling ping for postgres.")
	db, err := psql.db.DB()
	if err != nil {
		psql.logger.Errorf("Failed to get postgres client %s.", err)
		return false
	}
	err = db.PingContext(ctx)
	if err == nil {
		psql.logger.Errorf("Failed to ping postgres client %s.", err)
		return false
	}
	return false
}
func (psql *postgresConnector) Disconnect(ctx context.Context) error {
	psql.logger.Debug("Disconnecting with postgres client.")
	db, err := psql.db.DB()
	if err != nil {
		psql.logger.Errorf("Disconnecting with postgres client %s.", err)
		return err
	}
	err = db.Close()
	if err != nil {
		psql.logger.Debug("Disconnecting with postgres client %s.", err)
		return err
	}
	psql.db = nil
	return nil
}

func (psql *postgresConnector) Query(ctx context.Context, qry string, dest interface{}) error {
	tx := psql.db.Raw(qry).Scan(dest)
	if tx.Error != nil {
		return tx.Error
	}
	return nil
}
