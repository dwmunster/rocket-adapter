package rocket

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-joe/joe"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func joeConf(t *testing.T) *joe.Config {
	joeConf := new(joe.Config)
	joeConf.Name = "testname"
	require.NoError(t, joe.WithLogger(zaptest.NewLogger(t)).Apply(joeConf))
	return joeConf
}

func TestDefaultConfig(t *testing.T) {
	conf, err := newConf("fake@email", "password", "url", "botUser", joeConf(t), []Option{})
	require.NoError(t, err)
	assert.NotNil(t, conf.Logger)
	assert.Equal(t, "testname", conf.Name)
}

func TestWithLogger(t *testing.T) {
	logger := zaptest.NewLogger(t)
	conf, err := newConf("fake@email", "password", "url", "botUser", joeConf(t), []Option{
		WithLogger(logger),
	})

	require.NoError(t, err)
	assert.Equal(t, logger, conf.Logger)
}

func TestWithDebug(t *testing.T) {
	conf, err := newConf("fake@email", "password", "url", "botUser", joeConf(t), []Option{
		WithDebug(true),
	})

	require.NoError(t, err)
	assert.Equal(t, true, conf.Debug)

	conf, err = newConf("fake@email", "password", "url", "botUser", joeConf(t), []Option{
		WithDebug(false),
	})

	require.NoError(t, err)
	assert.Equal(t, false, conf.Debug)
}
