package main

import (
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"

	"github.com/retutils/gomitmproxy/cert"
	log "github.com/sirupsen/logrus"
)

// 生成假的/用于测试的服务器证书

type Config struct {
	commonName string
}

func loadConfig() *Config {
	config := new(Config)
	fs := flag.NewFlagSet("dummycert", flag.ExitOnError)
	fs.StringVar(&config.commonName, "commonName", "", "server commonName")
	fs.Parse(os.Args[1:])
	return config
}

func main() {
	config := loadConfig()
	if err := Run(config); err != nil {
		log.Fatal(err)
	}
}

func Run(config *Config) error {
	log.SetLevel(log.InfoLevel)
	log.SetReportCaller(false)
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	if config.commonName == "" {
		return fmt.Errorf("commonName required")
	}

	caApi, err := cert.NewSelfSignCA("")
	if err != nil {
		return err
	}
	ca := caApi.(*cert.SelfSignCA)

	certObj, err := ca.DummyCert(config.commonName)
	if err != nil {
		return err
	}

	os.Stdout.WriteString(fmt.Sprintf("%v-cert.pem\n", config.commonName))
	err = pem.Encode(os.Stdout, &pem.Block{Type: "CERTIFICATE", Bytes: certObj.Certificate[0]})
	if err != nil {
		return err
	}
	os.Stdout.WriteString(fmt.Sprintf("\n%v-key.pem\n", config.commonName))

	keyBytes, err := x509.MarshalPKCS8PrivateKey(&ca.PrivateKey)
	if err != nil {
		return err
	}
	err = pem.Encode(os.Stdout, &pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	if err != nil {
		return err
	}
	return nil
}
