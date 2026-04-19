package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestDatabaseConfig_Defaults(t *testing.T) {
	t.Setenv(testDBDataSourceEnv, "")

	cfg, err := testDatabaseConfig("oxygen_test_tmp")
	require.NoError(t, err)

	assert.Equal(t, dbHost, cfg.ConnConfig.Host)
	assert.Equal(t, dbUser, cfg.ConnConfig.User)
	assert.Equal(t, "oxygen_test_tmp", cfg.ConnConfig.Database)
}

func TestTestDatabaseConfig_UsesEnvSource(t *testing.T) {
	t.Setenv(testDBDataSourceEnv, "postgres://alice:secret@db.example.com:5433/custom?sslmode=require")

	cfg, err := testDatabaseConfig("oxygen_test_tmp")
	require.NoError(t, err)

	assert.Equal(t, "db.example.com", cfg.ConnConfig.Host)
	assert.Equal(t, "alice", cfg.ConnConfig.User)
	assert.Equal(t, uint16(5433), cfg.ConnConfig.Port)
	assert.Equal(t, "oxygen_test_tmp", cfg.ConnConfig.Database)
}

func TestTestDatabaseConfig_InvalidEnv(t *testing.T) {
	t.Setenv(testDBDataSourceEnv, "://bad")

	_, err := testDatabaseConfig("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), testDBDataSourceEnv)
}
