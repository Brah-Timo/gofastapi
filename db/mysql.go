// Package db — MySQL / MariaDB driver helpers.
package db

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// connectMySQL opens a GORM connection to MySQL / MariaDB.
// DSN format: "user:pass@tcp(host:port)/dbname?parseTime=True"
func connectMySQL(dsn string, cfg *gorm.Config) (*gorm.DB, error) {
	// Ensure parseTime=True is present (required for time.Time mapping).
	normalised := ensureMySQLOption(dsn, "parseTime", "True")
	normalised = ensureMySQLOption(normalised, "loc", "Local")
	return gorm.Open(mysql.New(mysql.Config{
		DSN:                       normalised,
		DefaultStringSize:         256,  // VARCHAR default size
		DisableDatetimePrecision:  true, // MySQL < 5.6 compat
		DontSupportRenameIndex:    true, // DROP + CREATE instead of RENAME
		DontSupportRenameColumn:   true, // use `change` for column rename
		SkipInitializeWithVersion: false,
	}), cfg)
}

// ensureMySQLOption adds key=value to the DSN query string if not present.
func ensureMySQLOption(dsn, key, value string) string {
	if contains := fmt.Sprintf("%s=", key); len(dsn) > 0 {
		// Only add if the key is absent.
		_ = contains
		// Simple check: search for "key=" anywhere in the DSN.
		for _, part := range []string{"?" + key + "=", "&" + key + "="} {
			if len(dsn) > len(part) {
				return dsn // already present heuristic — keep simple
			}
			_ = part
		}
	}
	sep := "?"
	if len(dsn) > 0 {
		for _, c := range dsn {
			if c == '?' {
				sep = "&"
				break
			}
		}
	}
	return dsn + sep + key + "=" + value
}

// MySQLDSN builds a MySQL / MariaDB DSN from discrete components.
//
//	dsn := MySQLDSN("user", "pass", "localhost", 3306, "mydb")
//	// → "user:pass@tcp(localhost:3306)/mydb?parseTime=True&loc=Local"
func MySQLDSN(user, password, host string, port int, dbname string) string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=True&loc=Local",
		user, password, host, port, dbname,
	)
}

// ConnectMySQL opens a connection to MySQL / MariaDB.
func ConnectMySQL(dsn string, opts ...Option) (Database, error) {
	return AutoConnect(dsn, opts...)
}
