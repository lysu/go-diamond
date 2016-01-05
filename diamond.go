package diamond

import (
	"bufio"
	"fmt"
	"github.com/lysu/httpclient"
	"golang.org/x/net/context"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang/glog"
)

type ConfigWatcher func(cfg string)

type DiamondManager struct {
	dataID     string
	group      string
	watchers   []ConfigWatcher
	subscriber *diamondSubscriber
}

func NewDiamondManager(group string, dataID string, watcher ...ConfigWatcher) (*DiamondManager, error) {

	addressEndpoint := "http://a.b.c:8080/diamond-server/diamond"

	manager := &DiamondManager{
		dataID:   dataID,
		group:    group,
		watchers: []ConfigWatcher{},
	}
	manager.watchers = append(manager.watchers, watcher...)
	curUser, err := user.Current()
	if err != nil {
		return nil, err
	}
	confRoot := filepath.Join(curUser.HomeDir, ".diamond")
	dataFolder := filepath.Join(confRoot, "data")
	err = os.MkdirAll(dataFolder, os.ModePerm)
	if err != nil {
		return nil, err
	}
	client, err := httpclient.NewHTTPClient(2*time.Second, 5*time.Second, 10)
	if err != nil {
		return nil, err
	}
	manager.subscriber = &diamondSubscriber{
		localFileSubscriber: &localFileSubscriber{
			dataFolder: dataFolder,
		},
		serverAddressSubscriber: &serverAddressSubscriber{
			configDir:       confRoot,
			req:             make(chan struct{}),
			resp:            make(chan []string),
			err:             make(chan error),
			client:          client,
			addressEndpoint: addressEndpoint,
		},
	}
	go manager.subscriber.start()
	return manager, nil
}

func (m *DiamondManager) AvailableConfigureInformation(timeout time.Duration) (string, error) {
	return "", nil
}

type diamondSubscriber struct {
	localFileSubscriber     *localFileSubscriber
	serverAddressSubscriber *serverAddressSubscriber
}

func (s *diamondSubscriber) start() {
	s.localFileSubscriber.start()
	go s.serverAddressSubscriber.start()
	s.serverAddressSubscriber.req <- struct{}{}
	s.serverAddressSubscriber.serverAddress = <-s.serverAddressSubscriber.resp
	s.serverAddressSubscriber.storeServerAddressToLocal()
}

type localFileSubscriber struct {
	dataFolder string
}

func (l *localFileSubscriber) start() {

}

type serverAddressSubscriber struct {
	configDir       string
	serverAddress   []string
	req             chan struct{}
	resp            chan []string
	err             chan error
	client          *httpclient.Client
	addressEndpoint string
}

func (s *serverAddressSubscriber) storeServerAddressToLocal() {
	err := os.MkdirAll(s.configDir, os.ModePerm)
	if err != nil {
		glog.Errorf("Create Conf dir %s failure: %v", s.configDir, err)
		return
	}
	serverAddressFile := filepath.Join(s.configDir, "ServerAddress")
	file, err := os.Create(serverAddressFile)
	if err != nil {
		glog.Errorf("Create ServerAddress file %s failure: %v", serverAddressFile, err)
		return
	}
	defer file.Close()
	for _, addr := range s.serverAddress {
		if _, err = file.WriteString(addr + "\n"); err != nil {
			glog.Errorf("Write ServerAddress file %s failure: %v", serverAddressFile, err)
			return
		}
	}
	return
}

func (s *serverAddressSubscriber) start() {
	for {
		select {
		case _ = <-s.req:
			ctx := context.Background()
			err := s.client.Get(ctx, s.addressEndpoint, func(resp *http.Response, err error) error {
				if err != nil {
					return err
				}
				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("Take server address failure with %s", resp.StatusCode)
				}
				domainNameList := make([]string, 0, 2)
				scanner := bufio.NewScanner(resp.Body)
				for scanner.Scan() {
					line := scanner.Text()
					line = strings.TrimSpace(line)
					if line != "" {
						domainNameList = append(domainNameList, line)
					}
				}
				err = scanner.Err()
				if err == io.EOF {
					err = nil
				}
				if err != nil {
					return err
				}
				s.resp <- domainNameList
				return nil
			})
			if err != nil {
				s.err <- err
			}
		}
	}
}

func (s *serverAddressSubscriber) syncAcquireServerAddress() {

}
