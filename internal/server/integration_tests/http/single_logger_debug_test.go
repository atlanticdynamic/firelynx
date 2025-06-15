//go:build integration

package http_test

import (
	_ "embed"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/atlanticdynamic/firelynx/internal/config"
	"github.com/atlanticdynamic/firelynx/internal/config/transaction"
	httplistener "github.com/atlanticdynamic/firelynx/internal/server/runnables/listeners/http"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/orchestrator"
	"github.com/atlanticdynamic/firelynx/internal/server/runnables/txmgr/txstorage"
	"github.com/atlanticdynamic/firelynx/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/single_logger.toml.tmpl
var singleLoggerTemplate string

func TestSingleLoggerFileOutput(t *testing.T) {
	t.Parallel()
	ctx := t.Context()

	tempDir := t.TempDir()
	port := testutil.GetRandomPort(t)
	logFile := filepath.Join(tempDir, "single.log")

	templateVars := struct {
		Port    int
		LogFile string
	}{
		Port:    port,
		LogFile: logFile,
	}

	tmpl, err := template.New("config").Parse(singleLoggerTemplate)
	require.NoError(t, err)

	var configBuffer strings.Builder
	err = tmpl.Execute(&configBuffer, templateVars)
	require.NoError(t, err)

	configData := configBuffer.String()
	t.Logf("Config:\n%s", configData)

	cfg, err := config.NewConfigFromBytes([]byte(configData))
	require.NoError(t, err)
	require.NoError(t, cfg.Validate())

	txStore := txstorage.NewMemoryStorage()
	saga := orchestrator.NewSagaOrchestrator(txStore, slog.Default().Handler())

	httpRunner, err := httplistener.NewRunner()
	require.NoError(t, err)

	err = saga.RegisterParticipant(httpRunner)
	require.NoError(t, err)

	runnerErrCh := make(chan error, 1)
	go func() {
		runnerErrCh <- httpRunner.Run(ctx)
	}()

	require.Eventually(t, func() bool {
		return httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)

	tx, err := transaction.FromTest(t.Name(), cfg, nil)
	require.NoError(t, err)

	err = tx.RunValidation()
	require.NoError(t, err)

	err = saga.ProcessTransaction(ctx, tx)
	require.NoError(t, err)

	require.Equal(t, "completed", tx.GetState())

	require.Eventually(t, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/debug-test", port))
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}, 10*time.Second, 100*time.Millisecond)

	assert.Eventually(t, func() bool {
		info, err := os.Stat(logFile)
		return err == nil && info.Size() > 0
	}, 5*time.Second, 100*time.Millisecond)

	content, err := os.ReadFile(logFile)
	require.NoError(t, err)
	t.Logf("Log content:\n%s", string(content))

	httpRunner.Stop()
	require.Eventually(t, func() bool {
		return !httpRunner.IsRunning()
	}, time.Second, 10*time.Millisecond)
}
