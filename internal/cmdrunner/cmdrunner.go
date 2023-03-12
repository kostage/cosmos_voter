package cmdrunner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	gracefulInterval = time.Second * 5
)

var (
	ErrTimedOut = fmt.Errorf("command timed out")
)

//go:generate mockgen -source cmdrunner.go -destination cmdrunner_mock.go -package cmdrunner
type CmdRunner interface {
	Run(context.Context, string, []string, []byte) ([]byte, []byte, error)
}

type cmdRunner struct {
	cmd     *exec.Cmd
	outPipe io.ReadCloser
	errPipe io.ReadCloser
	inPipe  io.WriteCloser
	out     []byte
	err     []byte
}

func NewCmdRunner() *cmdRunner {
	return &cmdRunner{}
}

func (c *cmdRunner) Run(
	ctx context.Context, command string, args []string, input []byte,
) ([]byte, []byte, error) {
	if err := c.start(command, args, (input != nil)); err != nil {
		return nil, nil, err
	}
	var cmdErr error
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		cmdErr = c.wait(ctx)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.readStdOut(); err != nil {
			log.Errorf("command '%s, %v' stdout stream failed: %v", command, args, err)
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.readStdErr(); err != nil {
			log.Errorf("command '%s, %v' stder stream failed: %v", command, args, err)
		}
	}()
	if input != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer c.inPipe.Close()
			if err := c.writeStdin(input); err != nil {
				log.Errorf("command '%s, %v' write stdin failed: %v", command, args, err)
			}
		}()
	}
	wg.Wait()
	return c.out, c.err, cmdErr
}

func (c *cmdRunner) start(
	command string,
	args []string,
	hasInput bool,
) error {
	c.reset()
	var err error
	log.Infof("Running command %s with args %v", command, args)
	c.cmd = exec.Command(command, args...)
	c.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.outPipe, err = c.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	c.errPipe, err = c.cmd.StderrPipe()
	if err != nil {
		return err
	}
	if hasInput {
		c.inPipe, err = c.cmd.StdinPipe()
		if err != nil {
			return err
		}
	}
	// Start defined command
	err = c.cmd.Start()
	if err != nil {
		return err
	}
	return nil
}

func (c *cmdRunner) wait(ctx context.Context) error {
	var cmdErr error
	cmdErrCh := make(chan error)
	cmdWg := sync.WaitGroup{}
	cmdWg.Add(1)
	defer cmdWg.Wait()
	go func() {
		defer cmdWg.Done()
		cmdErrCh <- c.cmd.Wait()
	}()

	cmdFinished := false
	select {
	case cmdErr, cmdFinished = <-cmdErrCh:
		break
	case <-ctx.Done():
		break
	}

	if cmdFinished {
		return cmdErr
	}

	// cmd not finished - we are here because of context
	timeout := (ctx.Err() == context.DeadlineExceeded)
	// terminate cmd gracefully
	c.sendSignal(syscall.SIGTERM)
	// wait result
	select {
	case cmdErr, cmdFinished = <-cmdErrCh:
		break
	case <-time.After(gracefulInterval):
		break
	}

	if cmdFinished {
		if timeout {
			return ErrTimedOut
		} else {
			return cmdErr
		}
	}

	// ok, go bad way
	c.sendSignal(syscall.SIGKILL)
	cmdErr = <-cmdErrCh
	if timeout {
		return ErrTimedOut
	}
	return cmdErr
}

func (c *cmdRunner) writeStdin(in []byte) error {
	if _, err := c.inPipe.Write(append(in, '\n')); err != nil {
		return err
	}
	return nil
}

func (c *cmdRunner) readStdOut() error {
	for {
		buf := make([]byte, 4096)
		n, err := c.outPipe.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		c.out = append(c.out, buf[:n]...)
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return nil
}

func (c *cmdRunner) readStdErr() error {
	for {
		buf := make([]byte, 4096)
		n, err := c.errPipe.Read(buf)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		c.err = append(c.err, buf[:n]...)
		if errors.Is(err, io.EOF) {
			break
		}
	}
	return nil
}

func (c *cmdRunner) sendSignal(sig syscall.Signal) {
	if c.cmd.Process == nil {
		log.Errorf("failed to send signal %v to process: not started", sig)
		return
	}
	if err := syscall.Kill(-c.cmd.Process.Pid, sig); err != nil {
		log.Errorf("failed to send signal %v to process: %v", sig, err)
	}
}

func (c *cmdRunner) reset() {
	c.cmd = nil
	c.outPipe = nil
	c.errPipe = nil
	c.inPipe = nil
	c.out = nil
	c.err = nil
}
