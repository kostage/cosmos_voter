package cmdrunner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCmd_SuccessWithStdoutAndStdErr(t *testing.T) {
	r := NewCmdRunner()
	stdout, stderr, err := r.Run(
		context.Background(),
		"sh", []string{"-c", "echo hello && echo 'darkness, my old friend' >& 2 "},
		nil,
	)
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", string(stdout))
	assert.Equal(t, "darkness, my old friend\n", string(stderr))
}

func TestCmd_FailWithStdoutAndStdErr(t *testing.T) {
	r := NewCmdRunner()
	stdout, stderr, err := r.Run(
		context.Background(),
		"sh", []string{"-c", "echo hello && echo 'darkness, my old friend' >& 2 && exit 1 "},
		nil,
	)
	assert.Error(t, err)
	assert.Equal(t, "exit status 1", err.Error())
	assert.Equal(t, "hello\n", string(stdout))
	assert.Equal(t, "darkness, my old friend\n", string(stderr))
}

func TestCmd_TimeoutWithStdoutAndStdErr(t *testing.T) {
	r := NewCmdRunner()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()
	stdout, stderr, err := r.Run(
		ctx,
		"sh", []string{"-c", "echo hello && sleep 0.1 && echo 'darkness, my old friend' >& 2 && sleep 1 && echo never printed"},
		nil,
	)
	assert.Error(t, err)
	assert.Equal(t, "command timed out", err.Error())
	assert.Equal(t, "hello\n", string(stdout))
	assert.Equal(t, "darkness, my old friend\n", string(stderr))
}

func TestCmd_SuccessWithStdoutAndStdErrAndStdIn(t *testing.T) {
	r := NewCmdRunner()
	stdout, stderr, err := r.Run(
		context.Background(),
		"sh", []string{
			"-c",
			`echo hello && \
			read input && \
			[ $input = password ] && \
			echo 'darkness, my old friend' >& 2 ||\
			exit 1`,
		},
		[]byte("password"),
	)
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", string(stdout))
	assert.Equal(t, "darkness, my old friend\n", string(stderr))
}

func TestCmd_SuccessReuse(t *testing.T) {
	r := NewCmdRunner()
	stdout, stderr, err := r.Run(
		context.Background(),
		"sh", []string{
			"-c",
			`echo hello && \
			read input && \
			[ $input = password ] && \
			echo 'darkness, my old friend' >& 2 ||\
			exit 1`,
		},
		[]byte("password"),
	)
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", string(stdout))
	assert.Equal(t, "darkness, my old friend\n", string(stderr))

	stdout, stderr, err = r.Run(
		context.Background(),
		"sh", []string{
			"-c",
			`echo hello && \
			read input && \
			[ $input = password ] && \
			echo 'darkness, my old friend' >& 2 ||\
			exit 1`,
		},
		[]byte("password"),
	)
	assert.NoError(t, err)
	assert.Equal(t, "hello\n", string(stdout))
	assert.Equal(t, "darkness, my old friend\n", string(stderr))
}
