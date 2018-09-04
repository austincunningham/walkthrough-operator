package main

import (
	"context"
	"flag"
	"github.com/integr8ly/walkthrough-operator/pkg/apis/integreatly/v1alpha1"
	"os"
	"runtime"
	"time"

	stub "github.com/integr8ly/walkthrough-operator/pkg/stub"
	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	k8sutil "github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	"github.com/sirupsen/logrus"
)

var (
	cfg v1alpha1.Config
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func init() {
	flagset := flag.CommandLine
	flagset.IntVar(&cfg.ResyncPeriod, "resync", 60, "change the resync period")
	flagset.StringVar(&cfg.LogLevel, "log-level", logrus.Level.String(logrus.InfoLevel), "Log level to use. Possible values: panic, fatal, error, warn, info, debug")
	flagset.Parse(os.Args[1:])
}

func main() {
	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logrus.Errorf("Failed to parse log level: %v", err)
	} else {
		logrus.SetLevel(logLevel)
	}
	printVersion()

	resource := "integreatly.aerogear.org/v1alpha1"
	kind := "Walkthrough"
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("Failed to get watch namespace: %v", err)
	}
	resyncDuration := time.Second * time.Duration(cfg.ResyncPeriod)
	logrus.Infof("Watching %s, %s, %s, %d", resource, kind, namespace, resyncDuration)
	sdk.Watch(resource, kind, namespace, resyncDuration)
	sdk.Handle(stub.NewHandler())
	sdk.Run(context.TODO())
}
