package main_test

import (
	"context"
	"errors"
	"io"
	"os/exec"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/gexec"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
)

type Session interface {
	G() (*gexec.Session, bool)
	Wait() error
}

type sessionWrapper struct {
	g   *gexec.Session
	cmd *exec.Cmd
}

func (s *sessionWrapper) G() (*gexec.Session, bool) {
	if s.g != nil {
		return s.g, true
	}
	return nil, false
}

func (s *sessionWrapper) Wait() error {
	if s == nil {
		return nil
	}
	if s.g != nil {
		ws := s.g.Wait()
		if ws.ExitCode() != 0 {
			return errors.New(string(ws.Err.Contents()))
		}
		return nil
	}
	return s.cmd.Wait()
}

func StartCmd(cmd *exec.Cmd) (Session, error) {
	session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	if err != nil {
		return nil, err
	}
	return &sessionWrapper{
		g:   session,
		cmd: cmd,
	}, nil
}

// nolint:unused
func streamLogs(ctx context.Context, namespace string, podName string) {
	logOptions := &corev1.PodLogOptions{
		Follow: true,
	}

	req := clientSet.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
	lo.Async(
		func() error {
			stream, err := req.Stream(ctx)
			if err != nil {
				return err
			}
			defer stream.Close()
			_, err = io.Copy(ginkgo.GinkgoWriter, stream)
			if err != nil {
				return err
			}
			return nil
		},
	)
}
