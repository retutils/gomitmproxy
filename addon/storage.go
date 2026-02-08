package addon

import (
	"github.com/retutils/gomitmproxy/proxy"
	"github.com/retutils/gomitmproxy/storage"
	log "github.com/sirupsen/logrus"
)

type StorageAddon struct {
	proxy.BaseAddon
	Service *storage.Service
}

func NewStorageAddon(storageDir string) (*StorageAddon, error) {
	svc, err := storage.NewService(storageDir)
	if err != nil {
		return nil, err
	}
	return &StorageAddon{
		Service: svc,
	}, nil
}

func (s *StorageAddon) Response(f *proxy.Flow) {
	// Save flow when response is received
	go func() {
		if err := s.Service.Save(f); err != nil {
			log.Errorf("StorageAddon: failed to save flow %s: %v", f.Id, err)
		}
	}()
}

func (s *StorageAddon) Close() {
	if s.Service != nil {
		s.Service.Close()
	}
}
