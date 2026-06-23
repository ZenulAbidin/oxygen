package cmd

import "testing"

func TestParseConnPreservesVerifiedTLSSettings(t *testing.T) {
	connCfg, err := parseConn("postgres://alice:secret@db.example.com:6543/oxygen?sslmode=verify-full&application_name=oxygen&pool_max_conns=7")
	if err != nil {
		t.Fatalf("parseConn() error = %v", err)
	}

	if connCfg.User != "alice" {
		t.Fatalf("user = %q, want alice", connCfg.User)
	}
	if connCfg.Password != "secret" {
		t.Fatalf("password = %q, want secret", connCfg.Password)
	}
	if connCfg.Host != "db.example.com" {
		t.Fatalf("host = %q, want db.example.com", connCfg.Host)
	}
	if connCfg.Port != 6543 {
		t.Fatalf("port = %d, want 6543", connCfg.Port)
	}
	if connCfg.Database != "oxygen" {
		t.Fatalf("database = %q, want oxygen", connCfg.Database)
	}
	if connCfg.Config.RuntimeParams["application_name"] != "oxygen" {
		t.Fatalf("application_name = %q, want oxygen", connCfg.Config.RuntimeParams["application_name"])
	}
	if _, ok := connCfg.Config.RuntimeParams["pool_max_conns"]; ok {
		t.Fatal("pool_max_conns should be stripped from migration connection runtime params")
	}
	if connCfg.Config.TLSConfig == nil {
		t.Fatal("TLSConfig is nil, want verified TLS config")
	}
	if connCfg.Config.TLSConfig.InsecureSkipVerify {
		t.Fatal("TLSConfig.InsecureSkipVerify = true, want false for sslmode=verify-full")
	}
	if connCfg.Config.TLSConfig.ServerName != "db.example.com" {
		t.Fatalf("TLSConfig.ServerName = %q, want db.example.com", connCfg.Config.TLSConfig.ServerName)
	}
}

func TestParseConnStripsPoolSettingsFromKeywordDSN(t *testing.T) {
	connCfg, err := parseConn("host=localhost port=5433 dbname=oxygen user=oxygen password=qwerty sslmode=disable pool_max_conns=32")
	if err != nil {
		t.Fatalf("parseConn() error = %v", err)
	}

	if connCfg.Host != "localhost" {
		t.Fatalf("host = %q, want localhost", connCfg.Host)
	}
	if connCfg.Port != 5433 {
		t.Fatalf("port = %d, want 5433", connCfg.Port)
	}
	if _, ok := connCfg.Config.RuntimeParams["pool_max_conns"]; ok {
		t.Fatal("pool_max_conns should be stripped from migration connection runtime params")
	}
	if connCfg.Config.TLSConfig != nil {
		t.Fatal("TLSConfig is not nil, want nil for sslmode=disable")
	}
}
